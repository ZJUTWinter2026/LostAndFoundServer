package tools

import (
	"app/comm/enum"
	"app/dao/model"
	"app/dao/repo"
	"context"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type ApplyClaimInput struct {
	PostID      int64    `json:"post_id" jsonschema:"description=发布ID,required"`
	Description string   `json:"description" jsonschema:"description=认领说明,描述为什么这是你的物品,required"`
	ProofImages []string `json:"proof_images" jsonschema:"description=证明图片URL列表"`
}

type ApplyClaimOutput struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

func applyClaimFunc(ctx context.Context, input *ApplyClaimInput) (*ApplyClaimOutput, error) {
	tc := GetToolContext(ctx)
	if tc == nil {
		return &ApplyClaimOutput{Success: false, Message: "工具上下文未初始化"}, nil
	}

	claimRepo := repo.NewClaimRepo()
	postRepo := repo.NewPostRepo()

	post, err := postRepo.FindById(ctx, input.PostID)
	if err != nil {
		return &ApplyClaimOutput{Success: false, Message: "查询发布记录失败"}, nil
	}

	if post == nil {
		return &ApplyClaimOutput{Success: false, Message: "发布记录不存在"}, nil
	}

	if post.PublisherID == tc.UserID {
		return &ApplyClaimOutput{Success: false, Message: "不能认领自己发布的物品"}, nil
	}

	if post.Status != enum.PostStatusApproved {
		return &ApplyClaimOutput{Success: false, Message: "该发布记录当前状态不允许认领"}, nil
	}

	hasPending, err := claimRepo.HasPendingOrMatchedClaim(ctx, input.PostID, tc.UserID)
	if err != nil {
		return &ApplyClaimOutput{Success: false, Message: "检查认领状态失败"}, nil
	}

	if hasPending {
		return &ApplyClaimOutput{Success: false, Message: "您已有待确认或已匹配的认领申请"}, nil
	}

	var proofImagesJSON string
	if len(input.ProofImages) > 0 {
		proofImagesJSON, _ = sonic.MarshalString(input.ProofImages)
	}

	claim := &model.Claim{
		PostID:      input.PostID,
		ClaimantID:  tc.UserID,
		Description: input.Description,
		ProofImages: proofImagesJSON,
		Status:      enum.ClaimStatusPending,
	}

	err = claimRepo.Create(ctx, claim)
	if err != nil {
		return &ApplyClaimOutput{Success: false, Message: "创建认领申请失败"}, nil
	}

	_ = postRepo.IncrementClaimCount(ctx, input.PostID)

	return &ApplyClaimOutput{
		Success: true,
		Message: "认领申请提交成功",
		Data: map[string]interface{}{
			"claim_id": claim.ID,
		},
	}, nil
}

func NewApplyClaimTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"apply_claim",
		"帮用户申请认领物品",
		applyClaimFunc,
	)
}
