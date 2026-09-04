package app

import (
	_ "backend/docs"
	"backend/internal/middleware"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (app *App) setupRoutes() {
	app.Gin.Use(middleware.CORS(app.Config.Server.AllowedOrigins))
	app.Gin.GET("/health", app.HealthHandler.HealthCheck)
	app.Gin.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := app.Gin.Group("/api")
	v1 := api.Group("/v1")
	v2 := api.Group("/v2")
	auth := v1.Group("/auth")
	{
		auth.POST("/register", app.UserAuthHandler.Register)
		auth.POST("/login", app.UserAuthHandler.Login)
		auth.POST("/refresh", app.UserAuthHandler.Refresh)
		auth.POST("/logout", app.UserAuthHandler.Logout)
		auth.POST("/verification-codes", app.VerificationHandler.SendCode)
	}

	v1.GET("/galgames", app.CatalogHandler.ListGalgames)
	v1.GET("/galgames/:id", app.CatalogHandler.GetGalgame)
	v1.GET("/galgames/:id/contributors", app.ContributionHandler.ListGalgameContributors)
	v1.GET("/galgames/:id/gallery", app.GalleryHandler.ListGalgameGallery)
	v1.GET("/resources/:id", app.ResourceHandler.GetResource)
	v1.GET("/posts", app.PostHandler.ListPosts)
	v1.GET("/posts/:id", app.PostHandler.GetPost)
	v1.GET("/banners", app.BannerHandler.ListBanners)
	v1.GET("/background-presets", app.BackgroundHandler.ListBackgroundPresets)
	v1.POST("/feedback", app.FeedbackHandler.CreateFeedback)
	v1.GET("/images/:id", app.ImageHandler.GetImage)
	v1.GET("/articles", app.ArticleHandler.ListArticles)
	v1.GET("/articles/:id", app.ArticleHandler.GetArticle)
	v1.GET("/home", app.HomeHandler.GetHome)
	v1.GET("/developers", app.CatalogHandler.ListDevelopers)
	v1.GET("/developers/:id", app.CatalogHandler.GetDeveloper)
	v1.GET("/tags", app.CatalogHandler.ListTags)
	v1.GET("/tags/:id", app.CatalogHandler.GetTag)
	v2.GET("/galgames/:id/resources", app.ResourceHandler.ListGalgameResources)
	v2.GET("/posts/:id/comments", app.CommentHandler.ListPostComments)
	v2.GET("/comments/:id/replies", app.CommentHandler.ListCommentReplies)

	publicUsers := v1.Group("/users", middleware.OptionalAuthWithUserChecker(
		app.Config.Auth.AccessTokenSecret,
		app.UserAuthRepository,
	))
	{
		publicUsers.GET("/me", app.UserProfileHandler.GetPublicProfile)
		publicUsers.GET("/:username", app.UserProfileHandler.GetPublicProfile)
		publicUsers.GET("/:username/posts", app.UserProfileHandler.ListUserPosts)
		publicUsers.GET("/:username/comments", app.UserProfileHandler.ListUserComments)
		publicUsers.GET("/:username/ratings", app.UserProfileHandler.ListUserRatings)
		publicUsers.GET("/:username/favorites", app.UserProfileHandler.ListUserFavorites)
		publicUsers.GET("/:username/activities", app.UserProfileHandler.ListUserActivities)
	}

	protected := v1.Group("", middleware.AuthWithUserChecker(
		app.Config.Auth.AccessTokenSecret,
		app.UserAuthRepository,
	))
	requirePermission := middleware.RequirePermission(app.RBACService)

	protected.POST("/galgames", requirePermission("galgame:create"), app.CatalogHandler.CreateGalgame)
	protected.PUT("/galgames/:id", requirePermission("galgame:update"), app.CatalogHandler.UpdateGalgame)
	protected.DELETE("/galgames/:id", requirePermission("galgame:delete"), app.CatalogHandler.DeleteGalgame)
	protected.POST("/developers", requirePermission("galgame:create"), app.CatalogHandler.CreateDeveloper)
	protected.PUT("/developers/:id", requirePermission("galgame:update"), app.CatalogHandler.UpdateDeveloper)
	protected.POST("/tags", requirePermission("galgame:create"), app.CatalogHandler.CreateTag)
	protected.PUT("/tags/:id", requirePermission("galgame:update"), app.CatalogHandler.UpdateTag)

	galgameRelations := protected.Group("/galgames/:id")
	{
		galgameRelations.PUT("/rating", app.UserRelationHandler.UpsertRating)
		galgameRelations.DELETE("/rating", app.UserRelationHandler.DeleteRating)
		galgameRelations.POST("/favorite", app.UserRelationHandler.AddFavorite)
		galgameRelations.DELETE("/favorite", app.UserRelationHandler.RemoveFavorite)
		galgameRelations.PUT("/state", app.UserRelationHandler.UpsertState)
		galgameRelations.DELETE("/state", app.UserRelationHandler.DeleteState)
		galgameRelations.GET("/me", app.UserRelationHandler.GetMyRelation)
	}

	resources := protected.Group("/resources")
	{
		resources.POST("", app.ResourceHandler.CreateResource)
		resources.PUT("/:id", app.ResourceHandler.UpdateResource)
		resources.DELETE("/:id", app.ResourceHandler.DeleteResource)
		resources.POST("/:id/reports", app.ReportHandler.CreateReport)
	}

	admin := protected.Group("/admin")
	{
		admin.GET("/galgames", requirePermission("galgame:review"), app.CatalogHandler.ListAdminGalgames)
		admin.GET("/galgames/:id", requirePermission("galgame:review"), app.CatalogHandler.GetAdminGalgame)
		admin.PUT("/galgames/:id/review", requirePermission("galgame:review"), app.CatalogHandler.ReviewGalgame)
		admin.PATCH("/galgames/batch", requirePermission("galgame:update"), app.CatalogHandler.BatchUpdateGalgames)
		admin.DELETE("/galgames/batch", requirePermission("galgame:delete"), app.CatalogHandler.BatchDeleteGalgames)
		admin.GET("/import/providers", requirePermission("galgame:import"), app.ImporterHandler.ListImportProviders)
		admin.GET("/import/games/search", requirePermission("galgame:import"), app.ImporterHandler.SearchImportGames)
		admin.GET("/import/games/:provider/:external_id", requirePermission("galgame:import"), app.ImporterHandler.PreviewImportGame)
		admin.POST("/import/games", requirePermission("galgame:import"), app.ImporterHandler.ImportGame)
		admin.POST("/import/batches", requirePermission("galgame:import:batch"), app.ImporterHandler.CreateImportBatch)
		admin.GET("/import/batches", requirePermission("galgame:import:batch"), app.ImporterHandler.ListImportBatches)
		admin.GET("/import/batches/:id", requirePermission("galgame:import:batch"), app.ImporterHandler.GetImportBatch)
		admin.GET("/import/enrich/stats", requirePermission("galgame:import"), app.ImporterHandler.GetEnrichStats)
		admin.POST("/import/enrich/batches", requirePermission("galgame:import:batch"), app.ImporterHandler.CreateEnrichBatch)
		admin.GET("/import/galgames/:id/external-candidates", requirePermission("galgame:import"), app.ImporterHandler.ListExternalCandidates)
		admin.POST("/import/galgames/:id/enrich", requirePermission("galgame:import"), app.ImporterHandler.EnrichGalgame)
		admin.GET("/import/matches", requirePermission("galgame:import"), app.ImporterHandler.ListMatchCandidates)
		admin.POST("/import/matches/:id/approve", requirePermission("galgame:import"), app.ImporterHandler.ApproveMatchCandidate)
		admin.POST("/import/matches/:id/reject", requirePermission("galgame:import"), app.ImporterHandler.RejectMatchCandidate)
		admin.POST("/import/matches/batch-approve", requirePermission("galgame:import"), app.ImporterHandler.BatchApproveMatchCandidates)
		admin.POST("/import/matches/batch-reject", requirePermission("galgame:import"), app.ImporterHandler.BatchRejectMatchCandidates)
		admin.GET("/galgames/:id/gallery", requirePermission("galgame_gallery:manage"), app.GalleryHandler.ListAdminGalgameGallery)
		admin.POST("/galgames/:id/gallery", requirePermission("galgame_gallery:manage"), app.GalleryHandler.CreateGalgameGalleryImage)
		admin.POST("/galgames/:id/gallery/batch", requirePermission("galgame_gallery:manage"), app.GalleryHandler.BatchCreateGalgameGalleryImages)
		admin.PUT("/galgames/:id/gallery/order", requirePermission("galgame_gallery:manage"), app.GalleryHandler.ReorderGalgameGallery)
		admin.PATCH("/galgames/:id/gallery/:galleryId", requirePermission("galgame_gallery:manage"), app.GalleryHandler.UpdateGalgameGalleryImage)
		admin.DELETE("/galgames/:id/gallery/:galleryId", requirePermission("galgame_gallery:manage"), app.GalleryHandler.DeleteGalgameGalleryImage)
		admin.GET("/gallery-images", requirePermission("galgame_gallery:review"), app.GalleryHandler.ListGalleryReviews)
		admin.POST("/gallery-images/batch-review", requirePermission("galgame_gallery:review"), app.GalleryHandler.BatchReviewGalleryImages)
		admin.POST("/gallery-images/:id/approve", requirePermission("galgame_gallery:review"), app.GalleryHandler.ApproveGalleryImage)
		admin.POST("/gallery-images/:id/reject", requirePermission("galgame_gallery:review"), app.GalleryHandler.RejectGalleryImage)
		admin.GET("/resource-reports", requirePermission("resource_report:list"), app.ReportHandler.ListReports)
		admin.PUT("/resource-reports/:id/handle", requirePermission("resource_report:handle"), app.ReportHandler.HandleReport)
		admin.GET("/feedback", requirePermission("feedback:read"), app.FeedbackHandler.ListAdminFeedback)
		admin.PUT("/feedback/:id/handle", requirePermission("feedback:handle"), app.FeedbackHandler.HandleFeedback)
		admin.GET("/resources", requirePermission("resource:review"), app.ResourceHandler.ListAdminResources)
		admin.PUT("/resources/:id/review", requirePermission("resource:review"), app.ResourceHandler.ReviewResource)
		admin.GET("/banners", requirePermission("banner:read"), app.BannerHandler.ListAdminBanners)
		admin.GET("/banners/:id", requirePermission("banner:read"), app.BannerHandler.GetAdminBanner)
		admin.POST("/banners", requirePermission("banner:create"), app.BannerHandler.CreateAdminBanner)
		admin.PUT("/banners/:id", requirePermission("banner:update"), app.BannerHandler.UpdateAdminBanner)
		admin.DELETE("/banners/:id", requirePermission("banner:delete"), app.BannerHandler.DeleteAdminBanner)
		admin.GET("/background-presets", requirePermission("background_preset:read"), app.BackgroundHandler.ListAdminBackgroundPresets)
		admin.POST("/background-presets", requirePermission("background_preset:create"), app.BackgroundHandler.CreateAdminBackgroundPreset)
		admin.PUT("/background-presets/:id", requirePermission("background_preset:update"), app.BackgroundHandler.UpdateAdminBackgroundPreset)
		admin.DELETE("/background-presets/:id", requirePermission("background_preset:delete"), app.BackgroundHandler.DeleteAdminBackgroundPreset)
		admin.GET("/articles", requirePermission("article:read"), app.ArticleHandler.ListAdminArticles)
		admin.GET("/articles/:id", requirePermission("article:read"), app.ArticleHandler.GetAdminArticle)
		admin.POST("/articles", requirePermission("article:create"), app.ArticleHandler.CreateAdminArticle)
		admin.PUT("/articles/:id", app.ArticleHandler.UpdateAdminArticle)
		admin.DELETE("/articles/:id", requirePermission("article:delete"), app.ArticleHandler.DeleteAdminArticle)
		admin.GET("/images", requirePermission("image:read"), app.ImageHandler.ListAdminImages)
		admin.GET("/images/:id", requirePermission("image:read"), app.ImageHandler.GetAdminImage)
		admin.DELETE("/images/:id", requirePermission("image:delete"), app.ImageHandler.DeleteAdminImage)
		admin.GET("/posts", requirePermission("post:moderate"), app.PostHandler.ListAdminPosts)
		admin.GET("/comments", requirePermission("comment:moderate"), app.CommentHandler.ListAdminComments)
		admin.GET("/users", requirePermission("user:list"), app.UserAdminHandler.List)
		admin.GET("/users/:id", requirePermission("user:read"), app.UserAdminHandler.Get)
		admin.POST("/users", requirePermission("user:create"), app.UserAdminHandler.Create)
		admin.PUT("/users/:id", requirePermission("user:update"), app.UserAdminHandler.Update)
		admin.DELETE("/users/:id", requirePermission("user:delete"), app.UserAdminHandler.Delete)
		admin.GET("/users/:id/roles", requirePermission("role:list"), app.AssignmentHandler.ListUserRoles)
		admin.PUT("/users/:id/roles", requirePermission("role:assign"), app.AssignmentHandler.UpdateUserRoles)
	}

	posts := protected.Group("/posts")
	{
		posts.POST("", app.PostHandler.CreatePost)
		posts.PUT("/:id", app.PostHandler.UpdatePost)
		posts.DELETE("/:id", app.PostHandler.DeletePost)
		posts.POST("/:id/like", app.InteractionHandler.LikePost)
		posts.DELETE("/:id/like", app.InteractionHandler.UnlikePost)
		posts.POST("/:id/favorite", app.InteractionHandler.FavoritePost)
		posts.DELETE("/:id/favorite", app.InteractionHandler.UnfavoritePost)
		posts.POST("/:id/comments", app.CommentHandler.CreateComment)
	}

	images := protected.Group("/images")
	{
		images.POST("/presign", app.ImageHandler.PresignUpload)
		images.POST("/:id/complete", app.ImageHandler.CompleteUpload)
		images.DELETE("/:id", app.ImageHandler.DeleteImage)
	}

	comments := protected.Group("/comments")
	{
		comments.PUT("/:id", app.CommentHandler.UpdateComment)
		comments.DELETE("/:id", app.CommentHandler.DeleteComment)
		comments.POST("/:id/like", app.InteractionHandler.LikeComment)
		comments.DELETE("/:id/like", app.InteractionHandler.UnlikeComment)
	}

	notifications := protected.Group("/notifications")
	{
		notifications.GET("", app.NotificationHandler.ListNotifications)
		notifications.GET("/unread-count", app.NotificationHandler.UnreadCount)
		notifications.PATCH("/:id/read", app.NotificationHandler.MarkRead)
		notifications.PATCH("/read-all", app.NotificationHandler.MarkAllRead)
	}

	roles := protected.Group("/roles")
	{
		roles.GET("", requirePermission("role:list"), app.RoleHandler.List)
		roles.POST("", requirePermission("role:create"), app.RoleHandler.Create)
		roles.GET("/:id", requirePermission("role:list"), app.RoleHandler.Get)
		roles.PUT("/:id", requirePermission("role:update"), app.RoleHandler.Update)
		roles.DELETE("/:id", requirePermission("role:delete"), app.RoleHandler.Delete)
		roles.GET("/:id/permissions", requirePermission("role:list"), app.RoleHandler.GetPermissions)
		roles.PUT("/:id/permissions", requirePermission("permission:assign"), app.RoleHandler.UpdatePermissions)
	}

	permissions := protected.Group("/permissions")
	{
		permissions.GET("", requirePermission("permission:list"), app.PermissionHandler.List)
		permissions.POST("", requirePermission("permission:create"), app.PermissionHandler.Create)
		permissions.PUT("/:id", requirePermission("permission:update"), app.PermissionHandler.Update)
		permissions.DELETE("/:id", requirePermission("permission:delete"), app.PermissionHandler.Delete)
	}

	protected.GET("/me/permissions", app.AssignmentHandler.MePermissions)
	protected.GET("/me/galgames", app.CatalogHandler.ListMyGalgames)
	protected.PATCH("/me", app.UserProfileHandler.UpdateMe)
	protected.GET("/me/preferences", app.UserProfileHandler.GetPreferences)
	protected.PATCH("/me/preferences", app.UserProfileHandler.UpdatePreferences)
	protected.GET("/users/me/profile", app.UserProfileHandler.GetMyProfile)
	protected.PATCH("/users/me/profile", app.UserProfileHandler.UpdateProfile)
	protected.GET("/users/me/privacy", app.UserProfileHandler.GetPrivacy)
	protected.PATCH("/users/me/privacy", app.UserProfileHandler.UpdatePrivacy)
}
