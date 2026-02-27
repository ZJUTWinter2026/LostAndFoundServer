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

func SessionHandler() gin.HandlerFunc {
	api := SessionApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfSession).Pointer()).Name()] = api
	return hfSession
}

type SessionApi struct {
	Info     struct{} `name:"创建会话" desc:"创建新的对话会话"`
	Request  SessionApiRequest
	Response SessionApiResponse
}

type SessionApiRequest struct {
	Body struct {
		Title string `json:"title" desc:"会话标题"`
	}
}

type SessionApiResponse struct {
	SessionID string `json:"session_id" desc:"会话ID"`
}

func (a *SessionApi) Run(ctx *gin.Context) kit.Code {
	request := a.Request.Body

	userID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	agentService := service.GetAgentService()
	sess, err := agentService.CreateSession(ctx, userID, request.Title)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("创建会话失败")
		return comm.CodeServerError
	}

	a.Response.SessionID = sess.SessionID
	return comm.CodeOK
}

func (a *SessionApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindJSON(&a.Request.Body)
	if err != nil {
		return err
	}
	return err
}

func hfSession(ctx *gin.Context) {
	api := &SessionApi{}
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

func SessionListHandler() gin.HandlerFunc {
	api := SessionListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfSessionList).Pointer()).Name()] = api
	return hfSessionList
}

type SessionListApi struct {
	Info     struct{} `name:"获取会话列表" desc:"获取用户的所有会话"`
	Request  struct{}
	Response SessionListApiResponse
}

type SessionListApiResponse struct {
	Sessions []SessionInfo `json:"sessions" desc:"会话列表"`
}

type SessionInfo struct {
	SessionID string `json:"session_id" desc:"会话ID"`
	Title     string `json:"title" desc:"会话标题"`
	CreatedAt string `json:"created_at" desc:"创建时间"`
	UpdatedAt string `json:"updated_at" desc:"更新时间"`
}

func (a *SessionListApi) Run(ctx *gin.Context) kit.Code {
	userID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	agentService := service.GetAgentService()
	sessions, err := agentService.ListSessions(ctx, userID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("获取会话列表失败")
		return comm.CodeServerError
	}

	var sessionInfos []SessionInfo
	for _, s := range sessions {
		sessionInfos = append(sessionInfos, SessionInfo{
			SessionID: s.SessionID,
			Title:     s.Title,
			CreatedAt: s.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: s.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	a.Response.Sessions = sessionInfos
	return comm.CodeOK
}

func (a *SessionListApi) Init(ctx *gin.Context) (err error) {
	return nil
}

func hfSessionList(ctx *gin.Context) {
	api := &SessionListApi{}
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
