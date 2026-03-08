package account

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/model"
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/ndb"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"
	"golang.org/x/crypto/bcrypt"
)

func ResetPasswordHandler() gin.HandlerFunc {
	api := ResetPasswordApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfResetPassword).Pointer()).Name()] = api
	return hfResetPassword
}

type ResetPasswordApi struct {
	Info     struct{} `name:"重置密码" desc:"系统管理员重置用户密码"`
	Request  ResetPasswordApiRequest
	Response ResetPasswordApiResponse
}

type ResetPasswordApiRequest struct {
	Body struct {
		ID int64 `json:"id" binding:"required" desc:"用户ID"`
	}
}

type ResetPasswordApiResponse struct {
	Result string `json:"result" desc:"重置的密码"`
}

func (a *ResetPasswordApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	req := a.Request.Body
	db := ndb.Pick().WithContext(ctx)

	var user model.User
	if err := db.First(&user, req.ID).Error; err != nil {
		return comm.CodeDataNotFound
	}

	password := "123456"
	if len(user.IDCard) >= 6 {
		password = user.IDCard[len(user.IDCard)-6:]
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("密码加密失败")
		return comm.CodeHashError
	}

	user.Password = string(hashedPwd)
	user.FirstLogin = user.Usertype == enum.UserTypeStudent

	if err := db.Save(&user).Error; err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("重置用户密码失败")
		return comm.CodeServerError
	}

	a.Response = ResetPasswordApiResponse{
		Result: password,
	}
	return comm.CodeOK
}

func (a *ResetPasswordApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfResetPassword(ctx *gin.Context) {
	api := &ResetPasswordApi{}
	err := api.Init(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("参数绑定校验错误")
		reply.Fail(ctx, comm.CodeParameterInvalid)
		return
	}
	code := api.Run(ctx)
	if !ctx.IsAborted() {
		if code == comm.CodeOK {
			reply.Success(ctx, struct{}{})
		} else {
			reply.Fail(ctx, code)
		}
	}
}
