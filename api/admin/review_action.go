package admin

import (
	"reflect"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/comm/enum"
	"app/dao/repo"
)

// ApproveHandler API router注册点
func ApproveHandler() gin.HandlerFunc {
	api := ApproveApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfApprove).Pointer()).Name()] = api
	return hfApprove
}

type ApproveApi struct {
	Info     struct{}           `name:"审核通过发布" desc:"审核通过发布"`
	Request  ApproveApiRequest  // API请求参数
	Response ApproveApiResponse // API响应数据
}

type ApproveApiRequest struct {
	Body struct {
		PostID int64 `json:"post_id" binding:"required" desc:"发布ID"`
	}
}

type ApproveApiResponse struct {
	Success bool `json:"success" desc:"是否成功"`
}

// Run Api业务逻辑执行点
func (a *ApproveApi) Run(ctx *gin.Context) kit.Code {
	request := a.Request.Body

	// 获取当前用户并验证是管理员
	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	adminID := cast.ToInt64(id)

	// 验证管理员权限
	urp := repo.NewUserRepo()
	user, err := urp.FindById(ctx, adminID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询用户失败")
		return comm.CodeServerError
	}
	if user == nil || (user.Usertype != enum.UserTypeAdmin && user.Usertype != enum.UserTypeSystemAdmin) {
		return comm.CodeAdminPermissionDenied
	}

	// 查询发布记录
	prp := repo.NewPostRepo()
	post, err := prp.FindById(ctx, request.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询发布记录失败")
		return comm.CodeServerError
	}
	if post == nil {
		return comm.CodeDataNotFound
	}

	// 必须是待审核状态
	if post.Status != enum.PostStatusPending {
		return comm.CodePostStatusInvalid
	}

	// 审核通过
	err = prp.ApprovePost(ctx, request.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("审核通过失败")
		return comm.CodeServerError
	}

	// 记录审计日志
	alr := repo.NewAuditLogRepo()
	err = alr.CreateAuditLog(ctx, adminID, enum.AuditLogTypeUpdate, "", request.PostID, post.Status, enum.PostStatusApproved)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("记录审计日志失败")
		return comm.CodeServerError
	}

	a.Response = ApproveApiResponse{Success: true}
	return comm.CodeOK
}

// Init Api初始化
func (a *ApproveApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

// hfApprove API执行入口
func hfApprove(ctx *gin.Context) {
	api := &ApproveApi{}
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

// RejectHandler API router注册点
func RejectHandler() gin.HandlerFunc {
	api := RejectApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfReject).Pointer()).Name()] = api
	return hfReject
}

type RejectApi struct {
	Info     struct{}          `name:"审核驳回发布" desc:"审核驳回发布"`
	Request  RejectApiRequest  // API请求参数
	Response RejectApiResponse // API响应数据
}

type RejectApiRequest struct {
	Body struct {
		PostID int64  `json:"post_id" binding:"required" desc:"发布ID"`
		Reason string `json:"reason" binding:"required,max=500" desc:"驳回理由"`
	}
}

type RejectApiResponse struct {
	Success bool `json:"success" desc:"是否成功"`
}

// Run Api业务逻辑执行点
func (r *RejectApi) Run(ctx *gin.Context) kit.Code {
	request := r.Request.Body

	// 验证驳回理由
	if strings.TrimSpace(request.Reason) == "" {
		return comm.CodeReviewReasonRequired
	}
	if utf8.RuneCountInString(request.Reason) > 500 {
		return comm.CodeReviewReasonTooLong
	}

	// 获取当前用户并验证是管理员
	id, err := jwt.GetIdentity[string](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}
	adminID := cast.ToInt64(id)

	// 验证管理员权限
	urp := repo.NewUserRepo()
	user, err := urp.FindById(ctx, adminID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询用户失败")
		return comm.CodeServerError
	}
	if user == nil || (user.Usertype != enum.UserTypeAdmin && user.Usertype != enum.UserTypeSystemAdmin) {
		return comm.CodeAdminPermissionDenied
	}

	// 查询发布记录
	prp := repo.NewPostRepo()
	post, err := prp.FindById(ctx, request.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询发布记录失败")
		return comm.CodeServerError
	}
	if post == nil {
		return comm.CodeDataNotFound
	}

	// 必须是待审核状态
	if post.Status != enum.PostStatusPending {
		return comm.CodePostStatusInvalid
	}

	// 审核驳回
	err = prp.RejectPost(ctx, request.PostID, request.Reason)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("审核驳回失败")
		return comm.CodeServerError
	}

	// 记录审计日志
	alr := repo.NewAuditLogRepo()
	err = alr.CreateAuditLog(ctx, adminID, enum.AuditLogTypeUpdate, request.Reason, request.PostID, post.Status, enum.PostStatusRejected)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("记录审计日志失败")
		return comm.CodeServerError
	}

	r.Response = RejectApiResponse{Success: true}
	return comm.CodeOK
}

// Init Api初始化
func (r *RejectApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindJSON(&r.Request.Body)
}

// hfReject API执行入口
func hfReject(ctx *gin.Context) {
	api := &RejectApi{}
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
