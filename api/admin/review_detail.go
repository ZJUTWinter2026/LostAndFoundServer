package admin

import (
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/comm/enum"
	"app/dao/repo"
)

// ReviewDetailHandler API router注册点
func ReviewDetailHandler() gin.HandlerFunc {
	api := ReviewDetailApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfReviewDetail).Pointer()).Name()] = api
	return hfReviewDetail
}

type ReviewDetailApi struct {
	Info     struct{}                `name:"待审核发布详情" desc:"待审核发布详情"`
	Request  ReviewDetailApiRequest  // API请求参数
	Response ReviewDetailApiResponse // API响应数据
}

type ReviewDetailApiRequest struct {
	Body struct {
		PostID int64 `uri:"post_id" binding:"required" desc:"发布ID"`
	}
}

type ReviewDetailApiResponse struct {
	ID            int64     `json:"id" desc:"发布ID"`
	PublishType   string    `json:"publish_type" desc:"发布类型 LOST/FOUND"`
	ItemName      string    `json:"item_name" desc:"物品名称"`
	ItemType      string    `json:"item_type" desc:"物品类型"`
	ItemTypeOther string    `json:"item_type_other" desc:"其它类型说明"`
	Location      string    `json:"location" desc:"地点"`
	EventTime     time.Time `json:"event_time" desc:"事件时间"`
	Features      string    `json:"features" desc:"物品特征"`
	ContactName   string    `json:"contact_name" desc:"联系人"`
	ContactPhone  string    `json:"contact_phone" desc:"联系电话"`
	HasReward     bool      `json:"has_reward" desc:"是否有悬赏"`
	Images        []string  `json:"images" desc:"图片列表"`
	Status        string    `json:"status" desc:"状态"`
	PublisherID   int64     `json:"publisher_id" desc:"发布者ID"`
	CreatedAt     time.Time `json:"created_at" desc:"创建时间"`
}

// Run Api业务逻辑执行点
func (r *ReviewDetailApi) Run(ctx *gin.Context) kit.Code {
	request := r.Request.Body

	// 获取当前用户并验证是管理员
	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	adminID := cast.ToInt64(id)

	// 验证管理员权限
	urp := repo.NewUserRepo()
	user, err := urp.FindById(ctx, adminID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询用户失败")
		return comm.CodeServerError
	}
	if user == nil || user.Usertype != enum.UserTypeAdmin {
		return comm.CodeAdminPermissionDenied
	}

	// 查询发布记录
	prp := repo.NewPostRepo()
	post, err := prp.FindById(ctx, request.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询发布记录失败")
		return comm.CodeServerError
	}
	if post == nil {
		return comm.CodeDataNotFound
	}

	// 必须是待审核状态
	if post.Status != enum.PostStatusPending {
		return comm.CodePostStatusInvalid
	}

	r.Response = ReviewDetailApiResponse{
		ID:            post.ID,
		PublishType:   post.PublishType,
		ItemName:      post.ItemName,
		ItemType:      post.ItemType,
		ItemTypeOther: post.ItemTypeOther,
		Location:      post.Location,
		EventTime:     post.EventTime,
		Features:      post.Features,
		ContactName:   post.ContactName,
		ContactPhone:  post.ContactPhone,
		HasReward:     post.HasReward,
		Images:        comm.UnmarshalImages(post.Images),
		Status:        post.Status,
		PublisherID:   post.PublisherID,
		CreatedAt:     post.CreatedAt,
	}
	return comm.CodeOK
}

// Init Api初始化
func (r *ReviewDetailApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindUri(&r.Request.Body)
}

// hfReviewDetail API执行入口
func hfReviewDetail(ctx *gin.Context) {
	api := &ReviewDetailApi{}
	err := api.Init(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("参数绑定校验错误")
		reply.Fail(ctx, comm.CodeParameterInvalid)
		return
	}
	code := api.Run(ctx)
	if !ctx.IsAborted() {
		if code == comm.CodeOK {
			reply.Success(ctx, api.Response)
		} else {
			reply.Fail(ctx, code)
		}
	}
}
