package router

import (
	"aiki/internal/handler"
	"aiki/internal/middleware"
	"aiki/internal/pkg/jwt"

	"github.com/labstack/echo/v4"
)

func Setup(
	e *echo.Echo,
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
	jobHandler *handler.JobHandler,
	homeHandler *handler.HomeHandler,
	notifHandler *handler.NotificationHandler,
	serpHandler *handler.SerpJobHandler,
	jwtManager *jwt.Manager,
) {
	api := e.Group("/api/v1")

	api.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok", "service": "aiki-api"})
	})

	// Auth
	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/logout", authHandler.Logout)
		auth.GET("/linkedin/login", authHandler.LinkedInLogin)
		auth.POST("/forgot-password", authHandler.ForgottenPassword)
		auth.POST("/forgot-password/validate", authHandler.ValidateForgottenPasswordOTP)
		auth.POST("/reset-password", authHandler.ResetPassword)
	}

	// Users
	users := api.Group("/users")
	users.Use(middleware.Auth(jwtManager))
	{
		users.GET("/me", userHandler.GetMe)
		users.PUT("/me", userHandler.UpdateMe)
		users.POST("/profile", userHandler.CreateProfile)
		users.PATCH("/profile", userHandler.UpdateProfile)
		users.POST("/upload/cv", userHandler.UploadCV)
	}

	// Jobs (manual tracker)
	jobs := api.Group("/jobs")
	jobs.Use(middleware.Auth(jwtManager))
	{
		jobs.POST("", jobHandler.CreateJob)
		jobs.GET("", jobHandler.GetAllJobs)
		jobs.GET("/:id", jobHandler.GetJob)
		jobs.PUT("/:id", jobHandler.UpdateJob)
		jobs.DELETE("/:id", jobHandler.DeleteJob)

		// Job recommendations via SerpApi
		jobs.GET("/recommended", serpHandler.GetRecommendedJobs)
		jobs.POST("/recommended/:id/save", serpHandler.SaveJobToTracker)
	}

	// Home screen
	home := api.Group("/home")
	home.Use(middleware.Auth(jwtManager))
	{
		home.GET("", homeHandler.GetHomeScreen)
	}

	// Focus sessions
	sessions := api.Group("/sessions")
	sessions.Use(middleware.Auth(jwtManager))
	{
		sessions.POST("", homeHandler.StartSession)
		sessions.GET("", homeHandler.GetSessionHistory)
		sessions.GET("/active", homeHandler.GetActiveSession)
		sessions.PATCH("/:id/pause", homeHandler.PauseSession)
		sessions.PATCH("/:id/resume", homeHandler.ResumeSession)
		sessions.PATCH("/:id/end", homeHandler.EndSession)
	}

	// Streaks
	streaks := api.Group("/streaks")
	streaks.Use(middleware.Auth(jwtManager))
	{
		streaks.GET("", homeHandler.GetStreak)
	}

	// Badges
	badges := api.Group("/badges")
	badges.Use(middleware.Auth(jwtManager))
	{
		badges.GET("", homeHandler.GetAllBadges)
		badges.GET("/me", homeHandler.GetUserBadges)
	}

	// Progress
	progress := api.Group("/progress")
	progress.Use(middleware.Auth(jwtManager))
	{
		progress.GET("", homeHandler.GetProgress)
	}

	// Notifications
	notifs := api.Group("/notifications")
	notifs.Use(middleware.Auth(jwtManager))
	{
		notifs.GET("", notifHandler.GetNotifications)
		notifs.GET("/unread-count", notifHandler.GetUnreadCount)
		notifs.PATCH("/read-all", notifHandler.MarkAllRead)
		notifs.PATCH("/:id/read", notifHandler.MarkRead)
		notifs.DELETE("/:id", notifHandler.DeleteNotification)
	}
}
