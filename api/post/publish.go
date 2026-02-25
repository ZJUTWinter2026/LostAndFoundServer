package post

import (
	"app/api/admin/system"
	"app/comm"
	"app/comm/enum"
	"app/dao/model"
	"app/dao/repo"
	"reflect"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
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
		PublishType     string   `json:"publish_type" binding:"required,oneof=LOST FOUND" desc:"发布类型 LOST失物 FOUND招领"`
		ItemName        string   `json:"item_name" binding:"required,max=50" desc:"物品名称"`
		ItemType        string   `json:"item_type" binding:"required,max=20" desc:"物品类型"`
		ItemTypeOther   string   `json:"item_type_other" binding:"max=15" desc:"其它类型说明"`
		Campus          string   `json:"campus" binding:"required,oneof=ZHAO_HUI PING_FENG MO_GAN_SHAN" desc:"校区"`
		Location        string   `json:"location" binding:"required,max=100" desc:"地点"`
		StorageLocation string   `json:"storage_location" binding:"max=100" desc:"存放地点"`
		EventTime       string   `json:"event_time" binding:"required" desc:"丢失/拾取时间"`
		Features        string   `json:"features" binding:"required,max=255" desc:"物品特征"`
		ContactName     string   `json:"contact_name" binding:"required,max=30" desc:"联系人"`
		ContactPhone    string   `json:"contact_phone" binding:"required,min=5,max=20" desc:"联系电话"`
		HasReward       bool     `json:"has_reward" desc:"是否有悬赏"`
		Images          []string `json:"images" binding:"max=3" desc:"图片列表"`
	}
}

type PublishApiResponse struct {
	Id int64 `json:"id" desc:"发布ID"`
}

func (p *PublishApi) Run(ctx *gin.Context) kit.Code {
	request := p.Request.Body

	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	publisherID := cast.ToInt64(id)

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

	if strings.TrimSpace(request.ItemType) == "其它类型" {
		if strings.TrimSpace(request.ItemTypeOther) == "" {
			return comm.CodeParameterInvalid
		}
		if utf8.RuneCountInString(request.ItemTypeOther) > 15 {
			return comm.CodeParameterInvalid
		}
	}

	if request.PublishType == enum.PostTypeFound {
		if len(request.Images) == 0 {
			return comm.CodeParameterInvalid
		}
	}

	eventTime, err := comm.ParseEventTime(request.EventTime)
	if err != nil {
		return comm.CodeParameterInvalid
	}

	imagesJSON, err := comm.MarshalImages(request.Images)
	if err != nil {
		return comm.CodeParameterInvalid
	}

	record := &model.Post{
		PublisherID:     publisherID,
		PublishType:     request.PublishType,
		ItemName:        strings.TrimSpace(request.ItemName),
		ItemType:        request.ItemType,
		ItemTypeOther:   request.ItemTypeOther,
		Campus:          request.Campus,
		Location:        request.Location,
		StorageLocation: request.StorageLocation,
		EventTime:       eventTime,
		Features:        request.Features,
		ContactName:     request.ContactName,
		ContactPhone:    request.ContactPhone,
		HasReward:       request.HasReward,
		Images:          imagesJSON,
		Status:          enum.PostStatusPending,
	}

	err = prp.Create(ctx, record)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("发布失败")
		return comm.CodeServerError
	}

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
