package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrVerificationCooldown  = errors.New("verification code was sent too recently")
	ErrVerificationRateLimit = errors.New("verification code rate limit exceeded")
	ErrResetTicketNotFound   = errors.New("password reset ticket not found")
	ErrResetTicketCollision  = errors.New("password reset ticket collision")
)

var reserveVerificationCodeScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[2]) == 1 then
    return 1
end

local ipCurrent = tonumber(redis.call("GET", KEYS[3]) or "0")
if ipCurrent >= tonumber(ARGV[4]) then
    return 2
end

local redisTime = redis.call("TIME")
local now = tonumber(redisTime[1]) * 1000 + math.floor(tonumber(redisTime[2]) / 1000)
redis.call("ZREMRANGEBYSCORE", KEYS[4], "-inf", now - tonumber(ARGV[7]))
if redis.call("ZCARD", KEYS[4]) >= tonumber(ARGV[6]) then
    return 2
end

redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2])
redis.call("SET", KEYS[2], ARGV[8], "PX", ARGV[3])
redis.call("DEL", KEYS[5])
ipCurrent = redis.call("INCR", KEYS[3])
if ipCurrent == 1 then
    redis.call("PEXPIRE", KEYS[3], ARGV[5])
end
redis.call("ZADD", KEYS[4], now, ARGV[8])
redis.call("PEXPIRE", KEYS[4], ARGV[7])

return 0
`)

var cancelVerificationCodeScript = redis.NewScript(`
local value = redis.call("GET", KEYS[1])
if not value or string.sub(value, 1, string.len(ARGV[1]) + 1) ~= ARGV[1] .. ":" then
    return 0
end

if redis.call("GET", KEYS[2]) ~= ARGV[1] then
    return 0
end

redis.call("DEL", KEYS[1], KEYS[2], KEYS[3])
return 1
`)

var consumeVerificationCodeScript = redis.NewScript(`
local value = redis.call("GET", KEYS[1])
if not value then
    return 0
end

local separator = string.find(value, ":", 1, true)
if separator and string.sub(value, separator + 1) == ARGV[1] then
    redis.call("DEL", KEYS[1], KEYS[2])
    return 1
end

local ttl = redis.call("PTTL", KEYS[1])
if ttl <= 0 then
    redis.call("DEL", KEYS[1], KEYS[2])
    return 0
end
local attempts = redis.call("INCR", KEYS[2])
if attempts == 1 then
    redis.call("PEXPIRE", KEYS[2], ttl)
end
if attempts >= tonumber(ARGV[2]) then
    redis.call("DEL", KEYS[1], KEYS[2])
end
return 0
`)

var consumePasswordResetTicketScript = redis.NewScript(`
local value = redis.call("GET", KEYS[1])
if not value then
    return false
end
redis.call("DEL", KEYS[1])
return value
`)

type VerificationReservation struct {
	CodeKey     string
	CooldownKey string
	AttemptsKey string
	RequestID   string
}

type VerificationRepository struct {
	rdb *redis.Client
}

func NewVerificationRepository(rdb *redis.Client) *VerificationRepository {
	return &VerificationRepository{rdb: rdb}
}

func (r *VerificationRepository) ReserveVerificationCode(
	ctx context.Context,
	identifier string,
	purpose string,
	ip string,
	requestID string,
	digest string,
	codeTTL time.Duration,
	resendInterval time.Duration,
	ipWindow time.Duration,
	ipLimit int,
	identifierWindow time.Duration,
	identifierLimit int,
) (*VerificationReservation, error) {
	identifierHash := hashKeyPart(identifier)
	reservation := &VerificationReservation{
		CodeKey:     fmt.Sprintf("verification:code:%s:%s", purpose, identifierHash),
		CooldownKey: fmt.Sprintf("verification:cooldown:%s:%s", purpose, identifierHash),
		AttemptsKey: fmt.Sprintf("verification:attempts:%s:%s", purpose, identifierHash),
		RequestID:   requestID,
	}
	ipKey := fmt.Sprintf("verification:ip:%s", hashKeyPart(ip))
	identifierKey := fmt.Sprintf("verification:identifier:%s", identifierHash)

	result, err := reserveVerificationCodeScript.Run(
		ctx,
		r.rdb,
		[]string{reservation.CodeKey, reservation.CooldownKey, ipKey, identifierKey, reservation.AttemptsKey},
		requestID+":"+digest,
		codeTTL.Milliseconds(),
		resendInterval.Milliseconds(),
		ipLimit,
		ipWindow.Milliseconds(),
		identifierLimit,
		identifierWindow.Milliseconds(),
		requestID,
	).Int()
	if err != nil {
		return nil, fmt.Errorf("reserve verification code: %w", err)
	}

	switch result {
	case 0:
		return reservation, nil
	case 1:
		return nil, ErrVerificationCooldown
	case 2:
		return nil, ErrVerificationRateLimit
	default:
		return nil, fmt.Errorf("reserve verification code: unexpected result %d", result)
	}
}

func (r *VerificationRepository) CancelVerificationCode(
	ctx context.Context,
	reservation *VerificationReservation,
) error {
	if reservation == nil {
		return nil
	}

	if err := cancelVerificationCodeScript.Run(
		ctx,
		r.rdb,
		[]string{reservation.CodeKey, reservation.CooldownKey, reservation.AttemptsKey},
		reservation.RequestID,
	).Err(); err != nil {
		return fmt.Errorf("cancel verification code: %w", err)
	}
	return nil
}

func (r *VerificationRepository) IsVerificationCodeCurrent(
	ctx context.Context,
	identifier string,
	purpose string,
	requestID string,
) (bool, error) {
	key := fmt.Sprintf("verification:code:%s:%s", purpose, hashKeyPart(identifier))
	value, err := r.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get verification code reservation: %w", err)
	}
	prefix := requestID + ":"
	return requestID != "" && len(value) >= len(prefix) && value[:len(prefix)] == prefix, nil
}

func (r *VerificationRepository) ConsumeVerificationCode(
	ctx context.Context,
	identifier string,
	purpose string,
	digest string,
	maxAttempts int,
) (bool, error) {
	identifierHash := hashKeyPart(identifier)
	result, err := consumeVerificationCodeScript.Run(
		ctx,
		r.rdb,
		[]string{
			fmt.Sprintf("verification:code:%s:%s", purpose, identifierHash),
			fmt.Sprintf("verification:attempts:%s:%s", purpose, identifierHash),
		},
		digest,
		maxAttempts,
	).Int()
	if err != nil {
		return false, fmt.Errorf("consume verification code: %w", err)
	}
	return result == 1, nil
}

func (r *VerificationRepository) CreatePasswordResetTicket(
	ctx context.Context,
	token string,
	userID uint,
	ttl time.Duration,
) error {
	created, err := r.rdb.SetNX(
		ctx,
		passwordResetTicketKey(token),
		strconv.FormatUint(uint64(userID), 10),
		ttl,
	).Result()
	if err != nil {
		return fmt.Errorf("create password reset ticket: %w", err)
	}
	if !created {
		return ErrResetTicketCollision
	}
	return nil
}

func (r *VerificationRepository) ConsumePasswordResetTicket(ctx context.Context, token string) (uint, error) {
	value, err := consumePasswordResetTicketScript.Run(
		ctx,
		r.rdb,
		[]string{passwordResetTicketKey(token)},
	).Text()
	if errors.Is(err, redis.Nil) {
		return 0, ErrResetTicketNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("consume password reset ticket: %w", err)
	}
	userID, err := strconv.ParseUint(value, 10, 64)
	if err != nil || userID == 0 {
		return 0, ErrResetTicketNotFound
	}
	return uint(userID), nil
}

func passwordResetTicketKey(token string) string {
	return "password_reset_ticket:" + hashKeyPart(token)
}

func hashKeyPart(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
