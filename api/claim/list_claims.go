package claim

import (
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/dao/repo"
)

// ListClaimsHandler API router注册点
func ListClaimsHandler() gin.HandlerFunc {
	api := ListApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfList).Pointer()).Name()] = api
	return hfList
}

type ListApi struct {
	Info     struct{}        `name:"认领申请列表" desc:"认领申请列表"`
	Request  ListApiRequest  // API请求参数 (Body/Header/Body/Body)
	Response ListApiResponse // API响应数据 (Body中的Data部分)
}

type ListApiRequest struct {
	Body struct {
		PostID int64 `json:"post_id" binding:"required" desc:"发布记录ID"`
	}
}

type ListApiResponse struct {
	List []ClaimItem `json:"list" desc:"认领申请列表"`
}

type ClaimItem struct {
	ID          int64     `json:"id" desc:"认领申请ID"`
	PostID      int64     `json:"post_id" desc:"发布记录ID"`
	Description string    `json:"description" desc:"补充说明"`
	ProofImages []string  `json:"proof_images" desc:"证明图片"`
	Status      string    `json:"status" desc:"状态"`
	CreatedAt   time.Time `json:"created_at" desc:"创建时间"`
}

// Run Api业务逻辑执行点
func (l *ListApi) Run(ctx *gin.Context) kit.Code {
	request := l.Request.Body

	// 检查发布记录是否存在
	prp := repo.NewPostRepo()
	post, err := prp.FindById(ctx, request.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询发布记录失败")
		return comm.CodeServerError
	}
	if post == nil {
		return comm.CodeDataNotFound
	}

	// 查询认领申请列表
	crp := repo.NewClaimRepo()
	claims, err := crp.ListByPostID(ctx, request.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询认领申请列表失败")
		return comm.CodeServerError
	}

	items := make([]ClaimItem, 0, len(claims))
	for _, claimRecord := range claims {
		items = append(items, ClaimItem{
			ID:          claimRecord.ID,
			PostID:      claimRecord.PostID,
			Description: claimRecord.Description,
			ProofImages: comm.UnmarshalImages(claimRecord.ProofImages),
			Status:      claimRecord.Status,
			CreatedAt:   claimRecord.CreatedAt,
		})
	}

	l.Response = ListApiResponse{List: items}
	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (l *ListApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindUri(&l.Request.Body)
}

// hfList API执行入口
func hfList(ctx *gin.Context) {
	api := &ListApi{}
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
