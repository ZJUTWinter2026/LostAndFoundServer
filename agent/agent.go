package agent

import (
	"app/agent/tools"
	"app/pkg/llm"
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type StreamEvent struct {
	Type    string      `json:"type"`
	Content string      `json:"content,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type ToolCallInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolResultInfo struct {
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
	Result     string `json:"result"`
}

type Agent struct {
	reactAgent *react.Agent
	tools      []tool.BaseTool
}

func NewAgent() *Agent {
	toolList := make([]tool.BaseTool, 0, 11)

	if t, err := tools.NewGetPostDetailTool(); err == nil {
		toolList = append(toolList, t)
	}
	if t, err := tools.NewSearchPostsTool(); err == nil {
		toolList = append(toolList, t)
	}
	if t, err := tools.NewGetMyPostsTool(); err == nil {
		toolList = append(toolList, t)
	}
	if t, err := tools.NewGetMyClaimsTool(); err == nil {
		toolList = append(toolList, t)
	}
	if t, err := tools.NewGetMyFeedbacksTool(); err == nil {
		toolList = append(toolList, t)
	}
	if t, err := tools.NewPublishPostTool(); err == nil {
		toolList = append(toolList, t)
	}
	if t, err := tools.NewApplyClaimTool(); err == nil {
		toolList = append(toolList, t)
	}
	if t, err := tools.NewCancelClaimTool(); err == nil {
		toolList = append(toolList, t)
	}
	if t, err := tools.NewReviewClaimTool(); err == nil {
		toolList = append(toolList, t)
	}
	if t, err := tools.NewSubmitFeedbackTool(); err == nil {
		toolList = append(toolList, t)
	}
	if t, err := tools.NewCancelPostTool(); err == nil {
		toolList = append(toolList, t)
	}

	return &Agent{
		tools: toolList,
	}
}

func (a *Agent) getOrCreateReactAgent(ctx context.Context, toolCtx *tools.ToolContext) (*react.Agent, error) {
	if a.reactAgent != nil {
		return a.reactAgent, nil
	}

	chatModel := llm.GetChatModel()

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: a.tools,
		},
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			systemPrompt := buildSystemPrompt(toolCtx)
			result := make([]*schema.Message, 0, len(input)+1)
			result = append(result, schema.SystemMessage(systemPrompt))
			result = append(result, input...)
			return result
		},
		MaxStep: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("创建ReAct Agent失败: %w", err)
	}

	a.reactAgent = agent
	return agent, nil
}

func (a *Agent) Stream(ctx context.Context, messages []ChatMessage, toolCtx *tools.ToolContext) (*schema.StreamReader[*schema.Message], error) {
	ctx = tools.WithToolContext(ctx, toolCtx)

	agent, err := a.getOrCreateReactAgent(ctx, toolCtx)
	if err != nil {
		return nil, err
	}

	schemaMessages := convertMessages(messages)

	stream, err := agent.Stream(ctx, schemaMessages)
	if err != nil {
		return nil, fmt.Errorf("AI对话失败: %w", err)
	}

	return stream, nil
}

func convertMessages(messages []ChatMessage) []*schema.Message {
	var schemaMessages []*schema.Message
	for _, msg := range messages {
		schemaMessages = append(schemaMessages, convertMessage(msg))
	}
	return schemaMessages
}

func convertMessage(msg ChatMessage) *schema.Message {
	switch msg.Role {
	case "user":
		return schema.UserMessage(msg.Content)
	case "assistant":
		return &schema.Message{
			Role:    schema.Assistant,
			Content: msg.Content,
		}
	case "system":
		return schema.SystemMessage(msg.Content)
	default:
		return &schema.Message{
			Role:    schema.RoleType(msg.Role),
			Content: msg.Content,
		}
	}
}

func ParseStreamMessage(msg *schema.Message) StreamEvent {
	switch msg.Role {
	case schema.Assistant:
		if len(msg.ToolCalls) > 0 {
			tc := msg.ToolCalls[0]
			return StreamEvent{
				Type: "tool_call",
				Data: ToolCallInfo{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
		return StreamEvent{
			Type:    "content",
			Content: msg.Content,
		}
	case schema.Tool:
		return StreamEvent{
			Type: "tool_result",
			Data: ToolResultInfo{
				ToolCallID: msg.ToolCallID,
				ToolName:   msg.ToolName,
				Result:     msg.Content,
			},
		}
	default:
		return StreamEvent{
			Type:    "content",
			Content: msg.Content,
		}
	}
}

func buildSystemPrompt(toolCtx *tools.ToolContext) string {
	var sb strings.Builder

	sb.WriteString("你是校园失物招领系统的AI助手，帮助用户处理失物招领相关事务。\n\n")

	sb.WriteString("## 工具调用规范（重要）\n")
	sb.WriteString("在调用任何工具之前，你必须遵循以下原则：\n")
	sb.WriteString("1. **参数完整性检查**：仔细检查工具所需的所有参数是否已经由用户提供。如果缺少必要信息，主动向用户询问，不要猜测或编造参数值。\n")
	sb.WriteString("2. **明确告知用户**：在调用工具前，用中文功能名称和字段名称，清楚说明即将进行的操作及对应填写内容。\n")
	sb.WriteString("3. **征求用户同意**：对于非查询类工具，在用户确认同意后再执行操作，不得在用户不知情的情况下直接调用。\n")
	sb.WriteString("4. **安全优先**：对于涉及数据修改的操作（如发布、认领、审核等），必须确保用户已提供完整且准确的信息。\n\n")

	sb.WriteString("## 可用工具\n")
	sb.WriteString("- get_post_detail: 获取发布详情\n")
	sb.WriteString("- search_posts: 搜索失物/招领信息\n")
	sb.WriteString("- get_my_posts: 获取我的发布列表\n")
	sb.WriteString("- get_my_claims: 获取我的认领申请\n")
	sb.WriteString("- get_my_feedbacks: 获取我的投诉反馈\n")
	sb.WriteString("- publish_post: 发布失物/招领信息\n")
	sb.WriteString("- apply_claim: 申请认领物品\n")
	sb.WriteString("- cancel_claim: 取消认领申请\n")
	sb.WriteString("- review_claim: 审核认领申请\n")
	sb.WriteString("- submit_feedback: 提交投诉反馈\n")
	sb.WriteString("- cancel_post: 取消发布\n\n")

	sb.WriteString("## 当前用户信息\n")
	sb.WriteString(fmt.Sprintf("用户ID: %d\n", toolCtx.UserID))
	sb.WriteString(fmt.Sprintf("用户类型: %s\n", toolCtx.UserType))

	return sb.String()
}
