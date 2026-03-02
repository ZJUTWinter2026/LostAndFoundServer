package tools

import (
	"app/comm/enum"
	daomodel "app/dao/model"
	"app/dao/repo"
	"app/pkg/vector"
	"context"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/zjutjh/mygo/nlog"
)

type PublishPostInput struct {
	PublishType       string   `json:"publish_type" jsonschema:"description=发布类型: LOST(寻物), FOUND(招领),required,enum=LOST,enum=FOUND"`
	ItemName          string   `json:"item_name" jsonschema:"description=物品名称,required"`
	ItemType          string   `json:"item_type" jsonschema:"description=物品类型,required"`
	Campus            string   `json:"campus" jsonschema:"description=校区,required,enum=ZHAO_HUI,enum=PING_FENG,enum=MO_GAN_SHAN"`
	Location          string   `json:"location" jsonschema:"description=地点,required"`
	StorageLocation   string   `json:"storage_location" jsonschema:"description=存放地点（仅招领有效）"`
	EventTime         string   `json:"event_time" jsonschema:"description=事件时间，格式: 2006-01-02 15:04:05,required"`
	Features          string   `json:"features" jsonschema:"description=物品特征描述,required"`
	ContactName       string   `json:"contact_name" jsonschema:"description=联系人姓名,required"`
	ContactPhone      string   `json:"contact_phone" jsonschema:"description=联系电话,required"`
	HasReward         bool     `json:"has_reward" jsonschema:"description=是否有悬赏"`
	RewardDescription string   `json:"reward_description" jsonschema:"description=悬赏说明(仅has_reward为true时有效)"`
	Images            []string `json:"images" jsonschema:"description=图片URL列表"`
}

type PublishPostOutput struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

func publishPostFunc(ctx context.Context, input *PublishPostInput) (*PublishPostOutput, error) {
	tc := GetToolContext(ctx)
	if tc == nil {
		nlog.Pick().WithContext(ctx).Warn("[Tool:publish_post] 工具上下文未初始化")
		return &PublishPostOutput{Success: false, Message: "工具上下文未初始化"}, nil
	}

	nlog.Pick().WithContext(ctx).Infof("[Tool:publish_post] 调用参数: user_id=%d, publish_type=%s, item_name=%s, item_type=%s, campus=%s", tc.UserID, input.PublishType, input.ItemName, input.ItemType, input.Campus)

	postRepo := repo.NewPostRepo()

	eventTime, err := time.ParseInLocation(time.DateTime, input.EventTime, time.Local)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:publish_post] 事件时间格式错误")
		return &PublishPostOutput{Success: false, Message: "事件时间格式错误"}, nil
	}

	imagesJSON, _ := sonic.MarshalString(input.Images)

	record := &daomodel.Post{
		PublisherID:       tc.UserID,
		PublishType:       input.PublishType,
		ItemName:          strings.TrimSpace(input.ItemName),
		ItemType:          input.ItemType,
		Campus:            input.Campus,
		Location:          input.Location,
		StorageLocation:   input.StorageLocation,
		EventTime:         eventTime,
		Features:          input.Features,
		ContactName:       input.ContactName,
		ContactPhone:      input.ContactPhone,
		HasReward:         input.HasReward,
		RewardDescription: input.RewardDescription,
		Images:            imagesJSON,
		Status:            enum.PostStatusPending,
	}

	err = postRepo.Create(ctx, record)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:publish_post] 创建发布记录失败")
		return &PublishPostOutput{Success: false, Message: "创建发布记录失败"}, nil
	}

	vectorSvc := vector.NewService()
	if err := vectorSvc.UpdatePostVector(ctx, record); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:publish_post] 更新向量失败")
		return &PublishPostOutput{Success: false, Message: "更新向量失败"}, nil
	}

	nlog.Pick().WithContext(ctx).Infof("[Tool:publish_post] 发布成功: post_id=%d", record.ID)
	return &PublishPostOutput{
		Success: true,
		Message: "发布成功，等待审核",
		Data: map[string]interface{}{
			"id": record.ID,
		},
	}, nil
}

func NewPublishPostTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"publish_post",
		"帮用户发布失物或招领信息",
		publishPostFunc,
	)
}
