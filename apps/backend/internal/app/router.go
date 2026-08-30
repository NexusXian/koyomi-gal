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

	protected := v1.Group("", middleware.Auth(app.Config.Auth.AccessTokenSecret))
	requirePermission := middleware.RequirePermission(app.RBACService)

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
