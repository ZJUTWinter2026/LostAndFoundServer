package announcement

import (
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/dao/repo"
)

func ListHandler() gin.HandlerFunc {
	api := ListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfList).Pointer()).Name()] = api
	return hfList
}

type ListApi struct {
	Info     struct{}       `name:"获取公告列表" desc:"获取已审核通过的公告列表"`
	Request  ListApiRequest `name:"获取公告列表" desc:"获取已审核通过的公告列表"`
	Response ListApiResponse
}

type ListApiRequest struct {
	Query struct {
		Page     int `form:"page" binding:"required,min=1" desc:"页码"`
		PageSize int `form:"page_size" binding:"required,min=1,max=50" desc:"每页数量"`
	}
}

type ListApiResponse struct {
	Total    int64            `json:"total" desc:"总数"`
	Page     int              `json:"page" desc:"页码"`
	PageSize int              `json:"page_size" desc:"每页数量"`
	List     []AnnouncementItem `json:"list" desc:"公告列表"`
}

type AnnouncementItem struct {
	ID        int64     `json:"id" desc:"公告ID"`
	Title     string    `json:"title" desc:"标题"`
	Content   string    `json:"content" desc:"内容"`
	Type      string    `json:"type" desc:"类型"`
	CreatedAt time.Time `json:"created_at" desc:"创建时间"`
}

func (a *ListApi) Run(ctx *gin.Context) kit.Code {
	req := a.Request.Query
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	userID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	offset := (page - 1) * pageSize
	arr := repo.NewAnnouncementRepo()
	announcements, total, err := arr.ListApprovedForUser(ctx, userID, offset, pageSize)
	if err != nil {
		return comm.CodeServerError
	}

	items := make([]AnnouncementItem, 0, len(announcements))
	for _, ann := range announcements {
		items = append(items, AnnouncementItem{
			ID:        ann.ID,
			Title:     ann.Title,
			Content:   ann.Content,
			Type:      ann.Type,
			CreatedAt: ann.CreatedAt,
		})
	}

	a.Response = ListApiResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}
	return comm.CodeOK
}

func (a *ListApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindQuery(&a.Request.Query)
}

func hfList(ctx *gin.Context) {
	api := &ListApi{}
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
