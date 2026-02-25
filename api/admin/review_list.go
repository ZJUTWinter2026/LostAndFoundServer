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

// ReviewListHandler API router注册点
func ReviewListHandler() gin.HandlerFunc {
	api := ReviewListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfReviewList).Pointer()).Name()] = api
	return hfReviewList
}

type ReviewListApi struct {
	Info     struct{}              `name:"待审核发布列表" desc:"待审核发布列表"`
	Request  ReviewListApiRequest  // API请求参数
	Response ReviewListApiResponse // API响应数据
}

type ReviewListApiRequest struct {
	Body struct {
		Page     int `form:"page" binding:"omitempty,min=1" desc:"页码"`
		PageSize int `form:"page_size" binding:"omitempty,min=1,max=50" desc:"每页数量"`
	}
}

type ReviewListApiResponse struct {
	Total    int64            `json:"total" desc:"总数"`
	Page     int              `json:"page" desc:"页码"`
	PageSize int              `json:"page_size" desc:"每页数量"`
	List     []ReviewListItem `json:"list" desc:"待审核列表"`
}

type ReviewListItem struct {
	ID          int64     `json:"id" desc:"发布ID"`
	PublishType string    `json:"publish_type" desc:"发布类型 LOST/FOUND"`
	ItemName    string    `json:"item_name" desc:"物品名称"`
	ItemType    string    `json:"item_type" desc:"物品类型"`
	Location    string    `json:"location" desc:"地点"`
	EventTime   time.Time `json:"event_time" desc:"事件时间"`
	ContactName string    `json:"contact_name" desc:"联系人"`
	CreatedAt   time.Time `json:"created_at" desc:"创建时间"`
}

// Run Api业务逻辑执行点
func (r *ReviewListApi) Run(ctx *gin.Context) kit.Code {
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
		return comm.CodeDatabaseError
	}
	if user == nil || user.Usertype != enum.UserTypeAdmin {
		return comm.CodeAdminPermissionDenied
	}

	page := request.Page
	pageSize := request.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize
	prp := repo.NewPostRepo()

	posts, total, err := prp.ListPendingReview(ctx, offset, pageSize)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询待审核列表失败")
		return comm.CodeDatabaseError
	}

	items := make([]ReviewListItem, 0, len(posts))
	for _, post := range posts {
		items = append(items, ReviewListItem{
			ID:          post.ID,
			PublishType: post.PublishType,
			ItemName:    post.ItemName,
			ItemType:    post.ItemType,
			Location:    post.Location,
			EventTime:   post.EventTime,
			ContactName: post.ContactName,
			CreatedAt:   post.CreatedAt,
		})
	}

	r.Response = ReviewListApiResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}
	return comm.CodeOK
}

// Init Api初始化
func (r *ReviewListApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindQuery(&r.Request.Body)
}

// hfReviewList API执行入口
func hfReviewList(ctx *gin.Context) {
	api := &ReviewListApi{}
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
