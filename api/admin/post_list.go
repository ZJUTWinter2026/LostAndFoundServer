package admin

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

func PostListHandler() gin.HandlerFunc {
	api := PostListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfPostList).Pointer()).Name()] = api
	return hfPostList
}

type PostListApi struct {
	Info     struct{}            `name:"管理员查询发布信息" desc:"管理员查询发布信息，支持按状态筛选"`
	Request  PostListApiRequest
	Response PostListApiResponse
}

type PostListApiRequest struct {
	Query PostListFilter
}

type PostListFilter struct {
	PublishType string    `form:"publish_type" binding:"omitempty,oneof=LOST FOUND" desc:"发布类型 LOST/FOUND"`
	ItemType    string    `form:"item_type" binding:"omitempty,max=20" desc:"物品类型(含其它)"`
	Campus      string    `form:"campus" binding:"omitempty,oneof=ZHAO_HUI PING_FENG MO_GAN_SHAN" desc:"校区"`
	Location    string    `form:"location" binding:"omitempty,max=100" desc:"地点"`
	Status      string    `form:"status" binding:"omitempty,oneof=PENDING APPROVED SOLVED CANCELLED REJECTED ARCHIVED" desc:"状态"`
	StartTime   time.Time `form:"start_time" desc:"时间范围起"`
	EndTime     time.Time `form:"end_time" desc:"时间范围止"`
	Page        int       `form:"page" binding:"required,min=1" desc:"页码"`
	PageSize    int       `form:"page_size" binding:"required,min=1,max=50" desc:"每页数量"`
}

type PostListApiResponse struct {
	Total    int64               `json:"total" desc:"总数"`
	Page     int                 `json:"page" desc:"页码"`
	PageSize int                 `json:"page_size" desc:"每页数量"`
	List     []AdminPostListItem `json:"list" desc:"列表"`
}

type AdminPostListItem struct {
	ID                int64     `json:"id" desc:"发布ID"`
	PublishType       string    `json:"publish_type" desc:"发布类型 LOST/FOUND"`
	ItemName          string    `json:"item_name" desc:"物品名称"`
	ItemType          string    `json:"item_type" desc:"物品类型"`
	Campus            string    `json:"campus" desc:"校区"`
	Location          string    `json:"location" desc:"地点"`
	EventTime         time.Time `json:"event_time" desc:"事件时间"`
	Features          string    `json:"features" desc:"物品特征"`
	HasReward         bool      `json:"has_reward" desc:"是否有悬赏"`
	RewardDescription string    `json:"reward_description" desc:"悬赏说明"`
	Status            string    `json:"status" desc:"状态"`
	Images            []string  `json:"images" desc:"图片"`
	PublisherID       int64     `json:"publisher_id" desc:"发布者ID"`
	CreatedAt         time.Time `json:"created_at" desc:"创建时间"`
}

func (p *PostListApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	request := p.Request.Query

	filter := repo.PostListFilter{
		PublishType: strings.TrimSpace(request.PublishType),
		ItemType:    strings.TrimSpace(request.ItemType),
		Campus:      strings.TrimSpace(request.Campus),
		Location:    strings.TrimSpace(request.Location),
		Status:      strings.TrimSpace(request.Status),
		StartTime:   request.StartTime,
		EndTime:     request.EndTime,
	}
	offset := (request.Page - 1) * request.PageSize

	prp := repo.NewPostRepo()
	records, total, err := prp.ListByFilter(ctx, filter, offset, request.PageSize)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询发布列表失败")
		return comm.CodeServerError
	}

	items := make([]AdminPostListItem, 0, len(records))
	for _, record := range records {
		var images []string
		if record.Images != "" {
			err = sonic.UnmarshalString(record.Images, &images)
			if err != nil {
				nlog.Pick().WithContext(ctx).WithError(err).Warn("解析图片列表失败")
				return comm.CodeServerError
			}
		}
		items = append(items, AdminPostListItem{
			ID:                record.ID,
			PublishType:       record.PublishType,
			ItemName:          record.ItemName,
			ItemType:          record.ItemType,
			Campus:            record.Campus,
			Location:          record.Location,
			EventTime:         record.EventTime,
			Features:          record.Features,
			HasReward:         record.HasReward,
			RewardDescription: record.RewardDescription,
			Status:            record.Status,
			Images:            images,
			PublisherID:       record.PublisherID,
			CreatedAt:         record.CreatedAt,
		})
	}

	p.Response = PostListApiResponse{
		Total:    total,
		Page:     request.Page,
		PageSize: request.PageSize,
		List:     items,
	}
	return comm.CodeOK
}

func (p *PostListApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindQuery(&p.Request.Query)
}

func hfPostList(ctx *gin.Context) {
	api := &PostListApi{}
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
