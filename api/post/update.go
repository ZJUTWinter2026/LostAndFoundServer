package post

import (
	"app/api/admin/system"
	"app/comm"
	"app/comm/enum"
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

func UpdateHandler() gin.HandlerFunc {
	api := UpdateApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfUpdate).Pointer()).Name()] = api
	return hfUpdate
}

type UpdateApi struct {
	Info     struct{}          `name:"修改我的发布信息" desc:"修改我的发布信息"`
	Request  UpdateApiRequest
	Response UpdateApiResponse
}

type UpdateApiRequest struct {
	Body struct {
		PostID          int64    `json:"post_id" binding:"required" desc:"发布ID"`
		ItemName        string   `json:"item_name" binding:"omitempty,max=50" desc:"物品名称"`
		ItemType        string   `json:"item_type" binding:"omitempty,max=20" desc:"物品类型"`
		ItemTypeOther   string   `json:"item_type_other" binding:"omitempty,max=15" desc:"其它类型说明"`
		Campus          string   `json:"campus" binding:"omitempty,oneof=ZHAO_HUI PING_FENG MO_GAN_SHAN" desc:"校区"`
		Location        string   `json:"location" binding:"omitempty,max=100" desc:"地点"`
		StorageLocation string   `json:"storage_location" binding:"omitempty,max=100" desc:"存放地点"`
		EventTime       string   `json:"event_time" binding:"omitempty" desc:"事件时间"`
		Features        string   `json:"features" binding:"omitempty,max=255" desc:"物品特征"`
		ContactName     string   `json:"contact_name" binding:"omitempty,max=30" desc:"联系人"`
		ContactPhone    string   `json:"contact_phone" binding:"omitempty,max=20" desc:"联系电话"`
		HasReward       *bool    `json:"has_reward" desc:"是否有悬赏"`
		Images          []string `json:"images" binding:"omitempty,dive,max=255" desc:"图片列表"`
	}
}

type UpdateApiResponse struct {
	Success bool `json:"success" desc:"是否成功"`
}

func (u *UpdateApi) Run(ctx *gin.Context) kit.Code {
	request := u.Request.Body

	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	publisherID := cast.ToInt64(id)

	prp := repo.NewPostRepo()
	post, err := prp.FindById(ctx, request.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询发布记录失败")
		return comm.CodeDatabaseError
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

	updates := make(map[string]interface{})

	if strings.TrimSpace(request.ItemName) != "" {
		updates["item_name"] = request.ItemName
	}
	if strings.TrimSpace(request.ItemType) != "" {
		if !system.IsValidItemType(ctx, request.ItemType) {
			return comm.CodeParameterInvalid
		}
		updates["item_type"] = request.ItemType
		if request.ItemType == "其它类型" {
			if strings.TrimSpace(request.ItemTypeOther) == "" {
				return comm.CodeParameterInvalid
			}
			if utf8.RuneCountInString(request.ItemTypeOther) > 15 {
				return comm.CodeParameterInvalid
			}
			updates["item_type_other"] = request.ItemTypeOther
		} else {
			updates["item_type_other"] = ""
		}
	}
	if strings.TrimSpace(request.Campus) != "" {
		updates["campus"] = request.Campus
	}
	if strings.TrimSpace(request.Location) != "" {
		updates["location"] = request.Location
	}
	if strings.TrimSpace(request.StorageLocation) != "" {
		updates["storage_location"] = request.StorageLocation
	}
	if strings.TrimSpace(request.EventTime) != "" {
		eventTime, err := comm.ParseEventTime(request.EventTime)
		if err != nil {
			return comm.CodeParameterInvalid
		}
		updates["event_time"] = eventTime
	}
	if strings.TrimSpace(request.Features) != "" {
		updates["features"] = request.Features
	}
	if strings.TrimSpace(request.ContactName) != "" {
		updates["contact_name"] = request.ContactName
	}
	if strings.TrimSpace(request.ContactPhone) != "" {
		updates["contact_phone"] = request.ContactPhone
	}
	if request.HasReward != nil {
		hasReward := int8(0)
		if *request.HasReward {
			hasReward = 1
		}
		updates["has_reward"] = hasReward
	}
	if len(request.Images) > 0 {
		imagesJSON, err := comm.MarshalImages(request.Images)
		if err != nil {
			return comm.CodeParameterInvalid
		}
		updates["images"] = string(imagesJSON)
	}

	if len(updates) == 0 {
		return comm.CodeParameterInvalid
	}

	err = prp.UpdatePost(ctx, request.PostID, publisherID, updates)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("更新发布记录失败")
		return comm.CodeDatabaseError
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
