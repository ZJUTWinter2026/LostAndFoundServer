package admin

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/repo"
	"errors"
	"reflect"
	"runtime"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func ApproveHandler() gin.HandlerFunc {
	api := ApproveApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfApprove).Pointer()).Name()] = api
	return hfApprove
}

type ApproveApi struct {
	Info     struct{} `name:"审核通过发布" desc:"审核通过发布"`
	Request  ApproveApiRequest
	Response ApproveApiResponse
}

type ApproveApiRequest struct {
	Body struct {
		PostID int64 `json:"post_id" binding:"required" desc:"发布ID"`
	}
}

type ApproveApiResponse struct {
	Success bool `json:"success" desc:"是否成功"`
}

func (a *ApproveApi) Run(ctx *gin.Context) kit.Code {
	request := a.Request.Body

	adminID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
	}

	urp := repo.NewUserRepo()
	user, err := urp.FindById(ctx, adminID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return comm.CodeAdminPermissionDenied
	}
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询用户失败")
		return comm.CodeServerError
	}
	if user == nil || (user.Usertype != enum.UserTypeAdmin && user.Usertype != enum.UserTypeSystemAdmin) {
		return comm.CodeAdminPermissionDenied
	}

	prp := repo.NewPostRepo()
	post, err := prp.FindById(ctx, request.PostID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return comm.CodeDataNotFound
	}
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询发布记录失败")
		return comm.CodeServerError
	}

	if post.Status != enum.PostStatusPending {
		return comm.CodePostStatusInvalid
	}

	if user.Usertype == enum.UserTypeAdmin && post.Campus != user.Campus {
		return comm.CodeAdminPermissionDenied
	}

	err = prp.ApprovePost(ctx, request.PostID, adminID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("审核通过失败")
		return comm.CodeServerError
	}

	a.Response = ApproveApiResponse{Success: true}
	return comm.CodeOK
}

func (a *ApproveApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

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
