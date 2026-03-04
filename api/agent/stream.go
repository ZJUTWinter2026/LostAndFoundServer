package agent

import (
	"app/agent"
	"app/comm"
	"app/dao/repo"
	"app/service"
	"io"
	"net/http"
	"reflect"
	"runtime"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func StreamHandler() gin.HandlerFunc {
	api := StreamApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfStream).Pointer()).Name()] = api
	return hfStream
}

type StreamApi struct {
	Info     struct{} `name:"Agent流式对话" desc:"发送消息给AI助手（流式输出）"`
	Request  StreamApiRequest
	Response StreamApiResponse
}

type StreamApiRequest struct {
	Body struct {
		SessionID string   `json:"session_id" binding:"required" desc:"会话ID"`
		Message   string   `json:"message" binding:"required" desc:"用户消息"`
		Images    []string `json:"images" desc:"图片URL列表"`
	}
}

type StreamApiResponse struct{}

func (a *StreamApi) Run(ctx *gin.Context) kit.Code {
	request := a.Request.Body

	userID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	userRepo := repo.NewUserRepo()
	user, err := userRepo.FindById(ctx, userID)
	if err != nil || user == nil {
		return comm.CodeServerError
	}

	agentService := service.GetAgentService()

	stream, err := agentService.Stream(ctx, request.SessionID, userID, user.Usertype, request.Message, request.Images)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("Agent流式对话失败")
		if err.Error() == "会话正在处理中" {
			return comm.CodeSessionProcessing
		}
		return comm.CodeServerError
	}

	a.handleStream(ctx, agentService, request.SessionID, userID, stream)
	return comm.CodeOK
}

func (a *StreamApi) handleStream(ctx *gin.Context, agentService *service.AgentService, sessionID string, userID int64, stream *schema.StreamReader[*schema.Message]) {
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Transfer-Encoding", "chunked")

	flusher, ok := ctx.Writer.(http.Flusher)
	if !ok {
		nlog.Pick().WithContext(ctx).Warn("不支持SSE")
		reply.Fail(ctx, comm.CodeServerError)
		return
	}

	sendEvent := func(event agent.StreamEvent) {
		eventBytes, err := sonic.Marshal(event)
		if err != nil {
			return
		}
		ctx.Writer.WriteString("data: ")
		ctx.Writer.Write(eventBytes)
		ctx.Writer.WriteString("\n\n")
		flusher.Flush()
	}

	var fullContent string
	// pendingToolCall 用于缓冲分片的 tool_call，待参数完整后再统一发送
	var pendingToolCall *agent.ToolCallInfo

	flushPendingToolCall := func() {
		if pendingToolCall != nil {
			sendEvent(agent.StreamEvent{
				Type: "tool_call",
				Data: *pendingToolCall,
			})
			pendingToolCall = nil
		}
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("流式读取错误")
			break
		}

		nlog.Pick().WithContext(ctx).WithField("msg", msg).Info("[eino stream] raw chunk")

		switch msg.Role {
		case schema.Assistant:
			if len(msg.ToolCalls) > 0 {
				tc := msg.ToolCalls[0]
				if tc.ID != "" {
					// 新的 tool_call 开始（第一个分片带有 ID 和 Name）
					// 先把上一个未发送的 tool_call 刷出去
					flushPendingToolCall()
					pendingToolCall = &agent.ToolCallInfo{
						ID:        tc.ID,
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					}
				} else if pendingToolCall != nil {
					// 后续分片只携带 Arguments 片段，累加
					pendingToolCall.Arguments += tc.Function.Arguments
				}
			} else {
				// 普通 content 消息：先把积压的 tool_call 发出去
				flushPendingToolCall()
				if msg.Content != "" {
					sendEvent(agent.StreamEvent{
						Type:    "content",
						Content: msg.Content,
					})
					fullContent += msg.Content
				}
			}
		case schema.Tool:
			// 工具调用结果到来：先把积压的 tool_call 发出去，再发 tool_result
			flushPendingToolCall()
			sendEvent(agent.StreamEvent{
				Type: "tool_result",
				Data: agent.ToolResultInfo{
					ToolCallID: msg.ToolCallID,
					ToolName:   msg.ToolName,
					Result:     msg.Content,
				},
			})
		default:
			if msg.Content != "" {
				flushPendingToolCall()
				sendEvent(agent.StreamEvent{
					Type:    "content",
					Content: msg.Content,
				})
			}
		}
	}

	// 流结束后，将残留的 tool_call（若有）发出去
	flushPendingToolCall()

	agentService.SaveAssistantMessage(ctx, sessionID, userID, fullContent)

	ctx.Writer.WriteString("data: [DONE]\n\n")
	flusher.Flush()
}

func (a *StreamApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindJSON(&a.Request.Body)
	if err != nil {
		return err
	}
	return err
}

func hfStream(ctx *gin.Context) {
	api := &StreamApi{}
	err := api.Init(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("参数绑定校验错误")
		reply.Fail(ctx, comm.CodeParameterInvalid)
		return
	}
	code := api.Run(ctx)
	if !ctx.IsAborted() && code != comm.CodeOK {
		reply.Fail(ctx, code)
	}
}
