package middleware

import (
	"app/comm"
	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
)

func AgentEnabled() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !comm.BizConf.Agent.Enable {
			reply.Fail(ctx, comm.CodeAgentDisabled)
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
