package tools

import (
	"app/comm/enum"
	"app/dao/repo"
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
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
		return &ReviewClaimOutput{Success: false, Message: "工具上下文未初始化"}, nil
	}

	claimRepo := repo.NewClaimRepo()
	postRepo := repo.NewPostRepo()

	claim, err := claimRepo.FindById(ctx, input.ClaimID)
	if err != nil {
		return &ReviewClaimOutput{Success: false, Message: "查询认领申请失败"}, nil
	}

	if claim == nil {
		return &ReviewClaimOutput{Success: false, Message: "认领申请不存在"}, nil
	}

	post, err := postRepo.FindById(ctx, claim.PostID)
	if err != nil {
		return &ReviewClaimOutput{Success: false, Message: "查询发布记录失败"}, nil
	}

	if post == nil {
		return &ReviewClaimOutput{Success: false, Message: "发布记录不存在"}, nil
	}

	isAdmin := tc.UserType == enum.UserTypeSystemAdmin || tc.UserType == enum.UserTypeAdmin
	isPublisher := post.PublisherID == tc.UserID

	if !isAdmin && !isPublisher {
		return &ReviewClaimOutput{Success: false, Message: "您没有权限审核该认领申请"}, nil
	}

	if claim.Status != enum.ClaimStatusPending {
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
		return &ReviewClaimOutput{Success: false, Message: "更新认领状态失败"}, nil
	}

	if input.Approved {
		postRepo.MarkAsSolved(ctx, claim.PostID)
	}

	return &ReviewClaimOutput{
		Success: true,
		Message: "审核完成",
	}, nil
}

func NewReviewClaimTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"review_claim",
		"帮用户（发布者或管理员）审核认领申请",
		reviewClaimFunc,
	)
}
