package tools

import (
	"app/comm/enum"
	"app/dao/repo"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/zjutjh/mygo/nlog"
)

type GetPostDetailInput struct {
	PostID int64 `json:"post_id" jsonschema:"description=发布ID,required"`
}

type GetPostDetailOutput struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type PostDetailData struct {
	ID                int64      `json:"id"`
	PublishType       string     `json:"publish_type"`
	ItemName          string     `json:"item_name"`
	ItemType          string     `json:"item_type"`
	Campus            string     `json:"campus"`
	Location          string     `json:"location"`
	StorageLocation   string     `json:"storage_location"`
	EventTime         time.Time  `json:"event_time"`
	Features          string     `json:"features"`
	ContactName       string     `json:"contact_name"`
	ContactPhone      string     `json:"contact_phone"`
	HasReward         bool       `json:"has_reward"`
	RewardDescription string     `json:"reward_description"`
	Images            []string   `json:"images"`
	Status            string     `json:"status"`
	CancelReason      string     `json:"cancel_reason"`
	RejectReason      string     `json:"reject_reason"`
	ClaimCount        int32      `json:"claim_count"`
	ArchiveMethod     string     `json:"archive_method"`
	ProcessedAt       *time.Time `json:"processed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

func getPostDetailFunc(ctx context.Context, input *GetPostDetailInput) (*GetPostDetailOutput, error) {
	nlog.Pick().WithContext(ctx).Infof("[Tool:get_post_detail] 调用参数: post_id=%d", input.PostID)
	tc := GetToolContext(ctx)
	if tc == nil {
		nlog.Pick().WithContext(ctx).Warn("[Tool:get_post_detail] 工具上下文未初始化")
		return &GetPostDetailOutput{Success: false, Message: "工具上下文未初始化"}, nil
	}

	postRepo := repo.NewPostRepo()
	userRepo := repo.NewUserRepo()

	post, err := postRepo.FindById(ctx, input.PostID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		nlog.Pick().WithContext(ctx).Infof("[Tool:get_post_detail] 发布记录不存在: post_id=%d", input.PostID)
		return &GetPostDetailOutput{Success: false, Message: "发布记录不存在"}, nil
	}
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:get_post_detail] 查询发布记录失败")
		return &GetPostDetailOutput{Success: false, Message: "查询发布记录失败"}, nil
	}

	user, err := userRepo.FindById(ctx, tc.UserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &GetPostDetailOutput{Success: false, Message: "当前用户不存在"}, nil
	}
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:get_post_detail] 查询当前用户失败")
		return &GetPostDetailOutput{Success: false, Message: "查询当前用户失败"}, nil
	}

	isAdmin := user.Usertype == enum.UserTypeAdmin || user.Usertype == enum.UserTypeSystemAdmin
	isOwner := post.PublisherID == tc.UserID
	if !isAdmin && !isOwner && post.Status != enum.PostStatusApproved {
		nlog.Pick().WithContext(ctx).Infof("[Tool:get_post_detail] 用户无权查看未公开帖子: user_id=%d, post_id=%d", tc.UserID, input.PostID)
		return &GetPostDetailOutput{Success: false, Message: "发布记录不存在"}, nil
	}

	var images []string
	if post.Images != "" {
		if err := sonic.UnmarshalString(post.Images, &images); err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("[Tool:get_post_detail] 解析图片失败")
			return &GetPostDetailOutput{Success: false, Message: "解析图片失败"}, nil
		}
	}

	contactName := ""
	contactPhone := ""
	if isAdmin || isOwner {
		contactName = post.ContactName
		contactPhone = post.ContactPhone
	}

	data := PostDetailData{
		ID:                post.ID,
		PublishType:       post.PublishType,
		ItemName:          post.ItemName,
		ItemType:          post.ItemType,
		Campus:            post.Campus,
		Location:          post.Location,
		StorageLocation:   post.StorageLocation,
		EventTime:         post.EventTime,
		Features:          post.Features,
		ContactName:       contactName,
		ContactPhone:      contactPhone,
		HasReward:         post.HasReward,
		RewardDescription: post.RewardDescription,
		Images:            images,
		Status:            post.Status,
		CancelReason:      post.CancelReason,
		RejectReason:      post.RejectReason,
		ClaimCount:        post.ClaimCount,
		ArchiveMethod:     post.ArchiveMethod,
		ProcessedAt:       post.ProcessedAt,
		CreatedAt:         post.CreatedAt,
	}

	nlog.Pick().WithContext(ctx).Infof("[Tool:get_post_detail] 查询成功: post_id=%d, item_name=%s", post.ID, post.ItemName)
	return &GetPostDetailOutput{
		Success: true,
		Data:    data,
	}, nil
}

func NewGetPostDetailTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"get_post_detail",
		"根据post_id获取失物/招领信息的详细内容",
		getPostDetailFunc,
	)
}
