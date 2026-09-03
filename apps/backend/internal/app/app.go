package app

import (
	"context"
	"fmt"
	"strings"

	"backend/config"
	articleHandler "backend/internal/article/handler"
	articleRepo "backend/internal/article/repository"
	articleService "backend/internal/article/service"
	backgroundHandler "backend/internal/background/handler"
	backgroundRepo "backend/internal/background/repository"
	backgroundService "backend/internal/background/service"
	bannerHandler "backend/internal/banner/handler"
	bannerRepo "backend/internal/banner/repository"
	bannerService "backend/internal/banner/service"
	communityHandler "backend/internal/community/handler"
	communityRepo "backend/internal/community/repository"
	communityService "backend/internal/community/service"
	feedbackHandler "backend/internal/feedback/handler"
	feedbackRepo "backend/internal/feedback/repository"
	feedbackService "backend/internal/feedback/service"
	galgameHandler "backend/internal/galgame/handler"
	galgameRepo "backend/internal/galgame/repository"
	galgameService "backend/internal/galgame/service"
	homeHandler "backend/internal/home/handler"
	homeService "backend/internal/home/service"
	imageHandler "backend/internal/image/handler"
	imageRepo "backend/internal/image/repository"
	imageService "backend/internal/image/service"
	"backend/internal/infrastructures/database"
	mailInfrastructure "backend/internal/infrastructures/mail"
	"backend/internal/infrastructures/queue"
	"backend/internal/infrastructures/storage"
	"backend/internal/migrations"
	notificationHandler "backend/internal/notification/handler"
	notificationRepo "backend/internal/notification/repository"
	notificationService "backend/internal/notification/service"
	rbacHandler "backend/internal/rbac/handler"
	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	resourceHandler "backend/internal/resource/handler"
	resourceRepo "backend/internal/resource/repository"
	resourceService "backend/internal/resource/service"
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
	UserAuthRepository  *userRepo.UserAuthRepository
	VerificationHandler *userHandler.VerificationHandler
	UserProfileHandler  *userHandler.UserProfileHandler
	UserAdminHandler    *userHandler.UserAdminHandler
	RBACService         *rbacService.RBACService
	RoleHandler         *rbacHandler.RoleHandler
	PermissionHandler   *rbacHandler.PermissionHandler
	AssignmentHandler   *rbacHandler.AssignmentHandler
	CatalogHandler      *galgameHandler.CatalogHandler
	UserRelationHandler *galgameHandler.UserRelationHandler
	GalleryHandler      *galgameHandler.GalleryHandler
	ResourceHandler     *resourceHandler.ResourceHandler
	ReportHandler       *resourceHandler.ReportHandler
	FeedbackHandler     *feedbackHandler.FeedbackHandler
	PostHandler         *communityHandler.PostHandler
	CommentHandler      *communityHandler.CommentHandler
	InteractionHandler  *communityHandler.InteractionHandler
	BannerHandler       *bannerHandler.BannerHandler
	BackgroundHandler   *backgroundHandler.BackgroundPresetHandler
	ArticleHandler      *articleHandler.ArticleHandler
	HomeHandler         *homeHandler.HomeHandler
	HealthHandler       *healthHandler.HealthHandler
	ImageHandler        *imageHandler.ImageHandler
	NotificationHandler *notificationHandler.NotificationHandler
	stopImageCleanup    func()
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
	notificationRepository := notificationRepo.NewNotificationRepository(postgresDB, cfg.R2.PublicURL)
	notificationSvc := notificationService.NewNotificationService(notificationRepository)

	galgameRepository := galgameRepo.NewGalgameRepository(postgresDB)
	developerRepository := galgameRepo.NewDeveloperRepository(postgresDB)
	tagRepository := galgameRepo.NewTagRepository(postgresDB)
	catalogService := galgameService.NewCatalogService(
		galgameRepository,
		developerRepository,
		tagRepository,
	)
	catalogService.SetNotificationDependencies(rbacSvc, notificationSvc)

	userRelationRepository := galgameRepo.NewUserRelationRepository(postgresDB)
	ratingService := galgameService.NewRatingService(galgameRepository, userRelationRepository)
	favoriteService := galgameService.NewFavoriteService(galgameRepository, userRelationRepository)
	userStateService := galgameService.NewUserStateService(galgameRepository, userRelationRepository)
	userRelationService := galgameService.NewUserRelationService(galgameRepository, userRelationRepository)
	galleryRepository := galgameRepo.NewGalleryRepository(postgresDB)

	resourceRepository := resourceRepo.NewResourceRepository(postgresDB)
	resourceSvc := resourceService.NewResourceService(resourceRepository, galgameRepository, rbacSvc)
	resourceSvc.SetNotificationDependencies(rbacSvc, notificationSvc)
	reportRepository := resourceRepo.NewReportRepository(postgresDB)
	reportSvc := resourceService.NewReportService(reportRepository, resourceRepository)
	reportSvc.SetNotificationDependencies(rbacSvc, notificationSvc)
	feedbackRepository := feedbackRepo.NewFeedbackRepository(postgresDB)
	feedbackSvc := feedbackService.NewFeedbackService(feedbackRepository, redisClient)

	postRepository := communityRepo.NewPostRepository(postgresDB, cfg.R2.PublicURL)
	commentRepository := communityRepo.NewCommentRepository(postgresDB, cfg.R2.PublicURL)
	postService := communityService.NewPostService(postRepository, galgameRepository, rbacSvc, notificationSvc)
	commentService := communityService.NewCommentService(commentRepository, postRepository, rbacSvc, notificationSvc)
	interactionService := communityService.NewInteractionService(postRepository, commentRepository, notificationSvc)

	bannerRepository := bannerRepo.NewBannerRepository(postgresDB)
	articleRepository := articleRepo.NewArticleRepository(postgresDB)
	bannerSvc := bannerService.NewBannerService(bannerRepository, redisClient)
	articleSvc := articleService.NewArticleService(articleRepository, rbacSvc, redisClient)
	backgroundPresetRepository := backgroundRepo.NewBackgroundPresetRepository(postgresDB)
	backgroundPresetSvc := backgroundService.NewBackgroundPresetService(backgroundPresetRepository, cfg.R2.PublicURL)

	r2Storage, err := storage.NewR2(cfg.R2)
	if err != nil {
		if sqlDB, dbErr := postgresDB.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, fmt.Errorf("init r2 storage: %w", err)
	}
	imageRepository := imageRepo.NewImageAssetRepository(postgresDB)
	imageSvc := imageService.NewImageAssetService(
		imageRepository,
		r2Storage,
		rbacSvc,
		redisClient,
		cfg.R2.PublicURL,
	)
	stopImageCleanup := imageSvc.StartCleanupLoop(context.Background())

	galleryService := galgameService.NewGalleryService(galgameRepository, galleryRepository, imageSvc)

	homeSvc := homeService.NewHomeService(
		bannerRepository,
		articleRepository,
		galgameRepository,
		postRepository,
		redisClient,
	)

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
	userPreferenceRepository := userRepo.NewUserPreferenceRepository(postgresDB)
	userAdminRepository := userRepo.NewUserAdminRepository(postgresDB)
	userProfileService := userService.NewUserProfileService(
		userAuthRepository,
		userPreferenceRepository,
		imageSvc,
	)
	userAuthService := userService.NewUserAuthService(
		userAuthRepository,
		refreshSessionRepository,
		verificationService,
		rbacSvc,
		imageSvc,
		cfg.Auth.AccessTokenSecret,
		cfg.Auth.AccessTokenTTL,
		cfg.Auth.RefreshTokenTTL,
	)
	userAdminService := userService.NewUserAdminService(userAdminRepository, rbacSvc)
	app := &App{
		Config:   cfg,
		Postgres: postgresDB,
		Redis:    redisClient,
		Queue:    verificationQueue,
		UserAuthHandler: userHandler.NewUserAuthHandler(
			userAuthService,
			cfg.Auth.RefreshTokenTTL,
		),
		UserAuthRepository:  userAuthRepository,
		VerificationHandler: userHandler.NewVerificationHandler(verificationService),
		UserProfileHandler:  userHandler.NewUserProfileHandler(userProfileService),
		UserAdminHandler:    userHandler.NewUserAdminHandler(userAdminService),
		RBACService:         rbacSvc,
		RoleHandler:         rbacHandler.NewRoleHandler(rbacSvc),
		PermissionHandler:   rbacHandler.NewPermissionHandler(rbacSvc),
		AssignmentHandler:   rbacHandler.NewAssignmentHandler(rbacSvc),
		CatalogHandler:      galgameHandler.NewCatalogHandler(catalogService),
		UserRelationHandler: galgameHandler.NewUserRelationHandler(
			ratingService,
			favoriteService,
			userStateService,
			userRelationService,
		),
		GalleryHandler:      galgameHandler.NewGalleryHandler(galleryService),
		ResourceHandler:     resourceHandler.NewResourceHandler(resourceSvc),
		ReportHandler:       resourceHandler.NewReportHandler(reportSvc),
		FeedbackHandler:     feedbackHandler.NewFeedbackHandler(feedbackSvc),
		PostHandler:         communityHandler.NewPostHandler(postService),
		CommentHandler:      communityHandler.NewCommentHandler(commentService),
		InteractionHandler:  communityHandler.NewInteractionHandler(interactionService),
		BannerHandler:       bannerHandler.NewBannerHandler(bannerSvc),
		BackgroundHandler:   backgroundHandler.NewBackgroundPresetHandler(backgroundPresetSvc),
		ArticleHandler:      articleHandler.NewArticleHandler(articleSvc),
		HomeHandler:         homeHandler.NewHomeHandler(homeSvc),
		HealthHandler:       healthHandler.NewHealthHandler(healthService),
		ImageHandler:        imageHandler.NewImageHandler(imageSvc),
		NotificationHandler: notificationHandler.NewNotificationHandler(notificationSvc),
		stopImageCleanup:    stopImageCleanup,
	}
	ginApp := gin.Default()
	if err := ginApp.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		app.Close()
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}
	app.Gin = ginApp
	app.setupRoutes()

	mailer := mailInfrastructure.NewSMTPMailer(workerCfg.SMTP)
	emailService := notificationService.NewEmailService(mailer, cfg.R2.PublicURL)
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
	if app.stopImageCleanup != nil {
		app.stopImageCleanup()
	}
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
