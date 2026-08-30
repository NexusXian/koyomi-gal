package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"

	"backend/internal/user/repository"
)

const (
	VerificationPurposeRegister      = "register"
	VerificationPurposePasswordReset = "password_reset"
)

var (
	ErrVerificationCooldown  = errors.New("verification code was sent too recently")
	ErrVerificationRateLimit = errors.New("verification code rate limit exceeded")
	ErrInvalidVerification   = errors.New("invalid verification request")
)

type VerificationEmailTask struct {
	RequestID string `json:"request_id"`
	Email     string `json:"email"`
	Purpose   string `json:"purpose"`
	Code      string `json:"code"`
	ExpiresAt int64  `json:"expires_at"`
}

type VerificationQueue interface {
	EnqueueVerificationEmail(context.Context, VerificationEmailTask) error
}

type VerificationService struct {
	repository     *repository.VerificationRepository
	queue          VerificationQueue
	secret         []byte
	codeTTL        time.Duration
	resendInterval time.Duration
	ipWindow       time.Duration
	ipLimit        int
}

func NewVerificationService(
	repository *repository.VerificationRepository,
	queue VerificationQueue,
	secret string,
	codeTTL time.Duration,
	resendInterval time.Duration,
	ipWindow time.Duration,
	ipLimit int,
) *VerificationService {
	return &VerificationService{
		repository:     repository,
		queue:          queue,
		secret:         []byte(secret),
		codeTTL:        codeTTL,
		resendInterval: resendInterval,
		ipWindow:       ipWindow,
		ipLimit:        ipLimit,
	}
}

func (s *VerificationService) SendCode(
	ctx context.Context,
	email string,
	purpose string,
	ip string,
) error {
	normalizedEmail, err := normalizeEmail(email)
	if err != nil || !validVerificationPurpose(purpose) {
		return ErrInvalidVerification
	}

	code, err := generateVerificationCode()
	if err != nil {
		return fmt.Errorf("generate verification code: %w", err)
	}
	requestID, err := generateRequestID()
	if err != nil {
		return fmt.Errorf("generate verification request ID: %w", err)
	}

	digest := s.codeDigest(normalizedEmail, purpose, code)
	reservation, err := s.repository.ReserveVerificationCode(
		ctx,
		normalizedEmail,
		purpose,
		ip,
		requestID,
		digest,
		s.codeTTL,
		s.resendInterval,
		s.ipWindow,
		s.ipLimit,
	)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrVerificationCooldown):
			return ErrVerificationCooldown
		case errors.Is(err, repository.ErrVerificationRateLimit):
			return ErrVerificationRateLimit
		default:
			return err
		}
	}

	task := VerificationEmailTask{
		RequestID: requestID,
		Email:     normalizedEmail,
		Purpose:   purpose,
		Code:      code,
		ExpiresAt: time.Now().Add(s.codeTTL).Unix(),
	}
	if err := s.queue.EnqueueVerificationEmail(ctx, task); err != nil {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		rollbackErr := s.repository.CancelVerificationCode(rollbackCtx, reservation)
		if rollbackErr != nil {
			return fmt.Errorf("enqueue verification email: %w; rollback failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("enqueue verification email: %w", err)
	}

	return nil
}

func (s *VerificationService) codeDigest(email string, purpose string, code string) string {
	hash := hmac.New(sha256.New, s.secret)
	hash.Write([]byte(purpose))
	hash.Write([]byte{0})
	hash.Write([]byte(email))
	hash.Write([]byte{0})
	hash.Write([]byte(code))
	return hex.EncodeToString(hash.Sum(nil))
}

func normalizeEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	address, err := mail.ParseAddress(value)
	if err != nil || !strings.EqualFold(address.Address, value) {
		return "", ErrInvalidVerification
	}
	return strings.ToLower(address.Address), nil
}

func validVerificationPurpose(purpose string) bool {
	return purpose == VerificationPurposeRegister || purpose == VerificationPurposePasswordReset
}

func generateVerificationCode() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func generateRequestID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}
