package tools

import (
	"app/comm/enum"
	"app/dao/repo"
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
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
		return &CancelClaimOutput{Success: false, Message: "工具上下文未初始化"}, nil
	}

	claimRepo := repo.NewClaimRepo()

	claim, err := claimRepo.FindById(ctx, input.ClaimID)
	if err != nil {
		return &CancelClaimOutput{Success: false, Message: "查询认领申请失败"}, nil
	}

	if claim == nil {
		return &CancelClaimOutput{Success: false, Message: "认领申请不存在"}, nil
	}

	if claim.ClaimantID != tc.UserID {
		return &CancelClaimOutput{Success: false, Message: "您没有权限取消该认领申请"}, nil
	}

	if claim.Status != enum.ClaimStatusPending {
		return &CancelClaimOutput{Success: false, Message: "该认领申请当前状态不允许取消"}, nil
	}

	err = claimRepo.Delete(ctx, input.ClaimID, tc.UserID)
	if err != nil {
		return &CancelClaimOutput{Success: false, Message: "取消认领申请失败"}, nil
	}

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
