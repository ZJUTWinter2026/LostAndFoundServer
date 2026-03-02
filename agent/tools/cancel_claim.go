package tools

import (
	"app/comm/enum"
	"app/dao/repo"
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/zjutjh/mygo/nlog"
)

type CancelClaimInput struct {
	ClaimID int64 `json:"claim_id" jsonschema:"description=认领申请ID,required"`
}

type CancelClaimOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func cancelClaimFunc(ctx context.Context, input *CancelClaimInput) (*CancelClaimOutput, error) {
	tc := GetToolContext(ctx)
	if tc == nil {
		nlog.Pick().WithContext(ctx).Warn("[Tool:cancel_claim] 工具上下文未初始化")
		return &CancelClaimOutput{Success: false, Message: "工具上下文未初始化"}, nil
	}

	nlog.Pick().WithContext(ctx).Infof("[Tool:cancel_claim] 调用参数: user_id=%d, claim_id=%d", tc.UserID, input.ClaimID)

	claimRepo := repo.NewClaimRepo()
	postRepo := repo.NewPostRepo()
	claim, err := claimRepo.FindById(ctx, input.ClaimID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:cancel_claim] 查询认领申请失败")
		return &CancelClaimOutput{Success: false, Message: "查询认领申请失败"}, nil
	}

	if claim == nil {
		nlog.Pick().WithContext(ctx).Infof("[Tool:cancel_claim] 认领申请不存在: claim_id=%d", input.ClaimID)
		return &CancelClaimOutput{Success: false, Message: "认领申请不存在"}, nil
	}

	if claim.ClaimantID != tc.UserID {
		nlog.Pick().WithContext(ctx).Infof("[Tool:cancel_claim] 用户没有权限取消该认领申请: user_id=%d, claim_id=%d, claimant_id=%d", tc.UserID, input.ClaimID, claim.ClaimantID)
		return &CancelClaimOutput{Success: false, Message: "您没有权限取消该认领申请"}, nil
	}

	if claim.Status != enum.ClaimStatusPending {
		nlog.Pick().WithContext(ctx).Infof("[Tool:cancel_claim] 认领申请状态不允许取消: claim_id=%d, status=%s", input.ClaimID, claim.Status)
		return &CancelClaimOutput{Success: false, Message: "该认领申请当前状态不允许取消"}, nil
	}

	err = claimRepo.Delete(ctx, input.ClaimID, tc.UserID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:cancel_claim] 取消认领申请失败")
		return &CancelClaimOutput{Success: false, Message: "取消认领申请失败"}, nil
	}

	// 认领取消后递减帖子的认领计数
	_ = postRepo.DecrementClaimCount(ctx, claim.PostID)

	nlog.Pick().WithContext(ctx).Infof("[Tool:cancel_claim] 认领申请已取消: claim_id=%d, user_id=%d", input.ClaimID, tc.UserID)
	return &CancelClaimOutput{
		Success: true,
		Message: "认领申请已取消",
	}, nil
}

func NewCancelClaimTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"cancel_claim",
		"帮用户取消认领申请",
		cancelClaimFunc,
	)
}
