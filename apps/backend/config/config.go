package config

import (
	"errors"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func LoadEnv() error {
	environment := strings.TrimSpace(os.Getenv("APP_ENV"))
	if environment == "" {
		environment = "development"
	}

	err := godotenv.Load(".env." + environment)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
