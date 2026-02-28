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

	sb.WriteString("\n请生成一段简洁的总结文本，包含时间、地点、物品特征等关键信息，便于语义搜索匹配。")

	var imageUrls []string
	if post.Images != "" {
		_ = sonic.UnmarshalString(post.Images, &imageUrls)
	}

	userMsg := s.buildUserMessage(sb.String(), imageUrls)

	messages := []*schema.Message{
		schema.SystemMessage("你是一个专业的失物招领信息总结助手。请根据用户提供的失物/招领信息（可能包含图片），生成一段简洁准确的总结文本，用于后续的语义搜索匹配。总结应包含时间、地点、物品特征等关键信息。如果提供了图片，请结合图片内容进行分析。"),
		userMsg,
	}

	chatModel := llm.GetChatModel()
	resp, err := chatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("生成总结失败: %w", err)
	}

	return strings.TrimSpace(resp.Content), nil
}

func (s *Service) buildUserMessage(text string, imageUrls []string) *schema.Message {
	if len(imageUrls) == 0 {
		return schema.UserMessage(text)
	}

	parts := make([]schema.MessageInputPart, 0, len(imageUrls)+1)
	parts = append(parts, schema.MessageInputPart{
		Type: schema.ChatMessagePartTypeText,
		Text: text,
	})

	for _, url := range imageUrls {
		parts = append(parts, schema.MessageInputPart{
			Type: schema.ChatMessagePartTypeImageURL,
			Image: &schema.MessageInputImage{
				MessagePartCommon: schema.MessagePartCommon{
					URL: &url,
				},
			},
		})
	}

	return &schema.Message{
		Role:                   schema.User,
		UserInputMultiContent: parts,
	}
}
