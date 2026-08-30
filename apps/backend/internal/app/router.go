package app

import "backend/internal/middleware"

func (app *App) setupRoutes() {
	app.Gin.Use(middleware.CORS())
    app.Gin.GET("/health",app.HealthHandler.HealthCheck)

	app.Gin.Group("/api")
	{
		app.Gin.Group("/v1")
        {

        }
	}
	// set up v1 routes

}
