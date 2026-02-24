package user

import (
	"app/dao/repo"
	"fmt"
	"github.com/zjutjh/mygo/jwt"
	"reflect"
	"runtime"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
)

// LoginHandler API router注册点
func LoginHandler() gin.HandlerFunc {
	api := LoginApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfLogin).Pointer()).Name()] = api
	return hfLogin
}

type LoginApi struct {
	Info     struct{}         `name:"登陆" desc:"登陆"`
	Request  LoginApiRequest  // API请求参数 (Body/Header/Body/Body)
	Response LoginApiResponse // API响应数据 (Body中的Data部分)
}

type LoginApiRequest struct {
	Body struct {
		Uid      int64  `json:"uid" binding:"required" desc:"学号"`
		Password string `json:"password" binding:"required,min=6,max=18" desc:"密码"`
	}
}

type LoginApiResponse struct {
	NeedUpdate int8   `json:"need_update" binding:"required" desc:"需要修改密码"`
	Id         int64  `json:"id" binding:"required" desc:"用户id"`
	UserType   int8   `json:"user_type" binding:"required" desc:"用户类型"`
	Token      string `json:"token" binding:"required" desc:"token"`
}

// Run Api业务逻辑执行点
func (l *LoginApi) Run(ctx *gin.Context) kit.Code {
	urp := repo.NewUserRepo()
	request := l.Request.Body

	user, err := urp.FindByUid(ctx, request.Uid)
	if err != nil {
		return comm.CodeDatabaseError
	}
	hash, _ := comm.HashPassword("123456")
	fmt.Println(hash)
	if user == nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("用户不存在")
		return comm.CodeUserNotExist
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
		UserType:   int8(user.Usertype),
		Token:      token,
	}
	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (l *LoginApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindJSON(&l.Request.Body)
	if err != nil {
		return err
	}
	return err
}

// hfLogin API执行入口
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
