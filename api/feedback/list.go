package feedback

import (
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/dao/model"
	"app/dao/repo"
)

// ListHandler API router注册点
func ListHandler() gin.HandlerFunc {
	api := ListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfList).Pointer()).Name()] = api
	return hfList
}

type ListApi struct {
	Info     struct{}        `name:"投诉反馈列表" desc:"投诉反馈列表"`
	Request  ListApiRequest  // API请求参数
	Response ListApiResponse // API响应数据
}

type ListApiRequest struct {
	Query struct {
		Status   *int8 `form:"status" binding:"omitempty,oneof=0 1" desc:"状态 0未处理 1已处理"`
		Page     int   `form:"page" binding:"omitempty,min=1" desc:"页码"`
		PageSize int   `form:"page_size" binding:"omitempty,min=1,max=50" desc:"每页数量"`
	}
}

type ListApiResponse struct {
	Total    int64          `json:"total" desc:"总数"`
	Page     int            `json:"page" desc:"页码"`
	PageSize int            `json:"page_size" desc:"每页数量"`
	List     []FeedbackItem `json:"list" desc:"投诉反馈列表"`
}

type FeedbackItem struct {
	ID          int64     `json:"id" desc:"投诉ID"`
	PostID      int64     `json:"post_id" desc:"物品ID"`
	ReporterID  int64     `json:"reporter_id" desc:"投诉者ID"`
	Type        string    `json:"type" desc:"投诉类型"`
	TypeOther   string    `json:"type_other" desc:"其它类型说明"`
	Description string    `json:"description" desc:"详细说明"`
	Status      int8      `json:"status" desc:"状态 0未处理 1已处理"`
	ProcessedBy int64     `json:"processed_by,omitempty" desc:"处理人ID"`
	ProcessedAt time.Time `json:"processed_at,omitempty" desc:"处理时间"`
	CreatedAt   time.Time `json:"created_at" desc:"创建时间"`
}

// Run Api业务逻辑执行点
func (l *ListApi) Run(ctx *gin.Context) kit.Code {
	request := l.Request.Query

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
	var err error

	if request.Status != nil {
		feedbacks, total, err = frp.ListByStatus(ctx, *request.Status, offset, pageSize)
	} else {
		feedbacks, total, err = frp.ListAll(ctx, offset, pageSize)
	}

	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询投诉反馈列表失败")
		return comm.CodeDatabaseError
	}

	items := make([]FeedbackItem, 0, len(feedbacks))
	for _, fb := range feedbacks {
		items = append(items, FeedbackItem{
			ID:          fb.ID,
			PostID:      fb.PostID,
			ReporterID:  fb.ReporterID,
			Type:        fb.Type,
			TypeOther:   fb.TypeOther,
			Description: fb.Description,
			Status:      fb.Status,
			ProcessedBy: fb.ProcessedBy,
			ProcessedAt: fb.ProcessedAt,
			CreatedAt:   fb.CreatedAt,
		})
	}

	l.Response = ListApiResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}
	return comm.CodeOK
}

// Init Api初始化
func (l *ListApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindQuery(&l.Request.Query)
}

// hfList API执行入口
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
