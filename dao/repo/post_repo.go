package repo

import (
	"app/dao/query"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/zjutjh/mygo/ndb"
	"gorm.io/gorm"

	"app/dao/model"
)

type PostRepo struct {
	query *query.Query
}

func NewPostRepo() *PostRepo {
	return &PostRepo{
		query: query.Use(ndb.Pick()),
	}
}

// Create 新增发布记录
func (r *PostRepo) Create(ctx context.Context, record *model.Post) error {
	return ndb.Pick().WithContext(ctx).Create(record).Error
}

type PostListFilter struct {
	ItemType  string
	Location  string
	Status    *int8
	StartTime *time.Time
	EndTime   *time.Time
}

// FindById 根据ID查询发布记录
func (r *PostRepo) FindById(ctx context.Context, id int64) (*model.Post, error) {
	p := r.query.Post
	record, err := p.WithContext(ctx).Where(p.ID.Eq(id)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

// ListByFilter 按条件查询发布记录列表
func (r *PostRepo) ListByFilter(ctx context.Context, filter PostListFilter, offset int, limit int) (records []*model.Post, total int64, err error) {
	db := ndb.Pick().WithContext(ctx).Model(&model.Post{})

	if strings.TrimSpace(filter.ItemType) != "" {
		db = db.Where("item_type = ?", strings.TrimSpace(filter.ItemType))
	}
	if strings.TrimSpace(filter.Location) != "" {
		like := "%" + strings.TrimSpace(filter.Location) + "%"
		db = db.Where("location LIKE ?", like)
	}
	if filter.Status != nil {
		db = db.Where("status = ?", *filter.Status)
	}
	if filter.StartTime != nil {
		db = db.Where("event_time >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		db = db.Where("event_time <= ?", *filter.EndTime)
	}

	if err = db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err = db.Order("event_time DESC").Offset(offset).Limit(limit).Find(&records).Error
	return records, total, err
}

// ListByPublisher 查询用户发布的记录列表
func (r *PostRepo) ListByPublisher(ctx context.Context, publisherID int64, publishType *int8, status *int8, offset int, limit int) (records []*model.Post, total int64, err error) {
	db := ndb.Pick().WithContext(ctx).Model(&model.Post{}).Where("publisher_id = ?", publisherID)

	if publishType != nil {
		db = db.Where("publish_type = ?", *publishType)
	}
	if status != nil {
		db = db.Where("status = ?", *status)
	}

	if err = db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err = db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&records).Error
	return records, total, err
}

// UpdatePost 更新发布记录
func (r *PostRepo) UpdatePost(ctx context.Context, postID int64, publisherID int64, updates map[string]interface{}) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("id = ? AND publisher_id = ?", postID, publisherID).
		Updates(updates).Error
}

// CancelPost 取消发布
func (r *PostRepo) CancelPost(ctx context.Context, postID int64, publisherID int64, reason string) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("id = ? AND publisher_id = ?", postID, publisherID).
		Updates(map[string]interface{}{
			"status":        4, // 已取消
			"cancel_reason": reason,
		}).Error
}

// DeletePost 删除发布记录（仅待审核状态可删除）
func (r *PostRepo) DeletePost(ctx context.Context, postID int64, publisherID int64) error {
	return ndb.Pick().WithContext(ctx).
		Where("id = ? AND publisher_id = ? AND status = 0", postID, publisherID).
		Delete(&model.Post{}).Error
}

// ListPendingReview 查询待审核的发布列表
func (r *PostRepo) ListPendingReview(ctx context.Context, offset int, limit int) ([]*model.Post, int64, error) {
	var posts []*model.Post
	db := ndb.Pick().WithContext(ctx).Model(&model.Post{}).Where("status = 0")

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&posts).Error
	return posts, total, err
}

// ApprovePost 审核通过发布
func (r *PostRepo) ApprovePost(ctx context.Context, postID int64) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("id = ? AND status = 0", postID).
		Updates(map[string]interface{}{
			"status":       1, // 已通过
			"processed_at": time.Now(),
		}).Error
}

// RejectPost 审核驳回发布
func (r *PostRepo) RejectPost(ctx context.Context, postID int64, reason string) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("id = ? AND status = 0", postID).
		Updates(map[string]interface{}{
			"status":        5, // 已驳回
			"reject_reason": reason,
			"processed_at":  time.Now(),
		}).Error
}

// MarkAsMatched 标记为已匹配
func (r *PostRepo) MarkAsMatched(ctx context.Context, postID int64) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("id = ?", postID).
		Updates(map[string]interface{}{
			"status":       2, // 已匹配
			"processed_at": time.Now(),
		}).Error
}
