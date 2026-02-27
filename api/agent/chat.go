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
	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func ChatHandler() gin.HandlerFunc {
	api := ChatApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfChat).Pointer()).Name()] = api
	return hfChat
}

type ChatApi struct {
	Info     struct{} `name:"Agent对话" desc:"发送消息给AI助手"`
	Request  ChatApiRequest
	Response ChatApiResponse
}

type ChatApiRequest struct {
	Body struct {
		SessionID string `json:"session_id" binding:"required" desc:"会话ID"`
		Message   string `json:"message" binding:"required" desc:"用户消息"`
		Stream    bool   `json:"stream" desc:"是否使用流式输出"`
	}
}

type ChatApiResponse struct {
	Response string `json:"response" desc:"AI回复"`
}

func (a *ChatApi) Run(ctx *gin.Context) kit.Code {
	request := a.Request.Body

	userID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	agentService := service.GetAgentService()

	if request.Stream {
		a.handleStream(ctx, agentService, request.SessionID, request.Message, userID)
		return comm.CodeOK
	}

	response, err := agentService.Chat(ctx, request.SessionID, userID, "", request.Message)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("Agent对话失败")
		return comm.CodeServerError
	}

	a.Response.Response = response
	return comm.CodeOK
}

func (a *ChatApi) handleStream(ctx *gin.Context, agentService *service.AgentService, sessionID, message string, userID int64) {
	stream, err := agentService.Stream(ctx, sessionID, userID, "", message)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("Agent流式对话失败")
		reply.Fail(ctx, comm.CodeServerError)
		return
	}

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

func (a *ChatApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindJSON(&a.Request.Body)
	if err != nil {
		return err
	}
	return err
}

func hfChat(ctx *gin.Context) {
	api := &ChatApi{}
	err := api.Init(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("参数绑定校验错误")
		reply.Fail(ctx, comm.CodeParameterInvalid)
		return
	}
	code := api.Run(ctx)
	if !ctx.IsAborted() {
		if code == comm.CodeOK && !api.Request.Body.Stream {
			reply.Success(ctx, api.Response)
		} else if code != comm.CodeOK {
			reply.Fail(ctx, code)
		}
	}
}
