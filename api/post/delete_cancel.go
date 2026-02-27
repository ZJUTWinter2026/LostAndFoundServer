package post

import (
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/comm/enum"
	"app/dao/repo"
)

// DeleteHandler API router注册点
func DeleteHandler() gin.HandlerFunc {
	api := DeleteApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfDelete).Pointer()).Name()] = api
	return hfDelete
}

type DeleteApi struct {
	Info     struct{}          `name:"删除我的发布信息" desc:"删除我的发布信息"`
	Request  DeleteApiRequest  // API请求参数
	Response DeleteApiResponse // API响应数据
}

type DeleteApiRequest struct {
	Body struct {
		PostID int64 `json:"post_id" binding:"required" desc:"发布ID"`
	}
}

type DeleteApiResponse struct {
	Success bool `json:"success" desc:"是否成功"`
}

// Run Api业务逻辑执行点
func (d *DeleteApi) Run(ctx *gin.Context) kit.Code {
	request := d.Request.Body

	// 获取当前用户ID
	publisherID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
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

	// 权限验证：仅发布者可删除
	if post.PublisherID != publisherID {
		return comm.CodePostNotOwner
	}

	// 状态验证：仅待审核状态可删除
	if post.Status != enum.PostStatusPending {
		return comm.CodePostStatusInvalid
	}

	// 删除发布记录
	err = prp.DeletePost(ctx, request.PostID, publisherID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("删除发布记录失败")
		return comm.CodeServerError
	}

	d.Response = DeleteApiResponse{Success: true}
	return comm.CodeOK
}

// Init Api初始化
func (d *DeleteApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindJSON(&d.Request.Body)
}

// hfDelete API执行入口
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

// CancelHandler API router注册点
func CancelHandler() gin.HandlerFunc {
	api := CancelApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfCancel).Pointer()).Name()] = api
	return hfCancel
}

type CancelApi struct {
	Info     struct{}          `name:"取消发布信息" desc:"取消发布信息"`
	Request  CancelApiRequest  // API请求参数
	Response CancelApiResponse // API响应数据
}

type CancelApiRequest struct {
	Body struct {
		PostID int64  `json:"post_id" binding:"required" desc:"发布ID"`
		Reason string `json:"reason" binding:"max=255" desc:"取消原因"`
	}
}

type CancelApiResponse struct {
	Success bool `json:"success" desc:"是否成功"`
}

// Run Api业务逻辑执行点
func (c *CancelApi) Run(ctx *gin.Context) kit.Code {
	request := c.Request.Body

	// 获取当前用户ID
	publisherID, err := session.GetIdentity[int64](ctx)
	if err != nil {
		return comm.CodeNotLoggedIn
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

	// 权限验证：仅发布者可取消
	if post.PublisherID != publisherID {
		return comm.CodePostNotOwner
	}

	// 状态验证：仅已通过状态可取消
	if post.Status != enum.PostStatusApproved {
		return comm.CodePostStatusInvalid
	}

	// 取消发布
	err = prp.CancelPost(ctx, request.PostID, publisherID, request.Reason)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("取消发布失败")
		return comm.CodeServerError
	}

	c.Response = CancelApiResponse{Success: true}
	return comm.CodeOK
}

// Init Api初始化
func (c *CancelApi) Init(ctx *gin.Context) (err error) {
	return ctx.ShouldBindJSON(&c.Request.Body)
}

// hfCancel API执行入口
func hfCancel(ctx *gin.Context) {
	api := &CancelApi{}
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
