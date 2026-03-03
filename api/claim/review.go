package claim

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/model"
	"app/dao/repo"
	"errors"
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/ndb"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
	"gorm.io/gorm"
)

// errClaimAlreadyMatchedTx 是事务内检测到竞态时返回的哨兵错误
var errClaimAlreadyMatchedTx = errors.New("claim already matched")

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
		if err == nil && user != nil && (user.Usertype == enum.UserTypeAdmin || user.Usertype == enum.UserTypeSystemAdmin) {
			isAdmin = true
		}
	}
	if !isPublisher && !isAdmin {
		return comm.CodePermissionDenied
	}

	// 根据操作类型执行对应逻辑
	if request.Approve {
		// 同意认领：在事务中完成竞态检查+状态更新，防止并发竞争
		txErr := ndb.Pick().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// 事务内再次检查是否已有已匹配认领（防止并发竞态）
			var matchedCount int64
			if e := tx.Model(&model.Claim{}).
				Where("post_id = ? AND status = ?", claimRecord.PostID, enum.ClaimStatusMatched).
				Count(&matchedCount).Error; e != nil {
				return e
			}
			if matchedCount > 0 {
				return errClaimAlreadyMatchedTx
			}
			now := time.Now()
			// 更新认领状态为 MATCHED
			if e := tx.Model(&model.Claim{}).
				Where("id = ?", request.ClaimID).
				Updates(map[string]interface{}{
					"status":      enum.ClaimStatusMatched,
					"reviewed_by": reviewerID,
					"reviewed_at": now,
				}).Error; e != nil {
				return e
			}
			// 更新帖子状态为 SOLVED
			if e := tx.Model(&model.Post{}).
				Where("id = ?", claimRecord.PostID).
				Update("status", enum.PostStatusSolved).Error; e != nil {
				return e
			}
			return nil
		})
		if errors.Is(txErr, errClaimAlreadyMatchedTx) {
			return comm.CodeClaimAlreadyMatched
		}
		if txErr != nil {
			nlog.Pick().WithContext(ctx).WithError(txErr).Warn("审核认领事务失败")
			return comm.CodeServerError
		}
	} else {
		// 拒绝认领：直接更新状态
		if err = crp.UpdateStatus(ctx, request.ClaimID, enum.ClaimStatusRejected, reviewerID); err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("更新认领状态失败")
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
