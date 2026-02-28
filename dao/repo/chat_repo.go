package repo

import (
	"context"
	"errors"
	"time"

	"github.com/bytedance/sonic"
	"github.com/zjutjh/mygo/ndb"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type ChatSession struct {
	ID        int64                 `gorm:"column:id;primaryKey;autoIncrement:true"`
	SessionID string                `gorm:"column:session_id;uniqueIndex;not null"`
	UserID    int64                 `gorm:"column:user_id;not null"`
	Title     string                `gorm:"column:title;not null;default:''"`
	CreatedAt time.Time             `gorm:"column:created_at;not null"`
	UpdatedAt time.Time             `gorm:"column:updated_at;not null"`
	DeletedAt soft_delete.DeletedAt `gorm:"column:deleted_at;softDelete:milli"`
}

func (*ChatSession) TableName() string {
	return "chat_session"
}

type ChatMessage struct {
	ID               int64                 `gorm:"column:id;primaryKey;autoIncrement:true"`
	SessionID        string                `gorm:"column:session_id;not null;index"`
	Role             string                `gorm:"column:role;not null"`
	Content          string                `gorm:"column:content;not null"`
	Images           string                `gorm:"column:images"`
	ImageDescriptions string               `gorm:"column:image_descriptions"`
	CreatedAt        time.Time             `gorm:"column:created_at;not null"`
	DeletedAt        soft_delete.DeletedAt `gorm:"column:deleted_at;softDelete:milli"`
}

func (*ChatMessage) TableName() string {
	return "chat_message"
}

type ChatMessageData struct {
	SessionID         string
	Role              string
	Content           string
	Images            []string
	ImageDescriptions []string
	CreatedAt         time.Time
}

type ChatRepo struct{}

func NewChatRepo() *ChatRepo {
	return &ChatRepo{}
}

func (r *ChatRepo) CreateSession(ctx context.Context, session *ChatSession) error {
	return ndb.Pick().WithContext(ctx).Create(session).Error
}

func (r *ChatRepo) GetSessionByID(ctx context.Context, sessionID string) (*ChatSession, error) {
	var session ChatSession
	err := ndb.Pick().WithContext(ctx).Where("session_id = ?", sessionID).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *ChatRepo) UpdateSessionTitle(ctx context.Context, sessionID string, title string) error {
	return ndb.Pick().WithContext(ctx).Model(&ChatSession{}).
		Where("session_id = ?", sessionID).
		Update("title", title).Error
}

func (r *ChatRepo) UpdateSessionUpdatedAt(ctx context.Context, sessionID string) error {
	return ndb.Pick().WithContext(ctx).Model(&ChatSession{}).
		Where("session_id = ?", sessionID).
		Update("updated_at", time.Now()).Error
}

func (r *ChatRepo) ListSessionsByUserID(ctx context.Context, userID int64) ([]*ChatSession, error) {
	var sessions []*ChatSession
	err := ndb.Pick().WithContext(ctx).Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&sessions).Error
	return sessions, err
}

func (r *ChatRepo) CreateMessage(ctx context.Context, data *ChatMessageData) error {
	imagesJSON, _ := sonic.MarshalString(data.Images)
	descJSON, _ := sonic.MarshalString(data.ImageDescriptions)

	msg := &ChatMessage{
		SessionID:         data.SessionID,
		Role:              data.Role,
		Content:           data.Content,
		Images:            imagesJSON,
		ImageDescriptions: descJSON,
		CreatedAt:         data.CreatedAt,
	}
	return ndb.Pick().WithContext(ctx).Create(msg).Error
}

func (r *ChatRepo) ListMessagesBySessionID(ctx context.Context, sessionID string) ([]*ChatMessageData, error) {
	var messages []*ChatMessage
	err := ndb.Pick().WithContext(ctx).Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&messages).Error
	if err != nil {
		return nil, err
	}

	result := make([]*ChatMessageData, 0, len(messages))
	for _, m := range messages {
		var images []string
		var descs []string
		if m.Images != "" {
			_ = sonic.UnmarshalString(m.Images, &images)
		}
		if m.ImageDescriptions != "" {
			_ = sonic.UnmarshalString(m.ImageDescriptions, &descs)
		}
		result = append(result, &ChatMessageData{
			SessionID:         m.SessionID,
			Role:              m.Role,
			Content:           m.Content,
			Images:            images,
			ImageDescriptions: descs,
			CreatedAt:         m.CreatedAt,
		})
	}
	return result, nil
}
