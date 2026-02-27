package admin

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/repo"
	"reflect"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func ClaimPostHandler() gin.HandlerFunc {
	api := ClaimPostApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfClaimPost).Pointer()).Name()] = api
	return hfClaimPost
}

type ClaimPostApi struct {
	Info     struct{}            `name:"标记已解决" desc:"管理员标记物品为已解决"`
	Request  ClaimPostApiRequest
	Response ClaimPostApiResponse
}

type ClaimPostApiRequest struct {
	Body struct {
		PostID int64 `json:"post_id" binding:"required" desc:"发布ID"`
	}
}

type ClaimPostApiResponse struct {
	Success bool `json:"success" desc:"是否成功"`
}

func (a *ClaimPostApi) Run(ctx *gin.Context) kit.Code {
	req := a.Request.Body

	adminID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	if code := checkAdminPermission(ctx, adminID); code != comm.CodeOK {
		return code
	}

	prp := repo.NewPostRepo()
	post, err := prp.FindById(ctx, req.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询发布记录失败")
		return comm.CodeServerError
	}
	if post == nil {
		return comm.CodeDataNotFound
	}

	if post.Status != enum.PostStatusApproved {
		return comm.CodePostStatusInvalid
	}

	err = prp.MarkAsSolved(ctx, req.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("标记已解决失败")
		return comm.CodeServerError
	}

	alr := repo.NewAuditLogRepo()
	err = alr.CreateAuditLog(ctx, adminID, enum.AuditLogTypeUpdate, "", req.PostID, post.Status, enum.PostStatusSolved)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("记录审计日志失败")
		return comm.CodeServerError
	}

	a.Response = ClaimPostApiResponse{Success: true}
	return comm.CodeOK
}

func (a *ClaimPostApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfClaimPost(ctx *gin.Context) {
	api := &ClaimPostApi{}
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

func ArchivePostHandler() gin.HandlerFunc {
	api := ArchivePostApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfArchivePost).Pointer()).Name()] = api
	return hfArchivePost
}

type ArchivePostApi struct {
	Info     struct{}             `name:"归档物品" desc:"管理员归档物品"`
	Request  ArchivePostApiRequest
	Response ArchivePostApiResponse
}

type ArchivePostApiRequest struct {
	Body struct {
		PostID        int64  `json:"post_id" binding:"required" desc:"发布ID"`
		ArchiveMethod string `json:"archive_method" binding:"required,max=100" desc:"物品处理方式"`
	}
}

type ArchivePostApiResponse struct {
	Success bool `json:"success" desc:"是否成功"`
}

func (a *ArchivePostApi) Run(ctx *gin.Context) kit.Code {
	req := a.Request.Body

	if strings.TrimSpace(req.ArchiveMethod) == "" {
		return comm.CodeArchiveReasonRequired
	}
	if utf8.RuneCountInString(req.ArchiveMethod) > 100 {
		return comm.CodeParameterInvalid
	}

	adminID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	if code := checkAdminPermission(ctx, adminID); code != comm.CodeOK {
		return code
	}

	prp := repo.NewPostRepo()
	post, err := prp.FindById(ctx, req.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询发布记录失败")
		return comm.CodeServerError
	}
	if post == nil {
		return comm.CodeDataNotFound
	}

	if post.Status != enum.PostStatusApproved {
		return comm.CodePostStatusInvalid
	}

	scr := repo.NewSystemConfigRepo()
	claimValidityDays, err := scr.GetClaimValidityDays(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("获取认领时效失败")
		return comm.CodeServerError
	}

	expiredTime := post.ProcessedAt.AddDate(0, 0, claimValidityDays)
	if time.Now().Before(expiredTime) {
		return comm.CodeArchiveNotExpired
	}

	err = prp.ArchivePost(ctx, req.PostID, req.ArchiveMethod)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("归档失败")
		return comm.CodeServerError
	}

	alr := repo.NewAuditLogRepo()
	err = alr.CreateAuditLog(ctx, adminID, enum.AuditLogTypeUpdate, req.ArchiveMethod, req.PostID, post.Status, enum.PostStatusArchived)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("记录审计日志失败")
		return comm.CodeServerError
	}

	a.Response = ArchivePostApiResponse{Success: true}
	return comm.CodeOK
}

func (a *ArchivePostApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfArchivePost(ctx *gin.Context) {
	api := &ArchivePostApi{}
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

func checkAdminPermission(ctx *gin.Context, adminID int64) kit.Code {
	urp := repo.NewUserRepo()
	user, err := urp.FindById(ctx, adminID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询用户失败")
		return comm.CodeServerError
	}
	if user == nil || (user.Usertype != enum.UserTypeAdmin && user.Usertype != enum.UserTypeSystemAdmin) {
		return comm.CodeAdminPermissionDenied
	}
	return comm.CodeOK
}
