package app

import (
	"backend/config"
	"backend/internal/infrastructures/database"
	userHandler "backend/internal/user/handler"
	userRepo "backend/internal/user/repository"
	userService "backend/internal/user/service"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	Config          *config.Config
	Postgres        *gorm.DB
	Redis           *redis.Client
	UserAuthHandler *userHandler.UserAuthHandler
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

	//Init the repository
	userAuthRepository := userRepo.NewUserAuthRepository(postgresDB)
	userAuthService := userService.NewUserAuthService(userAuthRepository)

	return &App{
		Config:          cfg,
		Postgres:        postgresDB,
		Redis:           redisClient,
		UserAuthHandler: userHandler.NewUserAuthHandler(userAuthService),
	}, nil
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
