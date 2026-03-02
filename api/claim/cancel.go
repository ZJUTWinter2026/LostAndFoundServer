package claim

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

func CancelHandler() gin.HandlerFunc {
	api := CancelApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfCancel).Pointer()).Name()] = api
	return hfCancel
}

type CancelApi struct {
	Info     struct{}       `name:"取消认领申请" desc:"取消认领申请，只能取消待确认的认领"`
	Request  CancelApiRequest
	Response CancelApiResponse
}

type CancelApiRequest struct {
	Body struct {
		ID int64 `json:"id" binding:"required" desc:"认领申请ID"`
	}
}

type CancelApiResponse struct {
	Success bool `json:"success" desc:"是否成功"`
}

func (c *CancelApi) Run(ctx *gin.Context) kit.Code {
	req := c.Request.Body

	claimantID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	crp := repo.NewClaimRepo()
	claimRecord, err := crp.FindById(ctx, req.ID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询认领申请失败")
		return comm.CodeServerError
	}
	if claimRecord == nil {
		return comm.CodeClaimNotFound
	}

	if claimRecord.ClaimantID != claimantID {
		return comm.CodePermissionDenied
	}

	err = crp.Delete(ctx, req.ID, claimantID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("取消认领申请失败")
		return comm.CodeServerError
	}

	prp := repo.NewPostRepo()
	if err = prp.DecrementClaimCount(ctx, claimRecord.PostID); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("更新认领人数失败")
	}

	c.Response = CancelApiResponse{Success: true}
	return comm.CodeOK
}

func (c *CancelApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&c.Request.Body)
}

func hfCancel(ctx *gin.Context) {
	api := &CancelApi{}
	if err := api.Init(ctx); err != nil {
		reply.Fail(ctx, comm.CodeParameterInvalid)
		return
	}
	code := api.Run(ctx)
	if code == comm.CodeOK {
		reply.Success(ctx, api.Response)
	} else {
		reply.Fail(ctx, code)
	}
}
