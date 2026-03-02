package account

import (
	"app/comm"
	"app/dao/model"
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/ndb"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"
)

func DisableHandler() gin.HandlerFunc {
	api := DisableApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfDisable).Pointer()).Name()] = api
	return hfDisable
}

type DisableApi struct {
	Info     struct{} `name:"禁用用户" desc:"禁用用户账号"`
	Request  DisableApiRequest
	Response struct{}
}

type DisableApiRequest struct {
	Body struct {
		ID       int64  `json:"id" binding:"required" desc:"用户ID"`
		Duration string `json:"duration" binding:"required,oneof=7days 1month 6months 1year" desc:"禁用时长"`
	}
}

func (a *DisableApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	req := a.Request.Body
	db := ndb.Pick().WithContext(ctx)

	var user model.User
	if err := db.First(&user, req.ID).Error; err != nil {
		return comm.CodeDataNotFound
	}

	switch req.Duration {
	case "7days":
		user.DisabledUntil = time.Now().AddDate(0, 0, 7)
	case "1month":
		user.DisabledUntil = time.Now().AddDate(0, 1, 0)
	case "6months":
		user.DisabledUntil = time.Now().AddDate(0, 6, 0)
	case "1year":
		user.DisabledUntil = time.Now().AddDate(1, 0, 0)
	}

	if err := db.Save(&user).Error; err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("禁用用户失败")
		return comm.CodeServerError
	}
	return comm.CodeOK
}

func (a *DisableApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfDisable(ctx *gin.Context) {
	api := &DisableApi{}
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
