package handler

import (
	"context"

	"backend/internal/community/dto"
	"backend/internal/community/service"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

// attachPostLevels fills the author level badges of a post list in one batch.
func attachPostLevels(ctx context.Context, levels service.LevelSummarizer, items []dto.PostData) {
	if levels == nil || len(items) == 0 {
		return
	}
	userIDs := make([]uint, 0, len(items))
	for i := range items {
		if items[i].Author != nil {
			userIDs = append(userIDs, items[i].Author.ID)
		}
	}
	summaries, err := levels.Summaries(ctx, userIDs)
	if err != nil {
		logger.Error("load post author levels", zap.Error(err))
		return
	}
	for i := range items {
		if items[i].Author == nil {
			continue
		}
		if summary, ok := summaries[items[i].Author.ID]; ok {
			summary := summary
			items[i].Author.Level = &summary
		}
	}
}

// attachCommentLevels fills the author and reply-to level badges of a comment
// list in one batch.
func attachCommentLevels(ctx context.Context, levels service.LevelSummarizer, items []dto.CommentData) {
	if levels == nil || len(items) == 0 {
		return
	}
	userIDs := make([]uint, 0, len(items)*2)
	for i := range items {
		if items[i].Author != nil {
			userIDs = append(userIDs, items[i].Author.ID)
		}
		if items[i].ReplyTo != nil {
			userIDs = append(userIDs, items[i].ReplyTo.ID)
		}
	}
	summaries, err := levels.Summaries(ctx, userIDs)
	if err != nil {
		logger.Error("load comment author levels", zap.Error(err))
		return
	}
	for i := range items {
		if items[i].Author != nil {
			if summary, ok := summaries[items[i].Author.ID]; ok {
				summary := summary
				items[i].Author.Level = &summary
			}
		}
		if items[i].ReplyTo != nil {
			if summary, ok := summaries[items[i].ReplyTo.ID]; ok {
				summary := summary
				items[i].ReplyTo.Level = &summary
			}
		}
	}
}
