package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

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
	classificationAgent "backend/internal/classification/agent"
	classificationHandler "backend/internal/classification/handler"
	classificationRepo "backend/internal/classification/repository"
	classificationService "backend/internal/classification/service"
	"backend/internal/classification/tools"
	communityHandler "backend/internal/community/handler"
	communityRepo "backend/internal/community/repository"
	communityService "backend/internal/community/service"
	contributionRepo "backend/internal/contribution/repository"
	contributionService "backend/internal/contribution/service"
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
	importerHandler "backend/internal/importer/handler"
	"backend/internal/importer/provider"
	importerRepo "backend/internal/importer/repository"
	importerService "backend/internal/importer/service"
	"backend/internal/infrastructures/database"
	mailInfrastructure "backend/internal/infrastructures/mail"
	"backend/internal/infrastructures/queue"
	"backend/internal/infrastructures/storage"
	"backend/internal/migrations"
	notificationHandler "backend/internal/notification/handler"
	notificationRepo "backend/internal/notification/repository"
	notificationService "backend/internal/notification/service"
	novelHandler "backend/internal/novel/handler"
	novelRepo "backend/internal/novel/repository"
	novelService "backend/internal/novel/service"
	rbacHandler "backend/internal/rbac/handler"
	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	relationRepo "backend/internal/relation/repository"
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
	Config                *config.Config
	Gin                   *gin.Engine
	Postgres              *gorm.DB
	Redis                 *redis.Client
	Queue                 *queue.VerificationClient
	ImportQueue           *queue.ImportClient
	ClassificationQueue   *queue.ClassificationClient
	MailWorker            *asynq.Server
	ImportWorker          *asynq.Server
	ClassificationWorker  *asynq.Server
	UserAuthHandler       *userHandler.UserAuthHandler
	UserAuthRepository    *userRepo.UserAuthRepository
	VerificationHandler   *userHandler.VerificationHandler
	UserProfileHandler    *userHandler.UserProfileHandler
	UserAdminHandler      *userHandler.UserAdminHandler
	RBACService           *rbacService.RBACService
	RoleHandler           *rbacHandler.RoleHandler
	PermissionHandler     *rbacHandler.PermissionHandler
	AssignmentHandler     *rbacHandler.AssignmentHandler
	CatalogHandler        *galgameHandler.CatalogHandler
	ImporterHandler       *importerHandler.ImporterHandler
	ContributionHandler   *galgameHandler.ContributionHandler
	NovelHandler          *novelHandler.NovelHandler
	NovelVolumeHandler    *novelHandler.VolumeHandler
	NovelAdminHandler     *novelHandler.AdminHandler
	UserRelationHandler   *galgameHandler.UserRelationHandler
	GalleryHandler        *galgameHandler.GalleryHandler
	ResourceHandler       *resourceHandler.ResourceHandler
	ReportHandler         *resourceHandler.ReportHandler
	FeedbackHandler       *feedbackHandler.FeedbackHandler
	ClassificationHandler *classificationHandler.ClassificationHandler
	PostHandler           *communityHandler.PostHandler
	CommentHandler        *communityHandler.CommentHandler
	InteractionHandler    *communityHandler.InteractionHandler
	BannerHandler         *bannerHandler.BannerHandler
	BackgroundHandler     *backgroundHandler.BackgroundPresetHandler
	ArticleHandler        *articleHandler.ArticleHandler
	HomeHandler           *homeHandler.HomeHandler
	HealthHandler         *healthHandler.HealthHandler
	ImageHandler          *imageHandler.ImageHandler
	NotificationHandler   *notificationHandler.NotificationHandler
	stopImageCleanup      func()
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
	contributionRepository := contributionRepo.NewContributionRepository(postgresDB, cfg.R2.PublicURL)
	contributionSvc := contributionService.NewContributionService(
		contributionRepository,
		galgameRepository,
		cfg.R2.PublicURL,
	)
	relationRepository := relationRepo.NewRelationRepository(postgresDB)
	developerRepository := galgameRepo.NewDeveloperRepository(postgresDB)
	tagRepository := galgameRepo.NewTagRepository(postgresDB)
	catalogService := galgameService.NewCatalogService(
		galgameRepository,
		developerRepository,
		tagRepository,
	)
	catalogService.SetContributionService(contributionSvc)
	catalogService.SetNotificationDependencies(rbacSvc, notificationSvc)
	catalogService.SetRelationRepository(relationRepository)

	novelRepository := novelRepo.NewNovelRepository(postgresDB)
	volumeRepository := novelRepo.NewVolumeRepository(postgresDB)
	novelSvc := novelService.NewNovelService(
		novelRepository,
		volumeRepository,
		tagRepository,
		galgameRepository,
		relationRepository,
	)
	novelSvc.SetContributionService(contributionSvc)
	novelSvc.SetNotificationDependencies(rbacSvc, notificationSvc)
	novelVolumeSvc := novelService.NewVolumeService(volumeRepository, novelRepository)
	novelVolumeSvc.SetContributionService(contributionSvc)
	novelVolumeSvc.SetNotificationDependencies(rbacSvc, notificationSvc)
	novelRelationSvc := novelService.NewRelationService(relationRepository, novelRepository, galgameRepository)
	novelRelationSvc.SetContributionService(contributionSvc)

	userRelationRepository := galgameRepo.NewUserRelationRepository(postgresDB)
	ratingService := galgameService.NewRatingService(galgameRepository, userRelationRepository)
	favoriteService := galgameService.NewFavoriteService(galgameRepository, userRelationRepository)
	userStateService := galgameService.NewUserStateService(galgameRepository, userRelationRepository)
	userRelationService := galgameService.NewUserRelationService(galgameRepository, userRelationRepository)
	galleryRepository := galgameRepo.NewGalleryRepository(postgresDB)

	resourceRepository := resourceRepo.NewResourceRepository(postgresDB, cfg.R2.PublicURL)
	resourceSvc := resourceService.NewResourceService(resourceRepository, galgameRepository, novelRepository, rbacSvc)
	resourceSvc.SetContributionService(contributionSvc)
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
	galleryService.SetContributionService(contributionSvc)

	importerRepository := importerRepo.NewRepository(postgresDB)
	vndbClient := &http.Client{Timeout: 15 * time.Second}
	importerSvc := importerService.NewService(
		importerRepository,
		map[string]provider.Provider{
			"vndb":    provider.NewVNDBProvider(vndbClient),
			"bangumi": provider.NewBangumiProvider(vndbClient, os.Getenv("BANGUMI_API_BASE_URL"), os.Getenv("BANGUMI_API_TOKEN")),
		},
		contributionSvc,
	)
	importQueueClient := queue.NewImportClient(cfg.Redis)
	importerSvc.SetBatchEnqueuer(importQueueClient.EnqueueVNDBBatch)
	importerSvc.SetEnrichEnqueuer(importQueueClient.EnqueueBangumiEnrich)

	// Age rating classification module: Eino agent + tools stay read-only; the
	// worker runs inside this process because it needs PostgreSQL and the LLM
	// configuration, mirroring the import worker.
	classificationRepository := classificationRepo.NewRepository(postgresDB)
	classificationQueueClient := queue.NewClassificationClient(cfg.Redis)
	ratingAgent, agentErr := classificationAgent.New(
		context.Background(),
		cfg.Classification,
		tools.NewCache(redisClient),
		vndbClient,
	)
	if agentErr != nil && !errors.Is(agentErr, classificationAgent.ErrAgentDisabled) {
		if sqlDB, dbErr := postgresDB.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, fmt.Errorf("init classification agent: %w", agentErr)
	}
	classificationSvc := classificationService.NewService(
		classificationRepository,
		galgameRepository,
		ratingAgent,
		classificationQueueClient,
		cfg.Classification,
	)

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
	userProfileRepository := userRepo.NewUserProfileRepository(postgresDB, cfg.R2.PublicURL)
	userActivityService := userService.NewUserActivityService(userProfileRepository)
	userProfileService := userService.NewUserProfileService(
		userAuthRepository,
		userPreferenceRepository,
		imageSvc,
		userProfileRepository,
	)
	postService.SetActivityRecorder(userActivityService)
	commentService.SetActivityRecorder(userActivityService)
	ratingService.SetActivityRecorder(userActivityService)
	favoriteService.SetActivityRecorder(userActivityService)
	resourceSvc.SetActivityRecorder(userActivityService)
	catalogService.SetActivityRecorder(userActivityService)
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
		ImporterHandler:     importerHandler.NewImporterHandler(importerSvc),
		ContributionHandler: galgameHandler.NewContributionHandler(contributionSvc),
		NovelHandler:        novelHandler.NewNovelHandler(novelSvc, novelRelationSvc),
		NovelVolumeHandler:  novelHandler.NewVolumeHandler(novelVolumeSvc),
		NovelAdminHandler:   novelHandler.NewAdminHandler(novelSvc, novelVolumeSvc),
		UserRelationHandler: galgameHandler.NewUserRelationHandler(
			ratingService,
			favoriteService,
			userStateService,
			userRelationService,
		),
		GalleryHandler:        galgameHandler.NewGalleryHandler(galleryService),
		ResourceHandler:       resourceHandler.NewResourceHandler(resourceSvc),
		ReportHandler:         resourceHandler.NewReportHandler(reportSvc),
		FeedbackHandler:       feedbackHandler.NewFeedbackHandler(feedbackSvc),
		ClassificationHandler: classificationHandler.NewClassificationHandler(classificationSvc),
		PostHandler:           communityHandler.NewPostHandler(postService),
		CommentHandler:        communityHandler.NewCommentHandler(commentService),
		InteractionHandler:    communityHandler.NewInteractionHandler(interactionService),
		BannerHandler:         bannerHandler.NewBannerHandler(bannerSvc),
		BackgroundHandler:     backgroundHandler.NewBackgroundPresetHandler(backgroundPresetSvc),
		ArticleHandler:        articleHandler.NewArticleHandler(articleSvc),
		HomeHandler:           homeHandler.NewHomeHandler(homeSvc),
		HealthHandler:         healthHandler.NewHealthHandler(healthService),
		ImageHandler:          imageHandler.NewImageHandler(imageSvc),
		NotificationHandler:   notificationHandler.NewNotificationHandler(notificationSvc),
		stopImageCleanup:      stopImageCleanup,
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

	importWorker := queue.NewImportServer(workerCfg.Redis, 1)
	importMux := asynq.NewServeMux()
	queue.RegisterImportTasks(importMux, importerSvc)
	if err := importWorker.Start(importMux); err != nil {
		app.Close()
		return nil, fmt.Errorf("start import worker: %w", err)
	}
	app.ImportWorker = importWorker
	app.ImportQueue = importQueueClient
	app.ClassificationQueue = classificationQueueClient

	if cfg.Classification.Enabled && classificationSvc.Enabled() {
		classificationWorker := queue.NewClassificationServer(
			cfg.Redis, cfg.Classification.QueueConcurrency,
		)
		classificationMux := asynq.NewServeMux()
		queue.RegisterClassificationTasks(classificationMux, classificationSvc)
		if err := classificationWorker.Start(classificationMux); err != nil {
			app.Close()
			return nil, fmt.Errorf("start classification worker: %w", err)
		}
		app.ClassificationWorker = classificationWorker
	}

	return app, nil
}

func (app *App) Close() {
	if app.stopImageCleanup != nil {
		app.stopImageCleanup()
	}
	if app.ImportWorker != nil {
		app.ImportWorker.Shutdown()
	}
	if app.ClassificationWorker != nil {
		app.ClassificationWorker.Shutdown()
	}
	if app.MailWorker != nil {
		app.MailWorker.Shutdown()
	}
	if app.Queue != nil {
		_ = app.Queue.Close()
	}
	if app.ImportQueue != nil {
		_ = app.ImportQueue.Close()
	}
	if app.ClassificationQueue != nil {
		_ = app.ClassificationQueue.Close()
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
