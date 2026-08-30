package service

import (
	"backend/internal/user/dto"
	"backend/internal/user/model"
	"backend/internal/user/repository"
	"backend/pkg/bcrypt"
	"backend/pkg/logger"
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
)

type UserAuthService struct {
	authRepo *repository.UserAuthRepository
}

func NewUserAuthService(authRepo *repository.UserAuthRepository) *UserAuthService {
	return &UserAuthService{
		authRepo: authRepo,
	}
}

func (s *UserAuthService) Create(
	ctx context.Context,
	req *dto.UserRegisterRequest,
) error {
	user, err := s.authRepo.FindUserByUsername(ctx, req.Username)
	if err != nil {
		logger.Error(
			"failed to find user by username",
			zap.String("username", req.Username),
			zap.Error(err),
		)
		return err
	}

	if user != nil {
		return errors.New("用户名已存在")
	}

	user, err = s.authRepo.FindUserByEmail(ctx, req.Email)
	if err != nil {
		logger.Error(
			"failed to find user by email",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		return err
	}

	if user != nil {
		return errors.New("邮箱已存在")
	}

	if req.Password != req.ConfirmPassword {
		return errors.New("两次输入的密码不一致")
	}

	hashedPassword, err := bcrypt.HashPassword(req.Password)
	if err != nil {
		logger.Error(
			"failed to hash password",
			zap.Error(err),
		)
		return err
	}

	now := time.Now()

	newUser := &model.User{
		Username:     req.Username,
		PasswordHash: hashedPassword,
		Email:        req.Email,
		IsBanned:     false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.authRepo.CreateUser(ctx, newUser); err != nil {
		logger.Error(
			"failed to create user",
			zap.String("username", req.Username),
			zap.String("email", req.Email),
			zap.Error(err),
		)
		return err
	}

	logger.Info(
		"user registered successfully",
		zap.String("username", newUser.Username),
		zap.String("email", newUser.Email),
	)

	return nil
}
