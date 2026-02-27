package tools

import (
	"app/dao/repo"
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type GetMyClaimsInput struct {
	Page     int `json:"page" jsonschema:"description=页码，默认1"`
	PageSize int `json:"page_size" jsonschema:"description=每页数量，默认10"`
}

type GetMyClaimsOutput struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Total   int64       `json:"total,omitempty"`
	Page    int         `json:"page,omitempty"`
}

func getMyClaimsFunc(ctx context.Context, input *GetMyClaimsInput) (*GetMyClaimsOutput, error) {
	tc := GetToolContext(ctx)
	if tc == nil {
		return &GetMyClaimsOutput{Success: false, Message: "工具上下文未初始化"}, nil
	}

	claimRepo := repo.NewClaimRepo()

	page := input.Page
	if page < 1 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	claims, total, err := claimRepo.ListByClaimant(ctx, tc.UserID, offset, pageSize)
	if err != nil {
		return &GetMyClaimsOutput{Success: false, Message: "查询失败"}, nil
	}

	return &GetMyClaimsOutput{
		Success: true,
		Data:    claims,
		Total:   total,
		Page:    page,
	}, nil
}

func NewGetMyClaimsTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"get_my_claims",
		"获取当前用户提交的认领申请列表",
		getMyClaimsFunc,
	)
}
