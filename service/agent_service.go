package service

import (
	"app/agent"
	"app/agent/tools"
	"app/dao/model"
	"app/dao/repo"
	"app/pkg/llm"
	"context"
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

type ChatMessageRecord struct {
	SessionID         string
	Role              string
	Content           string
	Images            []string
	ImageDescriptions []string
	CreatedAt         time.Time
}

type AgentService struct {
	agent    *agent.Agent
	chatRepo *repo.ChatRepo

	sessions   map[string]*ChatSession
	sessionMux sync.RWMutex
}

var (
	agentService     *AgentService
	agentServiceOnce sync.Once
)

func GetAgentService() *AgentService {
	agentServiceOnce.Do(func() {
		agentService = &AgentService{
			agent:    agent.NewAgent(),
			chatRepo: repo.NewChatRepo(),
			sessions: make(map[string]*ChatSession),
		}
	})
	return agentService
}

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

		msgRecords := make([]ChatMessageRecord, 0, len(messages))
		for _, m := range messages {
			msgRecords = append(msgRecords, ChatMessageRecord{
				SessionID:         m.SessionID,
				Role:              m.Role,
				Content:           m.Content,
				Images:            m.Images,
				ImageDescriptions: m.ImageDescriptions,
				CreatedAt:         m.CreatedAt,
			})
		}

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

func (s *AgentService) Stream(ctx context.Context, sessionID string, userID int64, userType string, userMessage string, images []string) (*schema.StreamReader[*schema.Message], error) {
	session, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	s.sessionMux.Lock()
	if session.IsProcessing {
		s.sessionMux.Unlock()
		return nil, fmt.Errorf("会话正在处理中")
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
	messages := make([]agent.ChatMessage, 0, len(session.Messages))
	for _, msg := range session.Messages {
		msgContent := msg.Content
		if len(msg.Images) > 0 && msg.Role == "user" {
			msgContent += llm.BuildImageContext(msg.Images, msg.ImageDescriptions)
		}
		messages = append(messages, agent.ChatMessage{
			Role:    msg.Role,
			Content: msgContent,
		})
	}
	s.sessionMux.Unlock()

	toolCtx := &tools.ToolContext{
		UserID: userID,
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

func (s *AgentService) SaveAssistantMessage(ctx context.Context, sessionID string, userID int64, content string) error {
	s.sessionMux.Lock()
	session, ok := s.sessions[sessionID]
	s.sessionMux.Unlock()

	if !ok {
		return fmt.Errorf("会话不存在")
	}

	if session.UserID != userID {
		return fmt.Errorf("无权访问该会话")
	}

	now := time.Now()
	assistantMsgRecord := ChatMessageRecord{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   content,
		CreatedAt: now,
	}

	if err := s.chatRepo.CreateMessage(ctx, &repo.ChatMessageData{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   content,
		CreatedAt: now,
	}); err != nil {
		s.sessionMux.Lock()
		session.IsProcessing = false
		s.sessionMux.Unlock()
		return err
	}

	s.sessionMux.Lock()
	session.Messages = append(session.Messages, assistantMsgRecord)
	session.UpdatedAt = now
	session.IsProcessing = false
	s.sessionMux.Unlock()

	_ = s.chatRepo.UpdateSessionUpdatedAt(ctx, sessionID)

	return nil
}

func (s *AgentService) GetChatHistory(ctx context.Context, sessionID string, userID int64) ([]ChatMessageRecord, error) {
	session, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	return session.Messages, nil
}

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
