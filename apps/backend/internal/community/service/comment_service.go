package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"backend/internal/community/dto"
	"backend/internal/community/model"
	"backend/internal/community/repository"
	notificationModel "backend/internal/notification/model"
	notificationService "backend/internal/notification/service"
	rbacService "backend/internal/rbac/service"
	userModel "backend/internal/user/model"
	userService "backend/internal/user/service"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

var (
	ErrCommentNotFound       = errors.New("comment not found")
	ErrForbiddenComment      = errors.New("not allowed to manage this comment")
	ErrInvalidCommentInput   = errors.New("invalid comment input")
	ErrInvalidCommentParent  = errors.New("comment parent must be a top-level comment of the same post")
	ErrInvalidCommentReplyTo = errors.New("reply target must belong to the same comment thread")
	ErrCommentNotTopLevel    = errors.New("comment is not top-level")
)

// PermissionCommentModerate manages comments authored by other users.
const PermissionCommentModerate = "comment:moderate"

type CommentService struct {
	comments      *repository.CommentRepository
	posts         *repository.PostRepository
	rbac          *rbacService.RBACService
	notifications *notificationService.NotificationService
	activities    userService.ActivityRecorder
}

func (s *CommentService) SetActivityRecorder(recorder userService.ActivityRecorder) {
	s.activities = recorder
}

func NewCommentService(
	comments *repository.CommentRepository,
	posts *repository.PostRepository,
	rbac *rbacService.RBACService,
	notifications ...*notificationService.NotificationService,
) *CommentService {
	service := &CommentService{comments: comments, posts: posts, rbac: rbac}
	if len(notifications) > 0 {
		service.notifications = notifications[0]
	}
	return service
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
	created, err := s.comments.FindByID(ctx, comment.ID)
	if err != nil {
		return nil, err
	}
	s.notifyCreated(ctx, authorID, post, created)
	if created != nil && s.activities != nil {
		metadata := map[string]any{"post_id": post.ID, "post_title": post.Title}
		if recordErr := s.activities.Record(ctx, authorID, userModel.ActivityCommentCreated, &created.ID, metadata); recordErr != nil {
			logger.Error("record comment activity", zap.Uint("comment_id", created.ID), zap.Error(recordErr))
		}
	}
	return created, nil
}

// ListByPost returns one page of top-level comments and their reply counts.
func (s *CommentService) ListByPost(
	ctx context.Context,
	postID uint,
	page, limit int,
) ([]model.Comment, map[uint]int64, int64, int, int, error) {
	post, err := s.posts.FindByID(ctx, postID)
	if err != nil {
		logger.Error("find post by id", zap.Uint("post_id", postID), zap.Error(err))
		return nil, nil, 0, page, limit, err
	}
	if post == nil {
		return nil, nil, 0, page, limit, ErrPostNotFound
	}
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}

	topLevel, total, err := s.comments.ListTopLevelByPost(ctx, postID, page, limit)
	if err != nil {
		logger.Error("list top-level comments", zap.Uint("post_id", postID), zap.Error(err))
		return nil, nil, 0, page, limit, err
	}
	parentIDs := make([]uint, 0, len(topLevel))
	for i := range topLevel {
		parentIDs = append(parentIDs, topLevel[i].ID)
	}
	replyCounts, err := s.comments.CountRepliesByParentIDs(ctx, parentIDs)
	if err != nil {
		logger.Error("count comment replies", zap.Uint("post_id", postID), zap.Error(err))
		return nil, nil, 0, page, limit, err
	}
	return topLevel, replyCounts, total, page, limit, nil
}

func (s *CommentService) ListReplies(
	ctx context.Context,
	parentID uint,
	page, limit int,
) ([]model.Comment, int64, int, int, error) {
	parent, err := s.comments.FindByID(ctx, parentID)
	if err != nil {
		logger.Error("find parent comment", zap.Uint("comment_id", parentID), zap.Error(err))
		return nil, 0, page, limit, err
	}
	if parent == nil {
		return nil, 0, page, limit, ErrCommentNotFound
	}
	if parent.ParentID != nil {
		return nil, 0, page, limit, ErrCommentNotTopLevel
	}
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}
	replies, total, err := s.comments.ListRepliesByParentID(ctx, parentID, page, limit)
	if err != nil {
		logger.Error("list comment replies", zap.Uint("comment_id", parentID), zap.Error(err))
		return nil, 0, page, limit, err
	}
	return replies, total, page, limit, nil
}

func (s *CommentService) ListAdmin(
	ctx context.Context,
	query *dto.AdminCommunityQuery,
) ([]model.Comment, int64, int, int, error) {
	page, limit := communityAdminPagination(query.Page, query.Limit)
	comments, total, err := s.comments.ListAdmin(ctx, repository.AdminCommunityListOptions{
		Keyword: strings.TrimSpace(query.Keyword), Page: page, Limit: limit,
	})
	if err != nil {
		logger.Error("list admin comments", zap.Error(err))
	}
	return comments, total, page, limit, err
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
	if comment.AuthorID != nil && *comment.AuthorID != actorID {
		s.notify(ctx, notificationService.CreateInput{
			RecipientID: *comment.AuthorID,
			ActorID:     &actorID,
			Category:    notificationModel.CategoryModeration,
			Type:        notificationModel.TypeCommentModerated,
			EntityType:  "comment",
			EntityID:    id,
			Title:       "评论已被处理",
			Content:     "你的评论已被管理员删除",
			TargetURL:   fmt.Sprintf("/posts/%d", comment.PostID),
			Metadata:    map[string]any{"preview": previewText(comment.Content)},
		})
	}
	return nil
}

func (s *CommentService) notifyCreated(ctx context.Context, actorID uint, post *model.Post, comment *model.Comment) {
	if s.notifications == nil || comment == nil {
		return
	}
	recipientID := post.AuthorID
	notificationType := notificationModel.TypePostCommented
	title := "新的评论"
	content := fmt.Sprintf("评论了你的帖子「%s」", post.Title)
	if comment.ReplyToUserID != nil {
		recipientID = comment.ReplyToUserID
		notificationType = notificationModel.TypeCommentReplied
		title = "新的回复"
		content = "回复了你的评论"
	}
	if recipientID == nil {
		return
	}
	s.notify(ctx, notificationService.CreateInput{
		RecipientID: *recipientID,
		ActorID:     &actorID,
		Category:    notificationModel.CategoryInteraction,
		Type:        notificationType,
		EntityType:  "comment",
		EntityID:    comment.ID,
		Title:       title,
		Content:     content,
		TargetURL:   fmt.Sprintf("/posts/%d?comment=%d", post.ID, comment.ID),
		Metadata:    map[string]any{"preview": previewText(comment.Content), "post_id": post.ID},
	})
}

func (s *CommentService) notify(ctx context.Context, input notificationService.CreateInput) {
	if s.notifications == nil {
		return
	}
	if _, err := s.notifications.Create(ctx, input); err != nil {
		logger.Error("create comment notification", zap.String("type", string(input.Type)), zap.Error(err))
	}
}

func previewText(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= 100 {
		return value
	}
	return string(runes[:100]) + "…"
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
