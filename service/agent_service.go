package service

import (
	"app/agent"
	"app/agent/tools"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

type ChatSession struct {
	SessionID string
	UserID    int64
	Title     string
	Messages  []ChatMessageRecord
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ChatMessageRecord struct {
	SessionID string
	Role      string
	Content   string
	Images    []string
	CreatedAt time.Time
}

type AgentService struct {
	agent      *agent.Agent
	sessions   map[string]*ChatSession
	sessionMux sync.RWMutex
}

var agentService *AgentService
var agentServiceOnce sync.Once

func GetAgentService() *AgentService {
	agentServiceOnce.Do(func() {
		agentService = &AgentService{
			agent:    agent.NewAgent(),
			sessions: make(map[string]*ChatSession),
		}
	})
	return agentService
}

func (s *AgentService) CreateSession(ctx context.Context, userID int64, title string) (*ChatSession, error) {
	sessionID := uuid.New().String()

	session := &ChatSession{
		SessionID: sessionID,
		UserID:    userID,
		Title:     title,
		Messages:  []ChatMessageRecord{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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
		return nil, fmt.Errorf("会话不存在")
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

	userMsgRecord := ChatMessageRecord{
		SessionID: sessionID,
		Role:      "user",
		Content:   userMessage,
		Images:    images,
		CreatedAt: time.Now(),
	}
	session.Messages = append(session.Messages, userMsgRecord)

	var messages []agent.ChatMessage
	for _, msg := range session.Messages {
		messages = append(messages, agent.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
			Images:  msg.Images,
		})
	}

	toolCtx := &tools.ToolContext{
		UserID:   userID,
		UserType: userType,
	}

	stream, err := s.agent.Stream(ctx, messages, toolCtx)
	if err != nil {
		return nil, fmt.Errorf("AI对话失败: %w", err)
	}

	if session.Title == "" && len(session.Messages) > 0 {
		session.Title = userMessage
		if len(session.Title) > 50 {
			session.Title = session.Title[:50] + "..."
		}
	}

	return stream, nil
}

func (s *AgentService) SaveAssistantMessage(ctx context.Context, sessionID string, userID int64, content string) error {
	session, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return err
	}

	assistantMsgRecord := ChatMessageRecord{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   content,
		CreatedAt: time.Now(),
	}
	session.Messages = append(session.Messages, assistantMsgRecord)
	session.UpdatedAt = time.Now()

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
	s.sessionMux.RLock()
	defer s.sessionMux.RUnlock()

	var sessions []*ChatSession
	for _, session := range s.sessions {
		if session.UserID == userID {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}
