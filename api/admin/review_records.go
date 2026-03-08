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
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func ReviewRecordsHandler() gin.HandlerFunc {
	api := ReviewRecordsApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfReviewRecords).Pointer()).Name()] = api
	return hfReviewRecords
}

type ReviewRecordsApi struct {
	Info     struct{} `name:"管理员审核记录列表" desc:"返回当前管理员审核过的发布记录列表"`
	Request  ReviewRecordsApiRequest
	Response ReviewRecordsApiResponse
}

type ReviewRecordsApiRequest struct {
	Query struct {
		Page     int `form:"page" binding:"required,min=1" desc:"页码"`
		PageSize int `form:"page_size" binding:"required,min=1,max=50" desc:"每页数量"`
	}
}

type ReviewRecordsApiResponse struct {
	Total    int64              `json:"total" desc:"总数"`
	Page     int                `json:"page" desc:"页码"`
	PageSize int                `json:"page_size" desc:"每页数量"`
	List     []ReviewRecordItem `json:"list" desc:"审核记录列表"`
}

type ReviewRecordItem struct {
	ID           int64      `json:"id" desc:"发布ID"`
	PublishType  string     `json:"publish_type" desc:"发布类型 LOST/FOUND"`
	ItemName     string     `json:"item_name" desc:"物品名称"`
	ItemType     string     `json:"item_type" desc:"物品类型"`
	Campus       string     `json:"campus" desc:"校区"`
	Location     string     `json:"location" desc:"地点"`
	Status       string     `json:"status" desc:"审核结果状态"`
	RejectReason string     `json:"reject_reason,omitempty" desc:"驳回原因"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty" desc:"审核时间"`
}

func (r *ReviewRecordsApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckAdminPermission(ctx); code != comm.CodeOK {
		return code
	}

	adminID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	request := r.Request.Query
	offset := (request.Page - 1) * request.PageSize

	prp := repo.NewPostRepo()
	records, total, err := prp.ListReviewedByAdmin(ctx, repo.AdminReviewRecordFilter{ReviewerAdminID: adminID}, offset, request.PageSize)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询管理员审核记录失败")
		return comm.CodeServerError
	}

	items := make([]ReviewRecordItem, 0, len(records))
	for _, record := range records {
		items = append(items, ReviewRecordItem{
			ID:           record.ID,
			PublishType:  record.PublishType,
			ItemName:     record.ItemName,
			ItemType:     record.ItemType,
			Campus:       record.Campus,
			Location:     record.Location,
			Status:       record.Status,
			RejectReason: record.RejectReason,
			ProcessedAt:  record.ProcessedAt,
		})
	}

	r.Response = ReviewRecordsApiResponse{
		Total:    total,
		Page:     request.Page,
		PageSize: request.PageSize,
		List:     items,
	}
	return comm.CodeOK
}

func (r *ReviewRecordsApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindQuery(&r.Request.Query)
}

func hfReviewRecords(ctx *gin.Context) {
	api := &ReviewRecordsApi{}
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
