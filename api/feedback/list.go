package feedback

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/model"
	"app/dao/repo"
	"errors"
	"reflect"
	"runtime"
	"time"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func ListHandler() gin.HandlerFunc {
	api := ListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfList).Pointer()).Name()] = api
	return hfList
}

type ListApi struct {
	Info     struct{} `name:"投诉反馈列表" desc:"投诉反馈列表"`
	Request  ListApiRequest
	Response ListApiResponse
}

type ListApiRequest struct {
	Query struct {
		Processed string `form:"processed" binding:"required,oneof=ALL YES NO" desc:"是否已处理"`
		Page      int    `form:"page" binding:"required,min=1" desc:"页码"`
		PageSize  int    `form:"page_size" binding:"required,min=1,max=50" desc:"每页数量"`
	}
}

type ListApiResponse struct {
	Total    int64          `json:"total" desc:"总数"`
	Page     int            `json:"page" desc:"页码"`
	PageSize int            `json:"page_size" desc:"每页数量"`
	List     []FeedbackItem `json:"list" desc:"投诉反馈列表"`
}

type FeedbackItem struct {
	ID          int64      `json:"id" desc:"投诉ID"`
	PostID      int64      `json:"post_id" desc:"物品ID"`
	ReporterID  int64      `json:"reporter_id" desc:"投诉者ID"`
	Type        string     `json:"type" desc:"投诉类型"`
	Description string     `json:"description" desc:"详细说明"`
	Processed   bool       `json:"processed" desc:"是否已处理"`
	ProcessedBy int64      `json:"processed_by" desc:"处理人ID"`
	ProcessedAt *time.Time `json:"processed_at,omitempty" desc:"处理时间"`
	CreatedAt   time.Time  `json:"created_at" desc:"创建时间"`
}

func (l *ListApi) Run(ctx *gin.Context) kit.Code {
	request := l.Request.Query

	adminID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	urp := repo.NewUserRepo()
	user, err := urp.FindById(ctx, adminID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return comm.CodeAdminPermissionDenied
	}
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询用户失败")
		return comm.CodeServerError
	}
	if user == nil || (user.Usertype != enum.UserTypeAdmin && user.Usertype != enum.UserTypeSystemAdmin) {
		return comm.CodeAdminPermissionDenied
	}

	page := request.Page
	pageSize := request.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize
	frp := repo.NewFeedbackRepo()

	var feedbacks []*model.Feedback
	var total int64

	if request.Processed != "ALL" {
		feedbacks, total, err = frp.ListByProcessed(ctx, request.Processed, offset, pageSize)
	} else {
		feedbacks, total, err = frp.ListAll(ctx, offset, pageSize)
	}

	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询投诉反馈列表失败")
		return comm.CodeServerError
	}

	items := make([]FeedbackItem, 0, len(feedbacks))
	for _, fb := range feedbacks {
		item := FeedbackItem{
			ID:          fb.ID,
			PostID:      fb.PostID,
			ReporterID:  fb.ReporterID,
			Type:        fb.Type,
			Description: fb.Description,
			Processed:   fb.Processed,
			ProcessedBy: fb.ProcessedBy,
			ProcessedAt: fb.ProcessedAt,
			CreatedAt:   fb.CreatedAt,
		}
		items = append(items, item)
	}

	l.Response = ListApiResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}
	return comm.CodeOK
}

func (l *ListApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindQuery(&l.Request.Query)
}

func hfList(ctx *gin.Context) {
	api := &ListApi{}
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
