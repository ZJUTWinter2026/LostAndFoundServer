package user

import (
	"app/dao/repo"
	"github.com/spf13/cast"
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

// UpdateHandler API router注册点
func UpdateHandler() gin.HandlerFunc {
	api := UpdateApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfUpdate).Pointer()).Name()] = api
	return hfUpdate
}

type UpdateApi struct {
	Info     struct{}          `name:"修改密码" desc:"修改密码"`
	Request  UpdateApiRequest  // API请求参数 (Body/Header/Body/Body)
	Response UpdateApiResponse // API响应数据 (Body中的Data部分)
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

// Run Api业务逻辑执行点
func (u *UpdateApi) Run(ctx *gin.Context) kit.Code {
	urp := repo.NewUserRepo()
	request := u.Request.Body

	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	uid := cast.ToInt64(id)

	user, err := urp.FindById(ctx, uid)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询用户失败")
		return comm.CodeDatabaseError
	}
	if user == nil {
		return comm.CodeUserNotExist
	}

	if !comm.CheckPassword(user.Password, request.OldPassword) {
		return comm.CodePasswordError
	}

	newHash, err := comm.HashPassword(request.NewPassword)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("新密码加密失败")
		return comm.CodeHashError
	}

	err = urp.UpdatePassword(ctx, uid, newHash)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("更新密码失败")
		return comm.CodeDatabaseError
	}

	err = urp.UpdateFirstLogin(ctx, uid)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("更新首次登录状态失败")
		return comm.CodeDatabaseError
	}

	token, err := jwt.Pick[string]().GenerateToken(strconv.FormatInt(uid, 10))
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("token生成失败")
		return comm.CodeTokenError
	}
	u.Response.Token = token
	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (u *UpdateApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindJSON(&u.Request.Body)
	if err != nil {
		return err
	}
	return err
}

// hfUpdate API执行入口
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
