package tools

import (
	"app/dao/repo"
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type GetPostDetailInput struct {
	PostID int64 `json:"post_id" jsonschema:"description=发布ID,required"`
}

type GetPostDetailOutput struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func getPostDetailFunc(ctx context.Context, input *GetPostDetailInput) (*GetPostDetailOutput, error) {
	postRepo := repo.NewPostRepo()

	post, err := postRepo.FindById(ctx, input.PostID)
	if err != nil {
		return &GetPostDetailOutput{Success: false, Message: "查询发布记录失败"}, nil
	}

	if post == nil {
		return &GetPostDetailOutput{Success: false, Message: "发布记录不存在"}, nil
	}

	return &GetPostDetailOutput{
		Success: true,
		Data:    post,
	}, nil
}

func NewGetPostDetailTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"get_post_detail",
		"根据post_id获取失物/招领信息的详细内容",
		getPostDetailFunc,
	)
}
