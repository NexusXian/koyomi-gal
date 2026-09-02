package repository

import (
	"backend/internal/user/model"
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var (
	ErrUsernameUniqueViolation = errors.New("username unique constraint violated")
	ErrEmailUniqueViolation    = errors.New("email unique constraint violated")
)

const (
	usernameLowerUniqueIndex = "idx_users_username_lower_unique"
	emailLowerUniqueIndex    = "idx_users_email_lower_unique"
	legacyEmailUniqueIndex   = "idx_users_email"
)

type UserAuthRepository struct {
	db *gorm.DB
}

func NewUserAuthRepository(db *gorm.DB) *UserAuthRepository {
	return &UserAuthRepository{db: db}
}

func (r *UserAuthRepository) CreateUser(ctx context.Context, user *model.User) error {
	return mapUserWriteError(r.db.WithContext(ctx).Create(user).Error)
}

func (r *UserAuthRepository) FindUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("LOWER(email) = LOWER(?)", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserAuthRepository) FindUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("LOWER(username) = LOWER(?)", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserAuthRepository) UpdateUser(ctx context.Context, user *model.User) error {
	return mapUserWriteError(r.db.WithContext(ctx).Updates(user).Error)
}

func (r *UserAuthRepository) UpdateAvatarAssetID(ctx context.Context, userID uint, assetID *uint) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Update("avatar_asset_id", assetID).Error
}

func (r *UserAuthRepository) FindUserByID(ctx context.Context, userID uint) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserAuthRepository) AccessUserStatus(ctx context.Context, userID uint) (bool, bool, error) {
	var status struct {
		IsBanned bool
	}
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Select("is_banned").
		Where("id = ?", userID).
		Take(&status).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return true, status.IsBanned, nil
}

func mapUserWriteError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}
	switch pgErr.ConstraintName {
	case usernameLowerUniqueIndex:
		return errors.Join(ErrUsernameUniqueViolation, err)
	case emailLowerUniqueIndex, legacyEmailUniqueIndex:
		return errors.Join(ErrEmailUniqueViolation, err)
	default:
		return err
	}
}
