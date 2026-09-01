package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/community/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostListOptions struct {
	GalgameID *uint
	Page      int
	Limit     int
}

type PostRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) Transaction(ctx context.Context, fn func(tx *PostRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&PostRepository{db: tx})
	})
}

func (r *PostRepository) Create(ctx context.Context, post *model.Post) error {
	if err := r.db.WithContext(ctx).Create(post).Error; err != nil {
		return fmt.Errorf("create post: %w", err)
	}
	return nil
}

func (r *PostRepository) Update(ctx context.Context, post *model.Post) error {
	post.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).
		Model(&model.Post{}).
		Where("id = ?", post.ID).
		Updates(map[string]any{
			"title":       post.Title,
			"content":     post.Content,
			"editor_mode": post.EditorMode,
			"updated_at":  post.UpdatedAt,
		}).Error
	if err != nil {
		return fmt.Errorf("update post: %w", err)
	}
	return nil
}

// Delete removes the post row and reports whether one was deleted; comments,
// likes, and favorites are removed by foreign key ON DELETE CASCADE.
func (r *PostRepository) Delete(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Delete(&model.Post{}, id)
	if result.Error != nil {
		return false, fmt.Errorf("delete post: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *PostRepository) FindByID(ctx context.Context, id uint) (*model.Post, error) {
	var post model.Post
	err := r.withNames(r.db.WithContext(ctx)).First(&post, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find post by id: %w", err)
	}
	return &post, nil
}

// withNames joins author usernames and galgame titles for display.
func (r *PostRepository) withNames(query *gorm.DB) *gorm.DB {
	return query.
		Model(&model.Post{}).
		Select(`posts.*, users.username AS author_name, users.avatar AS author_avatar,
galgames.title AS galgame_title, galgames.cover_url AS galgame_cover_url`).
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Joins("LEFT JOIN galgames ON galgames.id = posts.galgame_id")
}

func (r *PostRepository) ListHome(ctx context.Context, sort string, limit int) ([]model.Post, error) {
	order := "posts.created_at DESC"
	if sort == "popular" {
		order = "(posts.like_count * 3 + posts.comment_count * 5 + posts.favorite_count) DESC"
	}
	posts := make([]model.Post, 0)
	err := r.withNames(r.db.WithContext(ctx)).Order(order).Order("posts.id DESC").
		Limit(limit).Find(&posts).Error
	if err != nil {
		return nil, fmt.Errorf("list home posts: %w", err)
	}
	return posts, nil
}

func (r *PostRepository) List(ctx context.Context, options PostListOptions) ([]model.Post, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Post{})
	if options.GalgameID != nil {
		query = query.Where("galgame_id = ?", *options.GalgameID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count posts: %w", err)
	}

	var posts []model.Post
	err := r.withNames(query).
		Order("posts.id DESC").
		Offset((options.Page - 1) * options.Limit).
		Limit(options.Limit).
		Find(&posts).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list posts: %w", err)
	}
	return posts, total, nil
}

func (r *PostRepository) IncrementGalgamePostCount(ctx context.Context, galgameID uint) error {
	err := r.db.WithContext(ctx).Exec(
		"UPDATE galgames SET post_count = post_count + 1 WHERE id = ?",
		galgameID,
	).Error
	if err != nil {
		return fmt.Errorf("increment galgame post count: %w", err)
	}
	return nil
}

func (r *PostRepository) DecrementGalgamePostCount(ctx context.Context, galgameID uint) error {
	err := r.db.WithContext(ctx).Exec(
		"UPDATE galgames SET post_count = GREATEST(post_count - 1, 0) WHERE id = ?",
		galgameID,
	).Error
	if err != nil {
		return fmt.Errorf("decrement galgame post count: %w", err)
	}
	return nil
}

// AddPostLike inserts the like and reports whether a new row was created;
// false means the user already liked the post.
func (r *PostRepository) AddPostLike(ctx context.Context, postID, userID uint) (bool, error) {
	like := model.PostLike{PostID: postID, UserID: userID}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&like).Error
	if err != nil {
		return false, fmt.Errorf("add post like: %w", err)
	}
	return like.ID != 0, nil
}

// RemovePostLike deletes the like row and reports whether one was deleted.
func (r *PostRepository) RemovePostLike(ctx context.Context, postID, userID uint) (bool, error) {
	result := r.db.WithContext(ctx).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Delete(&model.PostLike{})
	if result.Error != nil {
		return false, fmt.Errorf("remove post like: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *PostRepository) IncrementPostLikeCount(ctx context.Context, postID uint) error {
	err := r.db.WithContext(ctx).Exec(
		"UPDATE posts SET like_count = like_count + 1 WHERE id = ?",
		postID,
	).Error
	if err != nil {
		return fmt.Errorf("increment post like count: %w", err)
	}
	return nil
}

func (r *PostRepository) DecrementPostLikeCount(ctx context.Context, postID uint) error {
	err := r.db.WithContext(ctx).Exec(
		"UPDATE posts SET like_count = GREATEST(like_count - 1, 0) WHERE id = ?",
		postID,
	).Error
	if err != nil {
		return fmt.Errorf("decrement post like count: %w", err)
	}
	return nil
}

// AddPostFavorite inserts the favorite and reports whether a new row was
// created; false means the user already favorited the post.
func (r *PostRepository) AddPostFavorite(ctx context.Context, postID, userID uint) (bool, error) {
	favorite := model.PostFavorite{PostID: postID, UserID: userID}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&favorite).Error
	if err != nil {
		return false, fmt.Errorf("add post favorite: %w", err)
	}
	return favorite.ID != 0, nil
}

// RemovePostFavorite deletes the favorite row and reports whether one was
// deleted.
func (r *PostRepository) RemovePostFavorite(ctx context.Context, postID, userID uint) (bool, error) {
	result := r.db.WithContext(ctx).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Delete(&model.PostFavorite{})
	if result.Error != nil {
		return false, fmt.Errorf("remove post favorite: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *PostRepository) IncrementPostFavoriteCount(ctx context.Context, postID uint) error {
	err := r.db.WithContext(ctx).Exec(
		"UPDATE posts SET favorite_count = favorite_count + 1 WHERE id = ?",
		postID,
	).Error
	if err != nil {
		return fmt.Errorf("increment post favorite count: %w", err)
	}
	return nil
}

func (r *PostRepository) DecrementPostFavoriteCount(ctx context.Context, postID uint) error {
	err := r.db.WithContext(ctx).Exec(
		"UPDATE posts SET favorite_count = GREATEST(favorite_count - 1, 0) WHERE id = ?",
		postID,
	).Error
	if err != nil {
		return fmt.Errorf("decrement post favorite count: %w", err)
	}
	return nil
}
