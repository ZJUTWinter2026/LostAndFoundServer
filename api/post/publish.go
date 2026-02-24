package post

import (
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

	"app/comm"
	"app/dao/model"
	"app/dao/repo"
)

const (
	statusPending int8 = 0
)

// PublishHandler API router注册点
func PublishHandler() gin.HandlerFunc {
	api := PublishApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfPublish).Pointer()).Name()] = api
	return hfPublish
}

type PublishApi struct {
	Info     struct{}           `name:"发布失物/招领" desc:"发布失物/招领"`
	Request  PublishApiRequest  // API请求参数 (Body/Header/Body/Body)
	Response PublishApiResponse // API响应数据 (Body中的Data部分)
}

type PublishApiRequest struct {
	Body struct {
		PublishType   int8     `json:"publish_type" binding:"required,oneof=1 2" desc:"发布类型 1失物 2招领"`
		ItemName      string   `json:"item_name" binding:"required,max=50" desc:"物品名称"`
		ItemType      string   `json:"item_type" binding:"required,max=20" desc:"物品类型"`
		ItemTypeOther string   `json:"item_type_other" binding:"max=15" desc:"其它类型说明"`
		Location      string   `json:"location" binding:"required,max=100" desc:"地点"`
		EventTime     string   `json:"event_time" binding:"required" desc:"丢失/拾取时间"`
		Features      string   `json:"features" binding:"required,max=255" desc:"物品特征"`
		ContactName   string   `json:"contact_name" binding:"required,max=30" desc:"联系人"`
		ContactPhone  string   `json:"contact_phone" binding:"required,min=5,max=20" desc:"联系电话"`
		HasReward     *bool    `json:"has_reward" desc:"是否有悬赏"`
		Images        []string `json:"images" binding:"omitempty,dive,max=255" desc:"图片列表"`
	}
}

type PublishApiResponse struct {
	Id int64 `json:"id" binding:"required" desc:"发布ID"`
}

// Run Api业务逻辑执行点
func (p *PublishApi) Run(ctx *gin.Context) kit.Code {
	request := p.Request.Body

	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	publisherID := cast.ToInt64(id)

	if strings.TrimSpace(request.ItemType) == "其它" {
		if strings.TrimSpace(request.ItemTypeOther) == "" {
			return comm.CodeParameterInvalid
		}
		if utf8.RuneCountInString(request.ItemTypeOther) > 15 {
			return comm.CodeParameterInvalid
		}
	}

	if request.PublishType == 2 && request.HasReward != nil && *request.HasReward {
		return comm.CodeParameterInvalid
	}

	eventTime, err := comm.ParseEventTime(request.EventTime)
	if err != nil {
		return comm.CodeParameterInvalid
	}

	imagesJSON, err := comm.MarshalImages(request.Images)
	if err != nil {
		return comm.CodeParameterInvalid
	}

	hasReward := int8(0)
	if request.HasReward != nil && *request.HasReward {
		hasReward = 1
	}

	record := &model.Post{
		PublisherID:   publisherID,
		PublishType:   request.PublishType,
		ItemName:      request.ItemName,
		ItemType:      request.ItemType,
		ItemTypeOther: request.ItemTypeOther,
		Location:      request.Location,
		EventTime:     eventTime,
		Features:      request.Features,
		ContactName:   request.ContactName,
		ContactPhone:  request.ContactPhone,
		HasReward:     hasReward,
		Images:        string(imagesJSON),
		Status:        statusPending,
	}

	prp := repo.NewPostRepo()
	err = prp.Create(ctx, record)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("发布记录写入失败")
		return comm.CodeDatabaseError
	}

	p.Response = PublishApiResponse{Id: record.ID}
	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (p *PublishApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindJSON(&p.Request.Body)
	if err != nil {
		return err
	}
	return err
}

// hfPublish API执行入口
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
