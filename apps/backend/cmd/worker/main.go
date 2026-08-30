package main

import (
	"os"

	"backend/config"
	"backend/internal/infrastructures/database"
	mailInfrastructure "backend/internal/infrastructures/mail"
	"backend/internal/infrastructures/queue"
	notificationService "backend/internal/notification/service"
	userRepository "backend/internal/user/repository"
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

	cfg, err := config.LoadWorker()
	if err != nil {
		panic(err)
	}
	redisClient, err := database.NewRedis(cfg.Redis)
	if err != nil {
		panic(err)
	}
	defer redisClient.Close()

	mailer := mailInfrastructure.NewSMTPMailer(cfg.SMTP)
	emailService := notificationService.NewEmailService(mailer)
	verificationRepository := userRepository.NewVerificationRepository(redisClient)
	server := queue.NewServer(cfg.Redis, cfg.Concurrency)

	logger.Info("mail worker started")
	if err := server.Run(queue.NewServeMux(
		emailService,
		verificationRepository,
		cfg.VerificationSecret,
	)); err != nil {
		panic(err)
	}
}
