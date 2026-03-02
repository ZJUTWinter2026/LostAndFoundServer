package user

import (
	"app/comm"
	"app/dao/repo"
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"
	"golang.org/x/crypto/bcrypt"
)

func ForgotPasswordHandler() gin.HandlerFunc {
	api := ForgotPasswordApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfForgotPassword).Pointer()).Name()] = api
	return hfForgotPassword
}

type ForgotPasswordApi struct {
	Info     struct{} `name:"忘记密码" desc:"通过用户名和身份证验证重置密码"`
	Request  ForgotPasswordApiRequest
	Response ForgotPasswordApiResponse
}

type ForgotPasswordApiRequest struct {
	Body struct {
		Username string `json:"username" binding:"required" desc:"用户名"`
		IDCard   string `json:"id_card" binding:"required,len=18" desc:"身份证号"`
	}
}

type ForgotPasswordApiResponse struct {
	Success bool `json:"success" desc:"是否成功"`
}

func (f *ForgotPasswordApi) Run(ctx *gin.Context) kit.Code {
	request := f.Request.Body

	urp := repo.NewUserRepo()
	user, err := urp.FindByUsernameAndIDCard(ctx, request.Username, request.IDCard)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询用户失败")
		return comm.CodeServerError
	}
	if user == nil {
		return comm.CodeUserNotExist
	}

	if len(user.IDCard) < 6 {
		return comm.CodeServerError
	}

	newPassword := user.IDCard[len(user.IDCard)-6:]
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("密码加密失败")
		return comm.CodeHashError
	}

	user.Password = string(newHash)
	user.FirstLogin = true

	if err := urp.Save(ctx, user); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("重置密码失败")
		return comm.CodeServerError
	}

	f.Response = ForgotPasswordApiResponse{Success: true}
	return comm.CodeOK
}

func (f *ForgotPasswordApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindJSON(&f.Request.Body)
}

func hfForgotPassword(ctx *gin.Context) {
	api := &ForgotPasswordApi{}
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
