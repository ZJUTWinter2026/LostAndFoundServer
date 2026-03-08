package user

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/repo"
	"errors"
	"reflect"
	"runtime"
	"time"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
	"golang.org/x/crypto/bcrypt"
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
		Username string `json:"username" binding:"required" desc:"用户名"`
		Password string `json:"password" binding:"required,min=6,max=18" desc:"密码"`
	}
}

type LoginApiResponse struct {
	NeedUpdate bool   `json:"need_update" desc:"需要修改密码"`
	Id         int64  `json:"id" desc:"用户id"`
	UserType   string `json:"user_type" desc:"用户类型"`
	Name       string `json:"name" desc:"姓名"`
	Campus     string `json:"campus" desc:"校区"`
}

func (l *LoginApi) Run(ctx *gin.Context) kit.Code {
	urp := repo.NewUserRepo()
	request := l.Request.Body

	user, err := urp.FindByUsername(ctx, request.Username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return comm.CodeUserNotExist
	}
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询用户失败")
		return comm.CodeServerError
	}

	if user.DisabledUntil != nil && user.DisabledUntil.After(time.Now()) {
		return comm.CodeUserDisabled
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)) != nil {
		nlog.Pick().WithContext(ctx).Warn("密码错误")
		return comm.CodePasswordError
	}

	err = session.SetIdentity(ctx, user.ID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("session设置失败")
		return comm.CodeTokenError
	}

	needUpdate := user.FirstLogin && user.Usertype == enum.UserTypeStudent

	l.Response = LoginApiResponse{
		NeedUpdate: needUpdate,
		Id:         user.ID,
		UserType:   user.Usertype,
		Name:       user.Name,
		Campus:     user.Campus,
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
