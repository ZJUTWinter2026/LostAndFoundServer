package user

import (
	"app/comm"
	"app/dao/repo"
	"errors"
	"reflect"
	"runtime"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
	"golang.org/x/crypto/bcrypt"
)

func UpdateHandler() gin.HandlerFunc {
	api := UpdateApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfUpdate).Pointer()).Name()] = api
	return hfUpdate
}

type UpdateApi struct {
	Info     struct{} `name:"修改密码" desc:"修改密码"`
	Request  UpdateApiRequest
	Response UpdateApiResponse
}

type UpdateApiRequest struct {
	Body struct {
		OldPassword string `json:"old_password" binding:"required,min=6,max=18" desc:"原密码"`
		NewPassword string `json:"new_password" binding:"required,min=6,max=12" desc:"新密码"`
	}
}

type UpdateApiResponse struct {
	Token string `json:"token" desc:"token"`
}

func (u *UpdateApi) Run(ctx *gin.Context) kit.Code {
	urp := repo.NewUserRepo()
	request := u.Request.Body

	uid, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	user, err := urp.FindById(ctx, uid)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return comm.CodeUserNotExist
	}
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询用户失败")
		return comm.CodeServerError
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.OldPassword)) != nil {
		return comm.CodePasswordError
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("新密码加密失败")
		return comm.CodeHashError
	}

	user.Password = string(newHash)
	user.FirstLogin = false

	if err := urp.Save(ctx, user); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("更新密码失败")
		return comm.CodeServerError
	}

	u.Response.Token = "logged_in"
	return comm.CodeOK
}

func (u *UpdateApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindJSON(&u.Request.Body)
}

func hfUpdate(ctx *gin.Context) {
	api := &UpdateApi{}
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
