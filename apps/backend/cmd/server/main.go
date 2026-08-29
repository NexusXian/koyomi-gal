package main

import (
	"backend/internal/app"
	"backend/pkg/logger"
)

func main() {
	if err := app.Init(); err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	logger.Info("application started")
}
