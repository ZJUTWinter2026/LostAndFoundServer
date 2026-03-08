package feedback

import (
	"app/comm"
	"app/dao/model"
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
)

func SubmitHandler() gin.HandlerFunc {
	api := SubmitApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfSubmit).Pointer()).Name()] = api
	return hfSubmit
}

type SubmitApi struct {
	Info     struct{} `name:"提交投诉反馈" desc:"提交投诉反馈"`
	Request  SubmitApiRequest
	Response SubmitApiResponse
}

type SubmitApiRequest struct {
	Body struct {
		PostID      int64  `json:"post_id" binding:"required" desc:"物品ID"`
		Type        string `json:"type" binding:"required,max=50" desc:"投诉类型"`
		Description string `json:"description" binding:"omitempty,max=500" desc:"详细说明"`
	}
}

type SubmitApiResponse struct {
	FeedbackID int64 `json:"feedback_id" desc:"投诉反馈ID"`
}

func (s *SubmitApi) Run(ctx *gin.Context) kit.Code {
	request := s.Request.Body

	reporterID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	scr := repo.NewSystemConfigRepo()
	if !scr.IsValidFeedbackType(ctx, request.Type) {
		return comm.CodeFeedbackTypeInvalid
	}

	prp := repo.NewPostRepo()
	_, err = prp.FindById(ctx, request.PostID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return comm.CodeDataNotFound
	}
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询物品失败")
		return comm.CodeServerError
	}

	feedback := &model.Feedback{
		PostID:      request.PostID,
		ReporterID:  reporterID,
		Type:        request.Type,
		Description: request.Description,
		Processed:   false,
	}

	frp := repo.NewFeedbackRepo()
	err = frp.Create(ctx, feedback)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("创建投诉反馈失败")
		return comm.CodeServerError
	}

	s.Response = SubmitApiResponse{FeedbackID: feedback.ID}
	return comm.CodeOK
}

func (s *SubmitApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindJSON(&s.Request.Body)
}

func hfSubmit(ctx *gin.Context) {
	api := &SubmitApi{}
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
