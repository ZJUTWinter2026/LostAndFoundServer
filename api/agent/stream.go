package agent

import (
	coreagent "app/agent"
	"app/comm"
	"app/dao/repo"
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"sync"
	"time"

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

// Run 执行流式对话请求并返回 SSE 响应。
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

	agentService := coreagent.GetAgentService()

	stream, err := agentService.Stream(ctx, request.SessionID, userID, request.Message, request.Images)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("Agent流式对话失败")
		if errors.Is(err, coreagent.ErrSessionProcessing) {
			return comm.CodeSessionProcessing
		}
		return comm.CodeServerError
	}

	a.handleStream(ctx, agentService, request.SessionID, userID, stream)
	return comm.CodeOK
}

// handleStream 负责 SSE 推送、流聚合和消息持久化。
func (a *StreamApi) handleStream(ctx *gin.Context, agentService *coreagent.AgentService, sessionID string, userID int64, stream *schema.StreamReader[*schema.Message]) {
	closeStreamOnce := sync.Once{}
	closeStream := func() {
		closeStreamOnce.Do(func() {
			stream.Close()
		})
	}
	defer closeStream()

	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Transfer-Encoding", "chunked")

	flusher, ok := ctx.Writer.(http.Flusher)
	if !ok {
		nlog.Pick().WithContext(ctx).Warn("不支持SSE")
		agentService.ResetProcessing(sessionID, userID)
		reply.Fail(ctx, comm.CodeServerError)
		return
	}

	seq := 0
	sendEvent := func(event coreagent.StreamEvent) {
		seq++
		event.Seq = seq
		event.EventID = fmt.Sprintf("%s-%d", sessionID, seq)
		event.TS = time.Now().UnixMilli()
		eventBytes, err := sonic.Marshal(event)
		if err != nil {
			return
		}
		ctx.Writer.WriteString("data: ")
		ctx.Writer.Write(eventBytes)
		ctx.Writer.WriteString("\n\n")
		flusher.Flush()
	}

	// 监听客户端断开：立即关闭 stream 解除 Recv 阻塞，后续统一走收尾逻辑重置状态
	streamCompleted := make(chan struct{})
	defer close(streamCompleted)
	go func() {
		select {
		case <-ctx.Request.Context().Done():
			closeStream()
			nlog.Pick().WithContext(ctx).Info("客户端断开SSE连接，已停止推理")
		case <-streamCompleted:
		}
	}()

	collectedMsgs, collectErr := agentService.CollectStreamMessages(stream, sendEvent)
	if collectErr != nil {
		nlog.Pick().WithContext(ctx).WithError(collectErr).Warn("流式读取错误")
	}

	// 批量落库（使用独立超时 context，防止 request ctx 取消导致保存失败）
	saveCtx, saveCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer saveCancel()
	if err := agentService.SaveConversationMessages(saveCtx, sessionID, userID, collectedMsgs); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("保存Agent会话消息失败")
		agentService.ResetProcessing(sessionID, userID)
	}

	// 仅在连接仍然有效时发送结束标记
	select {
	case <-ctx.Request.Context().Done():
		// 连接已断开，不再写入
	default:
		ctx.Writer.WriteString("data: [DONE]\n\n")
		flusher.Flush()
	}
}

// Init 绑定并校验请求参数。
func (a *StreamApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindJSON(&a.Request.Body)
	if err != nil {
		return err
	}
	return err
}

// hfStream 为 gin 入口函数，统一处理初始化与返回。
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
