package app

import (
	"fmt"
	"os"

	"backend/config"
	"backend/pkg/logger"
)

func Init() error {
	if err := config.LoadEnv(); err != nil {
		return fmt.Errorf("load environment: %w", err)
	}

	return logger.Init(os.Getenv("APP_ENV"))
}
