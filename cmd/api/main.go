package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"aiki/internal/ai"
	aiAnthropic "aiki/internal/ai/anthropic"
	aiOpenAI "aiki/internal/ai/openai"
	"aiki/internal/config"
	"aiki/internal/database"
	"aiki/internal/handler"
	"aiki/internal/middleware"
	"aiki/internal/pkg/jwt"
	"aiki/internal/pkg/mailer"
	"aiki/internal/pkg/scheduler"
	"aiki/internal/pkg/validator"
	"aiki/internal/repository"
	"aiki/internal/router"
	"aiki/internal/serp"
	"aiki/internal/service"

	_ "aiki/docs"

	"github.com/labstack/echo/v4"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           Aiki API
// @version         1.0
// @description     Backend API for Aiki — focus sessions, job tracking, streaks and badges.
// @contact.name    Aiki Support
// @contact.email   support@aiki.app
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     Type "Bearer" followed by your access token.

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := database.NewPostgresPool(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("✓ Database connection established")

	redis, err := database.NewRedisClient(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()
	log.Println("✓ Redis connection established")

	jwtManager := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry)

	// Repositories
	userRepo := repository.NewUserRepository(db)
	jobRepo := repository.NewJobRepository(db)
	homeRepo := repository.NewHomeRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	serpRepo := repository.NewSerpJobRepository(db)

	// Services
	serpClient := serp.NewClient(cfg.SerpAPI.Key)
	emailSender := mailer.NewResendSender(
		cfg.Email.ResendAPIKey,
		cfg.Email.FromEmail,
		cfg.Email.FromName,
	)
	authService := service.NewAuthService(userRepo, jwtManager)
	emailVerificationService := service.NewEmailVerificationService(
		userRepo,
		redis,
		emailSender,
		cfg.Server.Env,
	)
	passwordResetService := service.NewPasswordResetService(
		userRepo,
		redis,
		emailSender,
		cfg.Server.Env,
	)
	userService := service.NewUserService(userRepo)
	jobService := service.NewJobService(jobRepo)
	notifService := service.NewNotificationService(notifRepo)
	homeService := service.NewHomeService(homeRepo, notifService)
	serpJobService := service.NewSerpJobService(serpRepo, userRepo, jobRepo, serpClient)

	// AI providers & chat service
	aiRegistry := ai.NewRegistry()
	if cfg.AI.OpenAI.APIKey != "" {
		aiRegistry.Register(aiOpenAI.New(cfg.AI.OpenAI.APIKey, cfg.AI.OpenAI.DefaultModel))
		log.Println("✓ AI provider registered: openai")
	}
	if cfg.AI.Anthropic.APIKey != "" {
		aiRegistry.Register(aiAnthropic.New(cfg.AI.Anthropic.APIKey, cfg.AI.Anthropic.DefaultModel))
		log.Println("✓ AI provider registered: anthropic")
	}
	chatService := service.NewChatService(aiRegistry)

	// Echo
	e := echo.New()
	e.HideBanner = true
	e.Validator = validator.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.GET("/swagger/*", echo.WrapHandler(httpSwagger.WrapHandler))

	// Handlers
	authHandler := handler.NewAuthHandler(
		authService,
		emailVerificationService,
		passwordResetService,
		e.Validator,
		redis,
		*cfg,
	)
	userHandler := handler.NewUserHandler(userService, e.Validator)
	jobHandler := handler.NewJobHandler(jobService, e.Validator)
	homeHandler := handler.NewHomeHandler(homeService, e.Validator)
	notifHandler := handler.NewNotificationHandler(notifService)
	serpHandler := handler.NewSerpJobHandler(serpJobService)
	chatHandler := handler.NewChatHandler(chatService)

	// Routes
	router.Setup(e, authHandler, userHandler, jobHandler, homeHandler, notifHandler, serpHandler, chatHandler, jwtManager)

	// Scheduler
	sched := scheduler.NewScheduler(notifService)
	sched.Start()
	log.Println("✓ Notification scheduler started")

	serverAddr := fmt.Sprintf(":%s", cfg.Server.Port)
	go func() {
		log.Printf("✓ Server starting on %s", serverAddr)
		log.Printf("✓ Swagger UI at http://localhost%s/swagger/index.html", serverAddr)
		if err := e.Start(serverAddr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited!")
}
