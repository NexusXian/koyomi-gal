package ipgeo

import (
	"context"
	"errors"
	"time"

	"backend/pkg/logger"

	"go.uber.org/zap"
)

type LogQueue interface {
	EnqueueIPLog(ctx context.Context, entry UserIPLog) error
}

type ReliableEnqueuer struct {
	queue    LogQueue
	fallback *AuditService
}

func NewReliableEnqueuer(queue LogQueue, fallback *AuditService) *ReliableEnqueuer {
	return &ReliableEnqueuer{queue: queue, fallback: fallback}
}

func (e *ReliableEnqueuer) EnqueueIPLog(ctx context.Context, entry UserIPLog) error {
	if err := e.queue.EnqueueIPLog(ctx, entry); err != nil {
		logger.Warn("enqueue IP audit; writing synchronously", zap.Error(err))
		fallbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		if fallbackErr := e.fallback.Record(fallbackCtx, entry); fallbackErr != nil {
			return errors.Join(err, fallbackErr)
		}
	}
	return nil
}
