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

func EnableHandler() gin.HandlerFunc {
	api := EnableApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfEnable).Pointer()).Name()] = api
	return hfEnable
}

type EnableApi struct {
	Info     struct{} `name:"恢复用户" desc:"恢复用户账号"`
	Request  EnableApiRequest
	Response struct{}
}

type EnableApiRequest struct {
	Body struct {
		ID int64 `json:"id" binding:"required" desc:"用户ID"`
	}
}

func (a *EnableApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	req := a.Request.Body
	db := ndb.Pick().WithContext(ctx)

	var user model.User
	if err := db.First(&user, req.ID).Error; err != nil {
		return comm.CodeDataNotFound
	}

	if err := db.Model(&user).
		Update("disabled_until", nil).Error; err != nil {
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
