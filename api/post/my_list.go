package post

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
	"app/dao/repo"
)

// 发布状态常量
const (
	StatusPending   int8 = 0 // 待审核
	StatusApproved  int8 = 1 // 已通过
	StatusMatched   int8 = 2 // 已匹配
	StatusClaimed   int8 = 3 // 已认领
	StatusCancelled int8 = 4 // 已取消
	StatusRejected  int8 = 5 // 已驳回
)

// MyListHandler API router注册点
func MyListHandler() gin.HandlerFunc {
	api := MyListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfMyList).Pointer()).Name()] = api
	return hfMyList
}

type MyListApi struct {
	Info     struct{}          `name:"我发布的消息列表" desc:"我发布的消息列表"`
	Request  MyListApiRequest  // API请求参数
	Response MyListApiResponse // API响应数据
}

type MyListApiRequest struct {
	Query struct {
		PublishType *int8 `form:"publish_type" binding:"omitempty,oneof=1 2" desc:"发布类型 1失物 2招领"`
		Status      *int8 `form:"status" binding:"omitempty,oneof=0 1 2 3 4 5" desc:"状态 0待审核 1已通过 2已匹配 3已认领 4已取消 5已驳回"`
		Page        int   `form:"page" binding:"omitempty,min=1" desc:"页码"`
		PageSize    int   `form:"page_size" binding:"omitempty,min=1,max=50" desc:"每页数量"`
	}
}

type MyListApiResponse struct {
	Total    int64            `json:"total" desc:"总数"`
	Page     int              `json:"page" desc:"页码"`
	PageSize int              `json:"page_size" desc:"每页数量"`
	List     []MyPostListItem `json:"list" desc:"我的发布列表"`
}

type MyPostListItem struct {
	ID           int64     `json:"id" desc:"发布ID"`
	PublishType  int8      `json:"publish_type" desc:"发布类型 1失物 2招领"`
	ItemName     string    `json:"item_name" desc:"物品名称"`
	ItemType     string    `json:"item_type" desc:"物品类型"`
	Location     string    `json:"location" desc:"地点"`
	EventTime    time.Time `json:"event_time" desc:"事件时间"`
	Status       int8      `json:"status" desc:"状态"`
	StatusText   string    `json:"status_text" desc:"状态文本"`
	CancelReason string    `json:"cancel_reason,omitempty" desc:"取消原因"`
	RejectReason string    `json:"reject_reason,omitempty" desc:"驳回原因"`
	CreatedAt    time.Time `json:"created_at" desc:"创建时间"`
}

// Run Api业务逻辑执行点
func (m *MyListApi) Run(ctx *gin.Context) kit.Code {
	request := m.Request.Query

	// 获取当前用户ID
	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	publisherID := cast.ToInt64(id)

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
	prp := repo.NewPostRepo()

	records, total, err := prp.ListByPublisher(ctx, publisherID, request.PublishType, request.Status, offset, pageSize)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询我的发布列表失败")
		return comm.CodeDatabaseError
	}

	items := make([]MyPostListItem, 0, len(records))
	for _, post := range records {
		items = append(items, MyPostListItem{
			ID:           post.ID,
			PublishType:  post.PublishType,
			ItemName:     post.ItemName,
			ItemType:     post.ItemType,
			Location:     post.Location,
			EventTime:    post.EventTime,
			Status:       post.Status,
			StatusText:   getStatusText(post.Status),
			CancelReason: post.CancelReason,
			RejectReason: post.RejectReason,
			CreatedAt:    post.CreatedAt,
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

// getStatusText 获取状态文本
func getStatusText(status int8) string {
	statusMap := map[int8]string{
		0: "待审核",
		1: "已通过",
		2: "已匹配",
		3: "已认领",
		4: "已取消",
		5: "已驳回",
	}
	if text, ok := statusMap[status]; ok {
		return text
	}
	return "未知"
}
