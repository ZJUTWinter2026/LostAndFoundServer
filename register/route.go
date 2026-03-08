package register

import (
	"app/api/admin"
	"app/api/admin/account"
	"app/api/admin/system"
	"app/api/agent"
	"app/api/announcement"
	"app/api/claim"
	"app/api/feedback"
	"app/api/post"
	"app/api/user"
	"app/comm"
	"app/middleware"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/zjutjh/mygo/config"
	"github.com/zjutjh/mygo/middleware/cors"
	"github.com/zjutjh/mygo/session"
	"github.com/zjutjh/mygo/swagger"
)

func Route(router *gin.Engine) {
	router.Use(cors.Pick())
	router.Use(session.Pick())

	uploadDir := comm.BizConf.Upload.Dir
	router.Static("/"+uploadDir, "./"+uploadDir)

	r := router.Group(routePrefix())
	r.Use(middleware.CheckUserDisabled())
	{
		routeBase(r, router)

		userGroup := r.Group("/user")
		{
			userGroup.POST("/login", user.LoginHandler())
			userGroup.POST("/forgot-password", user.ForgotPasswordHandler())
			userGroup.POST("/update", user.UpdateHandler())
			userGroup.POST("upload", user.UploadHandler())
		}

		lostFoundGroup := r.Group("/post")
		{
			lostFoundGroup.POST("/publish", post.PublishHandler())
			lostFoundGroup.GET("/list", post.QueryHandler())
			lostFoundGroup.GET("/detail", post.DetailHandler())
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
			adminGroup.GET("/review-records", admin.ReviewRecordsHandler())
			adminGroup.GET("/statistics", admin.StatisticsHandler())
			adminGroup.POST("/claim", admin.ClaimPostHandler())
			adminGroup.POST("/archive", admin.ArchivePostHandler())
			adminGroup.GET("/export", admin.ExportDataHandler())
			adminGroup.GET("/expired-list", admin.ExpiredListHandler())
			adminGroup.DELETE("/expired-clean", admin.ExpiredCleanHandler())
			adminGroup.GET("/post-list", admin.PostListHandler())
			adminGroup.GET("/published-list", admin.PublishedListHandler())
		}

		claimGroup := r.Group("/claim")
		{
			claimGroup.POST("/apply", claim.ApplyHandler())
			claimGroup.GET("/list", claim.ListClaimsHandler())
			claimGroup.POST("/review", claim.ReviewHandler())
			claimGroup.GET("/my-list", claim.MyListHandler())
			claimGroup.POST("/cancel", claim.CancelHandler())
		}

		feedbackGroup := r.Group("/feedback")
		{
			feedbackGroup.POST("/submit", feedback.SubmitHandler())
			feedbackGroup.GET("/my-list", feedback.MyListHandler())
			feedbackGroup.GET("/list", feedback.ListHandler())
			feedbackGroup.GET("/detail", feedback.DetailHandler())
			feedbackGroup.POST("/process", feedback.ProcessHandler())
		}

		announcementGroup := r.Group("/announcement")
		{
			announcementGroup.GET("/list", announcement.ListHandler())
			announcementGroup.POST("/publish", announcement.PublishHandler())
			announcementGroup.GET("/review-list", announcement.ReviewListHandler())
			announcementGroup.POST("/approve", announcement.ApproveHandler())
			announcementGroup.DELETE("/delete", announcement.DeleteHandler())
			announcementGroup.GET("/all-list", announcement.AllListHandler())
		}

		systemGroup := r.Group("/system")
		{
			systemGroup.GET("/config", system.ConfigListHandler())
			systemGroup.PUT("/feedback-types", system.UpdateFeedbackTypesHandler())
			systemGroup.PUT("/item-types", system.UpdateItemTypesHandler())
			systemGroup.PUT("/claim-validity-days", system.UpdateClaimValidityDaysHandler())
			systemGroup.PUT("/publish-limit", system.UpdatePublishLimitHandler())
		}

		accountGroup := r.Group("/account")
		{
			accountGroup.GET("/list", account.ListHandler())
			accountGroup.POST("/create", account.CreateHandler())
			accountGroup.POST("/reset-password", account.ResetPasswordHandler())
			accountGroup.POST("/disable", account.DisableHandler())
			accountGroup.POST("/enable", account.EnableHandler())
		}

		agentGroup := r.Group("/agent")
		agentGroup.Use(middleware.AgentEnabled())
		{
			agentGroup.POST("/session", agent.SessionHandler())
			agentGroup.GET("/sessions", agent.SessionListHandler())
			agentGroup.POST("/stream", agent.StreamHandler())
			agentGroup.GET("/history", agent.HistoryHandler())
		}

		adminPostGroup := r.Group("/admin/post")
		{
			adminPostGroup.DELETE("/delete", admin.DeletePostHandler())
		}
	}
}

func routePrefix() string {
	return "/api"
}

func routeBase(r *gin.RouterGroup, router *gin.Engine) {
	if slices.Contains([]string{config.AppEnvDev, config.AppEnvTest}, config.AppEnv()) {
		r.GET("/swagger.json", swagger.DocumentHandler(router))
	}
}
