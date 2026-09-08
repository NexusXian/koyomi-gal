package service

import (
	imageService "backend/internal/image/service"
	dto "backend/internal/user/dto"
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
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

var (
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrAccountBanned           = errors.New("account banned")
	ErrInvalidRefreshToken     = errors.New("invalid refresh token")
	ErrInvalidUsername         = errors.New("invalid username")
	ErrUsernameExists          = errors.New("username already exists")
	ErrEmailExists             = errors.New("email already exists")
	ErrPasswordMismatch        = errors.New("password confirmation does not match")
	ErrInvalidVerificationCode = errors.New("verification code is invalid or expired")
	ErrInvalidResetToken       = errors.New("password reset token is invalid or expired")
	ErrSamePassword            = errors.New("new password matches current password")
	ErrUserNotFound            = errors.New("user not found")
)

const (
	accessTokenIssuer         = "koyomi-gal"
	maxTokenGenerationRetries = 3
	defaultRegisterRoleCode   = "user"
)

// RoleAssigner binds a role code to a user; implemented by the RBAC service.
type RoleAssigner interface {
	AssignRoleByCode(ctx context.Context, userID uint, roleCode string) error
}

type accessTokenClaims struct {
	TokenType   string `json:"token_type"`
	AuthVersion uint64 `json:"auth_version"`
	jwt.RegisteredClaims
}

type UserAuthService struct {
	authRepo            *repository.UserAuthRepository
	refreshSessionRepo  *repository.RefreshSessionRepository
	verificationService *VerificationService
	roleAssigner        RoleAssigner
	images              *imageService.ImageAssetService
	accessTokenSecret   []byte
	accessTokenTTL      time.Duration
	refreshTokenTTL     time.Duration
}

func NewUserAuthService(
	authRepo *repository.UserAuthRepository,
	refreshSessionRepo *repository.RefreshSessionRepository,
	verificationService *VerificationService,
	roleAssigner RoleAssigner,
	images *imageService.ImageAssetService,
	accessTokenSecret string,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
) *UserAuthService {
	return &UserAuthService{
		authRepo:            authRepo,
		refreshSessionRepo:  refreshSessionRepo,
		verificationService: verificationService,
		roleAssigner:        roleAssigner,
		images:              images,
		accessTokenSecret:   []byte(accessTokenSecret),
		accessTokenTTL:      accessTokenTTL,
		refreshTokenTTL:     refreshTokenTTL,
	}
}

func (s *UserAuthService) UserRegister(
	ctx context.Context,
	req *dto.UserRegisterRequest,
) error {
	username := strings.TrimSpace(req.Username)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if username == "" {
		return ErrInvalidUsername
	}

	user, err := s.authRepo.FindUserByUsername(ctx, username)
	if err != nil {
		logger.Error(
			"failed to find user by username",
			zap.String("username", username),
			zap.Error(err),
		)
		return err
	}

	if user != nil {
		return ErrUsernameExists
	}

	user, err = s.authRepo.FindUserByEmail(ctx, email)
	if err != nil {
		logger.Error(
			"failed to find user by email",
			zap.String("email", email),
			zap.Error(err),
		)
		return err
	}

	if user != nil {
		return ErrEmailExists
	}

	if err := ValidatePassword(req.Password); err != nil {
		return err
	}

	if req.Password != req.ConfirmPassword {
		return ErrPasswordMismatch
	}

	validCode, err := s.verificationService.ConsumeCode(
		ctx,
		email,
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
		return ErrInvalidVerificationCode
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
		Username:     username,
		PasswordHash: hashedPassword,
		Email:        email,
		IsBanned:     false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.authRepo.CreateUser(ctx, newUser); err != nil {
		if conflict := userConflictError(err); conflict != nil {
			return conflict
		}
		logger.Error(
			"failed to create user",
			zap.String("username", username),
			zap.String("email", email),
			zap.Error(err),
		)
		return err
	}

	logger.Info(
		"user registered successfully",
		zap.String("username", newUser.Username),
		zap.String("email", newUser.Email),
	)

	if s.roleAssigner != nil {
		if err := s.roleAssigner.AssignRoleByCode(ctx, newUser.ID, defaultRegisterRoleCode); err != nil {
			logger.Warn(
				"assign default register role",
				zap.Uint("user_id", newUser.ID),
				zap.String("role_code", defaultRegisterRoleCode),
				zap.Error(err),
			)
		}
	}

	return nil
}

func (s *UserAuthService) UserLogin(
	ctx context.Context,
	req *dto.UserLoginRequest,
) (*dto.AuthSession, string, error) {
	account := strings.TrimSpace(req.Account)
	user, err := s.authRepo.FindUserByEmail(ctx, strings.ToLower(account))
	if err != nil {
		return nil, "", fmt.Errorf("find login user: %w", err)
	}
	if user == nil {
		user, err = s.authRepo.FindUserByUsername(ctx, account)
		if err != nil {
			return nil, "", fmt.Errorf("find login user: %w", err)
		}
	}
	if user == nil || bcrypt.ComparePassword(user.PasswordHash, req.Password) != nil {
		return nil, "", ErrInvalidCredentials
	}
	if user.IsBanned {
		return nil, "", ErrAccountBanned
	}

	accessToken, err := s.issueAccessToken(user.ID, user.AuthVersion)
	if err != nil {
		return nil, "", err
	}
	refreshToken, err := s.createRefreshSession(ctx, user.ID, user.AuthVersion)
	if err != nil {
		return nil, "", err
	}

	return s.newAuthSession(ctx, user, accessToken), refreshToken, nil
}

func (s *UserAuthService) RefreshSession(
	ctx context.Context,
	refreshToken string,
) (*dto.AuthSession, string, error) {
	if refreshToken == "" {
		return nil, "", ErrInvalidRefreshToken
	}

	refreshSession, err := s.refreshSessionRepo.Find(ctx, refreshToken)
	if errors.Is(err, repository.ErrRefreshSessionNotFound) {
		return nil, "", ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, "", err
	}

	user, err := s.authRepo.FindUserByID(ctx, refreshSession.UserID)
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
	if user.AuthVersion != refreshSession.AuthVersion {
		if err := s.refreshSessionRepo.Revoke(ctx, refreshToken); err != nil {
			return nil, "", err
		}
		return nil, "", ErrInvalidRefreshToken
	}

	accessToken, err := s.issueAccessToken(user.ID, user.AuthVersion)
	if err != nil {
		return nil, "", err
	}
	replacementToken, err := s.rotateRefreshSession(ctx, refreshToken, refreshSession)
	if err != nil {
		return nil, "", err
	}

	return s.newAuthSession(ctx, user, accessToken), replacementToken, nil
}

func (s *UserAuthService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return s.refreshSessionRepo.Revoke(ctx, refreshToken)
}

func (s *UserAuthService) issueAccessToken(userID uint, authVersion uint64) (string, error) {
	now := time.Now()
	tokenID, err := randomToken(16)
	if err != nil {
		return "", fmt.Errorf("generate access token id: %w", err)
	}

	claims := accessTokenClaims{
		TokenType:   "access",
		AuthVersion: authVersion,
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

func (s *UserAuthService) createRefreshSession(ctx context.Context, userID uint, authVersion uint64) (string, error) {
	for range maxTokenGenerationRetries {
		token, err := randomToken(32)
		if err != nil {
			return "", fmt.Errorf("generate refresh token: %w", err)
		}
		err = s.refreshSessionRepo.Create(ctx, token, repository.RefreshSession{
			UserID:      userID,
			AuthVersion: authVersion,
		}, s.refreshTokenTTL)
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
	session repository.RefreshSession,
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
			session,
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

func (s *UserAuthService) newAuthSession(
	ctx context.Context,
	user *model.User,
	accessToken string,
) *dto.AuthSession {
	return &dto.AuthSession{
		Token: accessToken,
		User: dto.AuthUser{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Avatar:   resolveAvatarURL(ctx, s.images, user),
		},
	}
}
