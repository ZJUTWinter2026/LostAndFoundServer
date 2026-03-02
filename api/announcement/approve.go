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
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func ApproveHandler() gin.HandlerFunc {
	api := ApproveApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfApprove).Pointer()).Name()] = api
	return hfApprove
}

type ApproveApi struct {
	Info     struct{} `name:"审核公告" desc:"系统管理员审核公告"`
	Request  ApproveApiRequest
	Response struct{}
}

type ApproveApiRequest struct {
	Body struct {
		ID      int64 `json:"id" binding:"required" desc:"公告ID"`
		Approve bool  `json:"approve" binding:"required" desc:"true批准 false驳回"`
	}
}

func (a *ApproveApi) Run(ctx *gin.Context) kit.Code {
	reviewerID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	arr := repo.NewAnnouncementRepo()
	announcement, err := arr.FindById(ctx, a.Request.Body.ID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询公告失败")
		return comm.CodeServerError
	}
	if announcement == nil {
		return comm.CodeDataNotFound
	}

	if announcement.Status != "PENDING" {
		return comm.CodePostStatusInvalid
	}

	if a.Request.Body.Approve {
		err = arr.Approve(ctx, a.Request.Body.ID, reviewerID)
	} else {
		err = arr.Reject(ctx, a.Request.Body.ID, reviewerID)
	}
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("审核公告失败")
		return comm.CodeServerError
	}

	return comm.CodeOK
}

func (a *ApproveApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfApprove(ctx *gin.Context) {
	api := &ApproveApi{}
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
