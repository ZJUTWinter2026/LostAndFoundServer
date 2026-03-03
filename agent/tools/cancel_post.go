package tools

import (
	"app/comm/enum"
	"app/dao/repo"
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/zjutjh/mygo/nlog"
)

type CancelPostInput struct {
	PostID int64  `json:"post_id" jsonschema:"description=发布ID,required"`
	Reason string `json:"reason" jsonschema:"description=取消原因（仅帖子是APPROVED状态才需要填）"`
}

type CancelPostOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func cancelPostFunc(ctx context.Context, input *CancelPostInput) (*CancelPostOutput, error) {
	tc := GetToolContext(ctx)
	if tc == nil {
		nlog.Pick().WithContext(ctx).Warn("[Tool:cancel_post] 工具上下文未初始化")
		return &CancelPostOutput{Success: false, Message: "工具上下文未初始化"}, nil
	}

	nlog.Pick().WithContext(ctx).Infof("[Tool:cancel_post] 调用参数: user_id=%d, post_id=%d, reason=%s", tc.UserID, input.PostID, input.Reason)

	postRepo := repo.NewPostRepo()
	vectorRepo := repo.NewVectorRepo()

	post, err := postRepo.FindById(ctx, input.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:cancel_post] 查询发布记录失败")
		return &CancelPostOutput{Success: false, Message: "查询发布记录失败"}, nil
	}

	if post.PublisherID != tc.UserID {
		nlog.Pick().WithContext(ctx).Infof("[Tool:cancel_post] 用户没有权限取消该发布记录: user_id=%d, post_id=%d, publisher_id=%d", tc.UserID, input.PostID, post.PublisherID)
		return &CancelPostOutput{Success: false, Message: "您没有权限取消该发布记录"}, nil
	}

	if post.Status != enum.PostStatusApproved && post.Status != enum.PostStatusPending {
		nlog.Pick().WithContext(ctx).Infof("[Tool:cancel_post] 发布记录状态不允许取消: post_id=%d, status=%s", input.PostID, post.Status)
		return &CancelPostOutput{Success: false, Message: "该发布记录当前状态不允许取消"}, nil
	}

	if post.Status == enum.PostStatusApproved {
		err = postRepo.CancelPost(ctx, input.PostID, tc.UserID, input.Reason)
	} else {
		err = postRepo.DeletePost(ctx, input.PostID, tc.UserID)
	}
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:cancel_post] 取消发布失败")
		return &CancelPostOutput{Success: false, Message: "取消发布失败"}, nil
	}

	err = vectorRepo.Delete(ctx, input.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:cancel_post] 删除向量索引失败")
	}

	nlog.Pick().WithContext(ctx).Infof("[Tool:cancel_post] 发布已取消: post_id=%d, user_id=%d", input.PostID, tc.UserID)
	return &CancelPostOutput{
		Success: true,
		Message: "发布已取消",
	}, nil
}

func NewCancelPostTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"cancel_post",
		"帮用户取消发布信息（支持待审核和已通过状态）",
		cancelPostFunc,
	)
}
