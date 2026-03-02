package agent

import (
	"app/agent/tools"
	"app/pkg/llm"
	"context"
	"fmt"
	"strings"
	"sync"

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
	mu         sync.Mutex
	reactAgent *react.Agent
	tools      []tool.BaseTool
}

func NewAgent() *Agent {
	toolList := make([]tool.BaseTool, 0, 12)

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
	if t, err := tools.NewGetSystemConfigTool(); err == nil {
		toolList = append(toolList, t)
	}

	return &Agent{
		tools: toolList,
	}
}

func (a *Agent) getOrCreateReactAgent(ctx context.Context) (*react.Agent, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

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
			// 每次请求从 ctx 动态取 toolCtx，避免首次创建时的用户ID被永久锁定
			tc := tools.GetToolContext(ctx)
			systemPrompt := buildSystemPrompt(tc)
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

	agent, err := a.getOrCreateReactAgent(ctx)
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

	sb.WriteString("你是校园失物招领系统的AI助手，帮助用户处理失物招领相关事务。\n")
	sb.WriteString("你的服务对象是校园里的学生和教师，即普通用户，而不是管理员。\n\n")

	sb.WriteString("## 能力边界说明（重要）\n")
	sb.WriteString("1. 你只能在系统已提供的功能范围内协助用户，不得虚构系统能力。\n")
	sb.WriteString("2. 不得承诺线下处理、人工干预、系统外部联系或任何未在可用工具中定义的操作。\n")
	sb.WriteString("3. 如果用户提出超出系统功能范围的请求，应明确告知当前系统不支持该操作，而不是编造解决方案。\n")
	sb.WriteString("4. 不得保证一定找回物品、一定通过审核等结果性承诺。\n")
	sb.WriteString("5. 你不具备管理员权限，不能执行管理员专属操作。\n\n")

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
	sb.WriteString("- cancel_post: 取消发布\n")
	sb.WriteString("- get_system_config: 获取系统配置（物品类型、反馈类型等）\n\n")

	sb.WriteString("## 当前用户信息\n")
	sb.WriteString(fmt.Sprintf("用户ID: %d\n", toolCtx.UserID))

	return sb.String()
}
