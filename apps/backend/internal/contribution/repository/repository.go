package repository

import (
	"context"
	"fmt"
	"strings"

	contributionModel "backend/internal/contribution/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ContributionRepository struct {
	db        *gorm.DB
	publicURL string
}

func NewContributionRepository(db *gorm.DB, publicURL string) *ContributionRepository {
	return &ContributionRepository{db: db, publicURL: strings.TrimRight(publicURL, "/")}
}

func (r *ContributionRepository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *ContributionRepository) CreateContribution(ctx context.Context, contribution *contributionModel.WorkContribution) error {
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(contribution).Error; err != nil {
		return fmt.Errorf("create work contribution: %w", err)
	}
	return nil
}

func (r *ContributionRepository) ListContributors(
	ctx context.Context,
	targetType string,
	targetID uint,
	page, pageSize int,
) ([]contributionModel.WorkContributor, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Table("work_contributions").
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Distinct("user_id").
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count work contributors: %w", err)
	}

	aggregated := r.db.WithContext(ctx).
		Table("work_contributions").
		Select(`user_id,
COUNT(*) AS contribution_count,
MIN(created_at) AS first_contributed_at,
MAX(created_at) AS last_contributed_at`).
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Group("user_id")

	contributors := make([]contributionModel.WorkContributor, 0)
	err := r.db.WithContext(ctx).
		Table("(?) AS contributions", aggregated).
		Select(`contributions.user_id,
COALESCE(users.username, '') AS username,
CASE WHEN avatar_assets.object_key IS NOT NULL THEN CAST(? AS text) || '/' || avatar_assets.object_key ELSE COALESCE(users.avatar, '') END AS avatar_url,
contributions.contribution_count,
contributions.first_contributed_at,
contributions.last_contributed_at`, r.publicURL).
		Joins("LEFT JOIN users ON users.id = contributions.user_id").
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.id").
		Joins("LEFT JOIN image_assets AS avatar_assets ON avatar_assets.id = user_profiles.avatar_asset_id AND avatar_assets.user_id = users.id AND avatar_assets.category = 'avatars' AND avatar_assets.status = 1").
		Order("contributions.contribution_count DESC").
		Order("contributions.last_contributed_at DESC").
		Order("contributions.user_id ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&contributors).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list work contributors: %w", err)
	}
	return contributors, total, nil
}

func (r *ContributionRepository) ListContributionsByUserID(
	ctx context.Context,
	userID uint,
	page, pageSize int,
) ([]contributionModel.WorkContribution, int64, error) {
	query := r.db.WithContext(ctx).Model(&contributionModel.WorkContribution{}).Where("user_id = ?", userID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count user contributions: %w", err)
	}
	items := make([]contributionModel.WorkContribution, 0)
	if err := query.Order("created_at DESC").Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list user contributions: %w", err)
	}
	return items, total, nil
}

func (r *ContributionRepository) ExistsBySource(ctx context.Context, sourceType string, sourceID uint) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&contributionModel.WorkContribution{}).
		Where("source_type = ? AND source_id = ?", sourceType, sourceID).
		Limit(1).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("check contribution source: %w", err)
	}
	return count > 0, nil
}
