package tools

import (
	"app/dao/repo"
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type GetMyFeedbacksInput struct {
	Page     int `json:"page" jsonschema:"description=页码，默认1"`
	PageSize int `json:"page_size" jsonschema:"description=每页数量，默认10"`
}

type GetMyFeedbacksOutput struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Total   int64       `json:"total,omitempty"`
	Page    int         `json:"page,omitempty"`
}

func getMyFeedbacksFunc(ctx context.Context, input *GetMyFeedbacksInput) (*GetMyFeedbacksOutput, error) {
	tc := GetToolContext(ctx)
	if tc == nil {
		return &GetMyFeedbacksOutput{Success: false, Message: "工具上下文未初始化"}, nil
	}

	feedbackRepo := repo.NewFeedbackRepo()

	page := input.Page
	if page < 1 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	feedbacks, total, err := feedbackRepo.ListByReporter(ctx, tc.UserID, offset, pageSize)
	if err != nil {
		return &GetMyFeedbacksOutput{Success: false, Message: "查询失败"}, nil
	}

	return &GetMyFeedbacksOutput{
		Success: true,
		Data:    feedbacks,
		Total:   total,
		Page:    page,
	}, nil
}

func NewGetMyFeedbacksTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"get_my_feedbacks",
		"获取当前用户提交的投诉反馈列表",
		getMyFeedbacksFunc,
	)
}
