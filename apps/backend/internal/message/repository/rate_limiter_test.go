package repository

import (
	"context"
	"errors"
	"testing"

	"backend/internal/testutil"
)

func TestRateLimiterFailsClosedWithoutRedis(t *testing.T) {
	limiter := NewRateLimiter(nil)
	if err := limiter.AllowMessage(context.Background(), 1001); !errors.Is(err, ErrRateLimitUnavailable) {
		t.Fatalf("expected unavailable message limiter, got %v", err)
	}
	if err := limiter.AllowNewConversation(context.Background(), 1001); !errors.Is(err, ErrRateLimitUnavailable) {
		t.Fatalf("expected unavailable conversation limiter, got %v", err)
	}
}

func TestRateLimiterEnforcesMessageAndConversationLimits(t *testing.T) {
	client := testutil.NewRedis(t)
	limiter := NewRateLimiter(client)
	ctx := context.Background()

	for range 10 {
		if err := limiter.AllowMessage(ctx, 1001); err != nil {
			t.Fatalf("allow message within limit: %v", err)
		}
	}
	if err := limiter.AllowMessage(ctx, 1001); !errors.Is(err, ErrMessageRateLimit) {
		t.Fatalf("expected message rate limit, got %v", err)
	}

	for range 10 {
		if err := limiter.AllowNewConversation(ctx, 2002); err != nil {
			t.Fatalf("allow conversation within limit: %v", err)
		}
	}
	if err := limiter.AllowNewConversation(ctx, 2002); !errors.Is(err, ErrConversationRateLimit) {
		t.Fatalf("expected conversation rate limit, got %v", err)
	}
}
