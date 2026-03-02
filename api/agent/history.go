package agent

import (
	"app/comm"
	"app/service"
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func HistoryHandler() gin.HandlerFunc {
	api := HistoryApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfHistory).Pointer()).Name()] = api
	return hfHistory
}

type HistoryApi struct {
	Info     struct{} `name:"获取对话历史" desc:"获取会话的聊天记录"`
	Request  HistoryApiRequest
	Response HistoryApiResponse
}

type HistoryApiRequest struct {
	Query struct {
		SessionID string `form:"session_id" binding:"required" desc:"会话ID"`
	}
}

type HistoryApiResponse struct {
	Messages []MessageInfo `json:"messages" desc:"消息列表"`
}

type MessageInfo struct {
	Role      string   `json:"role" desc:"角色: user, assistant"`
	Content   string   `json:"content" desc:"消息内容"`
	Images    []string `json:"images,omitempty" desc:"图片URL列表"`
	CreatedAt string   `json:"created_at" desc:"创建时间"`
}

func (a *HistoryApi) Run(ctx *gin.Context) kit.Code {
	sessionID := a.Request.Query.SessionID

	userID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	agentService := service.GetAgentService()
	messages, err := agentService.GetChatHistory(ctx, sessionID, userID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("获取聊天历史失败")
		return comm.CodeServerError
	}

	var messageInfos []MessageInfo
	for _, m := range messages {
		messageInfos = append(messageInfos, MessageInfo{
			Role:      m.Role,
			Content:   m.Content,
			Images:    m.Images,
			CreatedAt: m.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	a.Response.Messages = messageInfos
	return comm.CodeOK
}

func (a *HistoryApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindQuery(&a.Request.Query)
	if err != nil {
		return err
	}
	return err
}

func hfHistory(ctx *gin.Context) {
	api := &HistoryApi{}
	err := api.Init(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("参数绑定校验错误")
		reply.Fail(ctx, comm.CodeParameterInvalid)
		return
	}
	code := api.Run(ctx)
	if !ctx.IsAborted() {
		if code == comm.CodeOK {
			reply.Success(ctx, api.Response)
		} else {
			reply.Fail(ctx, code)
		}
	}
}
