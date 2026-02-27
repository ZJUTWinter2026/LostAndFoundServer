package agent

import (
	"app/agent/tools"
	"app/pkg/llm"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type ChatMessage struct {
	Role    string
	Content string
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

func (a *Agent) Chat(ctx context.Context, messages []ChatMessage, toolCtx *tools.ToolContext) (string, error) {
	ctx = tools.WithToolContext(ctx, toolCtx)

	agent, err := a.getOrCreateReactAgent(ctx, toolCtx)
	if err != nil {
		return "", err
	}

	var schemaMessages []*schema.Message
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			schemaMessages = append(schemaMessages, schema.UserMessage(msg.Content))
		case "assistant":
			schemaMessages = append(schemaMessages, &schema.Message{
				Role:    schema.Assistant,
				Content: msg.Content,
			})
		case "system":
			schemaMessages = append(schemaMessages, schema.SystemMessage(msg.Content))
		default:
			schemaMessages = append(schemaMessages, &schema.Message{
				Role:    schema.RoleType(msg.Role),
				Content: msg.Content,
			})
		}
	}

	resp, err := agent.Generate(ctx, schemaMessages)
	if err != nil {
		return "", fmt.Errorf("AI对话失败: %w", err)
	}

	return resp.Content, nil
}

func (a *Agent) Stream(ctx context.Context, messages []ChatMessage, toolCtx *tools.ToolContext) (*schema.StreamReader[*schema.Message], error) {
	ctx = tools.WithToolContext(ctx, toolCtx)

	agent, err := a.getOrCreateReactAgent(ctx, toolCtx)
	if err != nil {
		return nil, err
	}

	var schemaMessages []*schema.Message
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			schemaMessages = append(schemaMessages, schema.UserMessage(msg.Content))
		case "assistant":
			schemaMessages = append(schemaMessages, &schema.Message{
				Role:    schema.Assistant,
				Content: msg.Content,
			})
		case "system":
			schemaMessages = append(schemaMessages, schema.SystemMessage(msg.Content))
		default:
			schemaMessages = append(schemaMessages, &schema.Message{
				Role:    schema.RoleType(msg.Role),
				Content: msg.Content,
			})
		}
	}

	stream, err := agent.Stream(ctx, schemaMessages)
	if err != nil {
		return nil, fmt.Errorf("AI对话失败: %w", err)
	}

	return stream, nil
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

func CollectStreamContent(stream *schema.StreamReader[*schema.Message]) (string, error) {
	var content strings.Builder
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if msg.Role == schema.Assistant && len(msg.ToolCalls) == 0 {
			content.WriteString(msg.Content)
		}
	}
	return content.String(), nil
}

func buildSystemPrompt(toolCtx *tools.ToolContext) string {
	var sb strings.Builder

	sb.WriteString("你是校园失物招领系统的AI助手，帮助用户处理失物招领相关事务。\n\n")
	sb.WriteString("你可以使用以下工具来帮助用户：\n")
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
	sb.WriteString(fmt.Sprintf("当前用户ID: %d\n", toolCtx.UserID))
	sb.WriteString(fmt.Sprintf("用户类型: %s\n", toolCtx.UserType))

	return sb.String()
}
