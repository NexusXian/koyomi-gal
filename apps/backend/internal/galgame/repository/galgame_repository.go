package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/galgame/model"

	"gorm.io/gorm"
)

type GalgameListOptions struct {
	Keyword        string
	DeveloperID    *uint
	TagIDs         []uint
	ReleaseFrom    *int
	ReleaseTo      *int
	AgeRating      *int16
	CoverSensitive *bool
	Status         *int16
	Sort           string
	Page           int
	Limit          int
}

type GalgameRepository struct {
	db *gorm.DB
}

func NewGalgameRepository(db *gorm.DB) *GalgameRepository {
	return &GalgameRepository{db: db}
}

func (r *GalgameRepository) Transaction(ctx context.Context, fn func(tx *GalgameRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&GalgameRepository{db: tx})
	})
}

func (r *GalgameRepository) Create(ctx context.Context, galgame *model.Galgame) error {
	if err := r.db.WithContext(ctx).Create(galgame).Error; err != nil {
		return fmt.Errorf("create galgame: %w", err)
	}
	return nil
}

func (r *GalgameRepository) Update(ctx context.Context, galgame *model.Galgame) error {
	galgame.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).
		Model(&model.Galgame{}).
		Where("id = ?", galgame.ID).
		Updates(map[string]any{
			"title":              galgame.Title,
			"original_title":     galgame.OriginalTitle,
			"romaji_title":       galgame.RomajiTitle,
			"slug":               galgame.Slug,
			"description":        galgame.Description,
			"description_source": galgame.DescriptionSource,
			"cover_url":          galgame.CoverURL,
			"banner_url":         galgame.BannerURL,
			"developer_id":       galgame.DeveloperID,
			"release_date":       galgame.ReleaseDate,
			"age_rating":         galgame.AgeRating,
			"cover_sensitive":    galgame.CoverSensitive,
			"status":             galgame.Status,
			"updated_at":         galgame.UpdatedAt,
		}).Error
	if err != nil {
		return fmt.Errorf("update galgame: %w", err)
	}
	return nil
}

func (r *GalgameRepository) UpdateStatus(ctx context.Context, id uint, status int16) error {
	if err := r.db.WithContext(ctx).
		Model(&model.Galgame{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": status, "updated_at": time.Now()}).Error; err != nil {
		return fmt.Errorf("update galgame status: %w", err)
	}
	return nil
}

// BatchUpdate applies the whitelisted column updates to every matching id and
// returns the number of affected rows.
func (r *GalgameRepository) BatchUpdate(ctx context.Context, ids []uint, updates map[string]any) (int64, error) {
	if len(ids) == 0 || len(updates) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).
		Model(&model.Galgame{}).
		Where("id IN ?", ids).
		Updates(updates)
	if result.Error != nil {
		return 0, fmt.Errorf("batch update galgames: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *GalgameRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&model.Galgame{}, id).Error; err != nil {
		return fmt.Errorf("delete galgame: %w", err)
	}
	return nil
}

// BatchDelete removes every matching id and returns the number of deleted rows.
func (r *GalgameRepository) BatchDelete(ctx context.Context, ids []uint) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.Galgame{})
	if result.Error != nil {
		return 0, fmt.Errorf("batch delete galgames: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *GalgameRepository) FindByID(ctx context.Context, id uint) (*model.Galgame, error) {
	return r.findByID(ctx, id, false)
}

func (r *GalgameRepository) FindPublishedByID(ctx context.Context, id uint) (*model.Galgame, error) {
	return r.findByID(ctx, id, true)
}

func (r *GalgameRepository) findByID(ctx context.Context, id uint, publishedOnly bool) (*model.Galgame, error) {
	var galgame model.Galgame
	query := r.withAssociations(r.db.WithContext(ctx))
	if publishedOnly {
		query = query.Where("status = ?", model.GalgameStatusPublished)
	}
	err := query.First(&galgame, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find galgame by id: %w", err)
	}
	return &galgame, nil
}

func (r *GalgameRepository) FindBySlug(ctx context.Context, slug string) (*model.Galgame, error) {
	var galgame model.Galgame
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&galgame).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find galgame by slug: %w", err)
	}
	return &galgame, nil
}

func (r *GalgameRepository) ListPublished(
	ctx context.Context,
	options GalgameListOptions,
) ([]model.Galgame, int64, error) {
	countQuery := r.applyListFilters(
		r.db.WithContext(ctx).Table("galgames AS g"),
		options,
		true,
	)
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count published galgames: %w", err)
	}

	query := r.applyListFilters(
		r.db.WithContext(ctx).Model(&model.Galgame{}).Table("galgames AS g").Select("g.*"),
		options,
		true,
	)
	query = r.withAssociations(query).
		Order(galgameSort(options.Sort)).
		Order("g.id DESC").
		Offset((options.Page - 1) * options.Limit).
		Limit(options.Limit)

	var galgames []model.Galgame
	if err := query.Find(&galgames).Error; err != nil {
		return nil, 0, fmt.Errorf("list published galgames: %w", err)
	}
	return galgames, total, nil
}

// ListAdmin lists galgames of every status; an optional Status filter narrows
// the result. Used by the galgame:review admin queries.
func (r *GalgameRepository) ListAdmin(
	ctx context.Context,
	options GalgameListOptions,
) ([]model.Galgame, int64, error) {
	countQuery := r.applyListFilters(
		r.db.WithContext(ctx).Table("galgames AS g"),
		options,
		false,
	)
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count galgames: %w", err)
	}

	query := r.applyListFilters(
		r.db.WithContext(ctx).Model(&model.Galgame{}).Table("galgames AS g").Select("g.*"),
		options,
		false,
	)
	query = r.withAssociations(query).
		Order(galgameSort(options.Sort)).
		Order("g.id DESC").
		Offset((options.Page - 1) * options.Limit).
		Limit(options.Limit)

	var galgames []model.Galgame
	if err := query.Find(&galgames).Error; err != nil {
		return nil, 0, fmt.Errorf("list galgames: %w", err)
	}
	return galgames, total, nil
}

func (r *GalgameRepository) ListByCreator(
	ctx context.Context,
	userID uint,
	page int,
	limit int,
) ([]model.Galgame, int64, error) {
	base := r.db.WithContext(ctx).Table("galgames AS g").Where("g.created_by = ?", userID)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count user galgames: %w", err)
	}

	query := r.db.WithContext(ctx).
		Model(&model.Galgame{}).
		Table("galgames AS g").
		Select("g.*").
		Where("g.created_by = ?", userID)
	query = r.withAssociations(query).
		Order("g.created_at DESC").
		Order("g.id DESC").
		Offset((page - 1) * limit).
		Limit(limit)

	var galgames []model.Galgame
	if err := query.Find(&galgames).Error; err != nil {
		return nil, 0, fmt.Errorf("list user galgames: %w", err)
	}
	return galgames, total, nil
}

func (r *GalgameRepository) ListFavoritesByUser(
	ctx context.Context,
	userID uint,
	page int,
	limit int,
) ([]model.Galgame, int64, error) {
	base := r.db.WithContext(ctx).
		Table("galgames AS g").
		Joins("JOIN galgame_favorites AS f ON f.galgame_id = g.id").
		Where("f.user_id = ? AND g.status = ?", userID, model.GalgameStatusPublished)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count user favorite galgames: %w", err)
	}

	query := r.db.WithContext(ctx).
		Model(&model.Galgame{}).
		Table("galgames AS g").
		Select("g.*").
		Joins("JOIN galgame_favorites AS f ON f.galgame_id = g.id").
		Where("f.user_id = ? AND g.status = ?", userID, model.GalgameStatusPublished)
	query = r.withAssociations(query).
		Order("f.created_at DESC").
		Order("g.id DESC").
		Offset((page - 1) * limit).
		Limit(limit)

	var galgames []model.Galgame
	if err := query.Find(&galgames).Error; err != nil {
		return nil, 0, fmt.Errorf("list user favorite galgames: %w", err)
	}
	return galgames, total, nil
}

func (r *GalgameRepository) ReplaceAliases(ctx context.Context, galgameID uint, aliases []string) error {
	if err := r.db.WithContext(ctx).Where("galgame_id = ?", galgameID).Delete(&model.Alias{}).Error; err != nil {
		return fmt.Errorf("delete galgame aliases: %w", err)
	}
	if len(aliases) == 0 {
		return nil
	}
	rows := make([]model.Alias, 0, len(aliases))
	for _, alias := range aliases {
		rows = append(rows, model.Alias{GalgameID: galgameID, Alias: alias})
	}
	if err := r.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return fmt.Errorf("create galgame aliases: %w", err)
	}
	return nil
}

func (r *GalgameRepository) ReplaceTags(ctx context.Context, galgameID uint, tagIDs []uint) error {
	if err := r.db.WithContext(ctx).Where("galgame_id = ?", galgameID).Delete(&model.GalgameTag{}).Error; err != nil {
		return fmt.Errorf("delete galgame tags: %w", err)
	}
	if len(tagIDs) == 0 {
		return nil
	}
	rows := make([]model.GalgameTag, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		rows = append(rows, model.GalgameTag{GalgameID: galgameID, TagID: tagID})
	}
	if err := r.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return fmt.Errorf("create galgame tags: %w", err)
	}
	return nil
}

func (r *GalgameRepository) withAssociations(query *gorm.DB) *gorm.DB {
	return query.
		Preload("Developer").
		Preload("Aliases", func(db *gorm.DB) *gorm.DB { return db.Order("galgame_aliases.id") }).
		Preload("Tags", func(db *gorm.DB) *gorm.DB { return db.Order("tags.id") })
}

func (r *GalgameRepository) applyListFilters(
	query *gorm.DB,
	options GalgameListOptions,
	publishedOnly bool,
) *gorm.DB {
	if publishedOnly {
		query = query.Where("g.status = ?", model.GalgameStatusPublished)
	}
	if options.Status != nil {
		query = query.Where("g.status = ?", *options.Status)
	}
	if options.Keyword != "" {
		pattern := "%" + escapeLikePattern(options.Keyword) + "%"
		query = query.Where(`
(g.title ILIKE ? ESCAPE E'\\' OR g.original_title ILIKE ? ESCAPE E'\\' OR g.romaji_title ILIKE ? ESCAPE E'\\' OR EXISTS (
    SELECT 1 FROM galgame_aliases ga
    WHERE ga.galgame_id = g.id AND ga.alias ILIKE ? ESCAPE E'\\'
))`, pattern, pattern, pattern, pattern)
	}
	if options.DeveloperID != nil {
		query = query.Where("g.developer_id = ?", *options.DeveloperID)
	}
	if len(options.TagIDs) > 0 {
		query = query.Where(`g.id IN (
    SELECT gt.galgame_id
    FROM galgame_tags gt
    WHERE gt.tag_id IN ?
    GROUP BY gt.galgame_id
    HAVING COUNT(DISTINCT gt.tag_id) = ?
)`, options.TagIDs, len(options.TagIDs))
	}
	if options.ReleaseFrom != nil {
		query = query.Where("g.release_date >= make_date(?, 1, 1)", *options.ReleaseFrom)
	}
	if options.ReleaseTo != nil {
		query = query.Where("g.release_date < make_date(? + 1, 1, 1)", *options.ReleaseTo)
	}
	if options.AgeRating != nil {
		query = query.Where("g.age_rating = ?", *options.AgeRating)
	}
	if options.CoverSensitive != nil {
		query = query.Where("g.cover_sensitive = ?", *options.CoverSensitive)
	}
	return query
}

func escapeLikePattern(value string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	).Replace(value)
}

func galgameSort(sort string) string {
	switch sort {
	case "oldest":
		return "g.release_date ASC NULLS LAST"
	case "rating":
		return "g.rating_average DESC"
	case "favorite":
		return "g.favorite_count DESC"
	case "popular":
		return "(g.favorite_count + g.resource_count + g.post_count) DESC"
	case "updated":
		return "g.updated_at DESC"
	default:
		return "g.release_date DESC NULLS LAST"
	}
}
