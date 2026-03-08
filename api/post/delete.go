package post

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

func DeleteHandler() gin.HandlerFunc {
	api := DeleteApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfDelete).Pointer()).Name()] = api
	return hfDelete
}

type DeleteApi struct {
	Info     struct{} `name:"删除我的发布信息" desc:"删除我的发布信息（仅待审核状态可删除）"`
	Request  DeleteApiRequest
	Response DeleteApiResponse
}

type DeleteApiRequest struct {
	Body struct {
		PostID int64 `json:"post_id" binding:"required" desc:"发布ID"`
	}
}

type DeleteApiResponse struct{}

func (d *DeleteApi) Run(ctx *gin.Context) kit.Code {
	request := d.Request.Body

	publisherID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
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

	if post.PublisherID != publisherID {
		return comm.CodePostNotOwner
	}

	if post.Status != enum.PostStatusPending {
		return comm.CodePostStatusInvalid
	}

	err = prp.DeletePost(ctx, request.PostID, publisherID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("删除发布记录失败")
		return comm.CodeServerError
	}

	vectorRepo := repo.NewVectorRepo()
	err = vectorRepo.Delete(ctx, request.PostID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("删除向量索引失败")
	}

	return comm.CodeOK
}

func (d *DeleteApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindJSON(&d.Request.Body)
}

func hfDelete(ctx *gin.Context) {
	api := &DeleteApi{}
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
