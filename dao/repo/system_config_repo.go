package repo

import (
	"app/dao/model"
	"context"
	"errors"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/zjutjh/mygo/ndb"
	"gorm.io/gorm"
)

const (
	ConfigKeyFeedbackTypes     = "feedback_types"
	ConfigKeyItemTypes         = "item_types"
	ConfigKeyClaimValidityDays = "claim_validity_days"
	ConfigKeyPublishLimit      = "publish_limit"
)

type SystemConfigRepo struct{}

func NewSystemConfigRepo() *SystemConfigRepo {
	return &SystemConfigRepo{}
}

func (r *SystemConfigRepo) GetByKey(ctx context.Context, key string) (*model.SystemConfig, error) {
	var config model.SystemConfig
	err := ndb.Pick().WithContext(ctx).Where("config_key = ?", key).First(&config).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *SystemConfigRepo) GetAll(ctx context.Context) ([]*model.SystemConfig, error) {
	var configs []*model.SystemConfig
	err := ndb.Pick().WithContext(ctx).Find(&configs).Error
	return configs, err
}

func (r *SystemConfigRepo) UpdateValue(ctx context.Context, key, value string) error {
	return ndb.Pick().WithContext(ctx).Model(&model.SystemConfig{}).
		Where("config_key = ?", key).
		Update("config_value", value).Error
}

func (r *SystemConfigRepo) GetFeedbackTypes(ctx context.Context) ([]string, error) {
	config, err := r.GetByKey(ctx, ConfigKeyFeedbackTypes)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return r.getDefaultFeedbackTypes(), nil
	}

	var types []string
	if err := sonic.UnmarshalString(config.ConfigValue, &types); err != nil {
		return r.getDefaultFeedbackTypes(), nil
	}
	return types, nil
}

func (r *SystemConfigRepo) GetItemTypes(ctx context.Context) ([]string, error) {
	config, err := r.GetByKey(ctx, ConfigKeyItemTypes)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return r.getDefaultItemTypes(), nil
	}

	var types []string
	if err := sonic.UnmarshalString(config.ConfigValue, &types); err != nil {
		return r.getDefaultItemTypes(), nil
	}
	return types, nil
}

func (r *SystemConfigRepo) GetClaimValidityDays(ctx context.Context) (int, error) {
	config, err := r.GetByKey(ctx, ConfigKeyClaimValidityDays)
	if err != nil {
		return 30, err
	}
	if config == nil {
		return 30, nil
	}

	days, err := strconv.Atoi(config.ConfigValue)
	if err != nil {
		return 30, nil
	}
	return days, nil
}

func (r *SystemConfigRepo) UpdateFeedbackTypes(ctx context.Context, types []string) error {
	data, err := sonic.MarshalString(types)
	if err != nil {
		return err
	}
	return r.UpdateValue(ctx, ConfigKeyFeedbackTypes, data)
}

func (r *SystemConfigRepo) UpdateItemTypes(ctx context.Context, types []string) error {
	data, err := sonic.MarshalString(types)
	if err != nil {
		return err
	}
	return r.UpdateValue(ctx, ConfigKeyItemTypes, data)
}

func (r *SystemConfigRepo) UpdateClaimValidityDays(ctx context.Context, days int) error {
	return r.UpdateValue(ctx, ConfigKeyClaimValidityDays, strconv.Itoa(days))
}

func (r *SystemConfigRepo) getDefaultFeedbackTypes() []string {
	return []string{"恶意发布", "信息不全", "不实消息", "恶心血腥", "涉黄信息"}
}

func (r *SystemConfigRepo) getDefaultItemTypes() []string {
	return []string{"电子", "饭卡", "文体", "证件", "衣包", "饰品"}
}

func (r *SystemConfigRepo) GetPublishLimit(ctx context.Context) (int, error) {
	config, err := r.GetByKey(ctx, ConfigKeyPublishLimit)
	if err != nil {
		return 10, err
	}
	if config == nil {
		return 10, nil
	}

	limit, err := strconv.Atoi(config.ConfigValue)
	if err != nil {
		return 10, nil
	}
	return limit, nil
}

func (r *SystemConfigRepo) UpdatePublishLimit(ctx context.Context, limit int) error {
	return r.UpdateValue(ctx, ConfigKeyPublishLimit, strconv.Itoa(limit))
}
