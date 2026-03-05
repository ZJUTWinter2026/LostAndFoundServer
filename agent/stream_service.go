package agent

import (
	"fmt"
	"io"

	"github.com/cloudwego/eino/schema"
)

// CollectStreamMessages 读取 Eino 流分片，输出稳定事件并收集可落库的标准消息。
func (s *AgentService) CollectStreamMessages(stream *schema.StreamReader[*schema.Message], emit func(StreamEvent)) ([]ChatMessageRecord, error) {
	var fullContent string
	collectedMsgs := make([]ChatMessageRecord, 0, 1)

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			if fullContent != "" {
				collectedMsgs = append(collectedMsgs, ChatMessageRecord{Role: "assistant", Content: fullContent})
			}
			return collectedMsgs, fmt.Errorf("流式读取错误: %w", err)
		}

		if msg.Content != "" {
			emit(StreamEvent{Content: msg.Content})
			fullContent += msg.Content
		}
	}

	if fullContent != "" {
		collectedMsgs = append(collectedMsgs, ChatMessageRecord{
			Role:    "assistant",
			Content: fullContent,
		})
	}

	return collectedMsgs, nil
}
