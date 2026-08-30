package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrRefreshSessionNotFound = errors.New("refresh session not found")
	ErrRefreshTokenCollision  = errors.New("refresh token collision")
)

var rotateRefreshSessionScript = redis.NewScript(`
local value = redis.call("GET", KEYS[1])
if not value or value ~= ARGV[1] then
    return 0
end
if redis.call("EXISTS", KEYS[2]) == 1 then
    return 2
end
redis.call("SET", KEYS[2], value, "PX", ARGV[2])
redis.call("DEL", KEYS[1])
return 1
`)

type RefreshSessionRepository struct {
	rdb *redis.Client
}

func NewRefreshSessionRepository(rdb *redis.Client) *RefreshSessionRepository {
	return &RefreshSessionRepository{rdb: rdb}
}

func (r *RefreshSessionRepository) Create(
	ctx context.Context,
	token string,
	userID uint,
	ttl time.Duration,
) error {
	created, err := r.rdb.SetNX(
		ctx,
		refreshSessionKey(token),
		strconv.FormatUint(uint64(userID), 10),
		ttl,
	).Result()
	if err != nil {
		return fmt.Errorf("create refresh session: %w", err)
	}
	if !created {
		return ErrRefreshTokenCollision
	}
	return nil
}

func (r *RefreshSessionRepository) FindUserID(ctx context.Context, token string) (uint, error) {
	value, err := r.rdb.Get(ctx, refreshSessionKey(token)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, ErrRefreshSessionNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("find refresh session: %w", err)
	}

	userID, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse refresh session user id: %w", err)
	}
	return uint(userID), nil
}

func (r *RefreshSessionRepository) Rotate(
	ctx context.Context,
	currentToken string,
	replacementToken string,
	userID uint,
	ttl time.Duration,
) error {
	result, err := rotateRefreshSessionScript.Run(
		ctx,
		r.rdb,
		[]string{refreshSessionKey(currentToken), refreshSessionKey(replacementToken)},
		strconv.FormatUint(uint64(userID), 10),
		ttl.Milliseconds(),
	).Int()
	if err != nil {
		return fmt.Errorf("rotate refresh session: %w", err)
	}

	switch result {
	case 1:
		return nil
	case 0:
		return ErrRefreshSessionNotFound
	case 2:
		return ErrRefreshTokenCollision
	default:
		return fmt.Errorf("rotate refresh session: unexpected result %d", result)
	}
}

func (r *RefreshSessionRepository) Revoke(ctx context.Context, token string) error {
	if err := r.rdb.Del(ctx, refreshSessionKey(token)).Err(); err != nil {
		return fmt.Errorf("revoke refresh session: %w", err)
	}
	return nil
}

func refreshSessionKey(token string) string {
	return "auth:refresh:" + hashKeyPart(token)
}
