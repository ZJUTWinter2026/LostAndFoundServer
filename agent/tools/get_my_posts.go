package tools

import (
	"app/dao/repo"
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type GetMyPostsInput struct {
	PublishType string `json:"publish_type" jsonschema:"description=发布类型筛选: LOST(寻物), FOUND(招领)"`
	Status      string `json:"status" jsonschema:"description=状态筛选"`
	Page        int    `json:"page" jsonschema:"description=页码，默认1"`
	PageSize    int    `json:"page_size" jsonschema:"description=每页数量，默认10"`
}

type GetMyPostsOutput struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Total   int64       `json:"total,omitempty"`
	Page    int         `json:"page,omitempty"`
}

func getMyPostsFunc(ctx context.Context, input *GetMyPostsInput) (*GetMyPostsOutput, error) {
	tc := GetToolContext(ctx)
	if tc == nil {
		return &GetMyPostsOutput{Success: false, Message: "工具上下文未初始化"}, nil
	}

	postRepo := repo.NewPostRepo()

	page := input.Page
	if page < 1 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	posts, total, err := postRepo.ListByPublisher(ctx, tc.UserID, input.PublishType, input.Status, offset, pageSize)
	if err != nil {
		return &GetMyPostsOutput{Success: false, Message: "查询失败"}, nil
	}

	return &GetMyPostsOutput{
		Success: true,
		Data:    posts,
		Total:   total,
		Page:    page,
	}, nil
}

func NewGetMyPostsTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"get_my_posts",
		"获取当前用户发布的失物/招领信息列表",
		getMyPostsFunc,
	)
}
