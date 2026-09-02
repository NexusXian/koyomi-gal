package service

import (
	"context"
	"errors"
	"strings"
	"time"

	rbacService "backend/internal/rbac/service"
	"backend/internal/user/dto"
	"backend/internal/user/model"
	"backend/internal/user/repository"
	"backend/pkg/bcrypt"
	"backend/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrAdminUserNotFound       = errors.New("user not found")
	ErrInvalidEmail            = errors.New("invalid email")
	ErrInvalidPassword         = errors.New("invalid password")
	ErrSelfDelete              = errors.New("cannot delete current user")
	ErrSelfBan                 = errors.New("cannot ban current user")
	ErrSuperAdminUserProtected = errors.New("super admin user requires super admin")
	ErrLastSuperAdmin          = errors.New("cannot delete last super admin")
)

type UserAdminRBAC interface {
	RoleAssigner
	RunUserAdminMutation(
		ctx context.Context,
		actorID, targetID uint,
		deleting bool,
		mutate func(tx *gorm.DB) error,
	) error
}

type UserAdminService struct {
	users *repository.UserAdminRepository
	rbac  UserAdminRBAC
}

func NewUserAdminService(users *repository.UserAdminRepository, rbac UserAdminRBAC) *UserAdminService {
	return &UserAdminService{users: users, rbac: rbac}
}

func (s *UserAdminService) List(
	ctx context.Context,
	query *dto.AdminUserQuery,
) ([]model.User, int64, int, int, error) {
	page, limit := adminUserPagination(query.Page, query.Limit)
	users, total, err := s.users.List(ctx, strings.TrimSpace(query.Keyword), page, limit)
	if err != nil {
		logger.Error("list admin users", zap.Error(err))
	}
	return users, total, page, limit, err
}

func (s *UserAdminService) Get(ctx context.Context, id uint) (*model.User, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		logger.Error("get admin user", zap.Uint("user_id", id), zap.Error(err))
		return nil, err
	}
	if user == nil {
		return nil, ErrAdminUserNotFound
	}
	return user, nil
}

func (s *UserAdminService) Create(
	ctx context.Context,
	req *dto.CreateAdminUserRequest,
) (*model.User, error) {
	username := strings.TrimSpace(req.Username)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if username == "" {
		return nil, ErrInvalidUsername
	}
	if email == "" {
		return nil, ErrInvalidEmail
	}
	if len(req.Password) < 8 {
		return nil, ErrInvalidPassword
	}
	if err := s.ensureUnique(ctx, username, email, 0); err != nil {
		return nil, err
	}
	hash, err := bcrypt.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if req.IsBanned != nil {
		user.IsBanned = *req.IsBanned
	}
	if err := s.users.Create(ctx, user); err != nil {
		if conflict := userConflictError(err); conflict != nil {
			return nil, conflict
		}
		logger.Error("create admin user", zap.String("username", username), zap.String("email", email), zap.Error(err))
		return nil, err
	}
	if s.rbac != nil {
		if err := s.rbac.AssignRoleByCode(ctx, user.ID, defaultRegisterRoleCode); err != nil {
			logger.Warn("assign default admin-created user role", zap.Uint("user_id", user.ID), zap.Error(err))
		} else {
			return s.Get(ctx, user.ID)
		}
	}
	return user, nil
}

func (s *UserAdminService) Update(
	ctx context.Context,
	actorID, id uint,
	req *dto.UpdateAdminUserRequest,
) (*model.User, error) {
	if actorID == id && req.IsBanned != nil && *req.IsBanned {
		return nil, ErrSelfBan
	}
	user, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	values := make(map[string]any, 4)
	if req.Username != nil {
		username := strings.TrimSpace(*req.Username)
		if username == "" {
			return nil, ErrInvalidUsername
		}
		exists, err := s.users.UsernameExists(ctx, username, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrUsernameExists
		}
		values["username"] = username
	}
	if req.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*req.Email))
		if email == "" {
			return nil, ErrInvalidEmail
		}
		exists, err := s.users.EmailExists(ctx, email, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrEmailExists
		}
		values["email"] = email
	}
	if req.Password != nil {
		if len(*req.Password) < 8 {
			return nil, ErrInvalidPassword
		}
		hash, err := bcrypt.HashPassword(*req.Password)
		if err != nil {
			return nil, err
		}
		values["password_hash"] = hash
	}
	if req.IsBanned != nil {
		values["is_banned"] = *req.IsBanned
	}
	if len(values) == 0 {
		if err := s.runAdminMutation(ctx, actorID, id, false, func(*gorm.DB) error { return nil }); err != nil {
			return nil, err
		}
		return user, nil
	}
	values["updated_at"] = time.Now()
	err = s.runAdminMutation(ctx, actorID, id, false, func(tx *gorm.DB) error {
		return s.users.UpdateTx(ctx, tx, id, values)
	})
	if err != nil {
		if conflict := userConflictError(err); conflict != nil {
			return nil, conflict
		}
		logger.Error("update admin user", zap.Uint("user_id", id), zap.Error(err))
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *UserAdminService) Delete(ctx context.Context, actorID, id uint) error {
	if actorID == id {
		return ErrSelfDelete
	}
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	var deleted bool
	err := s.runAdminMutation(ctx, actorID, id, true, func(tx *gorm.DB) error {
		var err error
		deleted, err = s.users.DeleteTx(ctx, tx, id)
		return err
	})
	if err != nil {
		logger.Error("delete admin user", zap.Uint("user_id", id), zap.Uint("actor_id", actorID), zap.Error(err))
		return err
	}
	if !deleted {
		return ErrAdminUserNotFound
	}
	return nil
}

func (s *UserAdminService) runAdminMutation(
	ctx context.Context,
	actorID, targetID uint,
	deleting bool,
	mutate func(tx *gorm.DB) error,
) error {
	if s.rbac == nil {
		return errors.New("user admin RBAC is not configured")
	}
	err := s.rbac.RunUserAdminMutation(ctx, actorID, targetID, deleting, mutate)
	switch {
	case errors.Is(err, rbacService.ErrSuperAdminUserGuard):
		return ErrSuperAdminUserProtected
	case errors.Is(err, rbacService.ErrLastSuperAdmin):
		return ErrLastSuperAdmin
	default:
		return err
	}
}

func userConflictError(err error) error {
	switch {
	case errors.Is(err, repository.ErrUsernameUniqueViolation):
		return ErrUsernameExists
	case errors.Is(err, repository.ErrEmailUniqueViolation):
		return ErrEmailExists
	default:
		return nil
	}
}

func (s *UserAdminService) ensureUnique(ctx context.Context, username, email string, excludeID uint) error {
	exists, err := s.users.UsernameExists(ctx, username, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return ErrUsernameExists
	}
	exists, err = s.users.EmailExists(ctx, email, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return ErrEmailExists
	}
	return nil
}

func adminUserPagination(page, limit int) (int, int) {
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
