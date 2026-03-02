package middleware

import (
	"app/comm"
	"app/dao/repo"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/session"
)

func CheckUserDisabled() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID, err := session.GetIdentity[int64](ctx)
		if err != nil {
			ctx.Next()
			return
		}

		userRepo := repo.NewUserRepo()
		user, err := userRepo.FindById(ctx, userID)
		if err != nil || user == nil {
			ctx.Next()
			return
		}

		if user.DisabledUntil != nil && user.DisabledUntil.After(time.Now()) {
			reply.Fail(ctx, comm.CodeUserDisabled)
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}
