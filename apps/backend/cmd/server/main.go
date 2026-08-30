package main

import (
	"fmt"
	"os"

	"backend/config"
	"backend/internal/app"
	"backend/pkg/logger"
)

//	@title			Koyomi Gal API
//	@version		1.0.0
//	@description	Koyomi Gal 后端 API
//	@host			localhost:8080
//	@BasePath		/

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
	workerCfg, err := config.LoadWorker()
	if err != nil {
		panic(err)
	}

	application, err := app.New(cfg, workerCfg)
	if err != nil {
		panic(err)
	}
	defer application.Close()

	logger.Info("application started")
	if err := application.Gin.Run(fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)); err != nil {
		panic(err)
	}
}
