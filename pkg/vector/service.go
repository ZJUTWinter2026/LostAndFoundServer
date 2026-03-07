package vector

import (
	"app/dao/model"
	"app/dao/repo"
	"app/pkg/llm"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/schema"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) UpdatePostVector(ctx context.Context, post *model.Post) error {
	summary, err := s.generateSummary(ctx, post)
	if err != nil {
		return fmt.Errorf("生成总结失败: %w", err)
	}

	postRepo := repo.NewPostRepo()
	if err := postRepo.UpdateSummary(ctx, post.ID, summary); err != nil {
		return fmt.Errorf("更新总结失败: %w", err)
	}

	embedModel := llm.GetEmbeddingModel()
	vectors, err := embedModel.EmbedStrings(ctx, []string{summary})
	if err != nil {
		return fmt.Errorf("向量化失败: %w", err)
	}

	if len(vectors) == 0 {
		return fmt.Errorf("向量化返回空结果")
	}

	vectorRepo := repo.NewVectorRepo()
	if err := vectorRepo.Update(ctx, post.ID, vectors[0]); err != nil {
		return fmt.Errorf("更新向量失败: %w", err)
	}

	return nil
}

func (s *Service) generateSummary(ctx context.Context, post *model.Post) (string, error) {
	var sb strings.Builder

	if post.PublishType == "LOST" {
		sb.WriteString("【寻物启事】\n")
	} else {
		sb.WriteString("【招领启事】\n")
	}

	sb.WriteString(fmt.Sprintf("物品名称: %s\n", post.ItemName))
	sb.WriteString(fmt.Sprintf("物品类型: %s\n", post.ItemType))
	sb.WriteString(fmt.Sprintf("校区: %s\n", post.Campus))
	sb.WriteString(fmt.Sprintf("地点: %s\n", post.Location))
	sb.WriteString(fmt.Sprintf("存放地点: %s\n", post.StorageLocation))
	sb.WriteString(fmt.Sprintf("事件时间: %s\n", post.EventTime.Format(time.DateTime)))
	sb.WriteString(fmt.Sprintf("物品特征: %s\n", post.Features))

	if post.HasReward {
		sb.WriteString(fmt.Sprintf("悬赏说明: %s\n", post.RewardDescription))
	}

	sb.WriteString("\n请根据以上信息生成一段简洁的纯文本总结，包含时间、地点、物品特征等关键信息，用于语义搜索。如果有图片，请结合图片内容进行分析。只描述客观信息，不要添加任何推测或额外说明。")

	var imageUrls []string
	if post.Images != "" {
		_ = sonic.UnmarshalString(post.Images, &imageUrls)
	}

	var parts []schema.MessageInputPart
	parts = append(parts, schema.MessageInputPart{
		Type: schema.ChatMessagePartTypeText,
		Text: sb.String(),
	})

	for _, url := range imageUrls {
		parts = append(parts, schema.MessageInputPart{
			Type: schema.ChatMessagePartTypeImageURL,
			Image: &schema.MessageInputImage{
				MessagePartCommon: schema.MessagePartCommon{
					URL: &url,
				},
				Detail: schema.ImageURLDetailHigh,
			},
		})
	}

	messages := []*schema.Message{
		schema.SystemMessage(`你是一个失物招领信息总结助手。你的任务是根据用户提供的信息（文字和图片）生成一段简洁的物品特征总结，用于语义搜索匹配。

规则：
1. 只使用用户提供的信息，不得添加任何推测或分析。
2. 未提供的信息直接忽略，不要猜测。
3. 输出为简洁的纯文本描述。

重点描述：
- 物品名称
- 颜色 / 外观
- 明显特征
- 附件或装饰

不要包含：
- 寻找或监控建议
- 信息真实性说明
- “可能”“建议”“推测”等表述
`),
		{
			Role:                  schema.User,
			UserInputMultiContent: parts,
		},
	}

	visionModel := llm.GetVisionModel()
	resp, err := visionModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("生成总结失败: %w", err)
	}

	return strings.TrimSpace(resp.Content), nil
}
