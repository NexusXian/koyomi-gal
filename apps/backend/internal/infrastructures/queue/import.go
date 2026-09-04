package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"backend/config"

	"github.com/hibiken/asynq"
)

const (
	// ImportVNDBBatchTaskType is the Asynq task type for VNDB batch imports.
	ImportVNDBBatchTaskType = "import:vndb:batch"
	// ImportBangumiEnrichTaskType is the Asynq task type for Bangumi batch
	// metadata enrichment.
	ImportBangumiEnrichTaskType = "import:bangumi:enrich"
	importQueueName             = "import"
)

type importBatchPayload struct {
	JobID int64 `json:"job_id"`
}

// ImportClient enqueues batch import tasks onto the import queue.
type ImportClient struct {
	client *asynq.Client
}

func NewImportClient(cfg *config.Redis) *ImportClient {
	return &ImportClient{client: asynq.NewClient(redisClientOpt(cfg))}
}

func (c *ImportClient) EnqueueVNDBBatch(ctx context.Context, jobID int64) error {
	payload, err := json.Marshal(importBatchPayload{JobID: jobID})
	if err != nil {
		return fmt.Errorf("encode import batch task: %w", err)
	}
	task := asynq.NewTask(ImportVNDBBatchTaskType, payload)
	_, err = c.client.EnqueueContext(ctx, task,
		asynq.Queue(importQueueName),
		asynq.TaskID(fmt.Sprintf("import:batch:job:%d", jobID)),
		asynq.MaxRetry(1),
		asynq.Timeout(2*time.Hour),
	)
	if err != nil {
		return fmt.Errorf("enqueue import batch task: %w", err)
	}
	return nil
}

func (c *ImportClient) EnqueueBangumiEnrich(ctx context.Context, jobID int64) error {
	payload, err := json.Marshal(importBatchPayload{JobID: jobID})
	if err != nil {
		return fmt.Errorf("encode import enrich task: %w", err)
	}
	task := asynq.NewTask(ImportBangumiEnrichTaskType, payload)
	_, err = c.client.EnqueueContext(ctx, task,
		asynq.Queue(importQueueName),
		asynq.TaskID(fmt.Sprintf("import:enrich:job:%d", jobID)),
		asynq.MaxRetry(1),
		asynq.Timeout(6*time.Hour),
	)
	if err != nil {
		return fmt.Errorf("enqueue import enrich task: %w", err)
	}
	return nil
}

func (c *ImportClient) Close() error {
	return c.client.Close()
}

// ImportBatchRunner is implemented by the importer service.
type ImportBatchRunner interface {
	RunImportJob(ctx context.Context, jobID int64) error
}

// NewImportServer builds a worker that consumes only the import queue.
// It runs inside the HTTP server process, which owns the database
// dependencies batch imports need; the standalone mail worker never
// reserves import tasks.
func NewImportServer(cfg *config.Redis, concurrency int) *asynq.Server {
	return asynq.NewServer(redisClientOpt(cfg), asynq.Config{
		Concurrency:     concurrency,
		Queues:          map[string]int{importQueueName: 1},
		ShutdownTimeout: 30 * time.Second,
	})
}

func RegisterImportTasks(mux *asynq.ServeMux, runner ImportBatchRunner) {
	runJob := func(ctx context.Context, task *asynq.Task) error {
		var payload importBatchPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil || payload.JobID <= 0 {
			return asynq.RevokeTask
		}
		return runner.RunImportJob(ctx, payload.JobID)
	}
	mux.HandleFunc(ImportVNDBBatchTaskType, runJob)
	mux.HandleFunc(ImportBangumiEnrichTaskType, runJob)
}
