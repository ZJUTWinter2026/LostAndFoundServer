package repo

import (
	"context"
	"errors"
	"time"

	"github.com/zjutjh/mygo/ndb"
	"gorm.io/gorm"

	"app/comm/enum"
	"app/dao/model"
)

type ClaimRepo struct{}

func NewClaimRepo() *ClaimRepo {
	return &ClaimRepo{}
}

// Create 创建认领申请
func (r *ClaimRepo) Create(ctx context.Context, claim *model.Claim) error {
	return ndb.Pick().WithContext(ctx).Create(claim).Error
}

// FindById 根据ID查询认领申请
func (r *ClaimRepo) FindById(ctx context.Context, id int64) (*model.Claim, error) {
	var claimRecord model.Claim
	err := ndb.Pick().WithContext(ctx).Where("id = ?", id).First(&claimRecord).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &claimRecord, nil
}

// HasPendingOrMatchedClaim 检查用户是否已有待确认或已匹配的申请
func (r *ClaimRepo) HasPendingOrMatchedClaim(ctx context.Context, postID int64, claimantID int64) (bool, error) {
	var count int64
	err := ndb.Pick().WithContext(ctx).Model(&model.Claim{}).
		Where("post_id = ? AND claimant_id = ? AND status IN (?, ?)", postID, claimantID, enum.ClaimStatusPending, enum.ClaimStatusMatched).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasMatchedClaim 检查物品是否已有已匹配的认领
func (r *ClaimRepo) HasMatchedClaim(ctx context.Context, postID int64) (bool, error) {
	var count int64
	err := ndb.Pick().WithContext(ctx).Model(&model.Claim{}).
		Where("post_id = ? AND status = ?", postID, enum.ClaimStatusMatched).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListByPostID 根据发布ID查询认领申请列表
func (r *ClaimRepo) ListByPostID(ctx context.Context, postID int64) ([]*model.Claim, error) {
	var claims []*model.Claim
	err := ndb.Pick().WithContext(ctx).
		Where("post_id = ?", postID).
		Order("created_at DESC").
		Find(&claims).Error
	return claims, err
}

// UpdateStatus 更新认领申请状态
func (r *ClaimRepo) UpdateStatus(ctx context.Context, id int64, status string, reviewedBy int64) error {
	now := time.Now()
	return ndb.Pick().WithContext(ctx).Model(&model.Claim{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      status,
			"reviewed_by": reviewedBy,
			"reviewed_at": now,
		}).Error
}

// ListByClaimant 根据认领者ID查询认领申请列表
func (r *ClaimRepo) ListByClaimant(ctx context.Context, claimantID int64, offset int, limit int) ([]*model.Claim, int64, error) {
	var claims []*model.Claim
	db := ndb.Pick().WithContext(ctx).Model(&model.Claim{}).Where("claimant_id = ?", claimantID)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&claims).Error
	return claims, total, err
}

// Delete 删除认领申请（仅待确认状态可删除）
func (r *ClaimRepo) Delete(ctx context.Context, id int64, claimantID int64) error {
	return ndb.Pick().WithContext(ctx).
		Where("id = ? AND claimant_id = ? AND status = ?", id, claimantID, enum.ClaimStatusPending).
		Delete(&model.Claim{}).Error
}

func (r *ClaimRepo) ListAll(ctx context.Context) ([]*model.Claim, error) {
	var claims []*model.Claim
	err := ndb.Pick().WithContext(ctx).Model(&model.Claim{}).
		Order("created_at DESC").
		Find(&claims).Error
	return claims, err
}
