package service

import (
	"backend/internal/user/dto"
	"backend/internal/user/model"
	"backend/internal/user/repository"
	"backend/pkg/bcrypt"
	"backend/pkg/logger"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrAccountBanned       = errors.New("account banned")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

const (
	accessTokenIssuer         = "koyomi-gal"
	maxTokenGenerationRetries = 3
)

type accessTokenClaims struct {
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

type UserAuthService struct {
	authRepo            *repository.UserAuthRepository
	refreshSessionRepo  *repository.RefreshSessionRepository
	verificationService *VerificationService
	accessTokenSecret   []byte
	accessTokenTTL      time.Duration
	refreshTokenTTL     time.Duration
}

func NewUserAuthService(
	authRepo *repository.UserAuthRepository,
	refreshSessionRepo *repository.RefreshSessionRepository,
	verificationService *VerificationService,
	accessTokenSecret string,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
) *UserAuthService {
	return &UserAuthService{
		authRepo:            authRepo,
		refreshSessionRepo:  refreshSessionRepo,
		verificationService: verificationService,
		accessTokenSecret:   []byte(accessTokenSecret),
		accessTokenTTL:      accessTokenTTL,
		refreshTokenTTL:     refreshTokenTTL,
	}
}

func (s *UserAuthService) UserRegister(
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

	validCode, err := s.verificationService.VerifyCode(
		ctx,
		req.Email,
		VerificationPurposeRegister,
		req.VerificationCode,
	)
	if err != nil {
		logger.Error(
			"failed to verify registration code",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		return err
	}
	if !validCode {
		return errors.New("验证码错误或已过期")
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

func (s *UserAuthService) UserLogin(
	ctx context.Context,
	req *dto.UserLoginRequest,
) (*dto.AuthSession, string, error) {
	user, err := s.authRepo.FindUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, "", fmt.Errorf("find login user: %w", err)
	}
	if user == nil || bcrypt.ComparePassword(user.PasswordHash, req.Password) != nil {
		return nil, "", ErrInvalidCredentials
	}
	if user.IsBanned {
		return nil, "", ErrAccountBanned
	}

	accessToken, err := s.issueAccessToken(user.ID)
	if err != nil {
		return nil, "", err
	}
	refreshToken, err := s.createRefreshSession(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}

	return newAuthSession(user, accessToken), refreshToken, nil
}

func (s *UserAuthService) RefreshSession(
	ctx context.Context,
	refreshToken string,
) (*dto.AuthSession, string, error) {
	if refreshToken == "" {
		return nil, "", ErrInvalidRefreshToken
	}

	userID, err := s.refreshSessionRepo.FindUserID(ctx, refreshToken)
	if errors.Is(err, repository.ErrRefreshSessionNotFound) {
		return nil, "", ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, "", err
	}

	user, err := s.authRepo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("find refresh session user: %w", err)
	}
	if user == nil {
		if err := s.refreshSessionRepo.Revoke(ctx, refreshToken); err != nil {
			return nil, "", err
		}
		return nil, "", ErrInvalidRefreshToken
	}
	if user.IsBanned {
		if err := s.refreshSessionRepo.Revoke(ctx, refreshToken); err != nil {
			return nil, "", err
		}
		return nil, "", ErrAccountBanned
	}

	accessToken, err := s.issueAccessToken(user.ID)
	if err != nil {
		return nil, "", err
	}
	replacementToken, err := s.rotateRefreshSession(ctx, refreshToken, user.ID)
	if err != nil {
		return nil, "", err
	}

	return newAuthSession(user, accessToken), replacementToken, nil
}

func (s *UserAuthService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return s.refreshSessionRepo.Revoke(ctx, refreshToken)
}

func (s *UserAuthService) issueAccessToken(userID uint) (string, error) {
	now := time.Now()
	tokenID, err := randomToken(16)
	if err != nil {
		return "", fmt.Errorf("generate access token id: %w", err)
	}

	claims := accessTokenClaims{
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    accessTokenIssuer,
			Subject:   strconv.FormatUint(uint64(userID), 10),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        tokenID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.accessTokenSecret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signedToken, nil
}

func (s *UserAuthService) createRefreshSession(ctx context.Context, userID uint) (string, error) {
	for range maxTokenGenerationRetries {
		token, err := randomToken(32)
		if err != nil {
			return "", fmt.Errorf("generate refresh token: %w", err)
		}
		err = s.refreshSessionRepo.Create(ctx, token, userID, s.refreshTokenTTL)
		if errors.Is(err, repository.ErrRefreshTokenCollision) {
			continue
		}
		if err != nil {
			return "", err
		}
		return token, nil
	}
	return "", errors.New("create unique refresh token: retry limit reached")
}

func (s *UserAuthService) rotateRefreshSession(
	ctx context.Context,
	currentToken string,
	userID uint,
) (string, error) {
	for range maxTokenGenerationRetries {
		replacementToken, err := randomToken(32)
		if err != nil {
			return "", fmt.Errorf("generate replacement refresh token: %w", err)
		}
		err = s.refreshSessionRepo.Rotate(
			ctx,
			currentToken,
			replacementToken,
			userID,
			s.refreshTokenTTL,
		)
		switch {
		case errors.Is(err, repository.ErrRefreshSessionNotFound):
			return "", ErrInvalidRefreshToken
		case errors.Is(err, repository.ErrRefreshTokenCollision):
			continue
		case err != nil:
			return "", err
		default:
			return replacementToken, nil
		}
	}
	return "", errors.New("rotate unique refresh token: retry limit reached")
}

func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func newAuthSession(user *model.User, accessToken string) *dto.AuthSession {
	return &dto.AuthSession{
		Token: accessToken,
		User: dto.AuthUser{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Avatar:   user.Avatar,
		},
	}
}
