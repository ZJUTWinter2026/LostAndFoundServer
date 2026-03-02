package tools

import (
	"app/comm/enum"
	"app/dao/model"
	"app/dao/repo"
	"app/pkg/llm"
	"context"
	"sort"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/zjutjh/mygo/nlog"
)

type SearchPostsInput struct {
	Query       string `json:"query" jsonschema:"description=搜索内容，使用自然语言描述,required"`
	PublishType string `json:"publish_type" jsonschema:"description=筛选发布类型: LOST(寻物), FOUND(招领),enum=LOST,enum=FOUND"`
	Campus      string `json:"campus" jsonschema:"description=校区筛选,enum=ZHAO_HUI,enum=PING_FENG,enum=MO_GAN_SHAN"`
	Limit       int    `json:"limit" jsonschema:"description=返回结果数量限制，默认10"`
}

type SearchPostsOutput struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Total   int         `json:"total,omitempty"`
}

func searchPostsFunc(ctx context.Context, input *SearchPostsInput) (*SearchPostsOutput, error) {
	nlog.Pick().WithContext(ctx).Infof("[Tool:search_posts] 调用参数: query=%s, publish_type=%s, campus=%s, limit=%d", input.Query, input.PublishType, input.Campus, input.Limit)

	postRepo := repo.NewPostRepo()
	vectorRepo := repo.NewVectorRepo()

	limit := input.Limit
	if limit < 1 {
		limit = 10
	}

	embedModel := llm.GetEmbeddingModel()
	vectors, err := embedModel.EmbedStrings(ctx, []string{input.Query})
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:search_posts] 向量化失败")
		return &SearchPostsOutput{Success: false, Message: "向量化失败"}, nil
	}

	if len(vectors) == 0 {
		nlog.Pick().WithContext(ctx).Warn("[Tool:search_posts] 向量化返回空结果")
		return &SearchPostsOutput{Success: false, Message: "向量化返回空结果"}, nil
	}

	searchResults, err := vectorRepo.Search(ctx, vectors[0], limit*2)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:search_posts] 向量搜索失败")
		return &SearchPostsOutput{Success: false, Message: "向量搜索失败"}, nil
	}

	// 建立 postID → score 映射，用于后续按相似度排序
	scoreMap := make(map[int64]float32, len(searchResults))
	var postIDs []int64
	for _, result := range searchResults {
		postIDs = append(postIDs, result.PostID)
		scoreMap[result.PostID] = result.Score
	}

	if len(postIDs) == 0 {
		nlog.Pick().WithContext(ctx).Infof("[Tool:search_posts] 未找到匹配结果")
		return &SearchPostsOutput{
			Success: true,
			Data:    []*model.Post{},
			Total:   0,
		}, nil
	}

	posts, err := postRepo.FindByIds(ctx, postIDs)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:search_posts] 查询发布记录失败")
		return &SearchPostsOutput{Success: false, Message: "查询发布记录失败"}, nil
	}

	var filteredPosts []*model.Post
	for _, post := range posts {
		if input.PublishType != "" && post.PublishType != input.PublishType {
			continue
		}
		if input.Campus != "" && post.Campus != input.Campus {
			continue
		}
		if post.Status != enum.PostStatusApproved {
			continue
		}
		filteredPosts = append(filteredPosts, post)
	}

	// 按向量相似度分数降序重排，保留语义搜索的顺序
	sort.Slice(filteredPosts, func(i, j int) bool {
		return scoreMap[filteredPosts[i].ID] > scoreMap[filteredPosts[j].ID]
	})

	if len(filteredPosts) > limit {
		filteredPosts = filteredPosts[:limit]
	}

	nlog.Pick().WithContext(ctx).Infof("[Tool:search_posts] 返回结果: total=%d", len(filteredPosts))
	return &SearchPostsOutput{
		Success: true,
		Data:    filteredPosts,
		Total:   len(filteredPosts),
	}, nil
}

func NewSearchPostsTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"search_posts",
		"通过自然语言query进行向量搜索，查找相关的失物/招领信息",
		searchPostsFunc,
	)
}
