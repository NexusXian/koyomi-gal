package service

import (
	"context"
	"errors"
	"strings"

	"backend/internal/community/dto"
	"backend/internal/community/model"
	"backend/internal/community/repository"
	galgameRepository "backend/internal/galgame/repository"
	rbacService "backend/internal/rbac/service"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

var (
	ErrGalgameNotFound  = errors.New("galgame not found")
	ErrPostNotFound     = errors.New("post not found")
	ErrForbiddenPost    = errors.New("not allowed to manage this post")
	ErrInvalidPostInput = errors.New("invalid post input")
)

// PermissionPostModerate manages posts authored by other users.
const PermissionPostModerate = "post:moderate"

type PostService struct {
	posts    *repository.PostRepository
	galgames *galgameRepository.GalgameRepository
	rbac     *rbacService.RBACService
}

func NewPostService(
	posts *repository.PostRepository,
	galgames *galgameRepository.GalgameRepository,
	rbac *rbacService.RBACService,
) *PostService {
	return &PostService{posts: posts, galgames: galgames, rbac: rbac}
}

// Create inserts a post; when it discusses a galgame, galgames.post_count is
// atomically incremented in the same transaction.
func (s *PostService) Create(
	ctx context.Context,
	authorID uint,
	req *dto.CreatePostRequest,
) (*model.Post, error) {
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if title == "" || content == "" {
		return nil, ErrInvalidPostInput
	}
	editorMode := model.EditorModePlain
	if req.EditorMode != "" {
		editorMode = req.EditorMode
	}
	if !model.IsValidEditorMode(editorMode) {
		return nil, ErrInvalidPostInput
	}
	if req.GalgameID != nil {
		if err := s.ensurePublishedGalgame(ctx, *req.GalgameID); err != nil {
			return nil, err
		}
	}

	post := &model.Post{
		GalgameID:  req.GalgameID,
		AuthorID:   &authorID,
		Title:      title,
		Content:    content,
		EditorMode: editorMode,
	}
	err := s.posts.Transaction(ctx, func(tx *repository.PostRepository) error {
		if err := tx.Create(ctx, post); err != nil {
			return err
		}
		if req.GalgameID != nil {
			return tx.IncrementGalgamePostCount(ctx, *req.GalgameID)
		}
		return nil
	})
	if err != nil {
		logger.Error("create post", zap.Uint("author_id", authorID), zap.Error(err))
		return nil, err
	}
	return s.posts.FindByID(ctx, post.ID)
}

func (s *PostService) Get(ctx context.Context, id uint) (*model.Post, error) {
	post, err := s.posts.FindByID(ctx, id)
	if err != nil {
		logger.Error("find post by id", zap.Uint("post_id", id), zap.Error(err))
		return nil, err
	}
	if post == nil {
		return nil, ErrPostNotFound
	}
	return post, nil
}

func (s *PostService) List(
	ctx context.Context,
	query *dto.PostQuery,
) ([]model.Post, int64, int, int, error) {
	page := query.Page
	if page == 0 {
		page = 1
	}
	limit := query.Limit
	if limit == 0 {
		limit = 20
	}
	posts, total, err := s.posts.List(ctx, repository.PostListOptions{
		GalgameID: query.GalgameID,
		Page:      page,
		Limit:     limit,
	})
	if err != nil {
		logger.Error("list posts", zap.Error(err))
		return nil, 0, page, limit, err
	}
	return posts, total, page, limit, nil
}

func (s *PostService) ListAdmin(
	ctx context.Context,
	query *dto.AdminCommunityQuery,
) ([]model.Post, int64, int, int, error) {
	page, limit := communityAdminPagination(query.Page, query.Limit)
	posts, total, err := s.posts.ListAdmin(ctx, repository.AdminCommunityListOptions{
		Keyword: strings.TrimSpace(query.Keyword), Page: page, Limit: limit,
	})
	if err != nil {
		logger.Error("list admin posts", zap.Error(err))
	}
	return posts, total, page, limit, err
}

func communityAdminPagination(page, limit int) (int, int) {
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

// Update replaces title and content. The actor must be the author or hold
// the post:moderate permission.
func (s *PostService) Update(
	ctx context.Context,
	actorID, id uint,
	req *dto.UpdatePostRequest,
) (*model.Post, error) {
	post, err := s.posts.FindByID(ctx, id)
	if err != nil {
		logger.Error("find post by id", zap.Uint("post_id", id), zap.Error(err))
		return nil, err
	}
	if post == nil {
		return nil, ErrPostNotFound
	}
	if err := s.ensureCanManage(ctx, actorID, post); err != nil {
		return nil, err
	}

	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if title == "" || content == "" {
		return nil, ErrInvalidPostInput
	}
	// An omitted editor mode keeps the stored value so legacy clients cannot
	// accidentally downgrade a markdown post to plain.
	editorMode := post.EditorMode
	if req.EditorMode != "" {
		editorMode = req.EditorMode
	}
	if !model.IsValidEditorMode(editorMode) {
		return nil, ErrInvalidPostInput
	}
	post.Title = title
	post.Content = content
	post.EditorMode = editorMode
	if err := s.posts.Update(ctx, post); err != nil {
		logger.Error("update post", zap.Uint("post_id", id), zap.Uint("actor_id", actorID), zap.Error(err))
		return nil, err
	}
	return s.posts.FindByID(ctx, id)
}

// Delete removes the post and atomically decrements galgames.post_count when
// the post discussed a galgame. The actor must be the author or hold the
// post:moderate permission.
func (s *PostService) Delete(ctx context.Context, actorID, id uint) error {
	post, err := s.posts.FindByID(ctx, id)
	if err != nil {
		logger.Error("find post by id", zap.Uint("post_id", id), zap.Error(err))
		return err
	}
	if post == nil {
		return ErrPostNotFound
	}
	if err := s.ensureCanManage(ctx, actorID, post); err != nil {
		return err
	}

	err = s.posts.Transaction(ctx, func(tx *repository.PostRepository) error {
		removed, err := tx.Delete(ctx, id)
		if err != nil {
			return err
		}
		if !removed {
			return ErrPostNotFound
		}
		if post.GalgameID != nil {
			return tx.DecrementGalgamePostCount(ctx, *post.GalgameID)
		}
		return nil
	})
	if err != nil {
		logger.Error("delete post", zap.Uint("post_id", id), zap.Uint("actor_id", actorID), zap.Error(err))
		return err
	}
	return nil
}

// ensureCanManage allows authors to manage their own posts and falls back to
// the post:moderate permission for everyone else.
func (s *PostService) ensureCanManage(ctx context.Context, actorID uint, post *model.Post) error {
	if post.AuthorID != nil && *post.AuthorID == actorID {
		return nil
	}
	allowed, err := s.rbac.HasPermission(ctx, actorID, PermissionPostModerate)
	if err != nil {
		logger.Error("check post permission",
			zap.String("permission", PermissionPostModerate), zap.Uint("actor_id", actorID), zap.Error(err))
		return err
	}
	if !allowed {
		return ErrForbiddenPost
	}
	return nil
}

func (s *PostService) ensurePublishedGalgame(ctx context.Context, id uint) error {
	galgame, err := s.galgames.FindPublishedByID(ctx, id)
	if err != nil {
		logger.Error("find galgame by id", zap.Uint("galgame_id", id), zap.Error(err))
		return err
	}
	if galgame == nil {
		return ErrGalgameNotFound
	}
	return nil
}
