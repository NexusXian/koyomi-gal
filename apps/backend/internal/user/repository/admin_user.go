package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"backend/internal/user/model"

	"gorm.io/gorm"
)

type UserAdminRepository struct {
	db *gorm.DB
}

func NewUserAdminRepository(db *gorm.DB) *UserAdminRepository {
	return &UserAdminRepository{db: db}
}

func (r *UserAdminRepository) List(
	ctx context.Context,
	keyword string,
	page, limit int,
) ([]model.User, int64, error) {
	base := func() *gorm.DB {
		query := r.db.WithContext(ctx).Model(&model.User{})
		if keyword == "" {
			return query
		}
		pattern := "%" + escapeUserLikePattern(keyword) + "%"
		if id, err := strconv.ParseUint(keyword, 10, 64); err == nil && id > 0 {
			return query.Where(
				`(username ILIKE ? ESCAPE E'\\' OR email ILIKE ? ESCAPE E'\\' OR id = ?)`,
				pattern, pattern, id,
			)
		}
		return query.Where(
			`(username ILIKE ? ESCAPE E'\\' OR email ILIKE ? ESCAPE E'\\')`,
			pattern, pattern,
		)
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}
	users := make([]model.User, 0)
	if err := preloadUserRoles(base()).Order("id DESC").Offset((page - 1) * limit).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	return users, total, nil
}

func (r *UserAdminRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	err := preloadUserRoles(r.db.WithContext(ctx)).First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return &user, nil
}

func preloadUserRoles(db *gorm.DB) *gorm.DB {
	return db.Preload("Roles", func(db *gorm.DB) *gorm.DB {
		return db.Select("roles.id", "roles.name", "roles.code").Order("roles.id")
	})
}

func (r *UserAdminRepository) UsernameExists(ctx context.Context, username string, excludeID uint) (bool, error) {
	query := r.db.WithContext(ctx).Model(&model.User{}).Where("LOWER(username) = LOWER(?)", username)
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("check username exists: %w", err)
	}
	return count > 0, nil
}

func (r *UserAdminRepository) EmailExists(ctx context.Context, email string, excludeID uint) (bool, error) {
	query := r.db.WithContext(ctx).Model(&model.User{}).Where("LOWER(email) = LOWER(?)", email)
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("check email exists: %w", err)
	}
	return count > 0, nil
}

func (r *UserAdminRepository) Create(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("create user: %w", mapUserWriteError(err))
	}
	return nil
}

func (r *UserAdminRepository) Update(ctx context.Context, id uint, values map[string]any) error {
	return r.UpdateTx(ctx, r.db, id, values)
}

func (r *UserAdminRepository) UpdateTx(
	ctx context.Context,
	tx *gorm.DB,
	id uint,
	values map[string]any,
) error {
	if err := tx.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(values).Error; err != nil {
		return fmt.Errorf("update user: %w", mapUserWriteError(err))
	}
	return nil
}

func (r *UserAdminRepository) Delete(ctx context.Context, id uint) (bool, error) {
	deleted := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		deleted, err = r.DeleteTx(ctx, tx, id)
		return err
	})
	return deleted, err
}

func (r *UserAdminRepository) DeleteTx(ctx context.Context, tx *gorm.DB, id uint) (bool, error) {
	if err := tx.WithContext(ctx).Exec("DELETE FROM user_roles WHERE user_id = ?", id).Error; err != nil {
		return false, fmt.Errorf("delete user roles: %w", err)
	}
	result := tx.WithContext(ctx).Delete(&model.User{}, id)
	if result.Error != nil {
		return false, fmt.Errorf("delete user: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func escapeUserLikePattern(value string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	).Replace(value)
}
