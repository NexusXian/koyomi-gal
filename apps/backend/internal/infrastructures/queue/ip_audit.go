package queue

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"backend/config"
	"backend/internal/ipgeo"

	"github.com/hibiken/asynq"
)

const (
	ipLogTaskType  = "audit:ip-log"
	ipLogQueueName = "audit"
)

type IPLogClient struct {
	client *asynq.Client
	key    [sha256.Size]byte
}

func NewIPLogClient(cfg *config.Redis, secret string) *IPLogClient {
	return &IPLogClient{
		client: asynq.NewClient(redisClientOpt(cfg)),
		key:    sha256.Sum256([]byte("koyomi-gal:ip-audit-queue:" + secret)),
	}
}

func (c *IPLogClient) EnqueueIPLog(ctx context.Context, entry ipgeo.UserIPLog) error {
	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encode IP audit task: %w", err)
	}
	payload, err = encryptPayload(c.key, payload)
	if err != nil {
		return fmt.Errorf("encrypt IP audit task: %w", err)
	}
	task := asynq.NewTask(ipLogTaskType, payload)
	options := []asynq.Option{
		asynq.Queue(ipLogQueueName),
		asynq.MaxRetry(5),
		asynq.Timeout(30 * time.Second),
	}
	if entry.EntityID != nil {
		options = append(options, asynq.TaskID(fmt.Sprintf("ip-audit:%s:%d", entry.Action, *entry.EntityID)))
	}
	if _, err := c.client.EnqueueContext(ctx, task, options...); err != nil {
		return fmt.Errorf("enqueue IP audit task: %w", err)
	}
	return nil
}

func (c *IPLogClient) Close() error {
	return c.client.Close()
}

type IPLogRecorder interface {
	Record(ctx context.Context, entry ipgeo.UserIPLog) error
}

func NewIPLogServer(cfg *config.Redis, concurrency int) *asynq.Server {
	return asynq.NewServer(redisClientOpt(cfg), asynq.Config{
		Concurrency:     concurrency,
		Queues:          map[string]int{ipLogQueueName: 1},
		ShutdownTimeout: 30 * time.Second,
	})
}

func RegisterIPLogTasks(mux *asynq.ServeMux, recorder IPLogRecorder, secret string) {
	key := sha256.Sum256([]byte("koyomi-gal:ip-audit-queue:" + secret))
	mux.HandleFunc(ipLogTaskType, func(ctx context.Context, task *asynq.Task) error {
		payload, err := decryptPayload(key, task.Payload())
		if err != nil {
			return asynq.RevokeTask
		}
		var entry ipgeo.UserIPLog
		if err := json.Unmarshal(payload, &entry); err != nil {
			return asynq.RevokeTask
		}
		if entry.UserID == 0 || entry.IPAddress == "" || entry.Action == "" {
			return asynq.RevokeTask
		}
		return recorder.Record(ctx, entry)
	})
}
