package service

import (
	"context"
	"errors"

	"backend/internal/community/model"
	"backend/internal/community/repository"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

var (
	ErrAlreadyLiked     = errors.New("already liked")
	ErrLikeNotFound     = errors.New("like not found")
	ErrAlreadyFavorited = errors.New("post already favorited")
	ErrFavoriteNotFound = errors.New("post favorite not found")
)

// InteractionService covers post likes, post favorites, and comment likes.
// Unique indexes prevent duplicate relations and counters are updated with
// atomic SQL expressions inside one transaction per action.
type InteractionService struct {
	posts    *repository.PostRepository
	comments *repository.CommentRepository
}

func NewInteractionService(
	posts *repository.PostRepository,
	comments *repository.CommentRepository,
) *InteractionService {
	return &InteractionService{posts: posts, comments: comments}
}

func (s *InteractionService) LikePost(ctx context.Context, userID, postID uint) (*model.Post, error) {
	post, err := s.ensurePost(ctx, postID)
	if err != nil {
		return nil, err
	}
	err = s.posts.Transaction(ctx, func(tx *repository.PostRepository) error {
		inserted, err := tx.AddPostLike(ctx, postID, userID)
		if err != nil {
			return err
		}
		if !inserted {
			return ErrAlreadyLiked
		}
		return tx.IncrementPostLikeCount(ctx, postID)
	})
	if err != nil {
		if errors.Is(err, ErrAlreadyLiked) {
			return nil, err
		}
		logger.Error("like post", zap.Uint("post_id", postID), zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	return s.posts.FindByID(ctx, post.ID)
}

func (s *InteractionService) UnlikePost(ctx context.Context, userID, postID uint) (*model.Post, error) {
	if _, err := s.ensurePost(ctx, postID); err != nil {
		return nil, err
	}
	err := s.posts.Transaction(ctx, func(tx *repository.PostRepository) error {
		removed, err := tx.RemovePostLike(ctx, postID, userID)
		if err != nil {
			return err
		}
		if !removed {
			return ErrLikeNotFound
		}
		return tx.DecrementPostLikeCount(ctx, postID)
	})
	if err != nil {
		if errors.Is(err, ErrLikeNotFound) {
			return nil, err
		}
		logger.Error("unlike post", zap.Uint("post_id", postID), zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	return s.posts.FindByID(ctx, postID)
}

func (s *InteractionService) FavoritePost(ctx context.Context, userID, postID uint) (*model.Post, error) {
	post, err := s.ensurePost(ctx, postID)
	if err != nil {
		return nil, err
	}
	err = s.posts.Transaction(ctx, func(tx *repository.PostRepository) error {
		inserted, err := tx.AddPostFavorite(ctx, postID, userID)
		if err != nil {
			return err
		}
		if !inserted {
			return ErrAlreadyFavorited
		}
		return tx.IncrementPostFavoriteCount(ctx, postID)
	})
	if err != nil {
		if errors.Is(err, ErrAlreadyFavorited) {
			return nil, err
		}
		logger.Error("favorite post", zap.Uint("post_id", postID), zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	return s.posts.FindByID(ctx, post.ID)
}

func (s *InteractionService) UnfavoritePost(ctx context.Context, userID, postID uint) (*model.Post, error) {
	if _, err := s.ensurePost(ctx, postID); err != nil {
		return nil, err
	}
	err := s.posts.Transaction(ctx, func(tx *repository.PostRepository) error {
		removed, err := tx.RemovePostFavorite(ctx, postID, userID)
		if err != nil {
			return err
		}
		if !removed {
			return ErrFavoriteNotFound
		}
		return tx.DecrementPostFavoriteCount(ctx, postID)
	})
	if err != nil {
		if errors.Is(err, ErrFavoriteNotFound) {
			return nil, err
		}
		logger.Error("unfavorite post", zap.Uint("post_id", postID), zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	return s.posts.FindByID(ctx, postID)
}

func (s *InteractionService) LikeComment(ctx context.Context, userID, commentID uint) (*model.Comment, error) {
	comment, err := s.ensureComment(ctx, commentID)
	if err != nil {
		return nil, err
	}
	err = s.comments.Transaction(ctx, func(tx *repository.CommentRepository) error {
		inserted, err := tx.AddCommentLike(ctx, commentID, userID)
		if err != nil {
			return err
		}
		if !inserted {
			return ErrAlreadyLiked
		}
		return tx.IncrementCommentLikeCount(ctx, commentID)
	})
	if err != nil {
		if errors.Is(err, ErrAlreadyLiked) {
			return nil, err
		}
		logger.Error("like comment", zap.Uint("comment_id", commentID), zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	return s.comments.FindByID(ctx, comment.ID)
}

func (s *InteractionService) UnlikeComment(ctx context.Context, userID, commentID uint) (*model.Comment, error) {
	if _, err := s.ensureComment(ctx, commentID); err != nil {
		return nil, err
	}
	err := s.comments.Transaction(ctx, func(tx *repository.CommentRepository) error {
		removed, err := tx.RemoveCommentLike(ctx, commentID, userID)
		if err != nil {
			return err
		}
		if !removed {
			return ErrLikeNotFound
		}
		return tx.DecrementCommentLikeCount(ctx, commentID)
	})
	if err != nil {
		if errors.Is(err, ErrLikeNotFound) {
			return nil, err
		}
		logger.Error("unlike comment", zap.Uint("comment_id", commentID), zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	return s.comments.FindByID(ctx, commentID)
}

func (s *InteractionService) ensurePost(ctx context.Context, postID uint) (*model.Post, error) {
	post, err := s.posts.FindByID(ctx, postID)
	if err != nil {
		logger.Error("find post by id", zap.Uint("post_id", postID), zap.Error(err))
		return nil, err
	}
	if post == nil {
		return nil, ErrPostNotFound
	}
	return post, nil
}

func (s *InteractionService) ensureComment(ctx context.Context, commentID uint) (*model.Comment, error) {
	comment, err := s.comments.FindByID(ctx, commentID)
	if err != nil {
		logger.Error("find comment by id", zap.Uint("comment_id", commentID), zap.Error(err))
		return nil, err
	}
	if comment == nil {
		return nil, ErrCommentNotFound
	}
	return comment, nil
}
