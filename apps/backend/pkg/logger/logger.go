package logger

import (
	"strings"

	"go.uber.org/zap"
)

func New(environment string) (*zap.Logger, error) {
	if strings.EqualFold(strings.TrimSpace(environment), "production") {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}

func Init(environment string) error {
	log, err := New(environment)
	if err != nil {
		return err
	}

	zap.ReplaceGlobals(log.WithOptions(zap.AddCallerSkip(1)))
	return nil
}

func Sync() error {
	return zap.L().Sync()
}

func Debug(message string, fields ...zap.Field) {
	zap.L().Debug(message, fields...)
}

func Info(message string, fields ...zap.Field) {
	zap.L().Info(message, fields...)
}

func Warn(message string, fields ...zap.Field) {
	zap.L().Warn(message, fields...)
}

func Error(message string, fields ...zap.Field) {
	zap.L().Error(message, fields...)
}
