package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
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

type RefreshSession struct {
	UserID      uint
	AuthVersion uint64
}

type RefreshSessionRepository struct {
	rdb *redis.Client
}

func NewRefreshSessionRepository(rdb *redis.Client) *RefreshSessionRepository {
	return &RefreshSessionRepository{rdb: rdb}
}

func (r *RefreshSessionRepository) Create(
	ctx context.Context,
	token string,
	session RefreshSession,
	ttl time.Duration,
) error {
	created, err := r.rdb.SetNX(
		ctx,
		refreshSessionKey(token),
		refreshSessionValue(session),
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

func (r *RefreshSessionRepository) Find(ctx context.Context, token string) (RefreshSession, error) {
	value, err := r.rdb.Get(ctx, refreshSessionKey(token)).Result()
	if errors.Is(err, redis.Nil) {
		return RefreshSession{}, ErrRefreshSessionNotFound
	}
	if err != nil {
		return RefreshSession{}, fmt.Errorf("find refresh session: %w", err)
	}

	session, err := parseRefreshSessionValue(value)
	if err != nil {
		return RefreshSession{}, fmt.Errorf("parse refresh session: %w", err)
	}
	return session, nil
}

func (r *RefreshSessionRepository) Rotate(
	ctx context.Context,
	currentToken string,
	replacementToken string,
	session RefreshSession,
	ttl time.Duration,
) error {
	result, err := rotateRefreshSessionScript.Run(
		ctx,
		r.rdb,
		[]string{refreshSessionKey(currentToken), refreshSessionKey(replacementToken)},
		refreshSessionValue(session),
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

func parseRefreshSessionValue(value string) (RefreshSession, error) {
	userIDValue, authVersionValue, hasVersion := strings.Cut(value, ":")
	userID, err := strconv.ParseUint(userIDValue, 10, 64)
	if err != nil || userID == 0 {
		return RefreshSession{}, errors.New("invalid user id")
	}
	session := RefreshSession{UserID: uint(userID)}
	if !hasVersion {
		return session, nil
	}
	if authVersionValue == "" {
		return RefreshSession{}, errors.New("invalid auth version")
	}
	authVersion, err := strconv.ParseUint(authVersionValue, 10, 64)
	if err != nil {
		return RefreshSession{}, errors.New("invalid auth version")
	}
	session.AuthVersion = authVersion
	return session, nil
}

func refreshSessionValue(session RefreshSession) string {
	return strconv.FormatUint(uint64(session.UserID), 10) + ":" +
		strconv.FormatUint(session.AuthVersion, 10)
}

func refreshSessionKey(token string) string {
	return "auth:refresh:" + hashKeyPart(token)
}
