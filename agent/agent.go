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
	"github.com/zjutjh/mygo/nlog"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type StreamEvent struct {
	EventID string `json:"event_id,omitempty"`
	Seq     int    `json:"seq,omitempty"`
	TS      int64  `json:"ts,omitempty"`
	Content string `json:"content,omitempty"`
}

type Agent struct {
	mu         sync.Mutex
	reactAgent *react.Agent
	tools      []tool.BaseTool
}

// NewAgent 创建 Agent 实例并注册可用工具。
func NewAgent() *Agent {
	toolList := make([]tool.BaseTool, 0, 12)
	toolFuncList := []func() (tool.InvokableTool, error){
		tools.NewGetPostDetailTool,
		tools.NewSearchPostsTool,
		tools.NewGetMyPostsTool,
		tools.NewGetMyClaimsTool,
		tools.NewGetMyFeedbacksTool,
		tools.NewPublishPostTool,
		tools.NewApplyClaimTool,
		tools.NewCancelClaimTool,
		tools.NewReviewClaimTool,
		tools.NewSubmitFeedbackTool,
		tools.NewCancelPostTool,
	}

	for _, tf := range toolFuncList {
		t, err := tf()
		if err != nil {
			nlog.Pick().WithError(err).Warn("注册工具失败")
			continue
		}
		toolList = append(toolList, t)
	}

	return &Agent{
		tools: toolList,
	}
}

// getOrCreateReactAgent 懒加载并复用底层 ReAct Agent。
func (a *Agent) getOrCreateReactAgent(ctx context.Context) (*react.Agent, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.reactAgent != nil {
		return a.reactAgent, nil
	}

	chatModel := llm.GetChatModel()

	// 每次 LLM 调用前，统一注入完整 system prompt（静态守则 + 动态配置）
	messageModifier := func(ctx context.Context, messages []*schema.Message) []*schema.Message {
		toolCtx := tools.GetToolContext(ctx)
		dynamicPart := buildDynamicPrompt(toolCtx)
		staticPart := buildStaticPrompt()
		fullPrompt := staticPart + "\n\n" + dynamicPart

		result := make([]*schema.Message, len(messages))
		copy(result, messages)

		// 找到第一条 system 消息并更新为完整提示（静态+动态）
		for i, msg := range result {
			if msg.Role == schema.System {
				result[i] = schema.SystemMessage(fullPrompt)
				return result
			}
		}
		// 若历史中没有 system 消息则在最前面插入
		return append([]*schema.Message{schema.SystemMessage(fullPrompt)}, result...)
	}

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig:      compose.ToolsNodeConfig{Tools: a.tools},
		MaxStep:          10,
		MessageModifier:  messageModifier,
	})
	if err != nil {
		return nil, fmt.Errorf("创建ReAct Agent失败: %w", err)
	}

	a.reactAgent = agent
	return agent, nil
}

// Stream 发起流式对话并注入工具上下文。
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

// convertMessages 批量将内部消息转换为 Eino 消息格式。
func convertMessages(messages []ChatMessage) []*schema.Message {
	var schemaMessages []*schema.Message
	for _, msg := range messages {
		schemaMessages = append(schemaMessages, convertMessage(msg))
	}
	return schemaMessages
}

// convertMessage 将单条内部消息转换为 Eino 消息。
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

// buildStaticPrompt 返回固定不变的系统守则。
// 实际注入由 MessageModifier 统一处理。
func buildStaticPrompt() string {
	var sb strings.Builder

	sb.WriteString("你是校园失物招领系统的AI助手，帮助用户处理失物招领相关事务。\n")
	sb.WriteString("你的服务对象是浙江工业大学校园里的学生和教师，即普通用户，而不是管理员。\n")
	sb.WriteString("该大学有三个校区：朝晖、屏峰、莫干山。\n\n")

	sb.WriteString("## 能力边界说明（重要）\n")
	sb.WriteString("1. 你只能在系统已提供的功能范围内协助用户，不得虚构系统能力。\n")
	sb.WriteString("2. 不得承诺线下处理、人工干预、系统外部联系或任何未在可用工具中定义的操作。\n")
	sb.WriteString("3. 如果用户提出超出系统功能范围的请求，应明确告知当前系统不支持该操作，而不是编造解决方案。\n")
	sb.WriteString("4. 不得保证一定找回物品、一定通过审核等结果性承诺。\n")
	sb.WriteString("5. 你不具备管理员权限，不能执行管理员专属操作。\n\n")

	sb.WriteString("## 工具调用规范（重要）\n")
	sb.WriteString("在调用任何工具之前，你必须遵循以下原则：\n")
	sb.WriteString("1. **参数完整性检查**：仔细检查工具所需参数是否已经由用户提供。如果缺少必要信息，主动向用户询问，不要猜测或编造参数值。\n")
	sb.WriteString("2. **只说明业务操作**：用户无法看到系统内部工具调用记录，只需用自然语言说明将要进行的业务操作，不暴露工具名称、参数字段、报错信息和系统实现细节。\n")
	sb.WriteString("3. **征求用户同意**：对于非查询类工具，在用户确认同意后再执行操作，不得在用户不知情的情况下直接执行。\n")
	sb.WriteString("4. **安全优先**：对于涉及数据修改的操作（如发布、认领、审核等），必须确保用户已提供完整且准确的信息。\n")
	sb.WriteString("5. **保证兼容**：为了保证后端系统能够正常工作，在工具调用时严禁生成额外文本，且应该先调用工具再进行文字回答。\n\n")

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

	sb.WriteString("## 补充说明\n")
	sb.WriteString("当需要查看某条发布的详细信息时，可以提供链接 https://found.imjh.top/query/detail/?itemId={postId} 供用户直接打开，其中 postId 为对应的发布ID。\n\n")

	return sb.String()
}

// buildDynamicPrompt 返回每次 LLM 调用时需要更新的动态内容：
// 当前用户信息和从数据库读取的实时系统配置。
func buildDynamicPrompt(toolCtx *tools.ToolContext) string {
	if toolCtx == nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("## 当前用户信息\n")
	sb.WriteString(fmt.Sprintf("用户ID: %d\n\n", toolCtx.UserID))

	if cfg := toolCtx.SystemConfig; cfg != nil {
		sb.WriteString("## 系统当前配置\n")
		if len(cfg.ItemTypes) > 0 {
			sb.WriteString(fmt.Sprintf("- 物品类型（发布或搜索时使用）：%s\n", strings.Join(cfg.ItemTypes, "、")))
		}
		if len(cfg.FeedbackTypes) > 0 {
			sb.WriteString(fmt.Sprintf("- 反馈类型（提交反馈时使用）：%s\n", strings.Join(cfg.FeedbackTypes, "、")))
		}
		if cfg.ClaimValidityDays > 0 {
			sb.WriteString(fmt.Sprintf("- 认领申请有效期：%d 天\n", cfg.ClaimValidityDays))
		}
		if cfg.PublishLimit > 0 {
			sb.WriteString(fmt.Sprintf("- 每日发布上限：%d 条\n", cfg.PublishLimit))
		}
	}

	return sb.String()
}
