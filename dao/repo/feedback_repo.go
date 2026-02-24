package repo

import (
	"context"
	"errors"
	"time"

	"github.com/zjutjh/mygo/ndb"
	"gorm.io/gorm"

	"app/dao/model"
)

type FeedbackRepo struct{}

func NewFeedbackRepo() *FeedbackRepo {
	return &FeedbackRepo{}
}

// Create 创建投诉反馈
func (r *FeedbackRepo) Create(ctx context.Context, feedback *model.Feedback) error {
	return ndb.Pick().WithContext(ctx).Create(feedback).Error
}

// FindById 根据ID查询投诉反馈
func (r *FeedbackRepo) FindById(ctx context.Context, id int64) (*model.Feedback, error) {
	var feedback model.Feedback
	err := ndb.Pick().WithContext(ctx).Where("id = ?", id).First(&feedback).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &feedback, nil
}

// ListAll 查询所有投诉反馈（分页）
func (r *FeedbackRepo) ListAll(ctx context.Context, offset int, limit int) ([]*model.Feedback, int64, error) {
	var feedbacks []*model.Feedback
	db := ndb.Pick().WithContext(ctx).Model(&model.Feedback{})

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&feedbacks).Error
	return feedbacks, total, err
}

// ListByStatus 按状态查询投诉反馈
func (r *FeedbackRepo) ListByStatus(ctx context.Context, status int8, offset int, limit int) ([]*model.Feedback, int64, error) {
	var feedbacks []*model.Feedback
	db := ndb.Pick().WithContext(ctx).Model(&model.Feedback{}).Where("status = ?", status)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&feedbacks).Error
	return feedbacks, total, err
}

// UpdateStatus 更新投诉反馈状态
func (r *FeedbackRepo) UpdateStatus(ctx context.Context, id int64, status int8, processedBy int64) error {
	now := time.Now()
	return ndb.Pick().WithContext(ctx).Model(&model.Feedback{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       status,
			"processed_by": processedBy,
			"processed_at": now,
		}).Error
}

// ListByReporter 按投诉者查询投诉反馈（分页）
func (r *FeedbackRepo) ListByReporter(ctx context.Context, reporterID int64, offset int, limit int) ([]*model.Feedback, int64, error) {
	var feedbacks []*model.Feedback
	db := ndb.Pick().WithContext(ctx).Model(&model.Feedback{}).Where("reporter_id = ?", reporterID)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&feedbacks).Error
	return feedbacks, total, err
}

// ListByReporterAndStatus 按投诉者和状态查询投诉反馈
func (r *FeedbackRepo) ListByReporterAndStatus(ctx context.Context, reporterID int64, status int8, offset int, limit int) ([]*model.Feedback, int64, error) {
	var feedbacks []*model.Feedback
	db := ndb.Pick().WithContext(ctx).Model(&model.Feedback{}).
		Where("reporter_id = ? AND status = ?", reporterID, status)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&feedbacks).Error
	return feedbacks, total, err
}
