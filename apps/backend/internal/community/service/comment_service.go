package service

import (
	"context"
	"errors"
	"strings"

	"backend/internal/community/dto"
	"backend/internal/community/model"
	"backend/internal/community/repository"
	rbacService "backend/internal/rbac/service"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

var (
	ErrCommentNotFound       = errors.New("comment not found")
	ErrForbiddenComment      = errors.New("not allowed to manage this comment")
	ErrInvalidCommentInput   = errors.New("invalid comment input")
	ErrInvalidCommentParent  = errors.New("comment parent must be a top-level comment of the same post")
	ErrInvalidCommentReplyTo = errors.New("reply target must belong to the same comment thread")
)

// PermissionCommentModerate manages comments authored by other users.
const PermissionCommentModerate = "comment:moderate"

type CommentService struct {
	comments *repository.CommentRepository
	posts    *repository.PostRepository
	rbac     *rbacService.RBACService
}

func NewCommentService(
	comments *repository.CommentRepository,
	posts *repository.PostRepository,
	rbac *rbacService.RBACService,
) *CommentService {
	return &CommentService{comments: comments, posts: posts, rbac: rbac}
}

// Create inserts a comment and atomically increments posts.comment_count.
// parent_id must reference a top-level comment of the same post; when
// reply_to_comment_id targets another reply, its author is stored as
// reply_to_user_id so clients never submit user IDs.
func (s *CommentService) Create(
	ctx context.Context,
	authorID, postID uint,
	req *dto.CreateCommentRequest,
) (*model.Comment, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, ErrInvalidCommentInput
	}
	post, err := s.posts.FindByID(ctx, postID)
	if err != nil {
		logger.Error("find post by id", zap.Uint("post_id", postID), zap.Error(err))
		return nil, err
	}
	if post == nil {
		return nil, ErrPostNotFound
	}

	var replyToUserID *uint
	if req.ReplyToCommentID != nil {
		replyTo, err := s.comments.FindByID(ctx, *req.ReplyToCommentID)
		if err != nil {
			logger.Error("find reply target", zap.Uint("comment_id", *req.ReplyToCommentID), zap.Error(err))
			return nil, err
		}
		if replyTo == nil || replyTo.PostID != postID {
			return nil, ErrInvalidCommentReplyTo
		}
		rootID := replyTo.ID
		if replyTo.ParentID != nil {
			rootID = *replyTo.ParentID
		}
		if req.ParentID != nil && *req.ParentID != rootID {
			return nil, ErrInvalidCommentReplyTo
		}
		replyToUserID = replyTo.AuthorID
		if req.ParentID == nil {
			parentID := rootID
			req.ParentID = &parentID
		}
	}
	if req.ParentID != nil {
		parent, err := s.comments.FindByID(ctx, *req.ParentID)
		if err != nil {
			logger.Error("find parent comment", zap.Uint("comment_id", *req.ParentID), zap.Error(err))
			return nil, err
		}
		if parent == nil || parent.PostID != postID || parent.ParentID != nil {
			return nil, ErrInvalidCommentParent
		}
	}

	comment := &model.Comment{
		PostID:        postID,
		AuthorID:      &authorID,
		ParentID:      req.ParentID,
		ReplyToUserID: replyToUserID,
		Content:       content,
	}
	err = s.comments.Transaction(ctx, func(tx *repository.CommentRepository) error {
		if err := tx.Create(ctx, comment); err != nil {
			return err
		}
		return tx.IncrementPostCommentCount(ctx, postID)
	})
	if err != nil {
		logger.Error("create comment", zap.Uint("post_id", postID), zap.Uint("author_id", authorID), zap.Error(err))
		return nil, err
	}
	return s.comments.FindByID(ctx, comment.ID)
}

// ListByPost returns one page of top-level comments, each with its replies
// loaded in a single query.
func (s *CommentService) ListByPost(
	ctx context.Context,
	postID uint,
	page, limit int,
) ([]repository.CommentThread, int64, int, int, error) {
	post, err := s.posts.FindByID(ctx, postID)
	if err != nil {
		logger.Error("find post by id", zap.Uint("post_id", postID), zap.Error(err))
		return nil, 0, page, limit, err
	}
	if post == nil {
		return nil, 0, page, limit, ErrPostNotFound
	}

	topLevel, total, err := s.comments.ListTopLevelByPost(ctx, postID, page, limit)
	if err != nil {
		logger.Error("list top-level comments", zap.Uint("post_id", postID), zap.Error(err))
		return nil, 0, page, limit, err
	}
	parentIDs := make([]uint, 0, len(topLevel))
	for i := range topLevel {
		parentIDs = append(parentIDs, topLevel[i].ID)
	}
	replies, err := s.comments.ListRepliesByParentIDs(ctx, parentIDs)
	if err != nil {
		logger.Error("list comment replies", zap.Uint("post_id", postID), zap.Error(err))
		return nil, 0, page, limit, err
	}

	threads := make([]repository.CommentThread, 0, len(topLevel))
	for i := range topLevel {
		thread := repository.CommentThread{Comment: topLevel[i]}
		if threadReplies, ok := replies[topLevel[i].ID]; ok {
			thread.Replies = threadReplies
		} else {
			thread.Replies = []model.Comment{}
		}
		threads = append(threads, thread)
	}
	return threads, total, page, limit, nil
}

// Update replaces the comment content. The actor must be the author or hold
// the comment:moderate permission.
func (s *CommentService) Update(
	ctx context.Context,
	actorID, id uint,
	req *dto.UpdateCommentRequest,
) (*model.Comment, error) {
	comment, err := s.comments.FindByID(ctx, id)
	if err != nil {
		logger.Error("find comment by id", zap.Uint("comment_id", id), zap.Error(err))
		return nil, err
	}
	if comment == nil {
		return nil, ErrCommentNotFound
	}
	if err := s.ensureCanManage(ctx, actorID, comment); err != nil {
		return nil, err
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, ErrInvalidCommentInput
	}
	comment.Content = content
	if err := s.comments.Update(ctx, comment); err != nil {
		logger.Error("update comment", zap.Uint("comment_id", id), zap.Uint("actor_id", actorID), zap.Error(err))
		return nil, err
	}
	return s.comments.FindByID(ctx, id)
}

// Delete removes the comment (replies cascade) and atomically decrements
// posts.comment_count by the removed subtree size. The actor must be the
// author or hold the comment:moderate permission.
func (s *CommentService) Delete(ctx context.Context, actorID, id uint) error {
	comment, err := s.comments.FindByID(ctx, id)
	if err != nil {
		logger.Error("find comment by id", zap.Uint("comment_id", id), zap.Error(err))
		return err
	}
	if comment == nil {
		return ErrCommentNotFound
	}
	if err := s.ensureCanManage(ctx, actorID, comment); err != nil {
		return err
	}

	removedCount := int64(1)
	if comment.ParentID == nil {
		replies, err := s.comments.CountReplies(ctx, id)
		if err != nil {
			logger.Error("count comment replies", zap.Uint("comment_id", id), zap.Error(err))
			return err
		}
		removedCount += replies
	}
	err = s.comments.Transaction(ctx, func(tx *repository.CommentRepository) error {
		removed, err := tx.Delete(ctx, id)
		if err != nil {
			return err
		}
		if !removed {
			return ErrCommentNotFound
		}
		return tx.DecrementPostCommentCountBy(ctx, comment.PostID, removedCount)
	})
	if err != nil {
		logger.Error("delete comment", zap.Uint("comment_id", id), zap.Uint("actor_id", actorID), zap.Error(err))
		return err
	}
	return nil
}

// ensureCanManage allows authors to manage their own comments and falls back
// to the comment:moderate permission for everyone else.
func (s *CommentService) ensureCanManage(ctx context.Context, actorID uint, comment *model.Comment) error {
	if comment.AuthorID != nil && *comment.AuthorID == actorID {
		return nil
	}
	allowed, err := s.rbac.HasPermission(ctx, actorID, PermissionCommentModerate)
	if err != nil {
		logger.Error("check comment permission",
			zap.String("permission", PermissionCommentModerate), zap.Uint("actor_id", actorID), zap.Error(err))
		return err
	}
	if !allowed {
		return ErrForbiddenComment
	}
	return nil
}
