package feedback

import (
	"app/api/admin/system"
	"app/comm"
	"app/dao/model"
	"app/dao/repo"
	"reflect"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"
)

func SubmitHandler() gin.HandlerFunc {
	api := SubmitApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfSubmit).Pointer()).Name()] = api
	return hfSubmit
}

type SubmitApi struct {
	Info     struct{}          `name:"提交投诉反馈" desc:"提交投诉反馈"`
	Request  SubmitApiRequest
	Response SubmitApiResponse
}

type SubmitApiRequest struct {
	Body struct {
		PostID      int64  `json:"post_id" binding:"required" desc:"物品ID"`
		Type        string `json:"type" binding:"required,max=50" desc:"投诉类型"`
		TypeOther   string `json:"type_other" binding:"max=15" desc:"其它类型说明"`
		Description string `json:"description" binding:"omitempty,max=500" desc:"详细说明"`
	}
}

type SubmitApiResponse struct {
	FeedbackID int64 `json:"feedback_id" desc:"投诉反馈ID"`
}

func (s *SubmitApi) Run(ctx *gin.Context) kit.Code {
	request := s.Request.Body

	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	reporterID := cast.ToInt64(id)

	if !system.IsValidFeedbackType(ctx, request.Type) {
		return comm.CodeFeedbackTypeInvalid
	}

	if request.Type == "其它类型" {
		if strings.TrimSpace(request.TypeOther) == "" {
			return comm.CodeFeedbackTypeOther
		}
		if utf8.RuneCountInString(request.TypeOther) > 15 {
			return comm.CodeParameterInvalid
		}
	}

	prp := repo.NewPostRepo()
	post, err := prp.FindById(ctx, request.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询物品失败")
		return comm.CodeDatabaseError
	}
	if post == nil {
		return comm.CodeDataNotFound
	}

	feedback := &model.Feedback{
		PostID:      request.PostID,
		ReporterID:  reporterID,
		Type:        request.Type,
		TypeOther:   request.TypeOther,
		Description: request.Description,
		Processed:   false,
	}

	frp := repo.NewFeedbackRepo()
	err = frp.Create(ctx, feedback)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("创建投诉反馈失败")
		return comm.CodeDatabaseError
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
