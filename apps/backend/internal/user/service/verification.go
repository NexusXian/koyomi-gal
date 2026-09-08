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
	"strconv"
	"strings"
	"time"

	"backend/internal/user/repository"
)

const (
	VerificationPurposeRegister       = "register"
	VerificationPurposePasswordReset  = "password_reset"
	VerificationPurposeChangePassword = "change_password"

	verificationMaxAttempts   = 5
	identifierSendLimit       = 10
	identifierSendWindow      = 24 * time.Hour
	passwordResetTicketTTL    = 10 * time.Minute
	resetTicketCreateAttempts = 3
)

var (
	ErrVerificationCooldown  = errors.New("verification code was sent too recently")
	ErrVerificationRateLimit = errors.New("verification code rate limit exceeded")
	ErrInvalidVerification   = errors.New("invalid verification request")
)

type VerificationEmailTask struct {
	RequestID  string `json:"request_id"`
	Email      string `json:"email"`
	Identifier string `json:"identifier"`
	Purpose    string `json:"purpose"`
	Code       string `json:"code"`
	ExpiresAt  int64  `json:"expires_at"`
}

type VerificationQueue interface {
	EnqueueVerificationEmail(context.Context, VerificationEmailTask) error
}

type PreparedVerificationCode struct {
	task        VerificationEmailTask
	reservation *repository.VerificationReservation
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
	if purpose != VerificationPurposeRegister {
		return ErrInvalidVerification
	}
	prepared, err := s.PrepareCode(ctx, email, email, purpose, ip)
	if err != nil {
		return err
	}
	return s.EnqueuePrepared(ctx, prepared)
}

func (s *VerificationService) PrepareCode(
	ctx context.Context,
	email string,
	identifier string,
	purpose string,
	ip string,
) (*PreparedVerificationCode, error) {
	normalizedEmail, err := normalizeEmail(email)
	if err != nil {
		return nil, ErrInvalidVerification
	}
	normalizedIdentifier, err := normalizeVerificationIdentifier(identifier, purpose)
	if err != nil {
		return nil, ErrInvalidVerification
	}

	code, err := generateVerificationCode()
	if err != nil {
		return nil, fmt.Errorf("generate verification code: %w", err)
	}
	requestID, err := generateRequestID()
	if err != nil {
		return nil, fmt.Errorf("generate verification request ID: %w", err)
	}

	digest := s.codeDigest(normalizedIdentifier, purpose, code)
	reservation, err := s.repository.ReserveVerificationCode(
		ctx,
		normalizedIdentifier,
		purpose,
		ip,
		requestID,
		digest,
		s.codeTTL,
		s.resendInterval,
		s.ipWindow,
		s.ipLimit,
		identifierSendWindow,
		identifierSendLimit,
	)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrVerificationCooldown):
			return nil, ErrVerificationCooldown
		case errors.Is(err, repository.ErrVerificationRateLimit):
			return nil, ErrVerificationRateLimit
		default:
			return nil, err
		}
	}

	return &PreparedVerificationCode{
		task: VerificationEmailTask{
			RequestID:  requestID,
			Email:      normalizedEmail,
			Identifier: normalizedIdentifier,
			Purpose:    purpose,
			Code:       code,
			ExpiresAt:  time.Now().Add(s.codeTTL).Unix(),
		},
		reservation: reservation,
	}, nil
}

func (s *VerificationService) EnqueuePrepared(
	ctx context.Context,
	prepared *PreparedVerificationCode,
) error {
	if prepared == nil || prepared.reservation == nil {
		return ErrInvalidVerification
	}
	if err := s.queue.EnqueueVerificationEmail(ctx, prepared.task); err != nil {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		rollbackErr := s.repository.CancelVerificationCode(rollbackCtx, prepared.reservation)
		if rollbackErr != nil {
			return fmt.Errorf("enqueue verification email: %w; rollback failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("enqueue verification email: %w", err)
	}
	return nil
}

func (s *VerificationService) ConsumeCode(
	ctx context.Context,
	identifier string,
	purpose string,
	code string,
) (bool, error) {
	normalizedIdentifier, err := normalizeVerificationIdentifier(identifier, purpose)
	if err != nil {
		return false, nil
	}
	digest := s.codeDigest(normalizedIdentifier, purpose, code)
	return s.repository.ConsumeVerificationCode(
		ctx,
		normalizedIdentifier,
		purpose,
		digest,
		verificationMaxAttempts,
	)
}

func (s *VerificationService) IssuePasswordResetTicket(ctx context.Context, userID uint) (string, error) {
	for range resetTicketCreateAttempts {
		token, err := randomToken(32)
		if err != nil {
			return "", fmt.Errorf("generate password reset token: %w", err)
		}
		err = s.repository.CreatePasswordResetTicket(ctx, token, userID, passwordResetTicketTTL)
		if errors.Is(err, repository.ErrResetTicketCollision) {
			continue
		}
		if err != nil {
			return "", err
		}
		return token, nil
	}
	return "", errors.New("create unique password reset token: retry limit reached")
}

func (s *VerificationService) ConsumePasswordResetTicket(ctx context.Context, token string) (uint, error) {
	if strings.TrimSpace(token) == "" {
		return 0, repository.ErrResetTicketNotFound
	}
	return s.repository.ConsumePasswordResetTicket(ctx, token)
}

func (s *VerificationService) codeDigest(identifier string, purpose string, code string) string {
	hash := hmac.New(sha256.New, s.secret)
	hash.Write([]byte(purpose))
	hash.Write([]byte{0})
	hash.Write([]byte(identifier))
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

func normalizeVerificationIdentifier(identifier string, purpose string) (string, error) {
	switch purpose {
	case VerificationPurposeRegister, VerificationPurposePasswordReset:
		return normalizeEmail(identifier)
	case VerificationPurposeChangePassword:
		userID, err := strconv.ParseUint(strings.TrimSpace(identifier), 10, 64)
		if err != nil || userID == 0 {
			return "", ErrInvalidVerification
		}
		return strconv.FormatUint(userID, 10), nil
	default:
		return "", ErrInvalidVerification
	}
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
