package queue

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"

	"backend/config"
	notificationService "backend/internal/notification/service"
	userRepository "backend/internal/user/repository"
	userService "backend/internal/user/service"

	"github.com/hibiken/asynq"
)

const (
	verificationEmailTaskType = "email:verification"
	mailQueueName             = "mail"
)

type VerificationClient struct {
	client *asynq.Client
	key    [sha256.Size]byte
}

func NewVerificationClient(cfg *config.Redis, secret string) *VerificationClient {
	return &VerificationClient{
		client: asynq.NewClient(redisClientOpt(cfg)),
		key:    queueEncryptionKey(secret),
	}
}

func (c *VerificationClient) EnqueueVerificationEmail(
	ctx context.Context,
	payload userService.VerificationEmailTask,
) error {
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode verification email task: %w", err)
	}
	encryptedPayload, err := encryptPayload(c.key, encodedPayload)
	if err != nil {
		return fmt.Errorf("encrypt verification email task: %w", err)
	}

	task := asynq.NewTask(verificationEmailTaskType, encryptedPayload)
	_, err = c.client.EnqueueContext(
		ctx,
		task,
		asynq.Queue(mailQueueName),
		asynq.TaskID(payload.RequestID),
		asynq.MaxRetry(3),
		asynq.ProcessIn(4*time.Second),
		asynq.Timeout(time.Minute),
		asynq.Deadline(time.Unix(payload.ExpiresAt, 0)),
	)
	if err != nil {
		return fmt.Errorf("enqueue verification email task: %w", err)
	}
	return nil
}

func (c *VerificationClient) Close() error {
	return c.client.Close()
}

func NewServer(cfg *config.Redis, concurrency int) *asynq.Server {
	return asynq.NewServer(redisClientOpt(cfg), asynq.Config{
		Concurrency:     concurrency,
		Queues:          map[string]int{mailQueueName: 1},
		ShutdownTimeout: 30 * time.Second,
	})
}

func NewServeMux(
	emailService *notificationService.EmailService,
	verificationRepository *userRepository.VerificationRepository,
	secret string,
) *asynq.ServeMux {
	key := queueEncryptionKey(secret)
	mux := asynq.NewServeMux()
	mux.HandleFunc(verificationEmailTaskType, func(ctx context.Context, task *asynq.Task) error {
		decryptedPayload, err := decryptPayload(key, task.Payload())
		if err != nil {
			return asynq.RevokeTask
		}
		var payload userService.VerificationEmailTask
		if err := json.Unmarshal(decryptedPayload, &payload); err != nil {
			return asynq.RevokeTask
		}
		if payload.RequestID == "" || payload.Email == "" || payload.Code == "" {
			return asynq.RevokeTask
		}
		if !time.Now().Before(time.Unix(payload.ExpiresAt, 0)) {
			return asynq.RevokeTask
		}

		current, err := verificationRepository.IsVerificationCodeCurrent(
			ctx,
			payload.Email,
			payload.Purpose,
			payload.RequestID,
		)
		if err != nil {
			return err
		}
		if !current {
			return asynq.RevokeTask
		}

		if err := emailService.SendVerificationCode(
			ctx,
			payload.Email,
			payload.Purpose,
			payload.Code,
			time.Unix(payload.ExpiresAt, 0),
		); err != nil {
			return fmt.Errorf("send verification email: %w", err)
		}
		return nil
	})
	return mux
}

func queueEncryptionKey(secret string) [sha256.Size]byte {
	return sha256.Sum256([]byte("koyomi-gal:verification-queue:" + secret))
}

func encryptPayload(key [sha256.Size]byte, payload []byte) ([]byte, error) {
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return aead.Seal(nonce, nonce, payload, nil), nil
}

func decryptPayload(key [sha256.Size]byte, payload []byte) ([]byte, error) {
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}
	if len(payload) < aead.NonceSize() {
		return nil, fmt.Errorf("encrypted payload is too short")
	}
	nonce := payload[:aead.NonceSize()]
	return aead.Open(nil, nonce, payload[aead.NonceSize():], nil)
}

func newAEAD(key [sha256.Size]byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func redisClientOpt(cfg *config.Redis) asynq.RedisClientOpt {
	option := asynq.RedisClientOpt{
		Addr:     net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port))),
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.Database,
	}
	if cfg.TLS {
		option.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: cfg.Host,
		}
	}
	return option
}
