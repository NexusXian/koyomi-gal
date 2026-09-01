package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/article/model"

	"gorm.io/gorm"
)

type ArticleRepository struct {
	db *gorm.DB
}

func NewArticleRepository(db *gorm.DB) *ArticleRepository {
	return &ArticleRepository{db: db}
}

func (r *ArticleRepository) Create(ctx context.Context, article *model.Article) error {
	if err := r.db.WithContext(ctx).Create(article).Error; err != nil {
		return fmt.Errorf("create article: %w", err)
	}
	return nil
}

func (r *ArticleRepository) Update(ctx context.Context, article *model.Article) error {
	article.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Model(&model.Article{}).Where("id = ?", article.ID).Updates(map[string]any{
		"title": article.Title, "summary": article.Summary, "content": article.Content,
		"editor_mode": article.EditorMode,
		"cover_url":   article.CoverURL, "type": article.Type, "is_pinned": article.IsPinned,
		"is_published": article.IsPublished, "published_at": article.PublishedAt, "updated_at": article.UpdatedAt,
	}).Error
	if err != nil {
		return fmt.Errorf("update article: %w", err)
	}
	return nil
}

func (r *ArticleRepository) Delete(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Delete(&model.Article{}, id)
	if result.Error != nil {
		return false, fmt.Errorf("delete article: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *ArticleRepository) FindByID(ctx context.Context, id uint) (*model.Article, error) {
	var article model.Article
	err := r.db.WithContext(ctx).First(&article, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find article by id: %w", err)
	}
	return &article, nil
}

func (r *ArticleRepository) FindPublishedAndIncrementView(ctx context.Context, id uint) (*model.Article, error) {
	var article model.Article
	err := r.db.WithContext(ctx).Raw(`
UPDATE articles
SET view_count = view_count + 1
WHERE id = ? AND is_published = TRUE AND published_at IS NOT NULL AND published_at <= NOW()
RETURNING *
`, id).Scan(&article).Error
	if err != nil {
		return nil, fmt.Errorf("find published article and increment view: %w", err)
	}
	if article.ID == 0 {
		return nil, nil
	}
	return &article, nil
}

func (r *ArticleRepository) ListPublished(
	ctx context.Context,
	articleType string,
	page, limit int,
) ([]model.Article, int64, error) {
	base := func() *gorm.DB {
		query := r.db.WithContext(ctx).Model(&model.Article{}).
			Where("is_published = TRUE AND published_at IS NOT NULL AND published_at <= NOW()")
		if articleType != "" {
			query = query.Where("type = ?", articleType)
		}
		return query
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count published articles: %w", err)
	}
	articles := make([]model.Article, 0)
	err := base().Order("is_pinned DESC").Order("published_at DESC").Order("id DESC").
		Offset((page - 1) * limit).Limit(limit).Find(&articles).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list published articles: %w", err)
	}
	return articles, total, nil
}

func (r *ArticleRepository) ListAdmin(ctx context.Context, page, limit int) ([]model.Article, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.Article{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count articles: %w", err)
	}
	articles := make([]model.Article, 0)
	err := r.db.WithContext(ctx).Order("id DESC").Offset((page - 1) * limit).
		Limit(limit).Find(&articles).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list admin articles: %w", err)
	}
	return articles, total, nil
}
