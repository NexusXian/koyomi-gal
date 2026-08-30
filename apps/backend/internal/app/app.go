package app

import (
	"backend/config"
	"backend/internal/infrastructures/database"
	userHandler "backend/internal/user/handler"
	userRepo "backend/internal/user/repository"
	userService "backend/internal/user/service"

	healthHandler "backend/internal/health/handler"
	healthService "backend/internal/health/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	Config          *config.Config
	Gin             *gin.Engine
	Postgres        *gorm.DB
	Redis           *redis.Client
	UserAuthHandler *userHandler.UserAuthHandler
    HealthHandler   *healthHandler.HealthHandler
}

func New(cfg *config.Config) (*App, error) {
	postgresDB, err := database.NewPostgre(cfg.Postgres)
	if err != nil {
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
	userAuthService := userService.NewUserAuthService(userAuthRepository)


	app := &App{
		Config:          cfg,
		Postgres:        postgresDB,
		Redis:           redisClient,

		UserAuthHandler: userHandler.NewUserAuthHandler(userAuthService),
        HealthHandler:   healthHandler.NewHealthHandler(healthService),
	}
	ginApp := gin.Default()
	app.Gin = ginApp
	app.setupRoutes()
	return app, nil
}

func (app *App) Close() {
	if app.Redis != nil {
		_ = app.Redis.Close()
	}
	if app.Postgres != nil {
		if sqlDB, err := app.Postgres.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
}
