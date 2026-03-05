package repo

import (
	"context"
	"errors"
	"time"

	"app/dao/model"

	"github.com/bytedance/sonic"
	"github.com/zjutjh/mygo/ndb"
	"gorm.io/gorm"
)

type ChatMessageData struct {
	SessionID         string
	Role              string
	Content           string
	Images            []string
	ImageDescriptions []string
	CreatedAt         time.Time
}

type ChatRepo struct{}

// NewChatRepo 创建聊天仓储实例。
func NewChatRepo() *ChatRepo {
	return &ChatRepo{}
}

// CreateSession 创建会话记录。
func (r *ChatRepo) CreateSession(ctx context.Context, session *model.ChatSession) error {
	return ndb.Pick().WithContext(ctx).Create(session).Error
}

// GetSessionByID 按会话ID查询会话。
func (r *ChatRepo) GetSessionByID(ctx context.Context, sessionID string) (*model.ChatSession, error) {
	var session model.ChatSession
	err := ndb.Pick().WithContext(ctx).Where("session_id = ?", sessionID).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// UpdateSessionTitle 更新会话标题。
func (r *ChatRepo) UpdateSessionTitle(ctx context.Context, sessionID string, title string) error {
	return ndb.Pick().WithContext(ctx).Model(&model.ChatSession{}).
		Where("session_id = ?", sessionID).
		Update("title", title).Error
}

// UpdateSessionUpdatedAt 更新时间戳用于会话排序。
func (r *ChatRepo) UpdateSessionUpdatedAt(ctx context.Context, sessionID string) error {
	return ndb.Pick().WithContext(ctx).Model(&model.ChatSession{}).
		Where("session_id = ?", sessionID).
		Update("updated_at", time.Now()).Error
}

// ListSessionsByUserID 查询用户下所有会话。
func (r *ChatRepo) ListSessionsByUserID(ctx context.Context, userID int64) ([]*model.ChatSession, error) {
	var sessions []*model.ChatSession
	err := ndb.Pick().WithContext(ctx).Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&sessions).Error
	return sessions, err
}

// CreateMessage 创建聊天消息。
func (r *ChatRepo) CreateMessage(ctx context.Context, data *ChatMessageData) error {
	imagesJSON, _ := sonic.MarshalString(data.Images)
	descJSON, _ := sonic.MarshalString(data.ImageDescriptions)

	row := model.ChatMessage{
		SessionID:         data.SessionID,
		Role:              data.Role,
		Content:           data.Content,
		Images:            imagesJSON,
		ImageDescriptions: descJSON,
		CreatedAt:         data.CreatedAt,
	}
	return ndb.Pick().WithContext(ctx).Create(&row).Error
}

// ListMessagesBySessionID 按时间顺序查询会话消息，并解码 JSON 字段。
func (r *ChatRepo) ListMessagesBySessionID(ctx context.Context, sessionID string) ([]*ChatMessageData, error) {
	var rows []model.ChatMessage
	err := ndb.Pick().WithContext(ctx).
		Where("session_id = ? AND deleted_at = 0", sessionID).
		Order("created_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]*ChatMessageData, 0, len(rows))
	for _, row := range rows {
		var images []string
		var descs []string
		if row.Images != "" {
			_ = sonic.UnmarshalString(row.Images, &images)
		}
		if row.ImageDescriptions != "" {
			_ = sonic.UnmarshalString(row.ImageDescriptions, &descs)
		}
		d := &ChatMessageData{
			SessionID:         row.SessionID,
			Role:              row.Role,
			Content:           row.Content,
			Images:            images,
			ImageDescriptions: descs,
			CreatedAt:         row.CreatedAt,
		}
		result = append(result, d)
	}
	return result, nil
}
