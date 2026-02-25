package post

import (
	"reflect"
	"runtime"
	"time"

	"github.com/spf13/cast"
	"github.com/zjutjh/mygo/jwt"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/comm/enum"
	"app/dao/repo"
)

// DetailHandler API router注册点
func DetailHandler() gin.HandlerFunc {
	api := DetailApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfDetail).Pointer()).Name()] = api
	return hfDetail
}

type DetailApi struct {
	Info     struct{}          `name:"发布详情" desc:"发布详情"`
	Request  DetailApiRequest  // API请求参数 (Body/Header/Body/Body)
	Response DetailApiResponse // API响应数据 (Body中的Data部分)
}

type DetailApiRequest struct {
	Body struct {
		ID int64 `json:"id" binding:"required" desc:"发布ID"`
	}
}

type DetailApiResponse struct {
	ID              int64     `json:"id" desc:"发布ID"`
	PublishType     string    `json:"publish_type" desc:"发布类型 LOST/FOUND"`
	ItemName        string    `json:"item_name" desc:"物品名称"`
	ItemType        string    `json:"item_type" desc:"物品类型"`
	ItemTypeOther   string    `json:"item_type_other" desc:"其它类型说明"`
	Campus          string    `json:"campus" desc:"校区"`
	Location        string    `json:"location" desc:"地点"`
	StorageLocation string    `json:"storage_location" desc:"存放地点"`
	EventTime       time.Time `json:"event_time" desc:"事件时间"`
	Features        string    `json:"features" desc:"物品特征"`
	ContactName     string    `json:"contact_name" desc:"联系人"`
	ContactPhone    string    `json:"contact_phone" desc:"联系电话"`
	HasReward       bool      `json:"has_reward" desc:"是否有悬赏"`
	Images          []string  `json:"images" desc:"图片"`
	Status          string    `json:"status" desc:"状态"`
	CancelReason    string    `json:"cancel_reason" desc:"取消原因"`
	RejectReason    string    `json:"reject_reason" desc:"驳回原因"`
	ClaimCount      int32     `json:"claim_count" desc:"认领人数"`
	ArchiveMethod   string    `json:"archive_method" desc:"物品处理方式"`
	ProcessedAt     time.Time `json:"processed_at" desc:"处理时间"`
	CreatedAt       time.Time `json:"created_at" desc:"创建时间"`
}

// Run Api业务逻辑执行点
func (d *DetailApi) Run(ctx *gin.Context) kit.Code {
	request := d.Request.Body

	prp := repo.NewPostRepo()
	record, err := prp.FindById(ctx, request.ID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询发布详情失败")
		return comm.CodeServerError
	}
	if record == nil {
		return comm.CodeDataNotFound
	}

	resp := DetailApiResponse{
		ID:              record.ID,
		PublishType:     record.PublishType,
		ItemName:        record.ItemName,
		ItemType:        record.ItemType,
		ItemTypeOther:   record.ItemTypeOther,
		Campus:          record.Campus,
		Location:        record.Location,
		StorageLocation: record.StorageLocation,
		EventTime:       record.EventTime,
		Features:        record.Features,
		ContactName:     record.ContactName,
		ContactPhone:    record.ContactPhone,
		HasReward:       record.HasReward,
		Images:          comm.UnmarshalImages(record.Images),
		Status:          record.Status,
		CancelReason:    record.CancelReason,
		RejectReason:    record.RejectReason,
		ClaimCount:      record.ClaimCount,
		ArchiveMethod:   record.ArchiveMethod,
		ProcessedAt:     record.ProcessedAt,
		CreatedAt:       record.CreatedAt,
	}

	// 权限判断：发布者本人或管理员可查看联系方式
	id, err := jwt.GetIdentity[string](ctx)
	if err == nil {
		userID := cast.ToInt64(id)
		if userID == record.PublisherID {
			resp.ContactName = record.ContactName
			resp.ContactPhone = record.ContactPhone
		} else {
			// 判断是否为管理员
			urp := repo.NewUserRepo()
			user, err := urp.FindById(ctx, userID)
			if err == nil && user != nil && (user.Usertype == enum.UserTypeAdmin || user.Usertype == enum.UserTypeSystemAdmin) {
				resp.ContactName = record.ContactName
				resp.ContactPhone = record.ContactPhone
			}
		}
	}

	d.Response = resp
	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (d *DetailApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindUri(&d.Request.Body)
}

// hfDetail API执行入口
func hfDetail(ctx *gin.Context) {
	api := &DetailApi{}
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
