package claim

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/repo"
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
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
		Approve bool  `json:"approve" desc:"操作 true同意 false拒绝"`
	}
}

type ReviewApiResponse struct {
	Success bool `json:"success" desc:"是否成功"`
}

// Run Api业务逻辑执行点
func (r *ReviewApi) Run(ctx *gin.Context) kit.Code {
	request := r.Request.Body

	// 获取当前用户ID
	reviewerID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	// 查询认领申请
	crp := repo.NewClaimRepo()
	claimRecord, err := crp.FindById(ctx, request.ClaimID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询认领申请失败")
		return comm.CodeServerError
	}
	if claimRecord == nil {
		return comm.CodeClaimNotFound
	}

	// 只能审核待确认状态的申请
	if claimRecord.Status != enum.ClaimStatusPending {
		return comm.CodeClaimStatusInvalid
	}

	// 查询发布记录，验证权限
	prp := repo.NewPostRepo()
	post, err := prp.FindById(ctx, claimRecord.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询发布记录失败")
		return comm.CodeServerError
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
		if err == nil && user != nil && user.Usertype == enum.UserTypeAdmin {
			isAdmin = true
		}
	}
	if !isPublisher && !isAdmin {
		return comm.CodePermissionDenied
	}

	// 如果是同意操作，检查是否已有已匹配的认领
	if request.Approve {
		hasMatched, err := crp.HasMatchedClaim(ctx, claimRecord.PostID)
		if err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("检查已匹配认领失败")
			return comm.CodeServerError
		}
		if hasMatched {
			return comm.CodeClaimAlreadyMatched
		}
	}

	// 更新状态
	var targetStatus string
	if request.Approve {
		targetStatus = enum.ClaimStatusMatched
	} else {
		targetStatus = enum.ClaimStatusRejected
	}

	err = crp.UpdateStatus(ctx, request.ClaimID, targetStatus, reviewerID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("更新认领状态失败")
		return comm.CodeServerError
	}

	// 如果同意认领，更新发布记录状态为已解决
	if targetStatus == enum.ClaimStatusMatched {
		err = prp.UpdateStatus(ctx, claimRecord.PostID, enum.PostStatusSolved)
		if err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("更新发布状态失败")
			return comm.CodeServerError
		}

		err = prp.IncrementClaimCount(ctx, claimRecord.PostID)
		if err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("增加认领人数失败")
			return comm.CodeServerError
		}
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
