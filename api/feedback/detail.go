package feedback

import (
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/comm/enum"
	"app/dao/repo"
)

func DetailHandler() gin.HandlerFunc {
	api := DetailApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfDetail).Pointer()).Name()] = api
	return hfDetail
}

type DetailApi struct {
	Info     struct{} `name:"投诉反馈详情" desc:"系统管理员查看投诉反馈详情"`
	Request  DetailApiRequest
	Response DetailApiResponse
}

type DetailApiRequest struct {
	Query struct {
		ID int64 `form:"id" binding:"required" desc:"投诉反馈ID"`
	}
}

type DetailApiResponse struct {
	ID          int64                 `json:"id" desc:"投诉ID"`
	PostID      int64                 `json:"post_id" desc:"物品ID"`
	ReporterID  int64                 `json:"reporter_id" desc:"投诉者ID"`
	Type        string                `json:"type" desc:"投诉类型"`
	Description string                `json:"description" desc:"详细说明"`
	Processed   bool                  `json:"processed" desc:"是否已处理"`
	ProcessedBy int64                 `json:"processed_by" desc:"处理人ID"`
	ProcessedAt time.Time             `json:"processed_at" desc:"处理时间"`
	CreatedAt   time.Time             `json:"created_at" desc:"创建时间"`
	Post        *PostDetailInFeedback `json:"post,omitempty" desc:"关联物品信息"`
}

type PostDetailInFeedback struct {
	ID          int64     `json:"id"`
	ItemName    string    `json:"item_name"`
	ItemType    string    `json:"item_type"`
	Campus      string    `json:"campus"`
	Location    string    `json:"location"`
	Status      string    `json:"status"`
	PublisherID int64     `json:"publisher_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func (d *DetailApi) Run(ctx *gin.Context) kit.Code {
	adminID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	urp := repo.NewUserRepo()
	user, err := urp.FindById(ctx, adminID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询用户失败")
		return comm.CodeServerError
	}
	if user == nil || (user.Usertype != enum.UserTypeAdmin && user.Usertype != enum.UserTypeSystemAdmin) {
		return comm.CodeAdminPermissionDenied
	}

	frp := repo.NewFeedbackRepo()
	feedback, err := frp.FindById(ctx, d.Request.Query.ID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询投诉反馈失败")
		return comm.CodeServerError
	}
	if feedback == nil {
		return comm.CodeDataNotFound
	}

	response := DetailApiResponse{
		ID:          feedback.ID,
		PostID:      feedback.PostID,
		ReporterID:  feedback.ReporterID,
		Type:        feedback.Type,
		Description: feedback.Description,
		Processed:   feedback.Processed,
		ProcessedBy: feedback.ProcessedBy,
		ProcessedAt: *feedback.ProcessedAt,
		CreatedAt:   feedback.CreatedAt,
	}

	prp := repo.NewPostRepo()
	post, err := prp.FindById(ctx, feedback.PostID)
	if err == nil && post != nil {
		response.Post = &PostDetailInFeedback{
			ID:          post.ID,
			ItemName:    post.ItemName,
			ItemType:    post.ItemType,
			Campus:      post.Campus,
			Location:    post.Location,
			Status:      post.Status,
			PublisherID: post.PublisherID,
			CreatedAt:   post.CreatedAt,
		}
	}

	d.Response = response
	return comm.CodeOK
}

func (d *DetailApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindQuery(&d.Request.Query)
}

func hfDetail(ctx *gin.Context) {
	api := &DetailApi{}
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
