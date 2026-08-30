package main

import (
	"backend/config"
	"backend/internal/app"
	"backend/internal/infrastructures/database"
	"backend/pkg/logger"
)

func main() {
	if err := app.Init(); err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	db, err := database.NewPostgre(cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}

	_ = db

	logger.Info("application started")
}
