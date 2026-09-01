package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"backend/internal/article/dto"
	"backend/internal/article/model"
	"backend/internal/article/repository"
	homeService "backend/internal/home/service"
	rbacService "backend/internal/rbac/service"

	"github.com/redis/go-redis/v9"
)

var (
	ErrArticleNotFound      = errors.New("article not found")
	ErrInvalidArticleInput  = errors.New("invalid article input")
	ErrInvalidArticleType   = errors.New("invalid article type")
	ErrInvalidArticleEditor = errors.New("invalid article editor mode")
	ErrArticleForbidden     = errors.New("article permission denied")
)

const (
	PermissionArticleUpdate  = "article:update"
	PermissionArticlePublish = "article:publish"
)

type ArticleService struct {
	articles *repository.ArticleRepository
	rbac     *rbacService.RBACService
	cache    *redis.Client
}

func NewArticleService(
	articles *repository.ArticleRepository,
	rbac *rbacService.RBACService,
	cache *redis.Client,
) *ArticleService {
	return &ArticleService{articles: articles, rbac: rbac, cache: cache}
}

func (s *ArticleService) ListPublished(ctx context.Context, articleType string, page, limit int) ([]model.Article, int64, int, int, error) {
	if articleType != "" && !validArticleType(articleType) {
		return nil, 0, page, limit, ErrInvalidArticleType
	}
	page, limit = articlePagination(page, limit)
	articles, total, err := s.articles.ListPublished(ctx, articleType, page, limit)
	return articles, total, page, limit, err
}

func (s *ArticleService) GetPublished(ctx context.Context, id uint) (*model.Article, error) {
	article, err := s.articles.FindPublishedAndIncrementView(ctx, id)
	if err != nil {
		return nil, err
	}
	if article == nil {
		return nil, ErrArticleNotFound
	}
	return article, nil
}

func (s *ArticleService) ListAdmin(ctx context.Context, page, limit int) ([]model.Article, int64, int, int, error) {
	page, limit = articlePagination(page, limit)
	articles, total, err := s.articles.ListAdmin(ctx, page, limit)
	return articles, total, page, limit, err
}

func (s *ArticleService) Get(ctx context.Context, id uint) (*model.Article, error) {
	article, err := s.articles.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if article == nil {
		return nil, ErrArticleNotFound
	}
	return article, nil
}

func (s *ArticleService) Create(ctx context.Context, actorID uint, req *dto.CreateArticleRequest) (*model.Article, error) {
	isPublished := req.IsPublished != nil && *req.IsPublished
	publishedAt := publicationTime(isPublished, req.PublishedAt, nil)
	editorMode := model.EditorModePlain
	if req.EditorMode != "" {
		editorMode = req.EditorMode
	}
	article := &model.Article{
		Title: strings.TrimSpace(req.Title), Summary: strings.TrimSpace(req.Summary),
		Content: strings.TrimSpace(req.Content), EditorMode: editorMode,
		CoverURL: strings.TrimSpace(req.CoverURL),
		Type:     strings.TrimSpace(req.Type), IsPinned: req.IsPinned,
		IsPublished: isPublished, PublishedAt: publishedAt,
	}
	if err := validateArticle(article); err != nil {
		return nil, err
	}
	if article.IsPublished {
		if err := s.requirePermission(ctx, actorID, PermissionArticlePublish); err != nil {
			return nil, err
		}
	}
	if err := s.articles.Create(ctx, article); err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	return article, nil
}

func (s *ArticleService) Update(ctx context.Context, actorID, id uint, req *dto.UpdateArticleRequest) (*model.Article, error) {
	article, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	isPublished := req.IsPublished != nil && *req.IsPublished
	publishedAt := publicationTime(isPublished, req.PublishedAt, article.PublishedAt)
	// An omitted editor mode keeps the stored value so legacy clients cannot
	// accidentally downgrade a markdown article to plain.
	editorMode := article.EditorMode
	if req.EditorMode != "" {
		editorMode = req.EditorMode
	}
	desired := &model.Article{
		ID: article.ID, Title: strings.TrimSpace(req.Title), Summary: strings.TrimSpace(req.Summary),
		Content: strings.TrimSpace(req.Content), EditorMode: editorMode,
		CoverURL: strings.TrimSpace(req.CoverURL),
		Type:     strings.TrimSpace(req.Type), IsPinned: req.IsPinned,
		IsPublished: isPublished, PublishedAt: publishedAt,
		ViewCount: article.ViewCount, CreatedAt: article.CreatedAt,
	}
	if err := validateArticle(desired); err != nil {
		return nil, err
	}
	publicationChanged := article.IsPublished != desired.IsPublished || !sameTime(article.PublishedAt, desired.PublishedAt)
	if articleFieldsChanged(article, desired) || !publicationChanged {
		if err := s.requirePermission(ctx, actorID, PermissionArticleUpdate); err != nil {
			return nil, err
		}
	}
	if publicationChanged {
		if err := s.requirePermission(ctx, actorID, PermissionArticlePublish); err != nil {
			return nil, err
		}
	}
	if err := s.articles.Update(ctx, desired); err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	return desired, nil
}

func (s *ArticleService) Delete(ctx context.Context, id uint) error {
	deleted, err := s.articles.Delete(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrArticleNotFound
	}
	s.invalidate(ctx)
	return nil
}

func (s *ArticleService) requirePermission(ctx context.Context, actorID uint, permission string) error {
	allowed, err := s.rbac.HasPermission(ctx, actorID, permission)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrArticleForbidden
	}
	return nil
}

func (s *ArticleService) invalidate(ctx context.Context) {
	if s.cache != nil {
		_ = s.cache.Del(context.WithoutCancel(ctx), homeService.CacheKey).Err()
	}
}

func validateArticle(article *model.Article) error {
	if article.Title == "" || article.Content == "" {
		return ErrInvalidArticleInput
	}
	if !validArticleType(article.Type) {
		return ErrInvalidArticleType
	}
	if !validEditorMode(article.EditorMode) {
		return ErrInvalidArticleEditor
	}
	return nil
}

func validArticleType(value string) bool {
	switch value {
	case model.TypeAnnouncement, model.TypeNews, model.TypeEvent, model.TypeUpdate:
		return true
	default:
		return false
	}
}

func validEditorMode(value string) bool {
	switch value {
	case model.EditorModePlain, model.EditorModeMarkdown:
		return true
	default:
		return false
	}
}

func articleFieldsChanged(current, desired *model.Article) bool {
	return current.Title != desired.Title || current.Summary != desired.Summary ||
		current.Content != desired.Content || current.EditorMode != desired.EditorMode ||
		current.CoverURL != desired.CoverURL ||
		current.Type != desired.Type || current.IsPinned != desired.IsPinned
}

func publicationTime(isPublished bool, requested, current *time.Time) *time.Time {
	if !isPublished {
		if requested != nil {
			return requested
		}
		return current
	}
	if requested != nil {
		return requested
	}
	if current != nil {
		return current
	}
	now := time.Now()
	return &now
}

func sameTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func articlePagination(page, limit int) (int, int) {
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}
	return page, limit
}
