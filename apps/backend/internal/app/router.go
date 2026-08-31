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
	v1.GET("/galgames/:id/resources", app.ResourceHandler.ListGalgameResources)
	v1.GET("/resources/:id", app.ResourceHandler.GetResource)
	v1.GET("/posts", app.PostHandler.ListPosts)
	v1.GET("/posts/:id", app.PostHandler.GetPost)
	v1.GET("/posts/:id/comments", app.CommentHandler.ListPostComments)
	v1.GET("/developers", app.CatalogHandler.ListDevelopers)
	v1.GET("/developers/:id", app.CatalogHandler.GetDeveloper)
	v1.GET("/tags", app.CatalogHandler.ListTags)
	v1.GET("/tags/:id", app.CatalogHandler.GetTag)

	protected := v1.Group("", middleware.Auth(app.Config.Auth.AccessTokenSecret))
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
		admin.GET("/resource-reports", requirePermission("resource_report:list"), app.ReportHandler.ListReports)
		admin.PUT("/resource-reports/:id/handle", requirePermission("resource_report:handle"), app.ReportHandler.HandleReport)
		admin.GET("/resources", requirePermission("resource:review"), app.ResourceHandler.ListAdminResources)
		admin.PUT("/resources/:id/review", requirePermission("resource:review"), app.ResourceHandler.ReviewResource)
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

	comments := protected.Group("/comments")
	{
		comments.PUT("/:id", app.CommentHandler.UpdateComment)
		comments.DELETE("/:id", app.CommentHandler.DeleteComment)
		comments.POST("/:id/like", app.InteractionHandler.LikeComment)
		comments.DELETE("/:id/like", app.InteractionHandler.UnlikeComment)
	}

	roles := protected.Group("/roles")
	{
		roles.GET("", requirePermission("role:list"), app.RoleHandler.List)
		roles.POST("", requirePermission("role:create"), app.RoleHandler.Create)
		roles.GET("/:id", requirePermission("role:list"), app.RoleHandler.Get)
		roles.PUT("/:id", requirePermission("role:update"), app.RoleHandler.Update)
		roles.DELETE("/:id", requirePermission("role:delete"), app.RoleHandler.Delete)
		roles.GET("/:id/permissions", requirePermission("role:list"), app.RoleHandler.GetPermissions)
		roles.PUT("/:id/permissions", requirePermission("role:assign"), app.RoleHandler.UpdatePermissions)
	}

	permissions := protected.Group("/permissions")
	{
		permissions.GET("", requirePermission("permission:list"), app.PermissionHandler.List)
		permissions.POST("", requirePermission("permission:create"), app.PermissionHandler.Create)
		permissions.PUT("/:id", requirePermission("permission:update"), app.PermissionHandler.Update)
		permissions.DELETE("/:id", requirePermission("permission:delete"), app.PermissionHandler.Delete)
	}

	users := protected.Group("/users")
	{
		users.GET("/:id/roles", requirePermission("role:list"), app.AssignmentHandler.ListUserRoles)
		users.PUT("/:id/roles", requirePermission("role:assign"), app.AssignmentHandler.UpdateUserRoles)
	}

	protected.GET("/me/permissions", app.AssignmentHandler.MePermissions)
}
