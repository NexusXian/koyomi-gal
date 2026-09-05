package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/config"
	"backend/internal/classification/agent"
	"backend/internal/classification/model"
	classificationRepo "backend/internal/classification/repository"
	galgameModel "backend/internal/galgame/model"
	galgameRepo "backend/internal/galgame/repository"
	"backend/internal/infrastructures/queue"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

var (
	// ErrAgentDisabled reports CLASSIFICATION_AGENT_ENABLED=false.
	ErrAgentDisabled = errors.New("AI 分级功能未启用")
	// ErrAlreadyRunning reports a game that already has a queued or processing run.
	ErrAlreadyRunning = errors.New("该游戏已有 AI 判断正在进行")
	// ErrNoClassification reports a game without any classification record.
	ErrNoClassification = errors.New("该游戏还没有 AI 判断结果")
	// ErrGameNotFound reports a game that does not exist in the catalog.
	ErrGameNotFound = errors.New("游戏不存在")
	// ErrInvalidState reports an action against a row whose status forbids it.
	ErrInvalidState = errors.New("当前状态不允许该操作")
	// ErrNotFailed reports a retry attempt against a row that did not fail.
	ErrNotFailed = errors.New("该游戏没有可重试的失败记录")
)

// evidenceWeights ranks sources for persistence ordering and display.
var evidenceWeights = map[string]int{
	model.SourceTypeOfficial:  10,
	model.SourceTypeCERO:      9,
	model.SourceTypeESRB:      9,
	model.SourceTypePEGI:      9,
	model.SourceTypeSteam:     8,
	model.SourceTypeVNDB:      6,
	model.SourceTypeBangumi:   5,
	model.SourceTypeWikipedia: 4,
	model.SourceTypeOther:     1,
}

type Service struct {
	repository *classificationRepo.Repository
	galgames   *galgameRepo.GalgameRepository
	agent      *agent.Agent // nil when the agent is disabled
	enqueuer   *queue.ClassificationClient
	cfg        *config.Classification
}

func NewService(
	repository *classificationRepo.Repository,
	galgames *galgameRepo.GalgameRepository,
	classificationAgent *agent.Agent,
	enqueuer *queue.ClassificationClient,
	cfg *config.Classification,
) *Service {
	return &Service{
		repository: repository,
		galgames:   galgames,
		agent:      classificationAgent,
		enqueuer:   enqueuer,
		cfg:        cfg,
	}
}

func (s *Service) Enabled() bool {
	return s.cfg != nil && s.cfg.Enabled && s.agent != nil
}

func (s *Service) AgentModelName() string {
	if s.cfg == nil {
		return ""
	}
	return s.cfg.LLMModel
}

// StartClassification queues an agent run for one game. Rows are inserted
// atomically only when no run is already active; the deterministic Asynq
// task id is the second layer of duplicate protection.
func (s *Service) StartClassification(ctx context.Context, gameID uint) (*model.GameClassification, error) {
	if !s.Enabled() {
		return nil, ErrAgentDisabled
	}
	if err := s.ensureGame(ctx, gameID); err != nil {
		return nil, err
	}
	row, err := s.repository.CreateQueued(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrAlreadyRunning
	}
	if err := s.enqueuer.EnqueueClassification(ctx, gameID); err != nil {
		_ = s.repository.MarkFailed(ctx, gameID, err.Error())
		logger.Error("enqueue classification task failed",
			zap.Uint("game_id", gameID), zap.Uint("classification_id", row.ID), zap.Error(err))
		return nil, err
	}
	logger.Info("classification queued",
		zap.Uint("game_id", gameID), zap.Uint("classification_id", row.ID))
	return row, nil
}

// RetryClassification re-queues the latest failed run of a game.
func (s *Service) RetryClassification(ctx context.Context, gameID uint) (*model.GameClassification, error) {
	if !s.Enabled() {
		return nil, ErrAgentDisabled
	}
	row, err := s.repository.FindLatest(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNoClassification
	}
	if row.Status != string(model.StatusFailed) {
		return nil, ErrNotFailed
	}
	if _, err := s.repository.ResetQueued(ctx, gameID); err != nil {
		return nil, err
	}
	if err := s.enqueuer.EnqueueClassification(ctx, gameID); err != nil {
		_ = s.repository.MarkFailed(ctx, gameID, err.Error())
		return nil, err
	}
	logger.Info("classification retried",
		zap.Uint("game_id", gameID), zap.Uint("classification_id", row.ID))
	return row, nil
}

// CancelClassification stops a queued or processing run for one game. The row
// is marked cancelled first; the Asynq task is then best-effort removed or
// signalled, so any task that slips through wakes up to a cancelled row and
// exits as stale. A finished run (pending_review or later) cannot be cancelled.
func (s *Service) CancelClassification(ctx context.Context, gameID uint) error {
	row, err := s.repository.FindLatest(ctx, gameID)
	if err != nil {
		return err
	}
	if row == nil {
		return ErrNoClassification
	}
	if !model.ClassificationStatus(row.Status).Active() {
		return fmt.Errorf("%w: 当前没有排队或进行中的 AI 判断任务", ErrInvalidState)
	}
	cancelled, err := s.repository.MarkCancelled(ctx, gameID)
	if err != nil {
		return err
	}
	if !cancelled {
		return fmt.Errorf("%w: 任务已结束，无法取消", ErrInvalidState)
	}
	if err := s.enqueuer.CancelTask(gameID); err != nil {
		logger.Warn("remove classification task from queue failed",
			zap.Uint("game_id", gameID), zap.Error(err))
	}
	logger.Info("classification cancelled",
		zap.Uint("game_id", gameID), zap.Uint("classification_id", row.ID))
	return nil
}

// ListQueue returns the paginated queue of latest runs, one per game, with
// catalog titles. Status and keyword are optional filters.
func (s *Service) ListQueue(
	ctx context.Context,
	page, limit int,
	status, keyword string,
) ([]model.ClassificationTask, int64, int, int, error) {
	page, limit = pagination(page, limit)
	tasks, total, err := s.repository.ListLatest(
		ctx, strings.TrimSpace(status), strings.TrimSpace(keyword), page, limit,
	)
	return tasks, total, page, limit, err
}

// RunClassification executes the agent pass. It is invoked by the Asynq
// handler; transient errors bubble up so Asynq retries them, while terminal
// outcomes mark the row and return nil.
func (s *Service) RunClassification(ctx context.Context, gameID uint) error {
	row, err := s.repository.ClaimQueued(ctx, gameID)
	if err != nil {
		return err
	}
	if row == nil {
		return nil // nothing queued; the task is stale
	}
	profile, err := s.repository.FindGameProfile(ctx, gameID)
	if err != nil {
		return err
	}
	if profile == nil {
		s.MarkClassificationFailed(ctx, gameID, "game not found")
		return nil
	}
	if s.agent == nil {
		s.MarkClassificationFailed(ctx, gameID, "classification agent is disabled")
		return nil
	}

	input := agent.GameAgentInput{
		GameID:        profile.GameID,
		Title:         profile.Title,
		OriginalTitle: profile.OriginalTitle,
		Developer:     profile.Developer,
		Publisher:     profile.Publisher,
		VNDBID:        profile.VNDBID,
		BangumiID:     profile.BangumiID,
	}
	result, stats, err := s.agent.Run(ctx, input)
	if err != nil {
		if errors.Is(err, agent.ErrInvalidModelOutput) || errors.Is(err, agent.ErrAgentDisabled) {
			s.MarkClassificationFailed(ctx, gameID, err.Error())
			return nil
		}
		logger.Warn("classification agent transient failure",
			zap.Uint("game_id", gameID),
			zap.Duration("llm_latency", stats.LLMLatency),
			zap.Int("searches", stats.Searches),
			zap.Int("fetches", stats.Fetches),
			zap.Int("vndb_lookups", stats.VNDBLookups),
			zap.Int("bangumi_lookups", stats.BangumiLookups),
			zap.Error(err))
		return err // Asynq retries
	}

	evidences := make([]model.GameClassificationEvidence, 0, len(result.Evidence))
	for _, item := range result.Evidence {
		evidences = append(evidences, model.GameClassificationEvidence{
			SourceType: item.SourceType,
			Title:      item.Title,
			URL:        item.URL,
			Evidence:   item.Evidence,
			Weight:     evidenceWeights[item.SourceType],
		})
	}
	if err := s.repository.SaveResult(
		ctx,
		row.ID,
		string(result.Classification),
		result.Confidence,
		result.Reason,
		result.Conflict,
		s.AgentModelName(),
		evidences,
	); err != nil {
		return err
	}

	logger.Info("classification finished",
		zap.Uint("game_id", gameID),
		zap.Uint("classification_id", row.ID),
		zap.Int("agent_iterations", 0),
		zap.Int("search_count", stats.Searches),
		zap.Int("fetch_count", stats.Fetches),
		zap.Int("vndb_lookup_count", stats.VNDBLookups),
		zap.Int("bangumi_lookup_count", stats.BangumiLookups),
		zap.Duration("llm_latency", stats.LLMLatency),
		zap.String("classification", string(result.Classification)),
		zap.Float64("confidence", result.Confidence),
	)
	return nil
}

// MarkClassificationFailed implements the queue runner contract.
func (s *Service) MarkClassificationFailed(ctx context.Context, gameID uint, message string) {
	if err := s.repository.MarkFailed(ctx, gameID, message); err != nil {
		logger.Error("mark classification failed",
			zap.Uint("game_id", gameID), zap.Error(err))
	}
}

// GetLatest returns the newest classification with evidence for a game.
func (s *Service) GetLatest(ctx context.Context, gameID uint) (*model.GameClassification, error) {
	return s.repository.FindLatest(ctx, gameID)
}

// DetailGame is the catalog summary embedded in classification responses.
type DetailGame struct {
	ID             uint
	Title          string
	OriginalTitle  string
	CoverURL       string
	AgeRating      int16
	CoverSensitive bool
}

// GetDetail returns the game summary plus its latest proposal for one game.
func (s *Service) GetDetail(ctx context.Context, gameID uint) (*DetailGame, *model.GameClassification, error) {
	game, err := s.galgames.FindByID(ctx, gameID)
	if err != nil {
		return nil, nil, err
	}
	if game == nil {
		return nil, nil, ErrGameNotFound
	}
	row, err := s.repository.FindLatest(ctx, gameID)
	if err != nil {
		return nil, nil, err
	}
	return &DetailGame{
		ID:             game.ID,
		Title:          game.Title,
		OriginalTitle:  game.OriginalTitle,
		CoverURL:       game.CoverURL,
		AgeRating:      game.AgeRating,
		CoverSensitive: game.CoverSensitive,
	}, row, nil
}

// Approve applies a pending_review verdict to the official galgame record.
// Only this path may write the age rating: the agent itself never touches the
// catalog. r18/r17/r15/r12 map to their age levels, non_r18 to all-ages.
func (s *Service) Approve(ctx context.Context, gameID uint, reviewerID uint) error {
	row, err := s.repository.FindLatest(ctx, gameID)
	if err != nil {
		return err
	}
	if row == nil {
		return ErrNoClassification
	}
	reviewed, err := s.repository.Review(ctx, row.ID, reviewerID, model.StatusApproved)
	if err != nil {
		return err
	}
	if !reviewed {
		return fmt.Errorf("%w: 该结果不是待审核状态", ErrInvalidState)
	}
	value := classificationToAgeRating(row.AsClassificationValue())
	if value == nil {
		// unknown cannot be approved; revert the row.
		_, _ = s.repository.Review(ctx, row.ID, reviewerID, model.StatusPending)
		return fmt.Errorf("%w: unknown 结果无法采用，请人工覆盖年龄分级", ErrInvalidState)
	}
	now := time.Now()
	if _, err := s.galgames.BatchUpdate(ctx, []uint{gameID},
		map[string]any{"age_rating": *value, "updated_at": now}); err != nil {
		_, _ = s.repository.Review(ctx, row.ID, reviewerID, model.StatusPending)
		return err
	}
	logger.Info("classification approved",
		zap.Uint("game_id", gameID),
		zap.Uint("classification_id", row.ID),
		zap.Uint("reviewer_id", reviewerID),
		zap.String("classification", row.Classification))
	return nil
}

// Reject declines the pending proposal; the official record stays untouched.
func (s *Service) Reject(ctx context.Context, gameID uint, reviewerID uint) error {
	row, err := s.repository.FindLatest(ctx, gameID)
	if err != nil {
		return err
	}
	if row == nil {
		return ErrNoClassification
	}
	reviewed, err := s.repository.Review(ctx, row.ID, reviewerID, model.StatusRejected)
	if err != nil {
		return err
	}
	if !reviewed {
		return fmt.Errorf("%w: 该结果不是待审核状态", ErrInvalidState)
	}
	logger.Info("classification rejected",
		zap.Uint("game_id", gameID), zap.Uint("reviewer_id", reviewerID))
	return nil
}

// Override replaces the latest proposal with a human verdict, keeping the
// record in pending_review until the admin confirms it.
func (s *Service) Override(
	ctx context.Context,
	gameID uint,
	value model.ClassificationValue,
	reason string,
) error {
	if !value.Valid() {
		return fmt.Errorf("%w: classification 必须是 r18 / r17 / r15 / r12 / non_r18 / unknown", ErrInvalidState)
	}
	row, err := s.repository.FindLatest(ctx, gameID)
	if err != nil {
		return err
	}
	if row == nil {
		return ErrNoClassification
	}
	overridden, err := s.repository.Override(ctx, gameID, string(value), reason)
	if err != nil {
		return err
	}
	if !overridden {
		return fmt.Errorf("%w: 已通过或进行中的结果无法覆盖", ErrInvalidState)
	}
	return nil
}

// BatchStart enqueues runs for many games and reports per-game outcomes.
func (s *Service) BatchStart(ctx context.Context, gameIDs []uint) (*BatchResult, error) {
	if !s.Enabled() {
		return nil, ErrAgentDisabled
	}
	result := &BatchResult{GameIDs: gameIDs}
	for _, gameID := range gameIDs {
		_, err := s.StartClassification(ctx, gameID)
		switch {
		case err == nil:
			result.Enqueued++
		case errors.Is(err, ErrAlreadyRunning):
			result.AlreadyRunning = append(result.AlreadyRunning, gameID)
		case errors.Is(err, ErrAgentDisabled):
			return nil, err
		default:
			result.Failed = append(result.Failed, BatchFailure{GameID: gameID, Reason: err.Error()})
		}
	}
	return result, nil
}

// BatchApprove adopts pending_review proposals that pass the strict bar:
// confidence >= 0.7, no conflicts, and a decisive verdict.
func (s *Service) BatchApprove(ctx context.Context, gameIDs []uint, reviewerID uint) (*BatchResult, error) {
	result := &BatchResult{GameIDs: gameIDs}
	rows, err := s.repository.ListReviewable(ctx, gameIDs)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		value := row.AsClassificationValue()
		switch {
		case classificationToAgeRating(value) == nil:
			result.Skipped = append(result.Skipped, BatchFailure{GameID: row.GameID, Reason: "unknown 结果"})
		case row.Conflict:
			result.Skipped = append(result.Skipped, BatchFailure{GameID: row.GameID, Reason: "证据冲突"})
		case row.Confidence < 0.7:
			result.Skipped = append(result.Skipped, BatchFailure{GameID: row.GameID, Reason: "置信度低于 70%"})
		case s.applyApproved(ctx, &row, reviewerID) == nil:
			result.Approved = append(result.Approved, row.GameID)
		default:
			result.Skipped = append(result.Skipped, BatchFailure{GameID: row.GameID, Reason: "写入失败"})
		}
	}
	return result, nil
}

func (s *Service) applyApproved(ctx context.Context, row *model.GameClassification, reviewerID uint) error {
	value := classificationToAgeRating(row.AsClassificationValue())
	if value == nil {
		return ErrInvalidState
	}
	reviewed, err := s.repository.Review(ctx, row.ID, reviewerID, model.StatusApproved)
	if err != nil || !reviewed {
		return fmt.Errorf("review: %w", err)
	}
	if _, err := s.galgames.BatchUpdate(ctx, []uint{row.GameID},
		map[string]any{"age_rating": *value, "updated_at": time.Now()}); err != nil {
		_, _ = s.repository.Review(ctx, row.ID, reviewerID, model.StatusPending)
		return err
	}
	return nil
}

func (s *Service) ensureGame(ctx context.Context, gameID uint) error {
	profile, err := s.repository.FindGameProfile(ctx, gameID)
	if err != nil {
		return err
	}
	if profile == nil {
		return ErrGameNotFound
	}
	return nil
}

func classificationToAgeRating(value model.ClassificationValue) *int16 {
	var result int16
	switch value {
	case model.ClassificationR18:
		result = galgameModel.AgeRatingR18
	case model.ClassificationR17:
		result = galgameModel.AgeRatingR17
	case model.ClassificationR15:
		result = galgameModel.AgeRatingR15
	case model.ClassificationR12:
		result = galgameModel.AgeRatingR12
	case model.ClassificationNonR18:
		result = galgameModel.AgeRatingAll
	default:
		return nil
	}
	return &result
}

func pagination(page, limit int) (int, int) {
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}
	return page, limit
}

type BatchFailure struct {
	GameID uint   `json:"game_id"`
	Reason string `json:"reason"`
}

type BatchResult struct {
	GameIDs        []uint         `json:"game_ids,omitempty"`
	Enqueued       int            `json:"enqueued"`
	Approved       []uint         `json:"approved,omitempty"`
	AlreadyRunning []uint         `json:"already_running,omitempty"`
	Skipped        []BatchFailure `json:"skipped,omitempty"`
	Failed         []BatchFailure `json:"failed,omitempty"`
}
