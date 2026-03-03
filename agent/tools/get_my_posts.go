package tools

import (
	"app/dao/repo"
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/zjutjh/mygo/nlog"
)

type GetMyPostsInput struct {
	PublishType string `json:"publish_type" jsonschema:"description=发布类型筛选（可选）: LOST(寻物), FOUND(招领),enum=,enum=LOST,enum=FOUND"`
	Status      string `json:"status" jsonschema:"description=状态筛选（可选）,enum=,enum=PENDING,enum=APPROVED,enum=SOLVED,enum=CANCELLED,enum=REJECTED"`
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
		nlog.Pick().WithContext(ctx).Warn("[Tool:get_my_posts] 工具上下文未初始化")
		return &GetMyPostsOutput{Success: false, Message: "工具上下文未初始化"}, nil
	}

	nlog.Pick().WithContext(ctx).Infof("[Tool:get_my_posts] 调用参数: user_id=%d, publish_type=%s, status=%s, page=%d, page_size=%d", tc.UserID, input.PublishType, input.Status, input.Page, input.PageSize)

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
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:get_my_posts] 查询失败")
		return &GetMyPostsOutput{Success: false, Message: "查询失败"}, nil
	}

	nlog.Pick().WithContext(ctx).Infof("[Tool:get_my_posts] 查询成功: total=%d, page=%d", total, page)
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
