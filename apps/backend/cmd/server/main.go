package main

import (
	"os"

	"backend/config"
	"backend/internal/app"
	"backend/pkg/logger"
)

func main() {
	if err := config.LoadEnv(); err != nil {
		panic(err)
	}
	if err := logger.Init(os.Getenv("APP_ENV")); err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	application, err := app.New(cfg)
	if err != nil {
		panic(err)
	}
	defer application.Close()

	logger.Info("application started")
}
