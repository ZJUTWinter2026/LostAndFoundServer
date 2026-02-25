package account

import (
	"app/comm"
	"app/dao/model"
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/ndb"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"
)

// UpdateHandler 更新账号信息
func UpdateHandler() gin.HandlerFunc {
	api := UpdateApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfUpdate).Pointer()).Name()] = api
	return hfUpdate
}

type UpdateApi struct {
	Info     struct{}         `name:"更新账号信息" desc:"更新账号信息(权限/重置密码)"`
	Request  UpdateApiRequest // API请求参数
	Response struct{}
}

type UpdateApiRequest struct {
	Body struct {
		ID            int64  `json:"id" binding:"required" desc:"用户ID"`
		UserType      string `json:"user_type" binding:"oneof=STUDENT ADMIN SYSTEM_ADMIN" desc:"用户类型"`
		ResetPassword bool   `json:"reset_password" desc:"是否重置密码(重置为身份证后六位/默认密码)"`
	}
}

func (a *UpdateApi) Run(ctx *gin.Context) kit.Code {
	if code := checkSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	req := a.Request.Body
	db := ndb.Pick().WithContext(ctx)

	var user model.User
	if err := db.First(&user, req.ID).Error; err != nil {
		return comm.CodeDataNotFound
	}

	updates := make(map[string]interface{})
	if req.UserType != "" {
		updates["usertype"] = req.UserType
	}
	if req.ResetPassword {
		hashedPwd, err := comm.HashPassword("123456")
		if err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("密码加密失败")
			return comm.CodeHashError
		}
		updates["password"] = hashedPwd
		updates["first_login"] = 1
	}

	if len(updates) > 0 {
		if err := db.Model(&user).Updates(updates).Error; err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("更新用户信息失败")
			return comm.CodeServerError
		}
	}

	return comm.CodeOK
}

func (a *UpdateApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfUpdate(ctx *gin.Context) {
	api := &UpdateApi{}
	if err := api.Init(ctx); err != nil {
		reply.Fail(ctx, comm.CodeParameterInvalid)
		return
	}
	code := api.Run(ctx)
	if code == comm.CodeOK {
		reply.Success(ctx, struct{}{})
	} else {
		reply.Fail(ctx, code)
	}
}
