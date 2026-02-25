package admin

import (
	"fmt"
	"reflect"
	"runtime"

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

// StatisticsHandler API router注册点
func StatisticsHandler() gin.HandlerFunc {
	api := StatisticsApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfStatistics).Pointer()).Name()] = api
	return hfStatistics
}

type StatisticsApi struct {
	Info     struct{}              `name:"获取统计数据" desc:"获取统计数据"`
	Request  StatisticsApiRequest  // API请求参数
	Response StatisticsApiResponse // API响应数据
}

type StatisticsApiRequest struct {
	Query struct{}
}

type StatisticsApiResponse struct {
	StatusCounts   map[string]int64  `json:"status_counts" desc:"各状态数量"`
	TypeCounts     map[string]int64  `json:"type_counts" desc:"各类型数量"`
	TypePercentage map[string]string `json:"type_percentage" desc:"各类型占比"`
}

// Run Api业务逻辑执行点
func (s *StatisticsApi) Run(ctx *gin.Context) kit.Code {
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

	prp := repo.NewPostRepo()

	// 获取状态统计
	statusCounts, err := prp.CountByStatus(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("获取状态统计失败")
		return comm.CodeServerError
	}

	// 获取类型统计
	typeCounts, err := prp.CountByItemType(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("获取类型统计失败")
		return comm.CodeServerError
	}

	// 计算总数和占比
	var total int64
	for _, count := range typeCounts {
		total += count
	}

	typePercentage := make(map[string]string)
	if total > 0 {
		for t, count := range typeCounts {
			percentage := float64(count) / float64(total) * 100
			typePercentage[t] = fmt.Sprintf("%.2f%%", percentage)
		}
	}

	s.Response = StatisticsApiResponse{
		StatusCounts:   statusCounts,
		TypeCounts:     typeCounts,
		TypePercentage: typePercentage,
	}
	return comm.CodeOK
}

// Init Api初始化
func (s *StatisticsApi) Init(ctx *gin.Context) (err error) {
	return nil
}

// hfStatistics API执行入口
func hfStatistics(ctx *gin.Context) {
	api := &StatisticsApi{}
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
