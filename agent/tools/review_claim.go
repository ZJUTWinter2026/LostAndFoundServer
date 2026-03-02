package tools

import (
	"app/comm/enum"
	"app/dao/repo"
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/zjutjh/mygo/nlog"
)

type ReviewClaimInput struct {
	ClaimID  int64 `json:"claim_id" jsonschema:"description=认领申请ID,required"`
	Approved bool  `json:"approved" jsonschema:"description=是否同意认领,required"`
}

type ReviewClaimOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func reviewClaimFunc(ctx context.Context, input *ReviewClaimInput) (*ReviewClaimOutput, error) {
	tc := GetToolContext(ctx)
	if tc == nil {
		nlog.Pick().WithContext(ctx).Warn("[Tool:review_claim] 工具上下文未初始化")
		return &ReviewClaimOutput{Success: false, Message: "工具上下文未初始化"}, nil
	}

	nlog.Pick().WithContext(ctx).Infof("[Tool:review_claim] 调用参数: user_id=%d, claim_id=%d, approved=%v", tc.UserID, input.ClaimID, input.Approved)

	claimRepo := repo.NewClaimRepo()
	postRepo := repo.NewPostRepo()

	claim, err := claimRepo.FindById(ctx, input.ClaimID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:review_claim] 查询认领申请失败")
		return &ReviewClaimOutput{Success: false, Message: "查询认领申请失败"}, nil
	}

	if claim == nil {
		nlog.Pick().WithContext(ctx).Infof("[Tool:review_claim] 认领申请不存在: claim_id=%d", input.ClaimID)
		return &ReviewClaimOutput{Success: false, Message: "认领申请不存在"}, nil
	}

	post, err := postRepo.FindById(ctx, claim.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:review_claim] 查询发布记录失败")
		return &ReviewClaimOutput{Success: false, Message: "查询发布记录失败"}, nil
	}

	if post == nil {
		nlog.Pick().WithContext(ctx).Infof("[Tool:review_claim] 发布记录不存在: post_id=%d", claim.PostID)
		return &ReviewClaimOutput{Success: false, Message: "发布记录不存在"}, nil
	}

	isPublisher := post.PublisherID == tc.UserID
	if !isPublisher {
		nlog.Pick().WithContext(ctx).Infof("[Tool:review_claim] 用户没有权限审核该认领申请: user_id=%d, claim_id=%d, publisher_id=%d", tc.UserID, input.ClaimID, post.PublisherID)
		return &ReviewClaimOutput{Success: false, Message: "您没有权限审核该认领申请"}, nil
	}

	if claim.Status != enum.ClaimStatusPending {
		nlog.Pick().WithContext(ctx).Infof("[Tool:review_claim] 认领申请状态不允许审核: claim_id=%d, status=%s", input.ClaimID, claim.Status)
		return &ReviewClaimOutput{Success: false, Message: "该认领申请当前状态不允许审核"}, nil
	}

	var newStatus string
	if input.Approved {
		newStatus = enum.ClaimStatusMatched
	} else {
		newStatus = enum.ClaimStatusRejected
	}

	err = claimRepo.UpdateStatus(ctx, input.ClaimID, newStatus, tc.UserID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:review_claim] 更新认领状态失败")
		return &ReviewClaimOutput{Success: false, Message: "更新认领状态失败"}, nil
	}

	if input.Approved {
		_ = postRepo.MarkAsSolved(ctx, claim.PostID)
		// 批量拒绝同一帖子下其他待处理的认领申请
		if err := claimRepo.RejectOtherPendingClaims(ctx, claim.PostID, input.ClaimID); err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:review_claim] 批量拒绝其他认领申请失败")
		}
	}

	nlog.Pick().WithContext(ctx).Infof("[Tool:review_claim] 审核完成: claim_id=%d, approved=%v, new_status=%s", input.ClaimID, input.Approved, newStatus)
	return &ReviewClaimOutput{
		Success: true,
		Message: "审核完成",
	}, nil
}

func NewReviewClaimTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"review_claim",
		"帮用户（发布者）审核认领申请",
		reviewClaimFunc,
	)
}
