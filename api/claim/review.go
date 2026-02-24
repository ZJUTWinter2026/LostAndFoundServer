package claim

import (
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/dao/repo"
)

// ReviewHandler API router注册点
func ReviewHandler() gin.HandlerFunc {
	api := ReviewApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfReview).Pointer()).Name()] = api
	return hfReview
}

type ReviewApi struct {
	Info     struct{}          `name:"审核认领申请" desc:"审核认领申请"`
	Request  ReviewApiRequest  // API请求参数 (Body/Header/Body/Body)
	Response ReviewApiResponse // API响应数据 (Body中的Data部分)
}

type ReviewApiRequest struct {
	Body struct {
		ClaimID int64 `json:"claim_id" binding:"required" desc:"认领申请ID"`
		Action  int8  `json:"action" binding:"required,oneof=1 2" desc:"操作 1同意 2拒绝"`
	}
}

type ReviewApiResponse struct {
	Success bool `json:"success" desc:"是否成功"`
}

// Run Api业务逻辑执行点
func (r *ReviewApi) Run(ctx *gin.Context) kit.Code {
	request := r.Request.Body

	// 获取当前用户ID
	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	reviewerID := cast.ToInt64(id)

	// 查询认领申请
	crp := repo.NewClaimRepo()
	claimRecord, err := crp.FindById(ctx, request.ClaimID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询认领申请失败")
		return comm.CodeDatabaseError
	}
	if claimRecord == nil {
		return comm.CodeClaimNotFound
	}

	// 只能审核待确认状态的申请
	if claimRecord.Status != statusPending {
		return comm.CodeClaimStatusInvalid
	}

	// 查询发布记录，验证权限
	prp := repo.NewPostRepo()
	post, err := prp.FindById(ctx, claimRecord.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询发布记录失败")
		return comm.CodeDatabaseError
	}
	if post == nil {
		return comm.CodeDataNotFound
	}

	// 权限判断：发布者本人或管理员
	isPublisher := post.PublisherID == reviewerID
	isAdmin := false
	if !isPublisher {
		urp := repo.NewUserRepo()
		user, err := urp.FindById(ctx, reviewerID)
		if err == nil && user != nil && user.Usertype == 1 {
			isAdmin = true
		}
	}
	if !isPublisher && !isAdmin {
		return comm.CodePermissionDenied
	}

	// 如果是同意操作，检查是否已有已匹配的认领
	if request.Action == 1 {
		hasMatched, err := crp.HasMatchedClaim(ctx, claimRecord.PostID)
		if err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("检查已匹配认领失败")
			return comm.CodeDatabaseError
		}
		if hasMatched {
			return comm.CodeClaimAlreadyMatched
		}
	}

	// 更新状态
	newStatus := statusMatched
	if request.Action == 2 {
		newStatus = statusRejected
	}

	err = crp.UpdateStatus(ctx, request.ClaimID, newStatus, reviewerID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("更新认领申请状态失败")
		return comm.CodeDatabaseError
	}

	r.Response = ReviewApiResponse{Success: true}
	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (r *ReviewApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindJSON(&r.Request.Body)
}

// hfReview API执行入口
func hfReview(ctx *gin.Context) {
	api := &ReviewApi{}
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
