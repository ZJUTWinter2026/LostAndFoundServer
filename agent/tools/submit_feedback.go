package tools

import (
	"app/dao/model"
	"app/dao/repo"
	"context"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type SubmitFeedbackInput struct {
	PostID  int64  `json:"post_id" jsonschema:"description=发布ID,required"`
	Type    string `json:"type" jsonschema:"description=投诉类型,required"`
	Content string `json:"content" jsonschema:"description=投诉内容,required"`
}

type SubmitFeedbackOutput struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

func submitFeedbackFunc(ctx context.Context, input *SubmitFeedbackInput) (*SubmitFeedbackOutput, error) {
	tc := GetToolContext(ctx)
	if tc == nil {
		return &SubmitFeedbackOutput{Success: false, Message: "工具上下文未初始化"}, nil
	}

	feedbackRepo := repo.NewFeedbackRepo()
	postRepo := repo.NewPostRepo()

	post, err := postRepo.FindById(ctx, input.PostID)
	if err != nil {
		return &SubmitFeedbackOutput{Success: false, Message: "查询发布记录失败"}, nil
	}

	if post == nil {
		return &SubmitFeedbackOutput{Success: false, Message: "发布记录不存在"}, nil
	}

	feedback := &model.Feedback{
		PostID:      input.PostID,
		ReporterID:  tc.UserID,
		Type:        input.Type,
		Description: input.Content,
		Processed:   false,
		ProcessedBy: 0,
		ProcessedAt: time.Now(),
	}

	err = feedbackRepo.Create(ctx, feedback)
	if err != nil {
		return &SubmitFeedbackOutput{Success: false, Message: "提交投诉反馈失败"}, nil
	}

	return &SubmitFeedbackOutput{
		Success: true,
		Message: "投诉反馈提交成功",
		Data: map[string]interface{}{
			"feedback_id": feedback.ID,
		},
	}, nil
}

func NewSubmitFeedbackTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"submit_feedback",
		"帮用户提交投诉反馈",
		submitFeedbackFunc,
	)
}
