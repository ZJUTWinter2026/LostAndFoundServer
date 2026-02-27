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
)

type PublishPostInput struct {
	PublishType       string   `json:"publish_type" jsonschema:"description=发布类型: LOST(寻物), FOUND(招领),required"`
	ItemName          string   `json:"item_name" jsonschema:"description=物品名称,required"`
	ItemType          string   `json:"item_type" jsonschema:"description=物品类型,required"`
	Campus            string   `json:"campus" jsonschema:"description=校区: ZHAO_HUI, PING_FENG, MO_GAN_SHAN,required"`
	Location          string   `json:"location" jsonschema:"description=地点,required"`
	StorageLocation   string   `json:"storage_location" jsonschema:"description=存放地点,required"`
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
		return &PublishPostOutput{Success: false, Message: "工具上下文未初始化"}, nil
	}

	postRepo := repo.NewPostRepo()

	eventTime, err := time.Parse(time.DateTime, input.EventTime)
	if err != nil {
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
		ProcessedAt:       time.Now(),
	}

	err = postRepo.Create(ctx, record)
	if err != nil {
		return &PublishPostOutput{Success: false, Message: "创建发布记录失败"}, nil
	}

	vectorSvc := vector.NewService()
	if err := vectorSvc.UpdatePostVector(ctx, record); err != nil {
		return &PublishPostOutput{Success: false, Message: "更新向量失败"}, nil
	}

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
