package agent

import (
	"app/agent/tools"
	"app/dao/model"
	"app/dao/repo"
	"app/pkg/llm"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

type ChatSession struct {
	SessionID    string
	UserID       int64
	Title        string
	Messages     []ChatMessageRecord
	IsProcessing bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

var ErrSessionProcessing = errors.New("会话正在处理中")

type ChatMessageRecord struct {
	SessionID         string
	Role              string
	Content           string
	Images            []string
	ImageDescriptions []string
	CreatedAt         time.Time
}

type AgentService struct {
	agent    *Agent
	chatRepo *repo.ChatRepo

	sessions   map[string]*ChatSession
	sessionMux sync.RWMutex
}

var (
	agentService     *AgentService
	agentServiceOnce sync.Once
)

// GetAgentService 返回 AgentService 单例。
func GetAgentService() *AgentService {
	agentServiceOnce.Do(func() {
		agentService = &AgentService{
			agent:    NewAgent(),
			chatRepo: repo.NewChatRepo(),
			sessions: make(map[string]*ChatSession),
		}
	})
	return agentService
}

// CreateSession 创建用户会话并写入数据库与内存缓存。
func (s *AgentService) CreateSession(ctx context.Context, userID int64, title string) (*ChatSession, error) {
	sessionID := uuid.New().String()
	now := time.Now()

	session := &ChatSession{
		SessionID: sessionID,
		UserID:    userID,
		Title:     title,
		Messages:  []ChatMessageRecord{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	dbSession := &model.ChatSession{
		SessionID: sessionID,
		UserID:    userID,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.chatRepo.CreateSession(ctx, dbSession); err != nil {
		return nil, err
	}

	s.sessionMux.Lock()
	s.sessions[sessionID] = session
	s.sessionMux.Unlock()

	return session, nil
}

// GetSession 获取会话，优先读内存缓存，未命中时回源数据库。
func (s *AgentService) GetSession(ctx context.Context, sessionID string, userID int64) (*ChatSession, error) {
	s.sessionMux.RLock()
	session, ok := s.sessions[sessionID]
	s.sessionMux.RUnlock()

	if !ok {
		dbSession, err := s.chatRepo.GetSessionByID(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		if dbSession == nil {
			return nil, fmt.Errorf("会话不存在")
		}
		if dbSession.UserID != userID {
			return nil, fmt.Errorf("无权访问该会话")
		}

		messages, err := s.chatRepo.ListMessagesBySessionID(ctx, sessionID)
		if err != nil {
			return nil, err
		}

		msgRecords := toChatMessageRecords(messages)

		session = &ChatSession{
			SessionID: dbSession.SessionID,
			UserID:    dbSession.UserID,
			Title:     dbSession.Title,
			Messages:  msgRecords,
			CreatedAt: dbSession.CreatedAt,
			UpdatedAt: dbSession.UpdatedAt,
		}

		s.sessionMux.Lock()
		s.sessions[sessionID] = session
		s.sessionMux.Unlock()
	}

	if session.UserID != userID {
		return nil, fmt.Errorf("无权访问该会话")
	}

	return session, nil
}

// Stream 写入用户消息并启动一轮 Agent 流式推理。
func (s *AgentService) Stream(ctx context.Context, sessionID string, userID int64, userMessage string, images []string) (*schema.StreamReader[*schema.Message], error) {
	session, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	s.sessionMux.Lock()
	if session.IsProcessing {
		s.sessionMux.Unlock()
		return nil, ErrSessionProcessing
	}
	session.IsProcessing = true
	s.sessionMux.Unlock()

	var imageDescriptions []string
	if len(images) > 0 {
		imageDescriptions, _ = llm.DescribeImages(ctx, images)
	}

	now := time.Now()
	userMsgRecord := ChatMessageRecord{
		SessionID:         sessionID,
		Role:              "user",
		Content:           userMessage,
		Images:            images,
		ImageDescriptions: imageDescriptions,
		CreatedAt:         now,
	}

	if err := s.chatRepo.CreateMessage(ctx, &repo.ChatMessageData{
		SessionID:         sessionID,
		Role:              "user",
		Content:           userMessage,
		Images:            images,
		ImageDescriptions: imageDescriptions,
		CreatedAt:         now,
	}); err != nil {
		s.sessionMux.Lock()
		session.IsProcessing = false
		s.sessionMux.Unlock()
		return nil, fmt.Errorf("保存消息失败: %w", err)
	}

	s.sessionMux.Lock()
	session.Messages = append(session.Messages, userMsgRecord)
	messages := toAgentMessages(session.Messages)
	s.sessionMux.Unlock()

	toolCtx := &tools.ToolContext{UserID: userID}

	// 查询系统配置，注入 ToolContext 供 MessageModifier 使用。
	{
		configRepo := repo.NewSystemConfigRepo()
		feedbackTypes, _ := configRepo.GetFeedbackTypes(ctx)
		feedbackTypes = append(feedbackTypes, "其它类型")
		itemTypes, _ := configRepo.GetItemTypes(ctx)
		itemTypes = append(itemTypes, "其它类型")
		claimValidityDays, _ := configRepo.GetClaimValidityDays(ctx)
		publishLimit, _ := configRepo.GetPublishLimit(ctx)
		toolCtx.SystemConfig = &tools.SystemConfig{
			FeedbackTypes:     feedbackTypes,
			ItemTypes:         itemTypes,
			ClaimValidityDays: claimValidityDays,
			PublishLimit:      publishLimit,
		}
	}

	stream, err := s.agent.Stream(ctx, messages, toolCtx)
	if err != nil {
		s.sessionMux.Lock()
		session.IsProcessing = false
		s.sessionMux.Unlock()
		return nil, fmt.Errorf("AI对话失败: %w", err)
	}

	s.sessionMux.Lock()
	if session.Title == "" && len(session.Messages) > 0 {
		session.Title = userMessage
		runes := []rune(session.Title)
		if len(runes) > 10 {
			session.Title = string(runes[:10]) + "..."
		}
		s.sessionMux.Unlock()
		_ = s.chatRepo.UpdateSessionTitle(ctx, sessionID, session.Title)
	} else {
		s.sessionMux.Unlock()
	}

	return stream, nil
}

// ResetProcessing 立即重置会话的处理状态，供客户端断开连接时调用。
func (s *AgentService) ResetProcessing(sessionID string, userID int64) {
	s.sessionMux.Lock()
	defer s.sessionMux.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return
	}
	if session.UserID != userID {
		return
	}
	session.IsProcessing = false
}

// SaveConversationMessages 批量保存本轮对话产生的所有消息并重置 IsProcessing。
func (s *AgentService) SaveConversationMessages(ctx context.Context, sessionID string, userID int64, msgs []ChatMessageRecord) error {
	s.sessionMux.RLock()
	session, ok := s.sessions[sessionID]
	s.sessionMux.RUnlock()
	if !ok {
		return fmt.Errorf("会话不存在")
	}
	if session.UserID != userID {
		return fmt.Errorf("无权访问该会话")
	}

	// 使用独立超时 context，防止 request ctx 已取消导致写入失败。
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()

	now := time.Now()
	for i := range msgs {
		if msgs[i].CreatedAt.IsZero() {
			msgs[i].CreatedAt = now
		}
	}

	persistedMsgs := make([]ChatMessageRecord, 0, len(msgs))
	var saveErr error
	for _, msg := range msgs {
		data := toRepoMessageData(sessionID, msg)
		if err := s.chatRepo.CreateMessage(dbCtx, data); err != nil {
			saveErr = fmt.Errorf("保存消息失败: %w", err)
			break
		}
		persistedMsgs = append(persistedMsgs, msg)
	}

	s.sessionMux.Lock()
	if liveSession, exists := s.sessions[sessionID]; exists && liveSession.UserID == userID {
		for _, msg := range persistedMsgs {
			msg.SessionID = sessionID
			liveSession.Messages = append(liveSession.Messages, msg)
		}
		if len(persistedMsgs) > 0 {
			liveSession.UpdatedAt = now
		}
		liveSession.IsProcessing = false
	} else {
		session.IsProcessing = false
	}
	s.sessionMux.Unlock()

	if len(persistedMsgs) > 0 {
		if err := s.chatRepo.UpdateSessionUpdatedAt(dbCtx, sessionID); err != nil && saveErr == nil {
			saveErr = fmt.Errorf("更新会话时间失败: %w", err)
		}
	}

	return saveErr
}

// toChatMessageRecords 将仓储消息转换为会话消息结构。
func toChatMessageRecords(messages []*repo.ChatMessageData) []ChatMessageRecord {
	msgRecords := make([]ChatMessageRecord, 0, len(messages))
	for _, m := range messages {
		rec := ChatMessageRecord{
			SessionID:         m.SessionID,
			Role:              m.Role,
			Content:           m.Content,
			Images:            m.Images,
			ImageDescriptions: m.ImageDescriptions,
			CreatedAt:         m.CreatedAt,
		}
		msgRecords = append(msgRecords, rec)
	}
	return msgRecords
}

// toAgentMessages 将会话消息转换为 Agent 输入消息。
func toAgentMessages(messages []ChatMessageRecord) []ChatMessage {
	agentMessages := make([]ChatMessage, 0, len(messages))
	for _, msg := range messages {
		msgContent := msg.Content
		if len(msg.Images) > 0 && msg.Role == "user" {
			msgContent += llm.BuildImageContext(msg.Images, msg.ImageDescriptions)
		}
		agentMessages = append(agentMessages, ChatMessage{
			Role:    msg.Role,
			Content: msgContent,
		})
	}
	return agentMessages
}

// toRepoMessageData 将会话消息转换为仓储写入结构。
func toRepoMessageData(sessionID string, msg ChatMessageRecord) *repo.ChatMessageData {
	return &repo.ChatMessageData{
		SessionID: sessionID,
		Role:      msg.Role,
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
	}
}

// GetChatHistory 获取会话聊天历史。
func (s *AgentService) GetChatHistory(ctx context.Context, sessionID string, userID int64) ([]ChatMessageRecord, error) {
	session, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	return session.Messages, nil
}

// ListSessions 列出用户所有会话。
func (s *AgentService) ListSessions(ctx context.Context, userID int64) ([]*ChatSession, error) {
	dbSessions, err := s.chatRepo.ListSessionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	sessions := make([]*ChatSession, 0, len(dbSessions))
	for _, dbSess := range dbSessions {
		s.sessionMux.RLock()
		cached, ok := s.sessions[dbSess.SessionID]
		s.sessionMux.RUnlock()

		if ok {
			sessions = append(sessions, cached)
		} else {
			sessions = append(sessions, &ChatSession{
				SessionID: dbSess.SessionID,
				UserID:    dbSess.UserID,
				Title:     dbSess.Title,
				Messages:  []ChatMessageRecord{},
				CreatedAt: dbSess.CreatedAt,
				UpdatedAt: dbSess.UpdatedAt,
			})
		}
	}

	return sessions, nil
}
