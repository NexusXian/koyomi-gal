package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const privateMessageChannelPrefix = "private_message:user:"

type RedisPublisher struct {
	redis *redis.Client
}

func NewRedisPublisher(client *redis.Client) *RedisPublisher {
	return &RedisPublisher{redis: client}
}

func (p *RedisPublisher) Publish(ctx context.Context, userIDs []uint, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal realtime event: %w", err)
	}
	seen := make(map[uint]struct{}, len(userIDs))
	var publishErrors []error
	for _, userID := range userIDs {
		if userID == 0 {
			continue
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		if err := p.redis.Publish(ctx, userChannel(userID), payload).Err(); err != nil {
			publishErrors = append(publishErrors, fmt.Errorf("publish realtime event to user %d: %w", userID, err))
		}
	}
	return errors.Join(publishErrors...)
}

func userChannel(userID uint) string {
	return privateMessageChannelPrefix + strconv.FormatUint(uint64(userID), 10)
}
