package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrMessageRateLimit      = errors.New("message rate limit exceeded")
	ErrConversationRateLimit = errors.New("conversation rate limit exceeded")
	ErrRateLimitUnavailable  = errors.New("message rate limiter unavailable")
)

var slidingWindowScript = redis.NewScript(`
local redisTime = redis.call("TIME")
local now = tonumber(redisTime[1]) * 1000 + math.floor(tonumber(redisTime[2]) / 1000)

for i = 1, #KEYS do
    local window = tonumber(ARGV[(i - 1) * 2 + 1])
    local limit = tonumber(ARGV[(i - 1) * 2 + 2])
    redis.call("ZREMRANGEBYSCORE", KEYS[i], "-inf", now - window)
    if redis.call("ZCARD", KEYS[i]) >= limit then
        return i
    end
end

local member = ARGV[#KEYS * 2 + 1]
for i = 1, #KEYS do
    local window = tonumber(ARGV[(i - 1) * 2 + 1])
    redis.call("ZADD", KEYS[i], now, member)
    redis.call("PEXPIRE", KEYS[i], window)
end
return 0
`)

type RateLimiter struct {
	redis *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{redis: client}
}

func (r *RateLimiter) AllowMessage(ctx context.Context, userID uint) error {
	keys := []string{
		fmt.Sprintf("messages:rate:{%d}:10s", userID),
		fmt.Sprintf("messages:rate:{%d}:1m", userID),
	}
	result, err := r.run(ctx, keys, []time.Duration{10 * time.Second, time.Minute}, []int{10, 60})
	if err != nil {
		return err
	}
	if result != 0 {
		return ErrMessageRateLimit
	}
	return nil
}

func (r *RateLimiter) AllowNewConversation(ctx context.Context, userID uint) error {
	keys := []string{fmt.Sprintf("messages:conversation-rate:{%d}:1h", userID)}
	result, err := r.run(ctx, keys, []time.Duration{time.Hour}, []int{10})
	if err != nil {
		return err
	}
	if result != 0 {
		return ErrConversationRateLimit
	}
	return nil
}

func (r *RateLimiter) run(
	ctx context.Context,
	keys []string,
	windows []time.Duration,
	limits []int,
) (int, error) {
	if r == nil || r.redis == nil {
		return 0, ErrRateLimitUnavailable
	}
	args := make([]any, 0, len(keys)*2+1)
	for i := range keys {
		args = append(args, windows[i].Milliseconds(), limits[i])
	}
	args = append(args, strconv.FormatInt(time.Now().UnixNano(), 10)+":"+uuid.NewString())
	result, err := slidingWindowScript.Run(ctx, r.redis, keys, args...).Int()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrRateLimitUnavailable, err)
	}
	return result, nil
}
