package feedback

import (
	"app/comm"
	"app/dao/model"
	"app/dao/repo"
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func MyListHandler() gin.HandlerFunc {
	api := MyListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfMyList).Pointer()).Name()] = api
	return hfMyList
}

type MyListApi struct {
	Info     struct{} `name:"我的投诉列表" desc:"我的投诉列表"`
	Request  MyListApiRequest
	Response MyListApiResponse
}

type MyListApiRequest struct {
	Query struct {
		Processed string `form:"processed" binding:"required,oneof=ALL YES NO" desc:"是否已处理"`
		Page      int    `form:"page" binding:"required,min=1" desc:"页码"`
		PageSize  int    `form:"page_size" binding:"required,min=1,max=50" desc:"每页数量"`
	}
}

type MyListApiResponse struct {
	Total    int64            `json:"total" desc:"总数"`
	Page     int              `json:"page" desc:"页码"`
	PageSize int              `json:"page_size" desc:"每页数量"`
	List     []MyFeedbackItem `json:"list" desc:"我的投诉列表"`
}

type MyFeedbackItem struct {
	ID          int64     `json:"id" desc:"投诉ID"`
	PostID      int64     `json:"post_id" desc:"物品ID"`
	Type        string    `json:"type" desc:"投诉类型"`
	Description string    `json:"description" desc:"详细说明"`
	Processed   bool      `json:"processed" desc:"是否已处理"`
	ProcessedAt time.Time `json:"processed_at" desc:"处理时间"`
	CreatedAt   time.Time `json:"created_at" desc:"创建时间"`
}

func (m *MyListApi) Run(ctx *gin.Context) kit.Code {
	request := m.Request.Query

	reporterID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
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
		feedbacks, total, err = frp.ListByReporterAndProcessed(ctx, reporterID, request.Processed, offset, pageSize)
	} else {
		feedbacks, total, err = frp.ListByReporter(ctx, reporterID, offset, pageSize)
	}

	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询我的投诉列表失败")
		return comm.CodeServerError
	}

	items := make([]MyFeedbackItem, 0, len(feedbacks))
	for _, fb := range feedbacks {
		item := MyFeedbackItem{
			ID:          fb.ID,
			PostID:      fb.PostID,
			Type:        fb.Type,
			Description: fb.Description,
			Processed:   fb.Processed,
			ProcessedAt: *fb.ProcessedAt,
			CreatedAt:   fb.CreatedAt,
		}
		items = append(items, item)
	}

	m.Response = MyListApiResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}
	return comm.CodeOK
}

func (m *MyListApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindQuery(&m.Request.Query)
}

func hfMyList(ctx *gin.Context) {
	api := &MyListApi{}
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
