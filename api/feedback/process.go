package feedback

import (
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/comm/enum"
	"app/dao/repo"
)

func ProcessHandler() gin.HandlerFunc {
	api := ProcessApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfProcess).Pointer()).Name()] = api
	return hfProcess
}

type ProcessApi struct {
	Info     struct{}           `name:"处理投诉反馈" desc:"处理投诉反馈"`
	Request  ProcessApiRequest
	Response ProcessApiResponse
}

type ProcessApiRequest struct {
	Body struct {
		FeedbackID int64 `json:"feedback_id" binding:"required" desc:"投诉ID"`
	}
}

type ProcessApiResponse struct {
	Success bool `json:"success" desc:"是否成功"`
}

func (p *ProcessApi) Run(ctx *gin.Context) kit.Code {
	request := p.Request.Body

	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	processorID := cast.ToInt64(id)

	urp := repo.NewUserRepo()
	user, err := urp.FindById(ctx, processorID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询用户失败")
		return comm.CodeDatabaseError
	}
	if user == nil || user.Usertype != enum.UserTypeAdmin {
		return comm.CodePermissionDenied
	}

	frp := repo.NewFeedbackRepo()
	feedback, err := frp.FindById(ctx, request.FeedbackID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询投诉记录失败")
		return comm.CodeDatabaseError
	}
	if feedback == nil {
		return comm.CodeDataNotFound
	}

	if feedback.Processed {
		return comm.CodeParameterInvalid
	}

	err = frp.MarkAsProcessed(ctx, request.FeedbackID, processorID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("处理投诉反馈失败")
		return comm.CodeDatabaseError
	}

	p.Response = ProcessApiResponse{Success: true}
	return comm.CodeOK
}

func (p *ProcessApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindJSON(&p.Request.Body)
}

func hfProcess(ctx *gin.Context) {
	api := &ProcessApi{}
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
