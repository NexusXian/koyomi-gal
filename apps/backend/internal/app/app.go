package app

import (
	"context"
	"fmt"
	"strings"

	"backend/config"
	"backend/internal/infrastructures/database"
	mailInfrastructure "backend/internal/infrastructures/mail"
	"backend/internal/infrastructures/queue"
	"backend/internal/migrations"
	notificationService "backend/internal/notification/service"
	rbacHandler "backend/internal/rbac/handler"
	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	userHandler "backend/internal/user/handler"
	userRepo "backend/internal/user/repository"
	userService "backend/internal/user/service"

	healthHandler "backend/internal/health/handler"
	healthService "backend/internal/health/service"
	"backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
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
	RBACService         *rbacService.RBACService
	RoleHandler         *rbacHandler.RoleHandler
	PermissionHandler   *rbacHandler.PermissionHandler
	AssignmentHandler   *rbacHandler.AssignmentHandler
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

	//Init RBAC module
	rbacRepository := rbacRepo.NewRBACRepository(postgresDB)
	rbacSvc := rbacService.NewRBACService(rbacRepository)
	bootstrapCtx := context.Background()
	if err := rbacSvc.SeedDefaults(bootstrapCtx); err != nil {
		app := &App{Config: cfg, Postgres: postgresDB, Redis: redisClient}
		app.Close()
		return nil, fmt.Errorf("seed rbac defaults: %w", err)
	}
	if err := bootstrapSuperAdmin(bootstrapCtx, cfg.RBAC.SuperAdminAccount, userAuthRepository, rbacSvc); err != nil {
		app := &App{Config: cfg, Postgres: postgresDB, Redis: redisClient}
		app.Close()
		return nil, err
	}

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
		rbacSvc,
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
		RBACService:         rbacSvc,
		RoleHandler:         rbacHandler.NewRoleHandler(rbacSvc),
		PermissionHandler:   rbacHandler.NewPermissionHandler(rbacSvc),
		AssignmentHandler:   rbacHandler.NewAssignmentHandler(rbacSvc),
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

// bootstrapSuperAdmin binds the configured account (email or username) to the
// super_admin role. It is idempotent and only ever promotes that one account.
func bootstrapSuperAdmin(
	ctx context.Context,
	account string,
	users *userRepo.UserAuthRepository,
	rbacSvc *rbacService.RBACService,
) error {
	account = strings.TrimSpace(account)
	if account == "" {
		return nil
	}

	user, err := users.FindUserByEmail(ctx, strings.ToLower(account))
	if err != nil {
		return fmt.Errorf("find super admin account by email: %w", err)
	}
	if user == nil {
		user, err = users.FindUserByUsername(ctx, account)
		if err != nil {
			return fmt.Errorf("find super admin account by username: %w", err)
		}
	}
	if user == nil {
		logger.Warn("rbac super admin account not found", zap.String("account", account))
		return nil
	}

	if err := rbacSvc.AssignRoleByCode(ctx, user.ID, rbacService.RoleCodeSuperAdmin); err != nil {
		return fmt.Errorf("assign super admin role: %w", err)
	}
	logger.Info(
		"rbac super admin bootstrapped",
		zap.Uint("user_id", user.ID),
		zap.String("username", user.Username),
	)
	return nil
}
