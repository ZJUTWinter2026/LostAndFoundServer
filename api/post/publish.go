package post

import (
	"app/api/admin/system"
	"app/comm"
	"app/comm/enum"
	"app/dao/model"
	"app/dao/repo"
	"app/pkg/vector"
	"context"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func PublishHandler() gin.HandlerFunc {
	api := PublishApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfPublish).Pointer()).Name()] = api
	return hfPublish
}

type PublishApi struct {
	Info     struct{} `name:"发布失物/招领" desc:"发布失物/招领"`
	Request  PublishApiRequest
	Response PublishApiResponse
}

type PublishApiRequest struct {
	Body struct {
		PublishType       string    `json:"publish_type" binding:"required,oneof=LOST FOUND" desc:"发布类型 LOST失物 FOUND招领"`
		ItemName          string    `json:"item_name" binding:"required,max=50" desc:"物品名称"`
		ItemType          string    `json:"item_type" binding:"required,max=20" desc:"物品类型"`
		Campus            string    `json:"campus" binding:"required,oneof=ZHAO_HUI PING_FENG MO_GAN_SHAN" desc:"校区"`
		Location          string    `json:"location" binding:"required,max=100" desc:"地点"`
		StorageLocation   string    `json:"storage_location" binding:"omitempty,max=100" desc:"存放地点"`
		EventTime         time.Time `json:"event_time" binding:"required" desc:"丢失/拾取时间"`
		Features          string    `json:"features" binding:"required,max=255" desc:"物品特征"`
		ContactName       string    `json:"contact_name" binding:"required,max=30" desc:"联系人"`
		ContactPhone      string    `json:"contact_phone" binding:"required" desc:"联系电话"`
		HasReward         bool      `json:"has_reward" desc:"是否有悬赏"`
		RewardDescription string    `json:"reward_description" binding:"omitempty,max=255" desc:"悬赏说明(仅has_reward为true时有效)"`
		Images            []string  `json:"images" binding:"omitempty,max=3" desc:"图片列表"`
	}
}

type PublishApiResponse struct {
	Id int64 `json:"id" desc:"发布ID"`
}

func (p *PublishApi) Run(ctx *gin.Context) kit.Code {
	request := p.Request.Body

	publisherID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	prp := repo.NewPostRepo()
	scr := repo.NewSystemConfigRepo()

	publishLimit, err := scr.GetPublishLimit(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("获取发布限制失败")
		return comm.CodeServerError
	}

	todayCount, err := prp.CountTodayByPublisher(ctx, publisherID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("统计今日发布数量失败")
		return comm.CodeServerError
	}
	if int(todayCount) >= publishLimit {
		return comm.CodePublishLimitExceeded
	}

	if !system.IsValidItemType(ctx, request.ItemType) {
		return comm.CodeParameterInvalid
	}

	imagesJSON, err := sonic.MarshalString(request.Images)
	if err != nil {
		return comm.CodeParameterInvalid
	}

	record := &model.Post{
		PublisherID:       publisherID,
		PublishType:       request.PublishType,
		ItemName:          strings.TrimSpace(request.ItemName),
		ItemType:          request.ItemType,
		Campus:            request.Campus,
		Location:          request.Location,
		StorageLocation:   request.StorageLocation,
		EventTime:         request.EventTime,
		Features:          request.Features,
		ContactName:       request.ContactName,
		ContactPhone:      request.ContactPhone,
		HasReward:         request.HasReward,
		RewardDescription: request.RewardDescription,
		Images:            imagesJSON,
		Status:            enum.PostStatusPending,
	}

	err = prp.Create(ctx, record)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("发布失败")
		return comm.CodeServerError
	}

	go func() {
		bgCtx := context.Background()
		vectorSvc := vector.NewService()
		if err := vectorSvc.UpdatePostVector(bgCtx, record); err != nil {
			nlog.Pick().WithContext(bgCtx).WithError(err).Warn("更新向量失败")
		}
	}()

	p.Response.Id = record.ID
	return comm.CodeOK
}

func (p *PublishApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindJSON(&p.Request.Body)
	if err != nil {
		return err
	}
	return err
}

func hfPublish(ctx *gin.Context) {
	api := &PublishApi{}
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
