package repo

import (
	"context"
	"time"

	"github.com/zjutjh/mygo/ndb"

	"app/dao/model"
)

type FeedbackRepo struct{}

func NewFeedbackRepo() *FeedbackRepo {
	return &FeedbackRepo{}
}

func (r *FeedbackRepo) Create(ctx context.Context, feedback *model.Feedback) error {
	return ndb.Pick().WithContext(ctx).Create(feedback).Error
}

func (r *FeedbackRepo) FindById(ctx context.Context, id int64) (*model.Feedback, error) {
	var feedback model.Feedback
	err := ndb.Pick().WithContext(ctx).Where("id = ?", id).First(&feedback).Error
	if err != nil {
		return nil, err
	}
	return &feedback, nil
}

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

func (r *FeedbackRepo) ListByProcessed(ctx context.Context, processed string, offset int, limit int) ([]*model.Feedback, int64, error) {
	var feedbacks []*model.Feedback
	db := ndb.Pick().WithContext(ctx).Model(&model.Feedback{}).Where("processed = ?", processed == "YES")

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&feedbacks).Error
	return feedbacks, total, err
}

func (r *FeedbackRepo) MarkAsProcessed(ctx context.Context, id int64, processedBy int64) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Feedback{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"processed":    true,
			"processed_by": processedBy,
			"processed_at": time.Now(),
		}).Error
}

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

func (r *FeedbackRepo) ListByReporterAndProcessed(ctx context.Context, reporterID int64, processed string, offset int, limit int) ([]*model.Feedback, int64, error) {
	var feedbacks []*model.Feedback
	db := ndb.Pick().WithContext(ctx).Model(&model.Feedback{}).
		Where("reporter_id = ? AND processed = ?", reporterID, processed == "YES")

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&feedbacks).Error
	return feedbacks, total, err
}

// MigrateTypeToOther 将指定投诉类型的所有数据迁移到其他类型
func (r *FeedbackRepo) MigrateTypeToOther(ctx context.Context, oldType, newType string) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Feedback{}).
		Where("type = ?", oldType).
		Updates(map[string]interface{}{
			"type": newType,
		}).Error
}

func (r *FeedbackRepo) ListAllData(ctx context.Context) ([]*model.Feedback, error) {
	var feedbacks []*model.Feedback
	err := ndb.Pick().WithContext(ctx).Model(&model.Feedback{}).
		Order("created_at DESC").
		Find(&feedbacks).Error
	return feedbacks, err
}
