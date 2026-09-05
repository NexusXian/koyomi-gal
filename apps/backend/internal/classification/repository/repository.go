package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/classification/model"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GameProfile is the minimal game metadata needed to build agent input. It is
// read from the galgame catalog and its external-source mappings.
type GameProfile struct {
	GameID        uint
	Title         string
	OriginalTitle string
	Developer     string
	Publisher     string
	VNDBID        string
	BangumiID     string
}

// FindGameProfile loads catalog identity plus known VNDB/Bangumi ids.
func (r *Repository) FindGameProfile(ctx context.Context, gameID uint) (*GameProfile, error) {
	type row struct {
		Title         string
		OriginalTitle string
		Developer     string
	}
	var game row
	err := r.db.WithContext(ctx).Raw(`
SELECT g.title AS title, g.original_title AS original_title,
       COALESCE(d.name, '') AS developer
FROM galgames g
LEFT JOIN developers d ON d.id = g.developer_id
WHERE g.id = ?
`, gameID).Scan(&game).Error
	if err != nil {
		return nil, fmt.Errorf("find galgame profile: %w", err)
	}
	if game.Title == "" {
		return nil, nil
	}

	var sources []struct {
		Source     string
		ExternalID string
	}
	if err := r.db.WithContext(ctx).
		Model(&model.GameClassification{}).
		Table("galgame_external_sources").
		Select("source, external_id").
		Where("galgame_id = ?", gameID).
		Scan(&sources).Error; err != nil {
		return nil, fmt.Errorf("list galgame external sources: %w", err)
	}
	profile := &GameProfile{
		GameID:        gameID,
		Title:         game.Title,
		OriginalTitle: game.OriginalTitle,
		Developer:     game.Developer,
	}
	for _, source := range sources {
		switch source.Source {
		case "vndb":
			profile.VNDBID = source.ExternalID
		case "bangumi":
			profile.BangumiID = source.ExternalID
		}
	}
	return profile, nil
}

// CreateQueued atomically inserts a queued classification row unless the game
// already has an active (queued/processing) row. Returns the created row, or
// nil when a run is already in progress.
func (r *Repository) CreateQueued(ctx context.Context, gameID uint) (*model.GameClassification, error) {
	var row model.GameClassification
	err := r.db.WithContext(ctx).Raw(`
INSERT INTO game_classifications (game_id, status, created_at, updated_at)
SELECT ?, 'queued', NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM game_classifications
    WHERE game_id = ? AND status IN ('queued', 'processing')
)
RETURNING id, game_id, classification, confidence, reason, conflict, status,
          model, error_message, reviewer_id, reviewed_at, created_at, updated_at
`, gameID, gameID).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("insert queued classification: %w", err)
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

// FindLatest returns the newest classification row of a game with evidences.
func (r *Repository) FindLatest(ctx context.Context, gameID uint) (*model.GameClassification, error) {
	var row model.GameClassification
	err := r.db.WithContext(ctx).
		Preload("Evidences", func(db *gorm.DB) *gorm.DB { return db.Order("game_classification_evidences.id") }).
		Where("game_id = ?", gameID).
		Order("id DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find latest classification: %w", err)
	}
	return &row, nil
}

// ClaimQueued atomically moves one queued row to processing. Returns false when
// no queued row exists for this game (e.g. duplicate task wakeup).
func (r *Repository) ClaimQueued(ctx context.Context, gameID uint) (*model.GameClassification, error) {
	var row model.GameClassification
	err := r.db.WithContext(ctx).Raw(`
UPDATE game_classifications
SET status = 'processing', error_message = '', updated_at = NOW()
WHERE id = (
    SELECT id FROM game_classifications
    WHERE game_id = ? AND status = 'queued'
    ORDER BY id DESC LIMIT 1
)
RETURNING id, game_id, classification, confidence, reason, conflict, status,
          model, error_message, reviewer_id, reviewed_at, created_at, updated_at
`, gameID).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("claim queued classification: %w", err)
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

// SaveResult persists the agent verdict and its evidence atomically and moves
// the row to pending_review. The guarded update runs first so a row cancelled
// or failed while the agent ran never receives the verdict or evidence.
func (r *Repository) SaveResult(
	ctx context.Context,
	classificationID uint,
	classification string,
	confidence float64,
	reason string,
	conflict bool,
	modelName string,
	evidences []model.GameClassificationEvidence,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.GameClassification{}).
			Where("id = ? AND status IN ?", classificationID,
				[]string{string(model.StatusQueued), string(model.StatusProcessing)}).
			Updates(map[string]any{
				"classification": classification,
				"confidence":     confidence,
				"reason":         reason,
				"conflict":       conflict,
				"status":         string(model.StatusPending),
				"model":          modelName,
				"error_message":  "",
				"updated_at":     time.Now(),
			})
		if result.Error != nil {
			return fmt.Errorf("save classification result: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return nil // the run is no longer active; drop its result
		}
		if err := tx.Where("classification_id = ?", classificationID).
			Delete(&model.GameClassificationEvidence{}).Error; err != nil {
			return fmt.Errorf("clear stale evidences: %w", err)
		}
		for i := range evidences {
			evidences[i].ClassificationID = classificationID
			evidences[i].ID = 0
		}
		if len(evidences) > 0 {
			if err := tx.Create(&evidences).Error; err != nil {
				return fmt.Errorf("save evidences: %w", err)
			}
		}
		return nil
	})
}

// MarkFailed records a permanent failure on the latest active row.
func (r *Repository) MarkFailed(ctx context.Context, gameID uint, message string) error {
	if err := r.db.WithContext(ctx).Raw(`
UPDATE game_classifications
SET status = 'failed', error_message = ?, updated_at = NOW()
WHERE id = (
    SELECT id FROM game_classifications
    WHERE game_id = ? AND status IN ('queued', 'processing')
    ORDER BY id DESC LIMIT 1
)
`, truncate(message, 4000), gameID).Error; err != nil {
		return fmt.Errorf("mark classification failed: %w", err)
	}
	return nil
}

// ListLatest returns one row per game — its newest classification run — joined
// with catalog identity. Keyword matches the game title or original title;
// a non-empty status filters on the latest row's status. Ordered newest first.
func (r *Repository) ListLatest(
	ctx context.Context,
	status, keyword string,
	page, limit int,
) ([]model.ClassificationTask, int64, error) {
	where := `WHERE gc.id = (
    SELECT MAX(id) FROM game_classifications
    WHERE game_id = gc.game_id
)`
	args := make([]any, 0, 4)
	if status != "" {
		where += ` AND gc.status = ?`
		args = append(args, status)
	}
	if keyword != "" {
		where += ` AND (g.title ILIKE ? OR g.original_title ILIKE ?)`
		pattern := "%" + keyword + "%"
		args = append(args, pattern, pattern)
	}

	var total int64
	if err := r.db.WithContext(ctx).Raw(`
SELECT COUNT(*)
FROM game_classifications gc
JOIN galgames g ON g.id = gc.game_id
`+where, args...).Scan(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count classification queue: %w", err)
	}

	tasks := make([]model.ClassificationTask, 0, limit)
	listQuery := `
SELECT gc.id, gc.game_id, gc.classification, gc.confidence, gc.reason,
       gc.conflict, gc.status, gc.model, gc.error_message, gc.reviewer_id,
       gc.reviewed_at, gc.created_at, gc.updated_at,
       g.title AS game_title, g.original_title AS original_title
FROM game_classifications gc
JOIN galgames g ON g.id = gc.game_id
` + where + ` ORDER BY gc.id DESC LIMIT ? OFFSET ?`
	args = append(args, limit, (page-1)*limit)
	if err := r.db.WithContext(ctx).Raw(listQuery, args...).Scan(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("list classification queue: %w", err)
	}
	return tasks, total, nil
}

// MarkCancelled stops the latest active (queued/processing) run of a game.
// Returns false when no active run exists (already finished or cancelled).
func (r *Repository) MarkCancelled(ctx context.Context, gameID uint) (bool, error) {
	result := r.db.WithContext(ctx).Raw(`
UPDATE game_classifications
SET status = 'cancelled', error_message = '', updated_at = NOW()
WHERE id = (
    SELECT id FROM game_classifications
    WHERE game_id = ? AND status IN ('queued', 'processing')
    ORDER BY id DESC LIMIT 1
)
`, gameID)
	if result.Error != nil {
		return false, fmt.Errorf("mark classification cancelled: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// ResetQueued moves the latest failed row back to queued for a retry. Returns
// false when there is no failed row.
func (r *Repository) ResetQueued(ctx context.Context, gameID uint) (bool, error) {
	result := r.db.WithContext(ctx).Raw(`
UPDATE game_classifications
SET status = 'queued', error_message = '', updated_at = NOW()
WHERE id = (
    SELECT id FROM game_classifications
    WHERE game_id = ? AND status = 'failed'
    ORDER BY id DESC LIMIT 1
)
`, gameID)
	if result.Error != nil {
		return false, fmt.Errorf("reset failed classification: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// Review transitions a pending_review row. Returns false when the row does not
// exist or is not pending_review.
func (r *Repository) Review(
	ctx context.Context,
	classificationID uint,
	reviewerID uint,
	status model.ClassificationStatus,
) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.GameClassification{}).
		Where("id = ? AND status = ?", classificationID, string(model.StatusPending)).
		Updates(map[string]any{
			"status":      string(status),
			"reviewer_id": reviewerID,
			"reviewed_at": time.Now(),
			"updated_at":  time.Now(),
		})
	if result.Error != nil {
		return false, fmt.Errorf("review classification: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// Override rewrites the verdict of the latest non-active row of a game so the
// admin can steer the proposal without running the agent again. Only pending,
// rejected or failed rows may be overridden; an approved row keeps its verdict.
func (r *Repository) Override(
	ctx context.Context,
	gameID uint,
	classification string,
	reason string,
) (bool, error) {
	result := r.db.WithContext(ctx).Raw(`
UPDATE game_classifications
SET classification = ?, confidence = 0, reason = ?, conflict = false,
    status = 'pending_review', model = 'manual_override',
    error_message = '', updated_at = NOW()
WHERE id = (
    SELECT id FROM game_classifications
    WHERE game_id = ? AND status IN ('pending_review', 'rejected', 'failed')
    ORDER BY id DESC LIMIT 1
)
`, classification, reason, gameID)
	if result.Error != nil {
		return false, fmt.Errorf("override classification: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// ListReviewable returns latest pending_review rows for the given games,
// ordered by classification id. Used by batch approval.
func (r *Repository) ListReviewable(
	ctx context.Context,
	gameIDs []uint,
) ([]model.GameClassification, error) {
	if len(gameIDs) == 0 {
		return nil, nil
	}
	var rows []model.GameClassification
	err := r.db.WithContext(ctx).Where(`
status = 'pending_review' AND game_id IN ? AND id IN (
    SELECT MAX(id) FROM game_classifications WHERE game_id IN ? GROUP BY game_id
)`, gameIDs, gameIDs).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list reviewable classifications: %w", err)
	}
	return rows, nil
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
