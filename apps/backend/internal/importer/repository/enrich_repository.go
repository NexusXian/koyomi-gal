package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	galgameModel "backend/internal/galgame/model"
	importerModel "backend/internal/importer/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ListExternalSources returns every external source row of one galgame.
func (r *Repository) ListExternalSources(
	ctx context.Context,
	galgameID uint,
) ([]importerModel.GalgameExternalSource, error) {
	var sources []importerModel.GalgameExternalSource
	err := r.db.WithContext(ctx).
		Where("galgame_id = ?", galgameID).
		Order("source").
		Find(&sources).Error
	if err != nil {
		return nil, fmt.Errorf("list external sources: %w", err)
	}
	return sources, nil
}

// ListGalgamesForEnrichment pages through galgames that already have the
// require source but not the missing one.
func (r *Repository) ListGalgamesForEnrichment(
	ctx context.Context,
	requireSource, missingSource string,
	afterID uint,
	limit int,
) ([]galgameModel.Galgame, error) {
	var games []galgameModel.Galgame
	err := r.db.WithContext(ctx).
		Model(&galgameModel.Galgame{}).
		Select("galgames.*").
		Joins("JOIN galgame_external_sources src_req ON src_req.galgame_id = galgames.id AND src_req.source = ?", requireSource).
		Joins("LEFT JOIN galgame_external_sources src_miss ON src_miss.galgame_id = galgames.id AND src_miss.source = ?", missingSource).
		Where("src_miss.id IS NULL AND galgames.id > ?", afterID).
		Preload("Aliases").
		Preload("Developer").
		Order("galgames.id").
		Limit(limit).
		Find(&games).Error
	if err != nil {
		return nil, fmt.Errorf("list galgames for enrichment: %w", err)
	}
	return games, nil
}

// CountGalgamesForEnrichment counts galgames eligible for enrichment.
func (r *Repository) CountGalgamesForEnrichment(
	ctx context.Context,
	requireSource, missingSource string,
) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&galgameModel.Galgame{}).
		Joins("JOIN galgame_external_sources src_req ON src_req.galgame_id = galgames.id AND src_req.source = ?", requireSource).
		Joins("LEFT JOIN galgame_external_sources src_miss ON src_miss.galgame_id = galgames.id AND src_miss.source = ?", missingSource).
		Where("src_miss.id IS NULL").
		Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("count galgames for enrichment: %w", err)
	}
	return total, nil
}

// EnrichOverview summarizes enrichment coverage for the admin panel.
type EnrichOverview struct {
	VndbCount         int64 `json:"vndb_count"`
	LinkedCount       int64 `json:"linked_count"`
	UnlinkedCount     int64 `json:"unlinked_count"`
	PendingCandidates int64 `json:"pending_candidates"`
}

func (r *Repository) EnrichOverviewStats(ctx context.Context, providerName string) (EnrichOverview, error) {
	var vndbCount int64
	err := r.db.WithContext(ctx).
		Model(&galgameModel.Galgame{}).
		Joins("JOIN galgame_external_sources src_req ON src_req.galgame_id = galgames.id AND src_req.source = ?", "vndb").
		Count(&vndbCount).Error
	if err != nil {
		return EnrichOverview{}, fmt.Errorf("count vndb galgames: %w", err)
	}
	var linkedCount int64
	err = r.db.WithContext(ctx).
		Model(&galgameModel.Galgame{}).
		Joins("JOIN galgame_external_sources src_req ON src_req.galgame_id = galgames.id AND src_req.source = ?", "vndb").
		Joins("JOIN galgame_external_sources src_target ON src_target.galgame_id = galgames.id AND src_target.source = ?", providerName).
		Count(&linkedCount).Error
	if err != nil {
		return EnrichOverview{}, fmt.Errorf("count linked galgames: %w", err)
	}
	var pending int64
	err = r.db.WithContext(ctx).
		Model(&importerModel.ExternalMatchCandidate{}).
		Where("status = ? AND provider = ?", importerModel.MatchCandidateStatusPending, providerName).
		Count(&pending).Error
	if err != nil {
		return EnrichOverview{}, fmt.Errorf("count pending match candidates: %w", err)
	}
	return EnrichOverview{
		VndbCount:         vndbCount,
		LinkedCount:       linkedCount,
		UnlinkedCount:     vndbCount - linkedCount,
		PendingCandidates: pending,
	}, nil
}

// UpsertMatchCandidate inserts a candidate or refreshes an identical pending
// one. Rejected or approved candidates are never overwritten.
func (r *Repository) UpsertMatchCandidate(ctx context.Context, candidate *importerModel.ExternalMatchCandidate) error {
	updated := r.db.WithContext(ctx).
		Model(&importerModel.ExternalMatchCandidate{}).
		Where(
			"galgame_id = ? AND provider = ? AND external_id = ? AND status = ?",
			candidate.GalgameID, candidate.Provider, candidate.ExternalID,
			importerModel.MatchCandidateStatusPending,
		).
		Updates(map[string]any{
			"confidence": candidate.Confidence,
			"reasons":    candidate.Reasons,
			"preview":    candidate.Preview,
		})
	if updated.Error != nil {
		return fmt.Errorf("refresh match candidate: %w", updated.Error)
	}
	if updated.RowsAffected > 0 {
		return nil
	}
	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(candidate)
	if result.Error != nil {
		return fmt.Errorf("upsert match candidate: %w", result.Error)
	}
	return nil
}

// FindMatchCandidate loads one candidate by ID.
func (r *Repository) FindMatchCandidate(
	ctx context.Context,
	id uint64,
) (*importerModel.ExternalMatchCandidate, error) {
	var candidate importerModel.ExternalMatchCandidate
	err := r.db.WithContext(ctx).First(&candidate, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find match candidate: %w", err)
	}
	return &candidate, nil
}

// ListMatchCandidates pages candidates, best confidence first, and attaches
// galgame titles for the review UI.
func (r *Repository) ListMatchCandidates(
	ctx context.Context,
	status *int16,
	page, limit int,
) ([]importerModel.ExternalMatchCandidate, int64, error) {
	query := r.db.WithContext(ctx).Model(&importerModel.ExternalMatchCandidate{})
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count match candidates: %w", err)
	}
	var candidates []importerModel.ExternalMatchCandidate
	err := query.
		Order("confidence DESC").
		Order("id DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&candidates).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list match candidates: %w", err)
	}
	if len(candidates) > 0 {
		ids := make([]uint, 0, len(candidates))
		for _, candidate := range candidates {
			ids = append(ids, candidate.GalgameID)
		}
		var games []galgameModel.Galgame
		if err := r.db.WithContext(ctx).
			Select("id", "title", "original_title").
			Where("id IN ?", ids).
			Find(&games).Error; err != nil {
			return nil, 0, fmt.Errorf("load match candidate galgames: %w", err)
		}
		titles := make(map[uint]galgameModel.Galgame, len(games))
		for _, game := range games {
			titles[game.ID] = game
		}
		for i := range candidates {
			if game, ok := titles[candidates[i].GalgameID]; ok {
				candidates[i].GalgameTitle = game.Title
				candidates[i].GalgameOriginalTitle = game.OriginalTitle
			}
		}
	}
	return candidates, total, nil
}

// ReviewMatchCandidate moves a pending candidate to the target status.
func (r *Repository) ReviewMatchCandidate(
	ctx context.Context,
	id uint64,
	status int16,
	reviewer *uint,
) error {
	updates := map[string]any{
		"status":      status,
		"reviewed_at": time.Now(),
	}
	if reviewer != nil {
		updates["reviewed_by"] = *reviewer
	}
	result := r.db.WithContext(ctx).
		Model(&importerModel.ExternalMatchCandidate{}).
		Where("id = ? AND status = ?", id, importerModel.MatchCandidateStatusPending).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("review match candidate: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("match candidate already reviewed")
	}
	return nil
}
