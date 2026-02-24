package feedback

import (
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/dao/model"
	"app/dao/repo"
)

// MyListHandler API router注册点
func MyListHandler() gin.HandlerFunc {
	api := MyListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfMyList).Pointer()).Name()] = api
	return hfMyList
}

type MyListApi struct {
	Info     struct{}          `name:"我的投诉列表" desc:"我的投诉列表"`
	Request  MyListApiRequest  // API请求参数
	Response MyListApiResponse // API响应数据
}

type MyListApiRequest struct {
	Query struct {
		Status   *int8 `form:"status" binding:"omitempty,oneof=0 1" desc:"状态 0未处理 1已处理"`
		Page     int   `form:"page" binding:"omitempty,min=1" desc:"页码"`
		PageSize int   `form:"page_size" binding:"omitempty,min=1,max=50" desc:"每页数量"`
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
	TypeOther   string    `json:"type_other" desc:"其它类型说明"`
	Description string    `json:"description" desc:"详细说明"`
	Status      int8      `json:"status" desc:"状态 0未处理 1已处理"`
	ProcessedAt time.Time `json:"processed_at,omitempty" desc:"处理时间"`
	CreatedAt   time.Time `json:"created_at" desc:"创建时间"`
}

// Run Api业务逻辑执行点
func (m *MyListApi) Run(ctx *gin.Context) kit.Code {
	request := m.Request.Query

	// 获取当前用户ID
	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	reporterID := cast.ToInt64(id)

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

	// 查询当前用户的投诉
	if request.Status != nil {
		feedbacks, total, err = frp.ListByReporterAndStatus(ctx, reporterID, *request.Status, offset, pageSize)
	} else {
		feedbacks, total, err = frp.ListByReporter(ctx, reporterID, offset, pageSize)
	}

	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询我的投诉列表失败")
		return comm.CodeDatabaseError
	}

	items := make([]MyFeedbackItem, 0, len(feedbacks))
	for _, fb := range feedbacks {
		items = append(items, MyFeedbackItem{
			ID:          fb.ID,
			PostID:      fb.PostID,
			Type:        fb.Type,
			TypeOther:   fb.TypeOther,
			Description: fb.Description,
			Status:      fb.Status,
			ProcessedAt: fb.ProcessedAt,
			CreatedAt:   fb.CreatedAt,
		})
	}

	m.Response = MyListApiResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}
	return comm.CodeOK
}

// Init Api初始化
func (m *MyListApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindQuery(&m.Request.Query)
}

// hfMyList API执行入口
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
