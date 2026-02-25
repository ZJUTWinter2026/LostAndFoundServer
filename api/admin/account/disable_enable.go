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

type DisableApi struct {
	Info     struct{}          `name:"禁用用户" desc:"禁用用户账号"`
	Request  DisableApiRequest `name:"禁用用户" desc:"禁用用户账号"`
	Response struct{}
}

type DisableApiRequest struct {
	Body struct {
		ID       int64  `json:"id" binding:"required" desc:"用户ID"`
		Duration string `json:"duration" binding:"required,oneof=7days 1month 6months 1year" desc:"禁用时长"`
	}
}

func (a *DisableApi) Run(ctx *gin.Context) kit.Code {
	if code := checkSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	req := a.Request.Body
	db := ndb.Pick().WithContext(ctx)

	var user model.User
	if err := db.First(&user, req.ID).Error; err != nil {
		return comm.CodeDataNotFound
	}

	var disabledUntil time.Time
	switch req.Duration {
	case "7days":
		disabledUntil = time.Now().AddDate(0, 0, 7)
	case "1month":
		disabledUntil = time.Now().AddDate(0, 1, 0)
	case "6months":
		disabledUntil = time.Now().AddDate(0, 6, 0)
	case "1year":
		disabledUntil = time.Now().AddDate(1, 0, 0)
	}

	if err := db.Model(&user).Updates(map[string]interface{}{
		"disabled_until": disabledUntil,
	}).Error; err != nil {
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

func DisableHandler() gin.HandlerFunc {
	api := DisableApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfDisable).Pointer()).Name()] = api
	return hfDisable
}

type EnableApi struct {
	Info     struct{}         `name:"恢复用户" desc:"恢复用户账号"`
	Request  EnableApiRequest `name:"恢复用户" desc:"恢复用户账号"`
	Response struct{}
}

type EnableApiRequest struct {
	Body struct {
		ID int64 `json:"id" binding:"required" desc:"用户ID"`
	}
}

func (a *EnableApi) Run(ctx *gin.Context) kit.Code {
	if code := checkSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	req := a.Request.Body
	db := ndb.Pick().WithContext(ctx)

	var user model.User
	if err := db.First(&user, req.ID).Error; err != nil {
		return comm.CodeDataNotFound
	}

	if err := db.Model(&user).Updates(map[string]interface{}{
		"disabled_until": nil,
	}).Error; err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("恢复用户失败")
		return comm.CodeServerError
	}

	return comm.CodeOK
}

func (a *EnableApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfEnable(ctx *gin.Context) {
	api := &EnableApi{}
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

func EnableHandler() gin.HandlerFunc {
	api := EnableApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfEnable).Pointer()).Name()] = api
	return hfEnable
}
