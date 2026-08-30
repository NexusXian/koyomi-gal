package service

import (
	"backend/internal/user/repository"
)

type UserAuthService struct {
	authRepo *repository.UserAuthRepository
}

func NewUserAuthService(authRepo *repository.UserAuthRepository) *UserAuthService {
	return &UserAuthService{authRepo: authRepo}
}
