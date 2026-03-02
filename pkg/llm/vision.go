package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

func DescribeImage(ctx context.Context, imageUrl string) (string, error) {
	visionModel := GetVisionModel()

	parts := []schema.MessageInputPart{
		{
			Type: schema.ChatMessagePartTypeText,
			Text: "请详细描述这张图片的内容，包括物品类型、颜色、形状、大小、特征等关键信息。输出一段中文纯文本，不超过150字。",
		},
		{
			Type: schema.ChatMessagePartTypeImageURL,
			Image: &schema.MessageInputImage{
				MessagePartCommon: schema.MessagePartCommon{
					URL: &imageUrl,
				},
				Detail: schema.ImageURLDetailHigh,
			},
		},
	}

	messages := []*schema.Message{
		{
			Role:                  schema.User,
			UserInputMultiContent: parts,
		},
	}

	resp, err := visionModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("图片描述失败: %w", err)
	}

	return strings.TrimSpace(resp.Content), nil
}

func DescribeImages(ctx context.Context, imageUrls []string) ([]string, error) {
	if len(imageUrls) == 0 {
		return nil, nil
	}

	descriptions := make([]string, len(imageUrls))
	for i, url := range imageUrls {
		desc, err := DescribeImage(ctx, url)
		if err != nil {
			descriptions[i] = "无法识别图片内容"
		} else {
			descriptions[i] = desc
		}
	}

	return descriptions, nil
}

func BuildImageContext(imageUrls []string, descriptions []string) string {
	if len(imageUrls) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n【用户上传的图片】\n")
	sb.WriteString("以下是用户在对话中上传的图片，你可以在工具调用中使用这些URL：\n")

	for i, url := range imageUrls {
		desc := ""
		if i < len(descriptions) && descriptions[i] != "" {
			desc = descriptions[i]
		}
		if desc != "" {
			sb.WriteString(fmt.Sprintf("%d. [描述: %s] URL: %s\n", i+1, desc, url))
		} else {
			sb.WriteString(fmt.Sprintf("%d. URL: %s\n", i+1, url))
		}
	}

	return sb.String()
}
