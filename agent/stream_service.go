package agent

import (
	"fmt"
	"io"

	"github.com/cloudwego/eino/schema"
)

// CollectStreamMessages 读取 Eino 流分片，输出稳定事件并收集可落库的标准消息。
func (s *AgentService) CollectStreamMessages(stream *schema.StreamReader[*schema.Message], emit func(StreamEvent)) ([]ChatMessageRecord, error) {
	var fullContent string
	pendingToolCalls := make(map[int]*ToolCallInfo)
	pendingToolCallOrder := make([]int, 0, 4)
	collectedMsgs := make([]ChatMessageRecord, 0, 8)
	currentToolCalls := make([]ToolCallInfo, 0, 4)

	flushPendingToolCalls := func() {
		if len(pendingToolCallOrder) == 0 {
			return
		}
		for _, idx := range pendingToolCallOrder {
			tc := pendingToolCalls[idx]
			if tc == nil || tc.ID == "" || tc.Name == "" {
				continue
			}
			emit(StreamEvent{Type: "tool_call", Data: *tc})
			currentToolCalls = append(currentToolCalls, *tc)
		}
		pendingToolCalls = make(map[int]*ToolCallInfo)
		pendingToolCallOrder = pendingToolCallOrder[:0]
	}

	flushCurrentToolCallMsg := func() {
		if len(currentToolCalls) == 0 {
			return
		}
		collectedMsgs = append(collectedMsgs, ChatMessageRecord{
			Role:      "assistant",
			ToolCalls: append([]ToolCallInfo(nil), currentToolCalls...),
		})
		currentToolCalls = currentToolCalls[:0]
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return collectedMsgs, fmt.Errorf("流式读取错误: %w", err)
		}

		switch msg.Role {
		case schema.Assistant:
			hasToolCalls := len(msg.ToolCalls) > 0
			if hasToolCalls {
				for i, tc := range msg.ToolCalls {
					idx := i
					if tc.Index != nil {
						idx = *tc.Index
					}

					pending, exists := pendingToolCalls[idx]
					if !exists {
						pending = &ToolCallInfo{}
						pendingToolCalls[idx] = pending
						pendingToolCallOrder = append(pendingToolCallOrder, idx)
					}

					if tc.ID != "" {
						pending.ID = tc.ID
					}
					if tc.Function.Name != "" {
						pending.Name = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						pending.Arguments += tc.Function.Arguments
					}
				}
			}

			if msg.Content != "" {
				if !hasToolCalls {
					flushPendingToolCalls()
				}
				emit(StreamEvent{Type: "content", Content: msg.Content})
				fullContent += msg.Content
			} else if !hasToolCalls {
				flushPendingToolCalls()
			}

		case schema.Tool:
			flushPendingToolCalls()
			flushCurrentToolCallMsg()
			emit(StreamEvent{
				Type: "tool_result",
				Data: ToolResultInfo{
					ToolCallID: msg.ToolCallID,
					ToolName:   msg.ToolName,
					Result:     msg.Content,
				},
			})
			collectedMsgs = append(collectedMsgs, ChatMessageRecord{
				Role:       "tool",
				Content:    msg.Content,
				ToolCallID: msg.ToolCallID,
				ToolName:   msg.ToolName,
			})

		default:
			if msg.Content != "" {
				flushPendingToolCalls()
				emit(StreamEvent{Type: "content", Content: msg.Content})
				fullContent += msg.Content
			}
		}
	}

	flushPendingToolCalls()
	flushCurrentToolCallMsg()

	if fullContent != "" {
		collectedMsgs = append(collectedMsgs, ChatMessageRecord{
			Role:    "assistant",
			Content: fullContent,
		})
	}

	return collectedMsgs, nil
}
