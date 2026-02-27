package post

import (
	"app/api/admin/system"
	"app/comm"
	"app/comm/enum"
	"app/dao/repo"
	"reflect"
	"runtime"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func UpdateHandler() gin.HandlerFunc {
	api := UpdateApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfUpdate).Pointer()).Name()] = api
	return hfUpdate
}

type UpdateApi struct {
	Info     struct{} `name:"修改我的发布信息" desc:"修改我的发布信息"`
	Request  UpdateApiRequest
	Response UpdateApiResponse
}

type UpdateApiRequest struct {
	Body struct {
		PostID            int64     `json:"post_id" binding:"required" desc:"发布ID"`
		ItemName          string    `json:"item_name" binding:"required,max=50" desc:"物品名称"`
		ItemType          string    `json:"item_type" binding:"required,max=20" desc:"物品类型"`
		ItemTypeOther     string    `json:"item_type_other" binding:"max=15" desc:"其它类型说明"`
		Campus            string    `json:"campus" binding:"required,oneof=ZHAO_HUI PING_FENG MO_GAN_SHAN" desc:"校区"`
		Location          string    `json:"location" binding:"required,max=100" desc:"地点"`
		StorageLocation   string    `json:"storage_location" binding:"required,max=100" desc:"存放地点"`
		EventTime         time.Time `json:"event_time" binding:"required" desc:"事件时间"`
		Features          string    `json:"features" binding:"required,max=200" desc:"物品特征"`
		ContactName       string    `json:"contact_name" binding:"required,max=30" desc:"联系人"`
		ContactPhone      string    `json:"contact_phone" binding:"required,max=20" desc:"联系电话"`
		HasReward         bool      `json:"has_reward" desc:"是否有悬赏"`
		RewardDescription string    `json:"reward_description" binding:"max=255" desc:"悬赏说明(仅has_reward为true时有效)"`
		Images            []string  `json:"images" binding:"required,max=3" desc:"图片列表"`
	}
}

type UpdateApiResponse struct {
	Success bool `json:"success" desc:"是否成功"`
}

func (u *UpdateApi) Run(ctx *gin.Context) kit.Code {
	request := u.Request.Body

	publisherID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	prp := repo.NewPostRepo()
	post, err := prp.FindById(ctx, request.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询发布记录失败")
		return comm.CodeServerError
	}
	if post == nil {
		return comm.CodeDataNotFound
	}

	if post.PublisherID != publisherID {
		return comm.CodePostNotOwner
	}

	if post.Status != enum.PostStatusPending && post.Status != enum.PostStatusApproved {
		return comm.CodePostCannotModify
	}

	if !system.IsValidItemType(ctx, request.ItemType) {
		return comm.CodeParameterInvalid
	}

	imagesJSON, err := sonic.MarshalString(request.Images)
	if err != nil {
		return comm.CodeParameterInvalid
	}

	post.ItemName = request.ItemName
	post.ItemType = request.ItemType
	post.ItemTypeOther = request.ItemTypeOther
	post.Campus = request.Campus
	post.Location = request.Location
	post.StorageLocation = request.StorageLocation
	post.EventTime = request.EventTime
	post.Features = request.Features
	post.ContactName = request.ContactName
	post.ContactPhone = request.ContactPhone
	post.HasReward = request.HasReward
	post.RewardDescription = request.RewardDescription
	post.Images = imagesJSON

	if err := prp.Save(ctx, post); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("更新发布记录失败")
		return comm.CodeServerError
	}

	u.Response = UpdateApiResponse{Success: true}
	return comm.CodeOK
}

func (u *UpdateApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindJSON(&u.Request.Body)
}

func hfUpdate(ctx *gin.Context) {
	api := &UpdateApi{}
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
