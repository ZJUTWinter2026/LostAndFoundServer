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

func DeletePostHandler() gin.HandlerFunc {
	api := DeletePostApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfDeletePost).Pointer()).Name()] = api
	return hfDeletePost
}

type DeletePostApi struct {
	Info     struct{} `name:"删除发布信息" desc:"系统管理员删除违规、虚假的发布信息"`
	Request  DeletePostApiRequest
	Response DeletePostApiResponse
}

type DeletePostApiRequest struct {
	Body struct {
		PostID int64 `json:"post_id" binding:"required" desc:"发布ID"`
	}
}

type DeletePostApiResponse struct {
	Success bool `json:"success" desc:"是否成功"`
}

func (a *DeletePostApi) Run(ctx *gin.Context) kit.Code {
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
	if user == nil || user.Usertype != enum.UserTypeSystemAdmin {
		return comm.CodeAdminPermissionDenied
	}

	prp := repo.NewPostRepo()
	_, err = prp.FindById(ctx, a.Request.Body.PostID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return comm.CodeDataNotFound
	}
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询发布记录失败")
		return comm.CodeServerError
	}

	err = prp.DeletePostByAdmin(ctx, a.Request.Body.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("删除发布信息失败")
		return comm.CodeServerError
	}

	a.Response = DeletePostApiResponse{Success: true}
	return comm.CodeOK
}

func (a *DeletePostApi) Init(ctx *gin.Context) error {
	return ctx.ShouldBindJSON(&a.Request.Body)
}

func hfDeletePost(ctx *gin.Context) {
	api := &DeletePostApi{}
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
