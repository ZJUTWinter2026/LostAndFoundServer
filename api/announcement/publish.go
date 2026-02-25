package announcement

import (
	"reflect"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/comm/enum"
	"app/dao/model"
	"app/dao/repo"
)

func PublishHandler() gin.HandlerFunc {
	api := PublishApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfPublish).Pointer()).Name()] = api
	return hfPublish
}

type PublishApi struct {
	Info     struct{}         `name:"发布公告" desc:"系统管理员发布全局公告"`
	Request  PublishApiRequest
	Response PublishApiResponse
}

type PublishApiRequest struct {
	Body struct {
		Title   string `json:"title" binding:"required,max=100" desc:"标题"`
		Content string `json:"content" binding:"required,max=5000" desc:"内容"`
		Type    string `json:"type" binding:"required,oneof=SYSTEM REGION" desc:"类型 SYSTEM/REGION"`
	}
}

type PublishApiResponse struct {
	ID int64 `json:"id" desc:"公告ID"`
}

func (a *PublishApi) Run(ctx *gin.Context) kit.Code {
	req := a.Request.Body

	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	publisherID := cast.ToInt64(id)

	urp := repo.NewUserRepo()
	user, err := urp.FindById(ctx, publisherID)
	if err != nil {
		return comm.CodeServerError
	}
	if user == nil || (user.Usertype != enum.UserTypeAdmin && user.Usertype != enum.UserTypeSystemAdmin) {
		return comm.CodeAdminPermissionDenied
	}

	if strings.TrimSpace(req.Title) == "" {
		return comm.CodeParameterInvalid
	}
	if utf8.RuneCountInString(req.Content) > 5000 {
		return comm.CodeParameterInvalid
	}

	announcement := &model.Announcement{
		Title:       strings.TrimSpace(req.Title),
		Content:     req.Content,
		Type:        req.Type,
		Status:      enum.AnnouncementStatusApproved,
		PublisherID: publisherID,
	}

	if user.Usertype == enum.UserTypeAdmin {
		announcement.Status = enum.AnnouncementStatusPending
	}

	arr := repo.NewAnnouncementRepo()
	err = arr.Create(ctx, announcement)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("发布公告失败")
		return comm.CodeServerError
	}

	a.Response = PublishApiResponse{ID: announcement.ID}
	return comm.CodeOK
}

func (a *PublishApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfPublish(ctx *gin.Context) {
	api := &PublishApi{}
	if err := api.Init(ctx); err != nil {
		reply.Fail(ctx, comm.CodeParameterInvalid)
		return
	}
	code := api.Run(ctx)
	if code == comm.CodeOK {
		reply.Success(ctx, api.Response)
	} else {
		reply.Fail(ctx, code)
	}
}

func ReviewListHandler() gin.HandlerFunc {
	api := ReviewListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfReviewList).Pointer()).Name()] = api
	return hfReviewList
}

type ReviewListApi struct {
	Info     struct{}            `name:"待审核公告列表" desc:"系统管理员获取待审核公告列表"`
	Request  ReviewListApiRequest
	Response ReviewListApiResponse
}

type ReviewListApiRequest struct {
	Query struct {
		Page     int `form:"page" binding:"min=1" desc:"页码"`
		PageSize int `form:"page_size" binding:"min=1,max=50" desc:"每页数量"`
	}
}

type ReviewListApiResponse struct {
	Total    int64                   `json:"total" desc:"总数"`
	Page     int                     `json:"page" desc:"页码"`
	PageSize int                     `json:"page_size" desc:"每页数量"`
	List     []ReviewAnnouncementItem `json:"list" desc:"公告列表"`
}

type ReviewAnnouncementItem struct {
	ID          int64     `json:"id" desc:"公告ID"`
	Title       string    `json:"title" desc:"标题"`
	Content     string    `json:"content" desc:"内容"`
	Type        string    `json:"type" desc:"类型"`
	PublisherID int64     `json:"publisher_id" desc:"发布者ID"`
	CreatedAt   time.Time `json:"created_at" desc:"创建时间"`
}

func (a *ReviewListApi) Run(ctx *gin.Context) kit.Code {
	if code := checkSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	req := a.Request.Query
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	arr := repo.NewAnnouncementRepo()
	announcements, total, err := arr.ListByStatus(ctx, enum.AnnouncementStatusPending, offset, pageSize)
	if err != nil {
		return comm.CodeServerError
	}

	items := make([]ReviewAnnouncementItem, 0, len(announcements))
	for _, ann := range announcements {
		items = append(items, ReviewAnnouncementItem{
			ID:          ann.ID,
			Title:       ann.Title,
			Content:     ann.Content,
			Type:        ann.Type,
			PublisherID: ann.PublisherID,
			CreatedAt:   ann.CreatedAt,
		})
	}

	a.Response = ReviewListApiResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}
	return comm.CodeOK
}

func (a *ReviewListApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindQuery(&a.Request.Query)
}

func hfReviewList(ctx *gin.Context) {
	api := &ReviewListApi{}
	if err := api.Init(ctx); err != nil {
		reply.Fail(ctx, comm.CodeParameterInvalid)
		return
	}
	code := api.Run(ctx)
	if code == comm.CodeOK {
		reply.Success(ctx, api.Response)
	} else {
		reply.Fail(ctx, code)
	}
}

func ApproveHandler() gin.HandlerFunc {
	api := ApproveApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfApprove).Pointer()).Name()] = api
	return hfApprove
}

type ApproveApi struct {
	Info     struct{}          `name:"审核公告" desc:"系统管理员审核通过公告"`
	Request  ApproveApiRequest
	Response struct{}
}

type ApproveApiRequest struct {
	Body struct {
		ID int64 `json:"id" binding:"required" desc:"公告ID"`
	}
}

func (a *ApproveApi) Run(ctx *gin.Context) kit.Code {
	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	reviewerID := cast.ToInt64(id)

	if code := checkSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	arr := repo.NewAnnouncementRepo()
	err = arr.Approve(ctx, a.Request.Body.ID, reviewerID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("审核公告失败")
		return comm.CodeServerError
	}

	return comm.CodeOK
}

func (a *ApproveApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfApprove(ctx *gin.Context) {
	api := &ApproveApi{}
	if err := api.Init(ctx); err != nil {
		reply.Fail(ctx, comm.CodeParameterInvalid)
		return
	}
	code := api.Run(ctx)
	if code == comm.CodeOK {
		reply.Success(ctx, struct{}{})
	} else {
		reply.Fail(ctx, code)
	}
}

func checkSysAdmin(ctx *gin.Context) kit.Code {
	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	adminID := cast.ToInt64(id)

	urp := repo.NewUserRepo()
	user, err := urp.FindById(ctx, adminID)
	if err != nil {
		return comm.CodeServerError
	}
	if user == nil || user.Usertype != enum.UserTypeSystemAdmin {
		return comm.CodeAdminPermissionDenied
	}
	return comm.CodeOK
}
