package post

import (
	"app/comm"
	"app/dao/repo"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"
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
	PublishType string    `form:"publish_type" binding:"omitempty,oneof=LOST FOUND" desc:"发布类型 LOST/FOUND"`
	ItemType    string    `form:"item_type" binding:"omitempty,max=20" desc:"物品类型(含其它)"`
	Campus      string    `form:"campus" binding:"omitempty,oneof=ZHAO_HUI PING_FENG MO_GAN_SHAN" desc:"校区"`
	Location    string    `form:"location" binding:"omitempty,max=100" desc:"地点"`
	Status      string    `form:"status" binding:"omitempty,oneof=PENDING APPROVED MATCHED CLAIMED CANCELLED REJECTED ARCHIVED" desc:"状态"`
	StartTime   time.Time `form:"start_time"  desc:"时间范围起"`
	EndTime     time.Time `form:"end_time" desc:"时间范围止"`
	Page        int       `form:"page" binding:"required,min=1" desc:"页码"`
	PageSize    int       `form:"page_size" binding:"required,min=1,max=50" desc:"每页数量"`
}

type QueryApiResponse struct {
	Total    int64          `json:"total" desc:"总数"`
	Page     int            `json:"page" desc:"页码"`
	PageSize int            `json:"page_size" desc:"每页数量"`
	List     []PostListItem `json:"list" desc:"列表"`
}

type PostListItem struct {
	ID                int64     `json:"id" desc:"发布ID"`
	PublishType       string    `json:"publish_type" desc:"发布类型 LOST/FOUND"`
	ItemName          string    `json:"item_name" desc:"物品名称"`
	ItemType          string    `json:"item_type" desc:"物品类型"`
	ItemTypeOther     string    `json:"item_type_other" desc:"其它类型说明"`
	Campus            string    `json:"campus" desc:"校区"`
	Location          string    `json:"location" desc:"地点"`
	EventTime         time.Time `json:"event_time" desc:"事件时间"`
	Features          string    `json:"features" desc:"物品特征"`
	HasReward         bool      `json:"has_reward" desc:"是否有悬赏"`
	RewardDescription string    `json:"reward_description" desc:"悬赏说明"`
	Status            string    `json:"status" desc:"状态"`
	Images            []string  `json:"images" desc:"图片"`
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

	filter := repo.PostListFilter{
		PublishType: strings.TrimSpace(request.PublishType),
		ItemType:    strings.TrimSpace(request.ItemType),
		Campus:      strings.TrimSpace(request.Campus),
		Location:    strings.TrimSpace(request.Location),
		Status:      request.Status,
		StartTime:   request.StartTime,
		EndTime:     request.EndTime,
	}
	offset := (page - 1) * pageSize

	prp := repo.NewPostRepo()
	records, total, err := prp.ListByFilter(ctx, filter, offset, pageSize)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询发布列表失败")
		return comm.CodeServerError
	}

	items := make([]PostListItem, 0, len(records))
	for _, record := range records {
		var images []string
		if record.Images != "" {
			err = sonic.UnmarshalString(record.Images, &images)
			if err != nil {
				nlog.Pick().WithContext(ctx).WithError(err).Warn("解析图片列表失败")
				return comm.CodeServerError
			}
		}
		items = append(items, PostListItem{
			ID:                record.ID,
			PublishType:       record.PublishType,
			ItemName:          record.ItemName,
			ItemType:          record.ItemType,
			ItemTypeOther:     record.ItemTypeOther,
			Location:          record.Location,
			EventTime:         record.EventTime,
			Features:          record.Features,
			HasReward:         record.HasReward,
			RewardDescription: record.RewardDescription,
			Status:            record.Status,
			Images:            images,
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
	if strings.TrimSpace(req.PublishType) != "" {
		return true
	}
	if strings.TrimSpace(req.ItemType) != "" {
		return true
	}
	if strings.TrimSpace(req.Campus) != "" {
		return true
	}
	if strings.TrimSpace(req.Location) != "" {
		return true
	}
	if strings.TrimSpace(req.Status) != "" {
		return true
	}
	if !req.StartTime.IsZero() || !req.EndTime.IsZero() {
		return true
	}
	return false
}
