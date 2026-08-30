package app

import (
	"context"
	"fmt"

	"backend/config"
	"backend/internal/infrastructures/database"
	mailInfrastructure "backend/internal/infrastructures/mail"
	"backend/internal/infrastructures/queue"
	"backend/internal/migrations"
	notificationService "backend/internal/notification/service"
	userHandler "backend/internal/user/handler"
	userRepo "backend/internal/user/repository"
	userService "backend/internal/user/service"

	healthHandler "backend/internal/health/handler"
	healthService "backend/internal/health/service"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	Config              *config.Config
	Gin                 *gin.Engine
	Postgres            *gorm.DB
	Redis               *redis.Client
	Queue               *queue.VerificationClient
	MailWorker          *asynq.Server
	UserAuthHandler     *userHandler.UserAuthHandler
	VerificationHandler *userHandler.VerificationHandler
	HealthHandler       *healthHandler.HealthHandler
}

func New(cfg *config.Config, workerCfg *config.WorkerConfig) (*App, error) {
	postgresDB, err := database.NewPostgre(cfg.Postgres)
	if err != nil {
		return nil, err
	}
	sqlDB, err := postgresDB.DB()
	if err != nil {
		return nil, fmt.Errorf("get PostgreSQL connection pool: %w", err)
	}
	if err := migrations.NewService(sqlDB).Up(context.Background()); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	redisClient, err := database.NewRedis(cfg.Redis)
	if err != nil {
		if sqlDB, dbErr := postgresDB.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, err
	}
	// Init Health Service module
	healthService := healthService.NewHealthService()

	//Init User module
	userAuthRepository := userRepo.NewUserAuthRepository(postgresDB)
	refreshSessionRepository := userRepo.NewRefreshSessionRepository(redisClient)
	verificationRepository := userRepo.NewVerificationRepository(redisClient)
	verificationQueue := queue.NewVerificationClient(cfg.Redis, cfg.Verification.Secret)
	verificationService := userService.NewVerificationService(
		verificationRepository,
		verificationQueue,
		cfg.Verification.Secret,
		cfg.Verification.CodeTTL,
		cfg.Verification.ResendInterval,
		cfg.Verification.IPWindow,
		cfg.Verification.IPLimit,
	)
	userAuthService := userService.NewUserAuthService(
		userAuthRepository,
		refreshSessionRepository,
		verificationService,
		cfg.Auth.AccessTokenSecret,
		cfg.Auth.AccessTokenTTL,
		cfg.Auth.RefreshTokenTTL,
	)
	app := &App{
		Config:   cfg,
		Postgres: postgresDB,
		Redis:    redisClient,
		Queue:    verificationQueue,
		UserAuthHandler: userHandler.NewUserAuthHandler(
			userAuthService,
			cfg.Auth.RefreshTokenTTL,
		),
		VerificationHandler: userHandler.NewVerificationHandler(verificationService),
		HealthHandler:       healthHandler.NewHealthHandler(healthService),
	}
	ginApp := gin.Default()
	if err := ginApp.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		app.Close()
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}
	app.Gin = ginApp
	app.setupRoutes()

	mailer := mailInfrastructure.NewSMTPMailer(workerCfg.SMTP)
	emailService := notificationService.NewEmailService(mailer)
	mailWorker := queue.NewServer(workerCfg.Redis, workerCfg.Concurrency)
	if err := mailWorker.Start(queue.NewServeMux(
		emailService,
		verificationRepository,
		workerCfg.VerificationSecret,
	)); err != nil {
		app.Close()
		return nil, fmt.Errorf("start mail worker: %w", err)
	}
	app.MailWorker = mailWorker

	return app, nil
}

func (app *App) Close() {
	if app.MailWorker != nil {
		app.MailWorker.Shutdown()
	}
	if app.Queue != nil {
		_ = app.Queue.Close()
	}
	if app.Redis != nil {
		_ = app.Redis.Close()
	}
	if app.Postgres != nil {
		if sqlDB, err := app.Postgres.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
}
