package tools

import (
	"app/comm/enum"
	"app/dao/repo"
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type CancelPostInput struct {
	PostID int64  `json:"post_id" jsonschema:"description=发布ID,required"`
	Reason string `json:"reason" jsonschema:"description=取消原因"`
}

type CancelPostOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func cancelPostFunc(ctx context.Context, input *CancelPostInput) (*CancelPostOutput, error) {
	tc := GetToolContext(ctx)
	if tc == nil {
		return &CancelPostOutput{Success: false, Message: "工具上下文未初始化"}, nil
	}

	postRepo := repo.NewPostRepo()
	vectorRepo := repo.NewVectorRepo()

	post, err := postRepo.FindById(ctx, input.PostID)
	if err != nil {
		return &CancelPostOutput{Success: false, Message: "查询发布记录失败"}, nil
	}

	if post == nil {
		return &CancelPostOutput{Success: false, Message: "发布记录不存在"}, nil
	}

	if post.PublisherID != tc.UserID {
		return &CancelPostOutput{Success: false, Message: "您没有权限取消该发布记录"}, nil
	}

	if post.Status != enum.PostStatusApproved {
		return &CancelPostOutput{Success: false, Message: "该发布记录当前状态不允许取消"}, nil
	}

	err = postRepo.CancelPost(ctx, input.PostID, tc.UserID, input.Reason)
	if err != nil {
		return &CancelPostOutput{Success: false, Message: "取消发布失败"}, nil
	}

	vectorRepo.DeletePostVector(ctx, input.PostID)

	return &CancelPostOutput{
		Success: true,
		Message: "发布已取消",
	}, nil
}

func NewCancelPostTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"cancel_post",
		"帮用户取消已通过的发布信息",
		cancelPostFunc,
	)
}
