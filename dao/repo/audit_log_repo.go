package repo

import (
	"app/dao/model"
	"context"

	"github.com/zjutjh/mygo/ndb"
)

type AuditLogRepo struct{}

func NewAuditLogRepo() *AuditLogRepo {
	return &AuditLogRepo{}
}

// CreateAuditLog 创建审计日志
func (r *AuditLogRepo) CreateAuditLog(ctx context.Context, adminID int64, actionType string, reason string, postID int64, oldStatus string, newStatus string) error {
	log := &model.AuditLog{
		AdminID:    adminID,
		ActionType: actionType,
		Reason:     reason,
		PostID:     postID,
		OldStatus:  oldStatus,
		NewStatus:  newStatus,
	}
	return ndb.Pick().WithContext(ctx).Create(log).Error
}
