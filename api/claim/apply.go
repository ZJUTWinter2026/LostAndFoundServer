package claim

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/model"
	"app/dao/repo"
	"reflect"
	"runtime"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"
)

// ApplyHandler API router注册点
func ApplyHandler() gin.HandlerFunc {
	api := ApplyApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfApply).Pointer()).Name()] = api
	return hfApply
}

type ApplyApi struct {
	Info     struct{}         `name:"认领申请" desc:"认领申请"`
	Request  ApplyApiRequest  // API请求参数 (Body/Header/Body/Body)
	Response ApplyApiResponse // API响应数据 (Body中的Data部分)
}

type ApplyApiRequest struct {
	Body struct {
		PostID      int64    `json:"post_id" binding:"required" desc:"发布记录ID"`
		Description string   `json:"description" binding:"required,max=500" desc:"补充说明"`
		ProofImages []string `json:"proof_images" binding:"max=255" desc:"证明图片"`
	}
}

type ApplyApiResponse struct {
	ClaimID int64 `json:"claim_id" desc:"认领申请ID"`
}

// Run Api业务逻辑执行点
func (a *ApplyApi) Run(ctx *gin.Context) kit.Code {
	request := a.Request.Body

	// 获取当前用户ID
	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	claimantID := cast.ToInt64(id)

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

	// 不能认领自己发布的物品
	if post.PublisherID == claimantID {
		return comm.CodeClaimOwnItem
	}

	crp := repo.NewClaimRepo()

	// 检查是否已有待确认或已匹配的申请（防重复）
	hasClaim, err := crp.HasPendingOrMatchedClaim(ctx, request.PostID, claimantID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("检查重复申请失败")
		return comm.CodeServerError
	}
	if hasClaim {
		return comm.CodeClaimDuplicate
	}

	// 序列化图片列表
	var proofImagesJSON string
	if len(request.ProofImages) > 0 {
		b, err := sonic.MarshalString(request.ProofImages)
		if err != nil {
			return comm.CodeParameterInvalid
		}
		proofImagesJSON = b
	}

	claimRecord := &model.Claim{
		PostID:      request.PostID,
		ClaimantID:  claimantID,
		Description: request.Description,
		ProofImages: proofImagesJSON,
		Status:      enum.ClaimStatusPending,
	}

	err = crp.Create(ctx, claimRecord)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("创建认领申请失败")
		return comm.CodeServerError
	}

	a.Response = ApplyApiResponse{ClaimID: claimRecord.ID}
	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (a *ApplyApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

// hfApply API执行入口
func hfApply(ctx *gin.Context) {
	api := &ApplyApi{}
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
