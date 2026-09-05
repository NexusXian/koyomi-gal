package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/novel/model"

	"gorm.io/gorm"
)

type NovelRepository struct {
	db *gorm.DB
}

func NewNovelRepository(db *gorm.DB) *NovelRepository {
	return &NovelRepository{db: db}
}

func (r *NovelRepository) Transaction(ctx context.Context, fn func(tx *NovelRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&NovelRepository{db: tx})
	})
}

func (r *NovelRepository) Create(ctx context.Context, novel *model.Novel) error {
	if err := r.db.WithContext(ctx).Create(novel).Error; err != nil {
		return fmt.Errorf("create novel: %w", err)
	}
	return nil
}

func (r *NovelRepository) Update(ctx context.Context, novel *model.Novel) error {
	novel.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).
		Model(&model.Novel{}).
		Where("id = ?", novel.ID).
		Updates(map[string]any{
			"title":              novel.Title,
			"original_title":     novel.OriginalTitle,
			"slug":               novel.Slug,
			"summary":            novel.Summary,
			"cover_url":          novel.CoverURL,
			"author":             novel.Author,
			"illustrator":        novel.Illustrator,
			"publisher":          novel.Publisher,
			"label":              novel.Label,
			"language":           novel.Language,
			"region":             novel.Region,
			"release_status":     novel.ReleaseStatus,
			"first_release_date": novel.FirstReleaseDate,
			"age_rating":         novel.AgeRating,
			"is_cover_sensitive": novel.IsCoverSensitive,
			"official_website":   novel.OfficialWebsite,
			"status":             novel.Status,
			"updated_at":         novel.UpdatedAt,
		}).Error
	if err != nil {
		return fmt.Errorf("update novel: %w", err)
	}
	return nil
}

func (r *NovelRepository) UpdateReview(
	ctx context.Context,
	id uint,
	status int16,
	reviewedBy *uint,
	reviewedAt *time.Time,
	rejectReason string,
) error {
	err := r.db.WithContext(ctx).
		Model(&model.Novel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        status,
			"reviewed_by":   reviewedBy,
			"reviewed_at":   reviewedAt,
			"reject_reason": rejectReason,
			"updated_at":    time.Now(),
		}).Error
	if err != nil {
		return fmt.Errorf("update novel review: %w", err)
	}
	return nil
}

// Delete soft-deletes the novel; gorm.DeletedAt filters it from later reads.
func (r *NovelRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&model.Novel{}, id).Error; err != nil {
		return fmt.Errorf("delete novel: %w", err)
	}
	return nil
}

func (r *NovelRepository) FindByID(ctx context.Context, id uint) (*model.Novel, error) {
	return r.findByID(ctx, id, false)
}

func (r *NovelRepository) FindPublishedByID(ctx context.Context, id uint) (*model.Novel, error) {
	return r.findByID(ctx, id, true)
}

func (r *NovelRepository) findByID(ctx context.Context, id uint, publishedOnly bool) (*model.Novel, error) {
	var novel model.Novel
	query := r.withAssociations(r.db.WithContext(ctx))
	if publishedOnly {
		query = query.Where("status = ?", model.NovelStatusPublished)
	}
	err := query.First(&novel, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find novel by id: %w", err)
	}
	return &novel, nil
}

func (r *NovelRepository) FindBySlug(ctx context.Context, slug string) (*model.Novel, error) {
	var novel model.Novel
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&novel).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find novel by slug: %w", err)
	}
	return &novel, nil
}

type NovelListOptions struct {
	Keyword       string
	TagIDs        []uint
	Author        string
	Publisher     string
	Label         string
	Language      string
	ReleaseStatus string
	Status        *int16
	Sort          string
	Page          int
	Limit         int
}

// ListPublished returns published novels paginated; admin listings pass a
// Status filter through the same query builder.
func (r *NovelRepository) ListPublished(
	ctx context.Context,
	options NovelListOptions,
) ([]model.Novel, int64, error) {
	base := func() *gorm.DB {
		query := r.db.WithContext(ctx).
			Table("novels AS n").
			Where("n.deleted_at IS NULL AND n.status = ?", model.NovelStatusPublished)
		return applyNovelFilters(query, options)
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count published novels: %w", err)
	}

	var novels []model.Novel
	err := r.withAssociations(base()).
		Select("n.*, (SELECT COUNT(*) FROM novel_volumes v WHERE v.novel_id = n.id AND v.status = 1 AND v.deleted_at IS NULL) AS volume_count").
		Order(novelSort(options.Sort)).
		Order("n.id DESC").
		Offset((options.Page - 1) * options.Limit).
		Limit(options.Limit).
		Find(&novels).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list published novels: %w", err)
	}
	return novels, total, nil
}

// ListAdmin returns novels of every status with an optional status filter.
func (r *NovelRepository) ListAdmin(
	ctx context.Context,
	options NovelListOptions,
) ([]model.Novel, int64, error) {
	base := func() *gorm.DB {
		query := r.db.WithContext(ctx).
			Table("novels AS n").
			Where("n.deleted_at IS NULL")
		if options.Status != nil {
			query = query.Where("n.status = ?", *options.Status)
		}
		return applyNovelFilters(query, options)
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count admin novels: %w", err)
	}

	var novels []model.Novel
	err := r.withAssociations(base()).
		Select("n.*, (SELECT COUNT(*) FROM novel_volumes v WHERE v.novel_id = n.id AND v.status = 1 AND v.deleted_at IS NULL) AS volume_count").
		Order(novelSort(options.Sort)).
		Order("n.id DESC").
		Offset((options.Page - 1) * options.Limit).
		Limit(options.Limit).
		Find(&novels).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list admin novels: %w", err)
	}
	return novels, total, nil
}

func applyNovelFilters(query *gorm.DB, options NovelListOptions) *gorm.DB {
	if options.Keyword != "" {
		pattern := "%" + escapeLikePattern(options.Keyword) + "%"
		query = query.Where(`(n.title ILIKE ? ESCAPE E'\\' OR n.original_title ILIKE ? ESCAPE E'\\' OR n.author ILIKE ? ESCAPE E'\\')`,
			pattern, pattern, pattern)
	}
	if options.Author != "" {
		query = query.Where("n.author ILIKE ?", "%"+escapeLikePattern(options.Author)+"%")
	}
	if options.Publisher != "" {
		query = query.Where("n.publisher ILIKE ?", "%"+escapeLikePattern(options.Publisher)+"%")
	}
	if options.Label != "" {
		query = query.Where("n.label ILIKE ?", "%"+escapeLikePattern(options.Label)+"%")
	}
	if options.Language != "" {
		query = query.Where("n.language = ?", options.Language)
	}
	if options.ReleaseStatus != "" {
		query = query.Where("n.release_status = ?", options.ReleaseStatus)
	}
	if len(options.TagIDs) > 0 {
		query = query.Where(`n.id IN (
    SELECT nt.novel_id
    FROM novel_tags nt
    WHERE nt.tag_id IN ?
    GROUP BY nt.novel_id
    HAVING COUNT(DISTINCT nt.tag_id) = ?
)`, options.TagIDs, len(options.TagIDs))
	}
	return query
}

func novelSort(sort string) string {
	switch sort {
	case "updated":
		return "n.updated_at DESC"
	case "release":
		return "n.first_release_date DESC NULLS LAST"
	case "release_asc":
		return "n.first_release_date ASC NULLS LAST"
	case "oldest":
		return "n.created_at ASC"
	default:
		return "n.created_at DESC"
	}
}

func (r *NovelRepository) ReplaceTags(ctx context.Context, novelID uint, tagIDs []uint) error {
	if err := r.db.WithContext(ctx).
		Where("novel_id = ?", novelID).
		Delete(&model.NovelTag{}).Error; err != nil {
		return fmt.Errorf("delete novel tags: %w", err)
	}
	if len(tagIDs) == 0 {
		return nil
	}
	rows := make([]model.NovelTag, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		rows = append(rows, model.NovelTag{NovelID: novelID, TagID: tagID})
	}
	if err := r.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return fmt.Errorf("create novel tags: %w", err)
	}
	return nil
}

// DeleteTags physically removes the novel's tag bindings; used when the novel
// itself is (soft-)deleted so no orphan join rows remain.
func (r *NovelRepository) DeleteTags(ctx context.Context, novelID uint) error {
	if err := r.db.WithContext(ctx).
		Where("novel_id = ?", novelID).
		Delete(&model.NovelTag{}).Error; err != nil {
		return fmt.Errorf("delete novel tags: %w", err)
	}
	return nil
}

// DeleteExternalMappings physically removes the novel's external ID mappings.
func (r *NovelRepository) DeleteExternalMappings(ctx context.Context, novelID uint) error {
	err := r.db.WithContext(ctx).
		Exec("DELETE FROM external_mappings WHERE target_type = ? AND target_id = ?", "novel", novelID).Error
	if err != nil {
		return fmt.Errorf("delete novel external mappings: %w", err)
	}
	return nil
}

// CountPublishedVolumes backs the detail response's dynamic volume count.
func (r *NovelRepository) CountPublishedVolumes(ctx context.Context, novelID uint) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&model.NovelVolume{}).
		Where("novel_id = ? AND status = ?", novelID, model.NovelStatusPublished).
		Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("count published volumes: %w", err)
	}
	return total, nil
}

func (r *NovelRepository) withAssociations(query *gorm.DB) *gorm.DB {
	return query.Preload("Tags", func(db *gorm.DB) *gorm.DB { return db.Order("tags.id") })
}

func escapeLikePattern(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}
