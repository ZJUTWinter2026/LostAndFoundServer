package post

import (
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/dao/repo"
)

// QueryHandler API router注册点
func QueryHandler() gin.HandlerFunc {
	api := QueryApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfQuery).Pointer()).Name()] = api
	return hfQuery
}

type QueryApi struct {
	Info     struct{}         `name:"失物/招领信息查询" desc:"失物/招领信息查询"`
	Request  QueryApiRequest  // API请求参数 (Body/Header/Body/Body)
	Response QueryApiResponse // API响应数据 (Body中的Data部分)
}

type QueryApiRequest struct {
	Query QueryFilter
}

type QueryFilter struct {
	ItemType  string `form:"item_type" binding:"omitempty,max=20" desc:"物品类型(含其它)"`
	Location  string `form:"location" binding:"omitempty,max=100" desc:"地点"`
	Status    *int8  `form:"status" binding:"omitempty,oneof=0 1 2" desc:"状态 0待审核 1已发布 2已认领"`
	StartTime string `form:"start_time" binding:"omitempty" desc:"时间范围起"`
	EndTime   string `form:"end_time" binding:"omitempty" desc:"时间范围止"`
	Page      int    `form:"page" binding:"omitempty,min=1" desc:"页码"`
	PageSize  int    `form:"page_size" binding:"omitempty,min=1,max=50" desc:"每页数量"`
}

type QueryApiResponse struct {
	Total    int64          `json:"total" desc:"总数"`
	Page     int            `json:"page" desc:"页码"`
	PageSize int            `json:"page_size" desc:"每页数量"`
	List     []PostListItem `json:"list" desc:"列表"`
}

type PostListItem struct {
	ID            int64     `json:"id" desc:"发布ID"`
	PublishType   int8      `json:"publish_type" desc:"发布类型 1失物 2招领"`
	ItemName      string    `json:"item_name" desc:"物品名称"`
	ItemType      string    `json:"item_type" desc:"物品类型"`
	ItemTypeOther string    `json:"item_type_other" desc:"其它类型说明"`
	Location      string    `json:"location" desc:"地点"`
	EventTime     time.Time `json:"event_time" desc:"事件时间"`
	Features      string    `json:"features" desc:"物品特征"`
	Status        int8      `json:"status" desc:"状态"`
	Images        []string  `json:"images" desc:"图片"`
}

// Run Api业务逻辑执行点
func (q *QueryApi) Run(ctx *gin.Context) kit.Code {
	request := q.Request.Query

	if !hasAnyFilter(request) {
		return comm.CodeParameterInvalid
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

	startTime, err := comm.ParseOptionalTime(request.StartTime, "")
	if err != nil {
		return comm.CodeParameterInvalid
	}
	endTime, err := comm.ParseOptionalTime(request.EndTime, "")
	if err != nil {
		return comm.CodeParameterInvalid
	}
	if startTime != nil && endTime != nil && startTime.After(*endTime) {
		return comm.CodeParameterInvalid
	}

	filter := repo.PostListFilter{
		ItemType:  strings.TrimSpace(request.ItemType),
		Location:  strings.TrimSpace(request.Location),
		Status:    request.Status,
		StartTime: startTime,
		EndTime:   endTime,
	}
	offset := (page - 1) * pageSize

	prp := repo.NewPostRepo()
	records, total, err := prp.ListByFilter(ctx, filter, offset, pageSize)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询发布列表失败")
		return comm.CodeDatabaseError
	}

	items := make([]PostListItem, 0, len(records))
	for _, record := range records {
		items = append(items, PostListItem{
			ID:            record.ID,
			PublishType:   record.PublishType,
			ItemName:      record.ItemName,
			ItemType:      record.ItemType,
			ItemTypeOther: record.ItemTypeOther,
			Location:      record.Location,
			EventTime:     record.EventTime,
			Features:      record.Features,
			Status:        record.Status,
			Images:        comm.UnmarshalImages(record.Images),
		})
	}

	q.Response = QueryApiResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}
	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (q *QueryApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindQuery(&q.Request.Query)
}

// hfQuery API执行入口
func hfQuery(ctx *gin.Context) {
	api := &QueryApi{}
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

func hasAnyFilter(req QueryFilter) bool {
	if strings.TrimSpace(req.ItemType) != "" {
		return true
	}
	if strings.TrimSpace(req.Location) != "" {
		return true
	}
	if req.Status != nil {
		return true
	}
	if strings.TrimSpace(req.StartTime) != "" || strings.TrimSpace(req.EndTime) != "" {
		return true
	}
	return false
}
