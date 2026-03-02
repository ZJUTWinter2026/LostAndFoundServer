package admin

import (
	"app/comm"
	"app/dao/repo"
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"
)

func ExpiredCleanHandler() gin.HandlerFunc {
	api := ExpiredCleanApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfExpiredClean).Pointer()).Name()] = api
	return hfExpiredClean
}

type ExpiredCleanApi struct {
	Info     struct{} `name:"清理过期无效数据" desc:"删除已归档、已删除、已取消的发布信息"`
	Request  ExpiredCleanApiRequest
	Response ExpiredCleanApiResponse
}

type ExpiredCleanApiRequest struct {
	Body struct{}
}

type ExpiredCleanApiResponse struct {
	DeletedCount int64 `json:"deleted_count" desc:"删除数量"`
}

func (e *ExpiredCleanApi) Run(ctx *gin.Context) kit.Code {
	if code := comm.CheckSysAdmin(ctx); code != comm.CodeOK {
		return code
	}

	prp := repo.NewPostRepo()

	posts, err := prp.ListExpired(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("查询过期无效数据失败")
		return comm.CodeServerError
	}

	count := int64(len(posts))

	if count > 0 {
		err = prp.DeleteExpired(ctx)
		if err != nil {
			nlog.Pick().WithContext(ctx).WithError(err).Warn("清理过期无效数据失败")
			return comm.CodeServerError
		}
	}

	e.Response = ExpiredCleanApiResponse{DeletedCount: count}
	return comm.CodeOK
}

func (e *ExpiredCleanApi) Init(ctx *gin.Context) error {
	return nil
}

func hfExpiredClean(ctx *gin.Context) {
	api := &ExpiredCleanApi{}
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
