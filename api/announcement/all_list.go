package announcement

import (
	"app/comm"
	"app/dao/repo"
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/swagger"
)

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
