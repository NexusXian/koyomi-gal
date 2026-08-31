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

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Transaction(ctx context.Context, fn func(tx *CommentRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&CommentRepository{db: tx})
	})
}

func (r *CommentRepository) Create(ctx context.Context, comment *model.Comment) error {
	if err := r.db.WithContext(ctx).Create(comment).Error; err != nil {
		return fmt.Errorf("create comment: %w", err)
	}
	return nil
}

func (r *CommentRepository) Update(ctx context.Context, comment *model.Comment) error {
	comment.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("id = ?", comment.ID).
		Updates(map[string]any{
			"content":    comment.Content,
			"updated_at": comment.UpdatedAt,
		}).Error
	if err != nil {
		return fmt.Errorf("update comment: %w", err)
	}
	return nil
}

// Delete removes the comment row and reports whether one was deleted; replies
// and likes are removed by foreign key ON DELETE CASCADE.
func (r *CommentRepository) Delete(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Delete(&model.Comment{}, id)
	if result.Error != nil {
		return false, fmt.Errorf("delete comment: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *CommentRepository) FindByID(ctx context.Context, id uint) (*model.Comment, error) {
	var comment model.Comment
	err := r.withAuthorName(r.db.WithContext(ctx)).First(&comment, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find comment by id: %w", err)
	}
	return &comment, nil
}

// withAuthorName joins the comment author's username for display.
func (r *CommentRepository) withAuthorName(query *gorm.DB) *gorm.DB {
	return query.
		Model(&model.Comment{}).
		Select("comments.*, authors.username AS author_name").
		Joins("LEFT JOIN users AS authors ON authors.id = comments.author_id")
}

// ListTopLevelByPost returns one page of top-level comments plus the total
// count of top-level comments for the post.
func (r *CommentRepository) ListTopLevelByPost(
	ctx context.Context,
	postID uint,
	page, limit int,
) ([]model.Comment, int64, error) {
	base := func() *gorm.DB {
		return r.db.WithContext(ctx).
			Model(&model.Comment{}).
			Where("post_id = ? AND parent_id IS NULL", postID)
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count top-level comments: %w", err)
	}

	var comments []model.Comment
	err := r.withAuthorName(base()).
		Order("comments.id ASC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&comments).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list top-level comments: %w", err)
	}
	return comments, total, nil
}

func (r *CommentRepository) CountRepliesByParentIDs(
	ctx context.Context,
	parentIDs []uint,
) (map[uint]int64, error) {
	counts := make(map[uint]int64, len(parentIDs))
	if len(parentIDs) == 0 {
		return counts, nil
	}
	var rows []struct {
		ParentID uint
		Count    int64
	}
	err := r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Select("parent_id, COUNT(*) AS count").
		Where("parent_id IN ?", parentIDs).
		Group("parent_id").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("count comment replies: %w", err)
	}
	for _, row := range rows {
		counts[row.ParentID] = row.Count
	}
	return counts, nil
}

func (r *CommentRepository) ListRepliesByParentID(
	ctx context.Context,
	parentID uint,
	page, limit int,
) ([]model.Comment, int64, error) {
	base := func() *gorm.DB {
		return r.db.WithContext(ctx).
			Model(&model.Comment{}).
			Where("parent_id = ?", parentID)
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count comment replies: %w", err)
	}
	var replies []model.Comment
	err := r.withAuthorName(base()).
		Order("comments.id ASC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&replies).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list comment replies: %w", err)
	}
	return replies, total, nil
}

// CountReplies returns the number of direct replies of a top-level comment;
// with only two levels this equals the subtree size minus one.
func (r *CommentRepository) CountReplies(ctx context.Context, parentID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("parent_id = ?", parentID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count comment replies: %w", err)
	}
	return count, nil
}

func (r *CommentRepository) IncrementPostCommentCount(ctx context.Context, postID uint) error {
	err := r.db.WithContext(ctx).Exec(
		"UPDATE posts SET comment_count = comment_count + 1 WHERE id = ?",
		postID,
	).Error
	if err != nil {
		return fmt.Errorf("increment post comment count: %w", err)
	}
	return nil
}

func (r *CommentRepository) DecrementPostCommentCountBy(ctx context.Context, postID uint, count int64) error {
	err := r.db.WithContext(ctx).Exec(
		"UPDATE posts SET comment_count = GREATEST(comment_count - ?, 0) WHERE id = ?",
		count, postID,
	).Error
	if err != nil {
		return fmt.Errorf("decrement post comment count: %w", err)
	}
	return nil
}

// AddCommentLike inserts the like and reports whether a new row was created;
// false means the user already liked the comment.
func (r *CommentRepository) AddCommentLike(ctx context.Context, commentID, userID uint) (bool, error) {
	like := model.CommentLike{CommentID: commentID, UserID: userID}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&like).Error
	if err != nil {
		return false, fmt.Errorf("add comment like: %w", err)
	}
	return like.ID != 0, nil
}

// RemoveCommentLike deletes the like row and reports whether one was deleted.
func (r *CommentRepository) RemoveCommentLike(ctx context.Context, commentID, userID uint) (bool, error) {
	result := r.db.WithContext(ctx).
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		Delete(&model.CommentLike{})
	if result.Error != nil {
		return false, fmt.Errorf("remove comment like: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *CommentRepository) IncrementCommentLikeCount(ctx context.Context, commentID uint) error {
	err := r.db.WithContext(ctx).Exec(
		"UPDATE comments SET like_count = like_count + 1 WHERE id = ?",
		commentID,
	).Error
	if err != nil {
		return fmt.Errorf("increment comment like count: %w", err)
	}
	return nil
}

func (r *CommentRepository) DecrementCommentLikeCount(ctx context.Context, commentID uint) error {
	err := r.db.WithContext(ctx).Exec(
		"UPDATE comments SET like_count = GREATEST(like_count - 1, 0) WHERE id = ?",
		commentID,
	).Error
	if err != nil {
		return fmt.Errorf("decrement comment like count: %w", err)
	}
	return nil
}
