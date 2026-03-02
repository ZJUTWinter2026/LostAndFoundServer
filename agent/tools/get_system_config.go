package tools

import (
	"app/dao/repo"
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/zjutjh/mygo/nlog"
)

type GetSystemConfigInput struct{}

type GetSystemConfigOutput struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

func getSystemConfigFunc(ctx context.Context, input *GetSystemConfigInput) (*GetSystemConfigOutput, error) {
	nlog.Pick().WithContext(ctx).Info("[Tool:get_system_config] 调用参数: 无")

	configRepo := repo.NewSystemConfigRepo()

	result := make(map[string]interface{})

	feedbackTypes, err := configRepo.GetFeedbackTypes(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:get_system_config] 获取反馈类型失败")
		return &GetSystemConfigOutput{Success: false, Message: "获取反馈类型失败"}, nil
	}
	result["feedback_types"] = feedbackTypes

	itemTypes, err := configRepo.GetItemTypes(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:get_system_config] 获取物品类型失败")
		return &GetSystemConfigOutput{Success: false, Message: "获取物品类型失败"}, nil
	}
	result["item_types"] = itemTypes

	claimValidityDays, err := configRepo.GetClaimValidityDays(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:get_system_config] 获取认领有效期失败")
		return &GetSystemConfigOutput{Success: false, Message: "获取认领有效期失败"}, nil
	}
	result["claim_validity_days"] = claimValidityDays

	publishLimit, err := configRepo.GetPublishLimit(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:get_system_config] 获取发布限制失败")
		return &GetSystemConfigOutput{Success: false, Message: "获取发布限制失败"}, nil
	}
	result["publish_limit"] = publishLimit

	nlog.Pick().WithContext(ctx).Infof("[Tool:get_system_config] 获取配置成功: feedback_types=%d, item_types=%d", len(feedbackTypes), len(itemTypes))
	return &GetSystemConfigOutput{
		Success: true,
		Data:    result,
	}, nil
}

func NewGetSystemConfigTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"get_system_config",
		"获取系统配置，包括feedback_types(反馈类型列表)、item_types(物品类型列表)、claim_validity_days(认领有效期天数)、publish_limit(每日发布限制)。发布物品或提交反馈前应先调用此工具获取可选值",
		getSystemConfigFunc,
	)
}
