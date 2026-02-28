package agent

import (
	"app/agent"
	"app/comm"
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

	agentService := service.GetAgentService()

	stream, err := agentService.Stream(ctx, request.SessionID, userID, "", request.Message, request.Images)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("Agent流式对话失败")
		return comm.CodeServerError
	}

	a.handleStream(ctx, agentService, request.SessionID, request.Message, request.Images, userID, stream)
	return comm.CodeOK
}

func (a *StreamApi) handleStream(ctx *gin.Context, agentService *service.AgentService, sessionID, message string, images []string, userID int64, stream *schema.StreamReader[*schema.Message]) {
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

	var fullContent string
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("流式读取错误")
			break
		}

		event := agent.ParseStreamMessage(msg)
		eventBytes, err := sonic.Marshal(event)
		if err != nil {
			continue
		}

		ctx.Writer.WriteString("data: ")
		ctx.Writer.Write(eventBytes)
		ctx.Writer.WriteString("\n\n")
		flusher.Flush()

		if msg.Role == "assistant" && len(msg.ToolCalls) == 0 {
			fullContent += msg.Content
		}
	}

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
