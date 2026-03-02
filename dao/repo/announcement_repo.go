package repo

import (
	"app/comm/enum"
	"app/dao/model"
	"context"
	"errors"
	"time"

	"github.com/zjutjh/mygo/ndb"
	"gorm.io/gorm"
)

type AnnouncementRepo struct{}

func NewAnnouncementRepo() *AnnouncementRepo {
	return &AnnouncementRepo{}
}

func (r *AnnouncementRepo) Create(ctx context.Context, announcement *model.Announcement) error {
	return ndb.Pick().WithContext(ctx).Create(announcement).Error
}

func (r *AnnouncementRepo) FindById(ctx context.Context, id int64) (*model.Announcement, error) {
	var announcement model.Announcement
	err := ndb.Pick().WithContext(ctx).Where("id = ?", id).First(&announcement).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &announcement, nil
}

func (r *AnnouncementRepo) ListApprovedForUser(ctx context.Context, userID int64, offset int, limit int) ([]*model.Announcement, int64, error) {
	var announcements []*model.Announcement
	db := ndb.Pick().WithContext(ctx).Model(&model.Announcement{}).
		Where("status = ?", enum.AnnouncementStatusApproved).
		Where("(target_user_id = 0 OR target_user_id = ?)", userID)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&announcements).Error
	return announcements, total, err
}

func (r *AnnouncementRepo) ListPending(ctx context.Context, campus string, offset int, limit int) ([]*model.Announcement, int64, error) {
	var announcements []*model.Announcement
	db := ndb.Pick().WithContext(ctx).Model(&model.Announcement{}).
		Where("status = ?", enum.AnnouncementStatusPending)

	if campus != "" {
		db = db.Where("campus = ?", campus)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&announcements).Error
	return announcements, total, err
}

func (r *AnnouncementRepo) Approve(ctx context.Context, id int64, reviewerID int64) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Announcement{}).
		Where("id = ? AND status = ?", id, enum.AnnouncementStatusPending).
		Updates(map[string]interface{}{
			"status":      enum.AnnouncementStatusApproved,
			"reviewed_by": reviewerID,
			"reviewed_at": time.Now(),
		}).Error
}

func (r *AnnouncementRepo) Reject(ctx context.Context, id int64, reviewerID int64) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Announcement{}).
		Where("id = ? AND status = ?", id, enum.AnnouncementStatusPending).
		Updates(map[string]interface{}{
			"status":      enum.AnnouncementStatusRejected,
			"reviewed_by": reviewerID,
			"reviewed_at": time.Now(),
		}).Error
}

func (r *AnnouncementRepo) Delete(ctx context.Context, id int64) error {
	return ndb.Pick().WithContext(ctx).Where("id = ?", id).Delete(&model.Announcement{}).Error
}

func (r *AnnouncementRepo) ListAll(ctx context.Context, offset int, limit int) ([]*model.Announcement, int64, error) {
	var announcements []*model.Announcement
	db := ndb.Pick().WithContext(ctx).Model(&model.Announcement{})

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&announcements).Error
	return announcements, total, err
}

func (r *AnnouncementRepo) ListAllData(ctx context.Context) ([]*model.Announcement, error) {
	var announcements []*model.Announcement
	err := ndb.Pick().WithContext(ctx).Model(&model.Announcement{}).
		Order("created_at DESC").
		Find(&announcements).Error
	return announcements, err
}
