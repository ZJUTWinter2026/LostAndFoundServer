package admin

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/repo"
	"reflect"
	"runtime"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func PublishedListHandler() gin.HandlerFunc {
	api := PublishedListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfPublishedList).Pointer()).Name()] = api
	return hfPublishedList
}

type PublishedListApi struct {
	Info     struct{} `name:"普通管理员获取已发布列表" desc:"普通管理员按所属校区获取已发布列表"`
	Request  PublishedListApiRequest
	Response PublishedListApiResponse
}

type PublishedListApiRequest struct {
	Query struct {
		Type     string `form:"type" binding:"required,oneof=LOST FOUND" desc:"发布类型"`
		Page     int    `form:"page" binding:"required,min=1" desc:"页码"`
		PageSize int    `form:"page_size" binding:"required,min=1,max=50" desc:"每页数量"`
	}
}

type PublishedListApiResponse struct {
	Total    int64               `json:"total" desc:"总数"`
	Page     int                 `json:"page" desc:"页码"`
	PageSize int                 `json:"page_size" desc:"每页数量"`
	List     []AdminPostListItem `json:"list" desc:"已发布列表"`
}

func (p *PublishedListApi) Run(ctx *gin.Context) kit.Code {
	adminID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	userRepo := repo.NewUserRepo()
	user, err := userRepo.FindById(ctx, adminID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询用户失败")
		return comm.CodeServerError
	}
	if user == nil || user.Usertype != enum.UserTypeAdmin {
		return comm.CodeAdminPermissionDenied
	}

	request := p.Request.Query
	filter := repo.PostListFilter{
		PublishType: strings.TrimSpace(request.Type),
		Campus:      user.Campus,
		Status:      enum.PostStatusApproved,
	}
	offset := (request.Page - 1) * request.PageSize

	postRepo := repo.NewPostRepo()
	records, total, err := postRepo.ListByFilter(ctx, filter, offset, request.PageSize)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询普通管理员已发布列表失败")
		return comm.CodeServerError
	}

	items := make([]AdminPostListItem, 0, len(records))
	for _, record := range records {
		var images []string
		if record.Images != "" {
			err = sonic.UnmarshalString(record.Images, &images)
			if err != nil {
				nlog.Pick().WithContext(ctx).WithError(err).Warn("解析图片列表失败")
				return comm.CodeServerError
			}
		}

		items = append(items, AdminPostListItem{
			ID:                record.ID,
			PublishType:       record.PublishType,
			ItemName:          record.ItemName,
			ItemType:          record.ItemType,
			Campus:            record.Campus,
			Location:          record.Location,
			EventTime:         record.EventTime,
			Features:          record.Features,
			HasReward:         record.HasReward,
			RewardDescription: record.RewardDescription,
			Status:            record.Status,
			Images:            images,
			PublisherID:       record.PublisherID,
			CreatedAt:         record.CreatedAt,
		})
	}

	p.Response = PublishedListApiResponse{
		Total:    total,
		Page:     request.Page,
		PageSize: request.PageSize,
		List:     items,
	}
	return comm.CodeOK
}

func (p *PublishedListApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindQuery(&p.Request.Query)
}

func hfPublishedList(ctx *gin.Context) {
	api := &PublishedListApi{}
	if err := api.Init(ctx); err != nil {
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
