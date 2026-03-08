package repo

import (
	"app/comm/enum"
	"app/dao/model"
	"app/dao/query"
	"context"
	"errors"
	"time"

	"github.com/zjutjh/mygo/ndb"
	"gorm.io/gorm"
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

func (r *PostRepo) Save(ctx context.Context, record *model.Post) error {
	return ndb.Pick().WithContext(ctx).Save(record).Error
}

type PostListFilter struct {
	PublishType string
	ItemType    string
	Campus      string
	Location    string
	Status      string
	StartTime   time.Time
	EndTime     time.Time
}

type AdminReviewRecordFilter struct {
	ReviewerAdminID int64
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

	if filter.PublishType != "" {
		db = db.Where("publish_type = ?", filter.PublishType)
	}
	if filter.ItemType != "" {
		db = db.Where("item_type = ?", filter.ItemType)
	}
	if filter.Campus != "" {
		db = db.Where("campus = ?", filter.Campus)
	}
	if filter.Location != "" {
		like := "%" + filter.Location + "%"
		db = db.Where("location LIKE ?", like)
	}
	if filter.Status != "" {
		db = db.Where("status = ?", filter.Status)
	}
	if !filter.StartTime.IsZero() {
		db = db.Where("event_time >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		db = db.Where("event_time <= ?", filter.EndTime)
	}

	if err = db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err = db.Order("event_time DESC").Offset(offset).Limit(limit).Find(&records).Error
	return records, total, err
}

// ListByPublisher 查询用户发布的记录列表
func (r *PostRepo) ListByPublisher(ctx context.Context, publisherID int64, publishType string, status string, offset int, limit int) (records []*model.Post, total int64, err error) {
	db := ndb.Pick().WithContext(ctx).Model(&model.Post{}).Where("publisher_id = ?", publisherID)

	if publishType != "" {
		db = db.Where("publish_type = ?", publishType)
	}
	if status != "" {
		db = db.Where("status = ?", status)
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
			"status":        enum.PostStatusCancelled,
			"cancel_reason": reason,
		}).Error
}

// DeletePost 删除发布记录（仅待审核状态可删除）
func (r *PostRepo) DeletePost(ctx context.Context, postID int64, publisherID int64) error {
	return ndb.Pick().WithContext(ctx).
		Where("id = ? AND publisher_id = ? AND status = ?", postID, publisherID, enum.PostStatusPending).
		Delete(&model.Post{}).Error
}

// ListPendingReview 查询待审核的发布列表
func (r *PostRepo) ListPendingReview(ctx context.Context, campus string, offset int, limit int) ([]*model.Post, int64, error) {
	var posts []*model.Post
	db := ndb.Pick().WithContext(ctx).Model(&model.Post{}).Where("status = ?", enum.PostStatusPending)
	if campus != "" {
		db = db.Where("campus = ?", campus)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&posts).Error
	return posts, total, err
}

// ApprovePost 审核通过发布
func (r *PostRepo) ApprovePost(ctx context.Context, postID int64, adminID int64) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("id = ? AND status = ?", postID, enum.PostStatusPending).
		Updates(map[string]interface{}{
			"status":            enum.PostStatusApproved,
			"reviewer_admin_id": adminID,
			"processed_at":      time.Now(),
		}).Error
}

// RejectPost 审核驳回发布
func (r *PostRepo) RejectPost(ctx context.Context, postID int64, reason string, adminID int64) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("id = ? AND status = ?", postID, enum.PostStatusPending).
		Updates(map[string]interface{}{
			"status":            enum.PostStatusRejected,
			"reject_reason":     reason,
			"reviewer_admin_id": adminID,
			"processed_at":      time.Now(),
		}).Error
}

// ListReviewedByAdmin 查询管理员审核过的发布记录
func (r *PostRepo) ListReviewedByAdmin(ctx context.Context, filter AdminReviewRecordFilter, offset int, limit int) (records []*model.Post, total int64, err error) {
	db := ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("reviewer_admin_id = ?", filter.ReviewerAdminID).
		Where("status IN ?", []string{enum.PostStatusApproved, enum.PostStatusRejected})

	if err = db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err = db.Order("processed_at DESC").Offset(offset).Limit(limit).Find(&records).Error
	return records, total, err
}

// CountByStatus 按状态统计数量
func (r *PostRepo) CountByStatus(ctx context.Context) (map[string]int64, error) {
	var results []struct {
		Status string
		Count  int64
	}
	err := ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, res := range results {
		counts[res.Status] = res.Count
	}
	return counts, nil
}

// CountByStatusAndCampus 按状态和校区统计数量
func (r *PostRepo) CountByStatusAndCampus(ctx context.Context, campus string) (map[string]int64, error) {
	var results []struct {
		Status string
		Count  int64
	}
	db := ndb.Pick().WithContext(ctx).Model(&model.Post{})
	if campus != "" {
		db = db.Where("campus = ?", campus)
	}
	err := db.Select("status, count(*) as count").
		Group("status").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, res := range results {
		counts[res.Status] = res.Count
	}
	return counts, nil
}

// CountByItemType 按物品类型统计数量
func (r *PostRepo) CountByItemType(ctx context.Context) (map[string]int64, error) {
	var results []struct {
		ItemType string
		Count    int64
	}
	err := ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Select("item_type, count(*) as count").
		Group("item_type").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, res := range results {
		counts[res.ItemType] = res.Count
	}
	return counts, nil
}

// CountByItemTypeAndCampus 按物品类型和校区统计数量
func (r *PostRepo) CountByItemTypeAndCampus(ctx context.Context, campus string) (map[string]int64, error) {
	var results []struct {
		ItemType string
		Count    int64
	}
	db := ndb.Pick().WithContext(ctx).Model(&model.Post{})
	if campus != "" {
		db = db.Where("campus = ?", campus)
	}
	err := db.Select("item_type, count(*) as count").
		Group("item_type").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, res := range results {
		counts[res.ItemType] = res.Count
	}
	return counts, nil
}

// UpdateStatus 更新发布状态
func (r *PostRepo) UpdateStatus(ctx context.Context, postID int64, status string) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("id = ?", postID).
		Update("status", status).Error
}

// MarkAsSolved 标记为已解决
func (r *PostRepo) MarkAsSolved(ctx context.Context, postID int64) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("id = ?", postID).
		Updates(map[string]interface{}{
			"status":       enum.PostStatusSolved,
			"processed_at": time.Now(),
		}).Error
}

// DeletePostByAdmin 管理员删除发布记录（不限状态）
func (r *PostRepo) DeletePostByAdmin(ctx context.Context, postID int64) error {
	return ndb.Pick().WithContext(ctx).
		Where("id = ?", postID).
		Delete(&model.Post{}).Error
}

// CountTodayByPublisher 统计用户当天发布数量
func (r *PostRepo) CountTodayByPublisher(ctx context.Context, publisherID int64) (int64, error) {
	var count int64
	today := time.Now().Format("2006-01-02")
	startTime, _ := time.ParseInLocation("2006-01-02", today, time.Local)
	endTime := startTime.Add(24 * time.Hour)

	err := ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("publisher_id = ? AND created_at >= ? AND created_at < ?", publisherID, startTime, endTime).
		Count(&count).Error
	return count, err
}

// IncrementClaimCount 增加认领人数
func (r *PostRepo) IncrementClaimCount(ctx context.Context, postID int64) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("id = ?", postID).
		Update("claim_count", gorm.Expr("claim_count + 1")).Error
}

// DecrementClaimCount 减少认领人数（最小保持为0）
func (r *PostRepo) DecrementClaimCount(ctx context.Context, postID int64) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("id = ? AND claim_count > 0", postID).
		Update("claim_count", gorm.Expr("claim_count - 1")).Error
}

// ArchivePost 归档发布记录
func (r *PostRepo) ArchivePost(ctx context.Context, postID int64, archiveMethod string) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("id = ?", postID).
		Updates(map[string]interface{}{
			"status":         enum.PostStatusArchived,
			"archive_method": archiveMethod,
			"processed_at":   time.Now(),
		}).Error
}

// MigrateItemTypeToOther 将指定物品类型的所有数据迁移到其他类型
func (r *PostRepo) MigrateItemTypeToOther(ctx context.Context, oldType, newType string) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("item_type = ?", oldType).
		Updates(map[string]interface{}{
			"item_type": newType,
		}).Error
}

// FindByIds 根据ID列表查询发布记录
func (r *PostRepo) FindByIds(ctx context.Context, ids []int64) ([]*model.Post, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var posts []*model.Post
	err := ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("id IN ?", ids).
		Find(&posts).Error
	return posts, err
}

// UpdateSummary 更新发布记录的总结文本
func (r *PostRepo) UpdateSummary(ctx context.Context, postID int64, summary string) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("id = ?", postID).
		Update("summary", summary).Error
}

func (r *PostRepo) ListAll(ctx context.Context) ([]*model.Post, error) {
	var posts []*model.Post
	err := ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Order("created_at DESC").
		Find(&posts).Error
	return posts, err
}

func (r *PostRepo) ListExpired(ctx context.Context) ([]*model.Post, error) {
	var posts []*model.Post
	err := ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("status IN ?", []string{enum.PostStatusArchived, enum.PostStatusCancelled, enum.PostStatusRejected}).
		Order("created_at DESC").
		Find(&posts).Error
	return posts, err
}

func (r *PostRepo) DeleteExpired(ctx context.Context) error {
	return ndb.Pick().WithContext(ctx).Model(&model.Post{}).
		Where("status IN ?", []string{enum.PostStatusArchived, enum.PostStatusCancelled, enum.PostStatusRejected}).
		Delete(&model.Post{}).Error
}
