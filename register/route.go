package register

import (
	"app/api/admin"
	"app/api/claim"
	"app/api/feedback"
	"app/api/post"
	"app/api/user"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/config"
	"github.com/zjutjh/mygo/middleware/cors"
	"github.com/zjutjh/mygo/swagger"

	"app/api"
)

func Route(router *gin.Engine) {
	router.Use(cors.Pick())

	r := router.Group(routePrefix())
	{
		routeBase(r, router)

		// 注册业务逻辑接口
		userGroup := r.Group("/user")
		{
			userGroup.POST("/login", user.LoginHandler())
			userGroup.POST("/update", user.UpdateHandler())
			userGroup.POST("upload", user.UploadHandler())
		}

		lostFoundGroup := r.Group("/post")
		{
			lostFoundGroup.POST("/publish", post.PublishHandler())
			lostFoundGroup.GET("/list", post.QueryHandler())
			lostFoundGroup.GET("/detail/:id", post.DetailHandler())
			lostFoundGroup.GET("/my-list", post.MyListHandler())
			lostFoundGroup.PUT("/update", post.UpdateHandler())
			lostFoundGroup.DELETE("/delete", post.DeleteHandler())
			lostFoundGroup.POST("/cancel", post.CancelHandler())
		}

		adminGroup := r.Group("/admin")
		{
			adminGroup.GET("/list", admin.ReviewListHandler())
			adminGroup.GET("/detail", admin.ReviewDetailHandler())
			adminGroup.POST("/approve", admin.ApproveHandler())
			adminGroup.POST("/reject", admin.RejectHandler())

		}

		claimGroup := r.Group("/claim")
		{
			claimGroup.POST("/apply", claim.ApplyHandler())
			claimGroup.GET("/list", claim.ListClaimsHandler())
			claimGroup.POST("/review", claim.ReviewHandler())
		}

		feedbackGroup := r.Group("/feedback")
		{
			feedbackGroup.POST("/submit", feedback.SubmitHandler())
			feedbackGroup.GET("/my-list", feedback.MyListHandler())
			feedbackGroup.GET("/list", feedback.ListHandler())
			feedbackGroup.POST("/process", feedback.ProcessHandler())
		}
	}
}

func routePrefix() string {
	return "/api"
}

func routeBase(r *gin.RouterGroup, router *gin.Engine) {
	// OpenAPI/Swagger 文档生成
	if slices.Contains([]string{config.AppEnvDev, config.AppEnvTest}, config.AppEnv()) {
		r.GET("/swagger.json", swagger.DocumentHandler(router))
	}

	// 健康检查
	r.GET("/health", api.HealthHandler())

}
