package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"backend/config"

	"github.com/hibiken/asynq"
)

const (
	// ClassificationGameTaskType is the Asynq task type running one game's
	// age-rating research pass.
	ClassificationGameTaskType = "classification:game"
	classificationQueueName    = "classification"
)

type classificationPayload struct {
	GameID uint `json:"game_id"`
}

// ClassificationClient enqueues classification tasks onto the dedicated
// classification queue so large admin batches never starve mail/import work.
type ClassificationClient struct {
	client *asynq.Client
}

func NewClassificationClient(cfg *config.Redis) *ClassificationClient {
	return &ClassificationClient{client: asynq.NewClient(redisClientOpt(cfg))}
}

// EnqueueClassification enqueues one task per game. The deterministic TaskID
// doubles as duplicate protection: a second enqueue for a game whose task is
// still queued fails with asynq.ErrTaskIDConflict.
func (c *ClassificationClient) EnqueueClassification(ctx context.Context, gameID uint) error {
	if gameID == 0 {
		return errors.New("game id is required")
	}
	payload, err := json.Marshal(classificationPayload{GameID: gameID})
	if err != nil {
		return fmt.Errorf("encode classification task: %w", err)
	}
	task := asynq.NewTask(ClassificationGameTaskType, payload)
	_, err = c.client.EnqueueContext(ctx, task,
		asynq.Queue(classificationQueueName),
		asynq.TaskID(fmt.Sprintf("classification:game:%d", gameID)),
		asynq.MaxRetry(2),
		asynq.Timeout(10*time.Minute),
	)
	if err != nil {
		return fmt.Errorf("enqueue classification task: %w", err)
	}
	return nil
}

func (c *ClassificationClient) Close() error {
	return c.client.Close()
}

// ClassificationRunner is implemented by the classification service. It keeps
// the queue layer decoupled from the database and the agent.
type ClassificationRunner interface {
	RunClassification(ctx context.Context, gameID uint) error
	MarkClassificationFailed(ctx context.Context, gameID uint, message string)
}

// NewClassificationServer builds a worker consuming only the classification
// queue. It runs inside the HTTP server process (like the import worker)
// because classification needs PostgreSQL and the LLM configuration.
func NewClassificationServer(cfg *config.Redis, concurrency int) *asynq.Server {
	return asynq.NewServer(redisClientOpt(cfg), asynq.Config{
		Concurrency:     concurrency,
		Queues:          map[string]int{classificationQueueName: 1},
		ShutdownTimeout: 30 * time.Second,
	})
}

// RegisterClassificationTasks wires the classification task to the runner.
// Transient failures are returned so Asynq retries them; once retries are
// exhausted the runner is told to mark the row failed and the task ends.
func RegisterClassificationTasks(mux *asynq.ServeMux, runner ClassificationRunner) {
	mux.HandleFunc(ClassificationGameTaskType, func(ctx context.Context, task *asynq.Task) error {
		var payload classificationPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil || payload.GameID == 0 {
			return asynq.RevokeTask
		}
		if err := runner.RunClassification(ctx, payload.GameID); err != nil {
			if retryCount, ok := asynq.GetRetryCount(ctx); ok {
				if maxRetry, ok := asynq.GetMaxRetry(ctx); ok && retryCount >= maxRetry {
					runner.MarkClassificationFailed(ctx, payload.GameID, err.Error())
					return nil
				}
			}
			return err
		}
		return nil
	})
}
