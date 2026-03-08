package announcement

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
)

func DeleteHandler() gin.HandlerFunc {
	api := DeleteApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfDelete).Pointer()).Name()] = api
	return hfDelete
}

type DeleteApi struct {
	Info     struct{} `name:"删除公告" desc:"系统管理员删除公告"`
	Request  DeleteApiRequest
	Response struct{}
}

type DeleteApiRequest struct {
	Body struct {
		ID int64 `json:"id" binding:"required" desc:"公告ID"`
	}
}

func (a *DeleteApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	arr := repo.NewAnnouncementRepo()
	err := arr.Delete(ctx, a.Request.Body.ID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("删除公告失败")
		return comm.CodeServerError
	}

	return comm.CodeOK
}

func (a *DeleteApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfDelete(ctx *gin.Context) {
	api := &DeleteApi{}
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
