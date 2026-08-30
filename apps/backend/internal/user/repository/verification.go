package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrVerificationCooldown  = errors.New("verification code was sent too recently")
	ErrVerificationRateLimit = errors.New("verification code rate limit exceeded")
)

var reserveVerificationCodeScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[2]) == 1 then
    return 1
end

local current = tonumber(redis.call("GET", KEYS[3]) or "0")
if current >= tonumber(ARGV[4]) then
    return 2
end

redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2])
redis.call("SET", KEYS[2], "1", "PX", ARGV[3])
current = redis.call("INCR", KEYS[3])
if current == 1 then
    redis.call("PEXPIRE", KEYS[3], ARGV[5])
end

return 0
`)

var cancelVerificationCodeScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) ~= ARGV[1] then
    return 0
end

redis.call("DEL", KEYS[1], KEYS[2])
return 1
`)

type VerificationReservation struct {
	CodeKey     string
	CooldownKey string
	Value       string
}

type VerificationRepository struct {
	rdb *redis.Client
}

func NewVerificationRepository(rdb *redis.Client) *VerificationRepository {
	return &VerificationRepository{rdb: rdb}
}

func (r *VerificationRepository) ReserveVerificationCode(
	ctx context.Context,
	email string,
	purpose string,
	ip string,
	requestID string,
	digest string,
	codeTTL time.Duration,
	resendInterval time.Duration,
	ipWindow time.Duration,
	ipLimit int,
) (*VerificationReservation, error) {
	emailHash := hashKeyPart(email)
	ipHash := hashKeyPart(ip)
	reservation := &VerificationReservation{
		CodeKey:     fmt.Sprintf("verification:code:%s:%s", purpose, emailHash),
		CooldownKey: fmt.Sprintf("verification:cooldown:%s:%s", purpose, emailHash),
		Value:       requestID + ":" + digest,
	}
	ipKey := fmt.Sprintf("verification:ip:%s", ipHash)

	result, err := reserveVerificationCodeScript.Run(
		ctx,
		r.rdb,
		[]string{reservation.CodeKey, reservation.CooldownKey, ipKey},
		reservation.Value,
		codeTTL.Milliseconds(),
		resendInterval.Milliseconds(),
		ipLimit,
		ipWindow.Milliseconds(),
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
		[]string{reservation.CodeKey, reservation.CooldownKey},
		reservation.Value,
	).Err(); err != nil {
		return fmt.Errorf("cancel verification code: %w", err)
	}
	return nil
}

func (r *VerificationRepository) IsVerificationCodeCurrent(
	ctx context.Context,
	email string,
	purpose string,
	requestID string,
) (bool, error) {
	key := fmt.Sprintf("verification:code:%s:%s", purpose, hashKeyPart(email))
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

func hashKeyPart(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
