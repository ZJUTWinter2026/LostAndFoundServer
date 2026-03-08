package tools

import (
	"app/dao/repo"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/zjutjh/mygo/nlog"
)

type GetReceivedClaimsInput struct {
	PostID   int64  `json:"post_id" jsonschema:"description=发布ID筛选（可选）"`
	Status   string `json:"status" jsonschema:"description=状态筛选（可选）,enum=,enum=PENDING,enum=MATCHED,enum=REJECTED"`
	Page     int    `json:"page" jsonschema:"description=页码，默认1"`
	PageSize int    `json:"page_size" jsonschema:"description=每页数量，默认10"`
}

type GetReceivedClaimsOutput struct {
	Success bool                `json:"success"`
	Message string              `json:"message,omitempty"`
	Data    []ReceivedClaimItem `json:"data,omitempty"`
	Total   int64               `json:"total,omitempty"`
	Page    int                 `json:"page,omitempty"`
}

type ReceivedClaimItem struct {
	ClaimID     int64     `json:"claim_id"`
	PostID      int64     `json:"post_id"`
	ItemName    string    `json:"item_name"`
	ClaimantID  int64     `json:"claimant_id"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

func getReceivedClaimsFunc(ctx context.Context, input *GetReceivedClaimsInput) (*GetReceivedClaimsOutput, error) {
	tc := GetToolContext(ctx)
	if tc == nil {
		nlog.Pick().WithContext(ctx).Warn("[Tool:get_received_claims] 工具上下文未初始化")
		return &GetReceivedClaimsOutput{Success: false, Message: "工具上下文未初始化"}, nil
	}

	nlog.Pick().WithContext(ctx).Infof("[Tool:get_received_claims] 调用参数: user_id=%d, post_id=%d, status=%s, page=%d, page_size=%d", tc.UserID, input.PostID, input.Status, input.Page, input.PageSize)

	page := input.Page
	if page < 1 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	claimRepo := repo.NewClaimRepo()
	postRepo := repo.NewPostRepo()
	claims, total, err := claimRepo.ListReceivedByPublisher(ctx, tc.UserID, input.PostID, input.Status, offset, pageSize)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:get_received_claims] 查询失败")
		return &GetReceivedClaimsOutput{Success: false, Message: "查询失败"}, nil
	}

	items := make([]ReceivedClaimItem, 0, len(claims))
	for _, claim := range claims {
		itemName := ""
		post, err := postRepo.FindById(ctx, claim.PostID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				err = nil
			} else {
				nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:get_received_claims] 查询关联发布失败")
				return &GetReceivedClaimsOutput{Success: false, Message: "查询关联发布失败"}, nil
			}
		}
		if err == nil && post != nil {
			itemName = post.ItemName
		}

		items = append(items, ReceivedClaimItem{
			ClaimID:     claim.ID,
			PostID:      claim.PostID,
			ItemName:    itemName,
			ClaimantID:  claim.ClaimantID,
			Description: claim.Description,
			Status:      claim.Status,
			CreatedAt:   claim.CreatedAt,
		})
	}

	return &GetReceivedClaimsOutput{
		Success: true,
		Data:    items,
		Total:   total,
		Page:    page,
	}, nil
}

func NewGetReceivedClaimsTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"get_received_claims",
		"获取当前用户作为发布者收到的认领申请列表",
		getReceivedClaimsFunc,
	)
}
