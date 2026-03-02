package admin

import (
	"app/comm"
	"app/dao/repo"
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"
)

func ExpiredListHandler() gin.HandlerFunc {
	api := ExpiredListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfExpiredList).Pointer()).Name()] = api
	return hfExpiredList
}

type ExpiredListApi struct {
	Info     struct{} `name:"查看过期无效数据" desc:"查看已归档、已删除、已取消的发布信息"`
	Request  ExpiredListApiRequest
	Response ExpiredListApiResponse
}

type ExpiredListApiRequest struct {
	Query struct {
		Page     int `form:"page" binding:"required,min=1" desc:"页码"`
		PageSize int `form:"page_size" binding:"required,min=1,max=50" desc:"每页数量"`
	}
}

type ExpiredListApiResponse struct {
	Total    int64             `json:"total" desc:"总数"`
	Page     int               `json:"page" desc:"页码"`
	PageSize int               `json:"page_size" desc:"每页数量"`
	List     []ExpiredPostItem `json:"list" desc:"过期无效数据列表"`
}

type ExpiredPostItem struct {
	ID            int64     `json:"id" desc:"发布ID"`
	PublishType   string    `json:"publish_type" desc:"发布类型 LOST/FOUND"`
	ItemName      string    `json:"item_name" desc:"物品名称"`
	ItemType      string    `json:"item_type" desc:"物品类型"`
	Campus        string    `json:"campus" desc:"校区"`
	Location      string    `json:"location" desc:"地点"`
	Status        string    `json:"status" desc:"状态"`
	CancelReason  string    `json:"cancel_reason,omitempty" desc:"取消原因"`
	RejectReason  string    `json:"reject_reason,omitempty" desc:"驳回原因"`
	ArchiveMethod string    `json:"archive_method,omitempty" desc:"归档处理方式"`
	PublisherID   int64     `json:"publisher_id" desc:"发布者ID"`
	CreatedAt     time.Time `json:"created_at" desc:"创建时间"`
	UpdatedAt     time.Time `json:"updated_at" desc:"更新时间"`
}

func (e *ExpiredListApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	req := e.Request.Query
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

	prp := repo.NewPostRepo()
	posts, err := prp.ListExpired(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询过期无效数据失败")
		return comm.CodeServerError
	}

	total := int64(len(posts))
	offset := (page - 1) * pageSize
	end := offset + pageSize
	if end > int(total) {
		end = int(total)
	}
	if offset > int(total) {
		offset = int(total)
	}

	items := make([]ExpiredPostItem, 0, pageSize)
	for i := offset; i < end; i++ {
		post := posts[i]
		items = append(items, ExpiredPostItem{
			ID:            post.ID,
			PublishType:   post.PublishType,
			ItemName:      post.ItemName,
			ItemType:      post.ItemType,
			Campus:        post.Campus,
			Location:      post.Location,
			Status:        post.Status,
			CancelReason:  post.CancelReason,
			RejectReason:  post.RejectReason,
			ArchiveMethod: post.ArchiveMethod,
			PublisherID:   post.PublisherID,
			CreatedAt:     post.CreatedAt,
			UpdatedAt:     post.UpdatedAt,
		})
	}

	e.Response = ExpiredListApiResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}
	return comm.CodeOK
}

func (e *ExpiredListApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindQuery(&e.Request.Query)
}

func hfExpiredList(ctx *gin.Context) {
	api := &ExpiredListApi{}
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
