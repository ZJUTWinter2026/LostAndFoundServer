package user

import (
	"app/comm"
	"app/dao/repo"
	"reflect"
	"runtime"
	"strconv"
	"time"

	"github.com/zjutjh/mygo/jwt"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"
)

func LoginHandler() gin.HandlerFunc {
	api := LoginApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfLogin).Pointer()).Name()] = api
	return hfLogin
}

type LoginApi struct {
	Info     struct{} `name:"登陆" desc:"登陆"`
	Request  LoginApiRequest
	Response LoginApiResponse
}

type LoginApiRequest struct {
	Body struct {
		Uid      int64  `json:"uid" binding:"required" desc:"学号"`
		Password string `json:"password" binding:"required,min=6,max=18" desc:"密码"`
	}
}

type LoginApiResponse struct {
	NeedUpdate bool   `json:"need_update" binding:"required" desc:"需要修改密码"`
	Id         int64  `json:"id" binding:"required" desc:"用户id"`
	UserType   string `json:"user_type" binding:"required" desc:"用户类型"`
	Token      string `json:"token" binding:"required" desc:"token"`
}

func (l *LoginApi) Run(ctx *gin.Context) kit.Code {
	urp := repo.NewUserRepo()
	request := l.Request.Body

	user, err := urp.FindByUid(ctx, request.Uid)
	if err != nil {
		return comm.CodeDatabaseError
	}
	if user == nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("用户不存在")
		return comm.CodeUserNotExist
	}

	if !user.DisabledUntil.IsZero() && user.DisabledUntil.After(time.Now()) {
		return comm.CodeUserDisabled
	}

	if !comm.CheckPassword(user.Password, request.Password) {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("密码错误")
		return comm.CodePasswordError
	}
	token, err := jwt.Pick[string]().GenerateToken(strconv.FormatInt(user.ID, 10))
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("token生成失败")
		return comm.CodeTokenError
	}
	l.Response = LoginApiResponse{
		NeedUpdate: user.FirstLogin,
		Id:         user.ID,
		UserType:   user.Usertype,
		Token:      token,
	}
	return comm.CodeOK
}

func (l *LoginApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindJSON(&l.Request.Body)
	if err != nil {
		return err
	}
	return err
}

func hfLogin(ctx *gin.Context) {
	api := &LoginApi{}
	err := api.Init(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("参数绑定校验错误")
		reply.Fail(ctx, comm.CodeParameterInvalid)
		return
	}
	code := api.Run(ctx)
	if !ctx.IsAborted() {
		if code == comm.CodeOK {
			reply.Success(ctx, api.Response)
		} else {
			reply.Fail(ctx, code)
		}
	}
}
