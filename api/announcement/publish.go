package announcement

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/model"
	"app/dao/repo"
	"reflect"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func PublishHandler() gin.HandlerFunc {
	api := PublishApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfPublish).Pointer()).Name()] = api
	return hfPublish
}

type PublishApi struct {
	Info     struct{} `name:"发布公告" desc:"系统管理员发布全局公告"`
	Request  PublishApiRequest
	Response PublishApiResponse
}

type PublishApiRequest struct {
	Body struct {
		Title        string `json:"title" binding:"required,max=100" desc:"标题"`
		Content      string `json:"content" binding:"required,max=5000" desc:"内容"`
		Type         string `json:"type" binding:"required,oneof=SYSTEM REGION" desc:"类型 SYSTEM系统公告/REGION区域公告"`
		Campus       string `json:"campus" binding:"omitempty,oneof=ZHAO_HUI PING_FENG MO_GAN_SHAN" desc:"所属校区: ZHAO_HUI, PING_FENG, MO_GAN_SHAN, 仅REGION类型有效"`
		TargetUserID int64  `json:"target_user_id" desc:"目标用户ID, 0表示全局公告/系统公告, 非0表示针对特定用户, 仅超管可用"`
	}
}

type PublishApiResponse struct {
	ID int64 `json:"id" desc:"公告ID"`
}

func (a *PublishApi) Run(ctx *gin.Context) kit.Code {
	req := a.Request.Body

	publisherID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

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

	isSysAdmin := user.Usertype == enum.UserTypeSystemAdmin

	if !isSysAdmin {
		if req.Type != enum.AnnouncementTypeRegion {
			return comm.CodeParameterInvalid
		}
		if user.Campus == "" {
			return comm.CodeParameterInvalid
		}
	}

	announcement := &model.Announcement{
		Title:        strings.TrimSpace(req.Title),
		Content:      req.Content,
		Type:         req.Type,
		PublisherID:  publisherID,
		TargetUserID: req.TargetUserID,
	}

	if req.Type == enum.AnnouncementTypeRegion {
		if isSysAdmin {
			announcement.Campus = req.Campus
		} else {
			announcement.Campus = user.Campus
		}
	}

	if isSysAdmin {
		announcement.Status = enum.AnnouncementStatusApproved
	} else {
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
	Info     struct{} `name:"待审核公告列表" desc:"系统管理员获取待审核公告列表"`
	Request  ReviewListApiRequest
	Response ReviewListApiResponse
}

type ReviewListApiRequest struct {
	Query struct {
		Page     int    `form:"page" binding:"required,min=1" desc:"页码"`
		PageSize int    `form:"page_size" binding:"required,min=1,max=50" desc:"每页数量"`
		Campus   string `form:"campus" binding:"omitempty,oneof=ZHAO_HUI PING_FENG MO_GAN_SHAN" desc:"校区筛选，可选"`
	}
}

type ReviewListApiResponse struct {
	Total    int64                    `json:"total" desc:"总数"`
	Page     int                      `json:"page" desc:"页码"`
	PageSize int                      `json:"page_size" desc:"每页数量"`
	List     []ReviewAnnouncementItem `json:"list" desc:"公告列表"`
}

type ReviewAnnouncementItem struct {
	ID          int64     `json:"id" desc:"公告ID"`
	Title       string    `json:"title" desc:"标题"`
	Content     string    `json:"content" desc:"内容"`
	Type        string    `json:"type" desc:"类型"`
	Campus      string    `json:"campus" desc:"校区"`
	PublisherID int64     `json:"publisher_id" desc:"发布者ID"`
	CreatedAt   time.Time `json:"created_at" desc:"创建时间"`
}

func (a *ReviewListApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
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

	announcements, total, err := arr.ListPending(ctx, req.Campus, offset, pageSize)
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
			Campus:      ann.Campus,
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
	Info     struct{} `name:"审核公告" desc:"系统管理员审核公告"`
	Request  ApproveApiRequest
	Response struct{}
}

type ApproveApiRequest struct {
	Body struct {
		ID      int64 `json:"id" binding:"required" desc:"公告ID"`
		Approve bool  `json:"approve" binding:"required" desc:"true批准 false驳回"`
	}
}

func (a *ApproveApi) Run(ctx *gin.Context) kit.Code {
	reviewerID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	arr := repo.NewAnnouncementRepo()
	announcement, err := arr.FindById(ctx, a.Request.Body.ID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询公告失败")
		return comm.CodeServerError
	}
	if announcement == nil {
		return comm.CodeDataNotFound
	}

	if announcement.Status != enum.AnnouncementStatusPending {
		return comm.CodePostStatusInvalid
	}

	if a.Request.Body.Approve {
		err = arr.Approve(ctx, a.Request.Body.ID, reviewerID)
	} else {
		err = arr.Reject(ctx, a.Request.Body.ID, reviewerID)
	}
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

func DeleteHandler() gin.HandlerFunc {
	api := DeleteApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfDelete).Pointer()).Name()] = api
	return hfDelete
}

type DeleteApi struct {
	Info     struct{} `name:"删除公告" desc:"系统管理员删除公告"`
	Request  DeleteApiRequest
	Response struct{}
}

type DeleteApiRequest struct {
	Body struct {
		ID int64 `json:"id" binding:"required" desc:"公告ID"`
	}
}

func (a *DeleteApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	arr := repo.NewAnnouncementRepo()
	announcement, err := arr.FindById(ctx, a.Request.Body.ID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询公告失败")
		return comm.CodeServerError
	}
	if announcement == nil {
		return comm.CodeDataNotFound
	}

	err = arr.Delete(ctx, a.Request.Body.ID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("删除公告失败")
		return comm.CodeServerError
	}

	return comm.CodeOK
}

func (a *DeleteApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfDelete(ctx *gin.Context) {
	api := &DeleteApi{}
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

func AllListHandler() gin.HandlerFunc {
	api := AllListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfAllList).Pointer()).Name()] = api
	return hfAllList
}

type AllListApi struct {
	Info     struct{} `name:"全部公告列表" desc:"系统管理员获取全部公告列表"`
	Request  AllListApiRequest
	Response AllListApiResponse
}

type AllListApiRequest struct {
	Query struct {
		Page     int `form:"page" binding:"required,min=1" desc:"页码"`
		PageSize int `form:"page_size" binding:"required,min=1,max=50" desc:"每页数量"`
	}
}

type AllListApiResponse struct {
	Total    int64                 `json:"total" desc:"总数"`
	Page     int                   `json:"page" desc:"页码"`
	PageSize int                   `json:"page_size" desc:"每页数量"`
	List     []AllAnnouncementItem `json:"list" desc:"公告列表"`
}

type AllAnnouncementItem struct {
	ID          int64     `json:"id" desc:"公告ID"`
	Title       string    `json:"title" desc:"标题"`
	Content     string    `json:"content" desc:"内容"`
	Type        string    `json:"type" desc:"类型"`
	Campus      string    `json:"campus" desc:"校区"`
	Status      string    `json:"status" desc:"状态"`
	PublisherID int64     `json:"publisher_id" desc:"发布者ID"`
	CreatedAt   time.Time `json:"created_at" desc:"创建时间"`
}

func (a *AllListApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
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

	announcements, total, err := arr.ListAll(ctx, offset, pageSize)
	if err != nil {
		return comm.CodeServerError
	}

	items := make([]AllAnnouncementItem, 0, len(announcements))
	for _, ann := range announcements {
		items = append(items, AllAnnouncementItem{
			ID:          ann.ID,
			Title:       ann.Title,
			Content:     ann.Content,
			Type:        ann.Type,
			Campus:      ann.Campus,
			Status:      ann.Status,
			PublisherID: ann.PublisherID,
			CreatedAt:   ann.CreatedAt,
		})
	}

	a.Response = AllListApiResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}
	return comm.CodeOK
}

func (a *AllListApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindQuery(&a.Request.Query)
}

func hfAllList(ctx *gin.Context) {
	api := &AllListApi{}
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
