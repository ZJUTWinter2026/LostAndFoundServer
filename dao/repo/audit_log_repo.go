package repo

import (
	"context"
	"github.com/zjutjh/mygo/ndb"

	"app/dao/model"
)

type AuditLogRepo struct{}

func NewAuditLogRepo() *AuditLogRepo {
	return &AuditLogRepo{}
}

// CreateAuditLog 创建审计日志
func (r *AuditLogRepo) CreateAuditLog(ctx context.Context, adminID int64, actionType int8, reason string, postID int64, oldStatus int8, newStatus int8) error {
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
