package admin

import (
	"app/comm"
	"app/comm/enum"
	"app/dao/repo"
	"errors"
	"reflect"
	"runtime"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func RejectHandler() gin.HandlerFunc {
	api := RejectApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfReject).Pointer()).Name()] = api
	return hfReject
}

type RejectApi struct {
	Info     struct{} `name:"审核驳回发布" desc:"审核驳回发布"`
	Request  RejectApiRequest
	Response RejectApiResponse
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

func (r *RejectApi) Run(ctx *gin.Context) kit.Code {
	request := r.Request.Body

	if strings.TrimSpace(request.Reason) == "" {
		return comm.CodeReviewReasonRequired
	}
	if utf8.RuneCountInString(request.Reason) > 500 {
		return comm.CodeReviewReasonTooLong
	}

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

	err = prp.RejectPost(ctx, request.PostID, request.Reason, adminID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("审核驳回失败")
		return comm.CodeServerError
	}

	r.Response = RejectApiResponse{Success: true}
	return comm.CodeOK
}

func (r *RejectApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindJSON(&r.Request.Body)
}

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
