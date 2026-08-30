package app

import "backend/internal/middleware"

func (app *App) setupRoutes() {
	app.Gin.Use(middleware.CORS())
	app.Gin.GET("/health", app.HealthHandler.HealthCheck)

	api := app.Gin.Group("/api")
	v1 := api.Group("/v1")
	auth := v1.Group("/auth")
	auth.POST("/verification-codes", app.VerificationHandler.SendCode)
}
