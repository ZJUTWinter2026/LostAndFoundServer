package system

import (
	"app/comm"
	"app/dao/repo"
	"context"
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

const OtherType = "其它类型"

func ConfigListHandler() gin.HandlerFunc {
	api := ConfigListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfConfigList).Pointer()).Name()] = api
	return hfConfigList
}

type ConfigListApi struct {
	Info     struct{} `name:"获取系统配置" desc:"获取系统配置"`
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
	PublishLimit      int      `json:"publish_limit" desc:"每日发布限制"`
}

func (a *ConfigListApi) Run(ctx *gin.Context) kit.Code {
	_, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	scr := repo.NewSystemConfigRepo()

	feedbackTypes, err := scr.GetFeedbackTypes(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("获取投诉类型失败")
		return comm.CodeServerError
	}

	itemTypes, err := scr.GetItemTypes(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("获取物品类型失败")
		return comm.CodeServerError
	}

	claimValidityDays, err := scr.GetClaimValidityDays(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("获取认领时效失败")
		return comm.CodeServerError
	}

	publishLimit, err := scr.GetPublishLimit(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("获取发布限制失败")
		return comm.CodeServerError
	}

	a.Response = ConfigListApiResponse{
		FeedbackTypes:     append(feedbackTypes, OtherType),
		ItemTypes:         append(itemTypes, OtherType),
		ClaimValidityDays: claimValidityDays,
		PublishLimit:      publishLimit,
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

func UpdateFeedbackTypesHandler() gin.HandlerFunc {
	api := UpdateFeedbackTypesApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfUpdateFeedbackTypes).Pointer()).Name()] = api
	return hfUpdateFeedbackTypes
}

type UpdateFeedbackTypesApi struct {
	Info     struct{} `name:"更新投诉反馈类型" desc:"更新投诉反馈类型"`
	Request  UpdateFeedbackTypesApiRequest
	Response struct{}
}

type UpdateFeedbackTypesApiRequest struct {
	Body struct {
		FeedbackTypes []string `json:"feedback_types" binding:"required" desc:"投诉反馈类型列表"`
	}
}

func (a *UpdateFeedbackTypesApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	req := a.Request.Body
	filteredTypes := filterOutOtherType(req.FeedbackTypes)
	if len(filteredTypes) == 0 {
		return comm.CodeParameterInvalid
	}

	scr := repo.NewSystemConfigRepo()

	oldTypes, err := scr.GetFeedbackTypes(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("获取原有投诉类型失败")
		return comm.CodeServerError
	}

	deletedTypes := findDeletedTypes(oldTypes, filteredTypes)
	if len(deletedTypes) > 0 {
		frr := repo.NewFeedbackRepo()
		for _, deletedType := range deletedTypes {
			if err := frr.MigrateTypeToOther(ctx, deletedType, OtherType); err != nil {
				nlog.Pick().WithContext(ctx).WithError(err).Warn("迁移投诉类型数据失败", "type", deletedType)
				return comm.CodeServerError
			}
		}
	}

	if err := scr.UpdateFeedbackTypes(ctx, filteredTypes); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("更新投诉类型失败")
		return comm.CodeServerError
	}

	return comm.CodeOK
}

func (a *UpdateFeedbackTypesApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfUpdateFeedbackTypes(ctx *gin.Context) {
	api := &UpdateFeedbackTypesApi{}
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

func UpdateItemTypesHandler() gin.HandlerFunc {
	api := UpdateItemTypesApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfUpdateItemTypes).Pointer()).Name()] = api
	return hfUpdateItemTypes
}

type UpdateItemTypesApi struct {
	Info     struct{} `name:"更新物品类型" desc:"更新物品类型"`
	Request  UpdateItemTypesApiRequest
	Response struct{}
}

type UpdateItemTypesApiRequest struct {
	Body struct {
		ItemTypes []string `json:"item_types" binding:"required" desc:"物品类型列表"`
	}
}

func (a *UpdateItemTypesApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	req := a.Request.Body
	filteredTypes := filterOutOtherType(req.ItemTypes)
	if len(filteredTypes) == 0 {
		return comm.CodeParameterInvalid
	}

	scr := repo.NewSystemConfigRepo()

	oldTypes, err := scr.GetItemTypes(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("获取原有物品类型失败")
		return comm.CodeServerError
	}

	deletedTypes := findDeletedTypes(oldTypes, filteredTypes)
	if len(deletedTypes) > 0 {
		prp := repo.NewPostRepo()
		for _, deletedType := range deletedTypes {
			if err := prp.MigrateItemTypeToOther(ctx, deletedType, OtherType); err != nil {
				nlog.Pick().WithContext(ctx).WithError(err).Warn("迁移物品类型数据失败", "type", deletedType)
				return comm.CodeServerError
			}
		}
	}

	if err := scr.UpdateItemTypes(ctx, filteredTypes); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("更新物品类型失败")
		return comm.CodeServerError
	}

	return comm.CodeOK
}

func (a *UpdateItemTypesApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfUpdateItemTypes(ctx *gin.Context) {
	api := &UpdateItemTypesApi{}
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

func UpdateClaimValidityDaysHandler() gin.HandlerFunc {
	api := UpdateClaimValidityDaysApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfUpdateClaimValidityDays).Pointer()).Name()] = api
	return hfUpdateClaimValidityDays
}

type UpdateClaimValidityDaysApi struct {
	Info     struct{} `name:"更新认领时效" desc:"更新认领时效"`
	Request  UpdateClaimValidityDaysApiRequest
	Response struct{}
}

type UpdateClaimValidityDaysApiRequest struct {
	Body struct {
		ClaimValidityDays int `json:"claim_validity_days" binding:"required,min=1" desc:"认领时效天数"`
	}
}

func (a *UpdateClaimValidityDaysApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	req := a.Request.Body
	scr := repo.NewSystemConfigRepo()

	if err := scr.UpdateClaimValidityDays(ctx, req.ClaimValidityDays); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("更新认领时效失败")
		return comm.CodeServerError
	}

	return comm.CodeOK
}

func (a *UpdateClaimValidityDaysApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfUpdateClaimValidityDays(ctx *gin.Context) {
	api := &UpdateClaimValidityDaysApi{}
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

func UpdatePublishLimitHandler() gin.HandlerFunc {
	api := UpdatePublishLimitApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfUpdatePublishLimit).Pointer()).Name()] = api
	return hfUpdatePublishLimit
}

type UpdatePublishLimitApi struct {
	Info     struct{} `name:"更新发布限制" desc:"更新每日发布限制"`
	Request  UpdatePublishLimitApiRequest
	Response struct{}
}

type UpdatePublishLimitApiRequest struct {
	Body struct {
		PublishLimit int `json:"publish_limit" binding:"required,min=1" desc:"每日发布限制"`
	}
}

func (a *UpdatePublishLimitApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	req := a.Request.Body
	scr := repo.NewSystemConfigRepo()

	if err := scr.UpdatePublishLimit(ctx, req.PublishLimit); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("更新发布限制失败")
		return comm.CodeServerError
	}

	return comm.CodeOK
}

func (a *UpdatePublishLimitApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfUpdatePublishLimit(ctx *gin.Context) {
	api := &UpdatePublishLimitApi{}
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

func findDeletedTypes(oldTypes, newTypes []string) []string {
	newTypeSet := make(map[string]bool)
	for _, t := range newTypes {
		newTypeSet[t] = true
	}

	var deleted []string
	for _, t := range oldTypes {
		if !newTypeSet[t] {
			deleted = append(deleted, t)
		}
	}
	return deleted
}

func filterOutOtherType(types []string) []string {
	var filtered []string
	for _, t := range types {
		if t != OtherType {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func IsValidItemType(ctx context.Context, itemType string) bool {
	if itemType == OtherType {
		return true
	}
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
	if feedbackType == OtherType {
		return true
	}
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
