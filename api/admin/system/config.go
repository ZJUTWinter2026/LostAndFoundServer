package system

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/repo"
	"context"
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"
)

func ConfigListHandler() gin.HandlerFunc {
	api := ConfigListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfConfigList).Pointer()).Name()] = api
	return hfConfigList
}

type ConfigListApi struct {
	Info     struct{}               `name:"获取系统配置" desc:"获取系统配置"`
	Request  ConfigListApiRequest
	Response ConfigListApiResponse
}

type ConfigListApiRequest struct {
	Query struct{}
}

type ConfigListApiResponse struct {
	FeedbackTypes     []string `json:"feedback_types" desc:"投诉反馈类型"`
	ItemTypes         []string `json:"item_types" desc:"物品类型分类"`
	ClaimValidityDays int      `json:"claim_validity_days" desc:"认领时效(天)"`
}

func (a *ConfigListApi) Run(ctx *gin.Context) kit.Code {
	if code := checkSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	scr := repo.NewSystemConfigRepo()

	feedbackTypes, err := scr.GetFeedbackTypes(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("获取投诉类型失败")
		return comm.CodeDatabaseError
	}

	itemTypes, err := scr.GetItemTypes(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("获取物品类型失败")
		return comm.CodeDatabaseError
	}

	claimValidityDays, err := scr.GetClaimValidityDays(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("获取认领时效失败")
		return comm.CodeDatabaseError
	}

	a.Response = ConfigListApiResponse{
		FeedbackTypes:     feedbackTypes,
		ItemTypes:         itemTypes,
		ClaimValidityDays: claimValidityDays,
	}
	return comm.CodeOK
}

func hfConfigList(ctx *gin.Context) {
	api := &ConfigListApi{}
	code := api.Run(ctx)
	if code == comm.CodeOK {
		reply.Success(ctx, api.Response)
	} else {
		reply.Fail(ctx, code)
	}
}

func ConfigUpdateHandler() gin.HandlerFunc {
	api := ConfigUpdateApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfConfigUpdate).Pointer()).Name()] = api
	return hfConfigUpdate
}

type ConfigUpdateApi struct {
	Info     struct{}               `name:"更新系统配置" desc:"更新系统配置"`
	Request  ConfigUpdateApiRequest
	Response struct{}
}

type ConfigUpdateApiRequest struct {
	Body struct {
		ConfigKey         string   `json:"config_key" binding:"required,oneof=feedback_types item_types claim_validity_days" desc:"配置键名"`
		FeedbackTypes     []string `json:"feedback_types" desc:"投诉反馈类型(当config_key为feedback_types时必填)"`
		ItemTypes         []string `json:"item_types" desc:"物品类型分类(当config_key为item_types时必填)"`
		ClaimValidityDays *int     `json:"claim_validity_days" desc:"认领时效天数(当config_key为claim_validity_days时必填)"`
	}
}

func (a *ConfigUpdateApi) Run(ctx *gin.Context) kit.Code {
	if code := checkSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	req := a.Request.Body
	scr := repo.NewSystemConfigRepo()

	switch req.ConfigKey {
	case "feedback_types":
		if len(req.FeedbackTypes) == 0 {
			return comm.CodeParameterInvalid
		}
		if err := scr.UpdateFeedbackTypes(ctx, req.FeedbackTypes); err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("更新投诉类型失败")
			return comm.CodeDatabaseError
		}

	case "item_types":
		if len(req.ItemTypes) == 0 {
			return comm.CodeParameterInvalid
		}
		if err := scr.UpdateItemTypes(ctx, req.ItemTypes); err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("更新物品类型失败")
			return comm.CodeDatabaseError
		}

	case "claim_validity_days":
		if req.ClaimValidityDays == nil || *req.ClaimValidityDays <= 0 {
			return comm.CodeParameterInvalid
		}
		if err := scr.UpdateClaimValidityDays(ctx, *req.ClaimValidityDays); err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("更新认领时效失败")
			return comm.CodeDatabaseError
		}
	}

	return comm.CodeOK
}

func (a *ConfigUpdateApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfConfigUpdate(ctx *gin.Context) {
	api := &ConfigUpdateApi{}
	if err := api.Init(ctx); err != nil {
		reply.Fail(ctx, comm.CodeParameterInvalid)
		return
	}
	code := api.Run(ctx)
	if code == comm.CodeOK {
		reply.Success(ctx, struct{}{})
	} else {
		reply.Fail(ctx, code)
	}
}

func checkSysAdmin(ctx *gin.Context) kit.Code {
	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	adminID := cast.ToInt64(id)

	urp := repo.NewUserRepo()
	user, err := urp.FindById(ctx, adminID)
	if err != nil {
		return comm.CodeDatabaseError
	}
	if user == nil || user.Usertype != enum.UserTypeSystemAdmin {
		return comm.CodeAdminPermissionDenied
	}
	return comm.CodeOK
}

func PublicConfigHandler() gin.HandlerFunc {
	api := PublicConfigApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfPublicConfig).Pointer()).Name()] = api
	return hfPublicConfig
}

type PublicConfigApi struct {
	Info     struct{}                  `name:"获取公开配置" desc:"获取公开配置(物品类型等)"`
	Request  PublicConfigApiRequest
	Response PublicConfigApiResponse
}

type PublicConfigApiRequest struct {
	Query struct{}
}

type PublicConfigApiResponse struct {
	ItemTypes []string `json:"item_types" desc:"物品类型分类"`
}

func (a *PublicConfigApi) Run(ctx *gin.Context) kit.Code {
	scr := repo.NewSystemConfigRepo()

	itemTypes, err := scr.GetItemTypes(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("获取物品类型失败")
		return comm.CodeDatabaseError
	}

	a.Response = PublicConfigApiResponse{
		ItemTypes: itemTypes,
	}
	return comm.CodeOK
}

func hfPublicConfig(ctx *gin.Context) {
	api := &PublicConfigApi{}
	code := api.Run(ctx)
	if code == comm.CodeOK {
		reply.Success(ctx, api.Response)
	} else {
		reply.Fail(ctx, code)
	}
}

func IsValidItemType(ctx context.Context, itemType string) bool {
	scr := repo.NewSystemConfigRepo()
	itemTypes, err := scr.GetItemTypes(ctx)
	if err != nil {
		return false
	}
	for _, t := range itemTypes {
		if t == itemType {
			return true
		}
	}
	return false
}

func IsValidFeedbackType(ctx context.Context, feedbackType string) bool {
	scr := repo.NewSystemConfigRepo()
	feedbackTypes, err := scr.GetFeedbackTypes(ctx)
	if err != nil {
		return false
	}
	for _, t := range feedbackTypes {
		if t == feedbackType {
			return true
		}
	}
	return false
}
