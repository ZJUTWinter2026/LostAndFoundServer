package claim

import (
	"app/comm"
	"app/dao/repo"
	"reflect"
	"runtime"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func MyListHandler() gin.HandlerFunc {
	api := MyListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfMyList).Pointer()).Name()] = api
	return hfMyList
}

type MyListApi struct {
	Info     struct{} `name:"我的认领申请列表" desc:"我的认领申请列表"`
	Request  MyListApiRequest
	Response MyListApiResponse
}

type MyListApiRequest struct {
	Query struct {
		Page     int `form:"page" binding:"required,min=1" desc:"页码"`
		PageSize int `form:"page_size" binding:"required,min=1,max=50" desc:"每页数量"`
	}
}

type MyListApiResponse struct {
	Total    int64         `json:"total" desc:"总数"`
	Page     int           `json:"page" desc:"页码"`
	PageSize int           `json:"page_size" desc:"每页数量"`
	List     []MyClaimItem `json:"list" desc:"认领申请列表"`
}

type MyClaimItem struct {
	ID          int64     `json:"id" desc:"认领申请ID"`
	PostID      int64     `json:"post_id" desc:"发布记录ID"`
	ItemName    string    `json:"item_name" desc:"物品名称"`
	PublishType string    `json:"publish_type" desc:"发布类型"`
	Description string    `json:"description" desc:"补充说明"`
	ProofImages []string  `json:"proof_images" desc:"证明图片"`
	Status      string    `json:"status" desc:"状态"`
	CreatedAt   time.Time `json:"created_at" desc:"创建时间"`
}

func (m *MyListApi) Run(ctx *gin.Context) kit.Code {
	req := m.Request.Query

	claimantID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

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

	offset := (page - 1) * pageSize
	crp := repo.NewClaimRepo()

	claims, total, err := crp.ListByClaimant(ctx, claimantID, offset, pageSize)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询我的认领申请列表失败")
		return comm.CodeServerError
	}

	prp := repo.NewPostRepo()
	items := make([]MyClaimItem, 0, len(claims))
	for _, claimRecord := range claims {
		var proofImages []string
		if claimRecord.ProofImages != "" {
			err = sonic.UnmarshalString(claimRecord.ProofImages, &proofImages)
			if err != nil {
				nlog.Pick().WithContext(ctx).WithError(err).Warn("解析证明图片列表失败")
				return comm.CodeServerError
			}
		}

		post, err := prp.FindById(ctx, claimRecord.PostID)
		if err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("查询认领关联发布失败")
			continue
		}

		items = append(items, MyClaimItem{
			ID:          claimRecord.ID,
			PostID:      claimRecord.PostID,
			ItemName:    post.ItemName,
			PublishType: post.PublishType,
			Description: claimRecord.Description,
			ProofImages: proofImages,
			Status:      claimRecord.Status,
			CreatedAt:   claimRecord.CreatedAt,
		})
	}

	m.Response = MyListApiResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}
	return comm.CodeOK
}

func (m *MyListApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindQuery(&m.Request.Query)
}

func hfMyList(ctx *gin.Context) {
	api := &MyListApi{}
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
