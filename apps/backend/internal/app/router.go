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
}
