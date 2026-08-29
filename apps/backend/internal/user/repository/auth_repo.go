package repository

import (
	"backend/internal/user/model"
	"context"

	"gorm.io/gorm"
)

type UserAuthRepository struct {
	db *gorm.DB
}

func NewUserAuthRepository(db *gorm.DB) *UserAuthRepository {
	return &UserAuthRepository{db: db}
}

func (r *UserAuthRepository) Create(ctx context.Context,user *model.User) error {
    return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserAuthRepository) FindByEmail(email string) (*model.User, error) {
    var user model.User
    err := r.db.Where("email = ?", email).First(&user).Error
    if err != nil {
        return nil, err
    }
    return &user, nil
}
