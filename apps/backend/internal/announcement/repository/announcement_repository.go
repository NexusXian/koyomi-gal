package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/announcement/model"

	"gorm.io/gorm"
)

type AnnouncementRepository struct {
	db *gorm.DB
}

func NewAnnouncementRepository(db *gorm.DB) *AnnouncementRepository {
	return &AnnouncementRepository{db: db}
}

func (r *AnnouncementRepository) Create(ctx context.Context, value *model.Announcement) error {
	if err := r.db.WithContext(ctx).Create(value).Error; err != nil {
		return fmt.Errorf("create announcement: %w", err)
	}
	return nil
}

func (r *AnnouncementRepository) Update(ctx context.Context, value *model.Announcement) error {
	value.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Model(&model.Announcement{}).Where("id = ?", value.ID).Updates(map[string]any{
		"title": value.Title, "content": value.Content, "type": value.Type,
		"display_mode": value.DisplayMode, "target": value.Target, "priority": value.Priority,
		"starts_at": value.StartsAt, "ends_at": value.EndsAt, "dismissible": value.Dismissible,
		"published": value.Published, "updated_at": value.UpdatedAt,
	}).Error
	if err != nil {
		return fmt.Errorf("update announcement: %w", err)
	}
	return nil
}

func (r *AnnouncementRepository) SetPublished(ctx context.Context, id uint, published bool) error {
	if err := r.db.WithContext(ctx).Model(&model.Announcement{}).Where("id = ?", id).
		Updates(map[string]any{"published": published, "updated_at": time.Now()}).Error; err != nil {
		return fmt.Errorf("set announcement publication: %w", err)
	}
	return nil
}

func (r *AnnouncementRepository) SyncGenerated(
	ctx context.Context,
	id uint,
	title, content string,
	startsAt *time.Time,
	dismissible, published bool,
) error {
	if err := r.db.WithContext(ctx).Model(&model.Announcement{}).Where("id = ?", id).
		Updates(map[string]any{
			"title": title, "content": content, "starts_at": startsAt,
			"dismissible": dismissible, "published": published, "updated_at": time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("sync generated announcement: %w", err)
	}
	return nil
}

func (r *AnnouncementRepository) Delete(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Delete(&model.Announcement{}, id)
	if result.Error != nil {
		return false, fmt.Errorf("delete announcement: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *AnnouncementRepository) FindByID(ctx context.Context, id uint) (*model.Announcement, error) {
	var value model.Announcement
	err := r.db.WithContext(ctx).First(&value, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find announcement: %w", err)
	}
	return &value, nil
}

func (r *AnnouncementRepository) ListActive(ctx context.Context, platform string) ([]model.Announcement, error) {
	values := make([]model.Announcement, 0)
	targets := []string{model.TargetAll, platform}
	if platform == "windows" || platform == "macos" || platform == "linux" {
		targets = append(targets, model.TargetDesktop)
	}
	err := r.db.WithContext(ctx).
		Where("published = TRUE").
		Where("starts_at IS NULL OR starts_at <= NOW()").
		Where("ends_at IS NULL OR ends_at > NOW()").
		Where("target IN ?", targets).
		Order("priority DESC").Order("updated_at DESC").Order("id DESC").
		Find(&values).Error
	if err != nil {
		return nil, fmt.Errorf("list active announcements: %w", err)
	}
	return values, nil
}

func (r *AnnouncementRepository) ListAdmin(ctx context.Context, page, limit int) ([]model.Announcement, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.Announcement{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count announcements: %w", err)
	}
	values := make([]model.Announcement, 0)
	err := r.db.WithContext(ctx).Order("updated_at DESC").Order("id DESC").
		Offset((page - 1) * limit).Limit(limit).Find(&values).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list announcements: %w", err)
	}
	return values, total, nil
}
