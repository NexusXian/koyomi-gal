package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	galgameModel "backend/internal/galgame/model"
	importerModel "backend/internal/importer/model"
	"backend/internal/importer/provider"
	importerRepository "backend/internal/importer/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Enrichable fields accepted by the enrich API.
const (
	EnrichFieldTitle       = "title"
	EnrichFieldDescription = "description"
	EnrichFieldAliases     = "aliases"
	EnrichFieldCover       = "cover"
	EnrichFieldTags        = "tags"
)

var (
	ErrGalgameNotFound        = errors.New("galgame not found")
	ErrInvalidEnrichField     = errors.New("invalid enrich field")
	ErrMatchCandidateNotFound = errors.New("match candidate not found")
	ErrMatchCandidateReviewed = errors.New("match candidate already reviewed")
	ErrMatchInputEmpty        = errors.New("galgame has no usable identity for matching")
)

// EnrichOptions selects which fields the enrichment may fill. Empty fields
// are filled only; Force additionally overwrites maintained values.
type EnrichOptions struct {
	FillTitle       bool
	FillDescription bool
	FillAliases     bool
	FillCover       bool
	FillTags        bool
	Force           bool
}

func DefaultEnrichOptions() EnrichOptions {
	return EnrichOptions{
		FillTitle:       true,
		FillDescription: true,
		FillAliases:     true,
		FillCover:       true,
		FillTags:        true,
	}
}

// ParseEnrichFields converts API field names into options. Unknown fields are
// rejected so callers notice typos.
func ParseEnrichFields(fields []string, force bool) (EnrichOptions, error) {
	if len(fields) == 0 {
		return DefaultEnrichOptions(), nil
	}
	opts := EnrichOptions{Force: force}
	for _, field := range fields {
		switch field {
		case EnrichFieldTitle:
			opts.FillTitle = true
		case EnrichFieldDescription:
			opts.FillDescription = true
		case EnrichFieldAliases:
			opts.FillAliases = true
		case EnrichFieldCover:
			opts.FillCover = true
		case EnrichFieldTags:
			opts.FillTags = true
		default:
			return opts, fmt.Errorf("%w: %s", ErrInvalidEnrichField, field)
		}
	}
	return opts, nil
}

// EnrichResult reports which fields were actually written.
type EnrichResult struct {
	GalgameID     uint      `json:"galgame_id"`
	Provider      string    `json:"provider"`
	ExternalID    string    `json:"external_id"`
	UpdatedFields []string  `json:"updated_fields"`
	AppliedAt     time.Time `json:"applied_at"`
}

// Enrich links an external subject to an existing galgame and fills the
// selected fields. Existing user-maintained values are never overwritten
// unless opts.Force is set.
func (s *Service) Enrich(
	ctx context.Context,
	galgameID uint,
	providerName, externalID string,
	opts EnrichOptions,
) (*EnrichResult, error) {
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	externalID = strings.TrimSpace(externalID)
	selected, err := s.provider(providerName)
	if err != nil {
		return nil, err
	}
	game, err := selected.Get(ctx, externalID)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, ErrExternalGameNotFound
	}

	result := &EnrichResult{
		GalgameID:     galgameID,
		Provider:      providerName,
		ExternalID:    externalID,
		UpdatedFields: []string{},
		AppliedAt:     time.Now(),
	}
	err = s.repository.Transaction(ctx, func(tx *gorm.DB) error {
		galgame, err := findGalgameWithRelations(ctx, tx, galgameID)
		if err != nil {
			return err
		}
		if galgame == nil {
			return ErrGalgameNotFound
		}
		updated, err := applyEnrichment(ctx, tx, galgame, game, opts)
		if err != nil {
			return err
		}
		if err := linkExternalSource(ctx, tx, galgameID, game); err != nil {
			return err
		}
		result.UpdatedFields = updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// applyEnrichment mutates the galgame row with fill-empty semantics and
// returns the list of changed fields.
func applyEnrichment(
	ctx context.Context,
	tx *gorm.DB,
	galgame *galgameModel.Galgame,
	game *provider.ExternalGame,
	opts EnrichOptions,
) ([]string, error) {
	now := time.Now()
	updates := map[string]any{
		"source_type":         mergedSourceType(galgame.SourceType, game.Source),
		"metadata_updated_at": now,
		"updated_at":          now,
	}
	updated := make([]string, 0, 5)

	setIfAllowed := func(field, current, incoming string) {
		if incoming == "" {
			return
		}
		if current != "" && !opts.Force {
			return
		}
		if current == incoming {
			return
		}
		updates[field] = incoming
		updated = append(updated, field)
	}
	if opts.FillTitle {
		setIfAllowed("title", galgame.Title, strings.TrimSpace(game.Title))
	}
	if opts.FillDescription {
		setIfAllowed("description", galgame.Description, strings.TrimSpace(game.Description))
	}
	if opts.FillCover {
		setIfAllowed("cover_url", galgame.CoverURL, strings.TrimSpace(game.CoverURL))
	}
	if err := tx.WithContext(ctx).Model(&galgameModel.Galgame{}).
		Where("id = ?", galgame.ID).
		Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("enrich galgame: %w", err)
	}

	if opts.FillTags {
		added, err := mergeTags(ctx, tx, galgame.ID, enrichmentTags(game))
		if err != nil {
			return nil, err
		}
		if added > 0 {
			updated = append(updated, EnrichFieldTags)
		}
	}
	if opts.FillAliases {
		added, err := mergeAliases(ctx, tx, galgame.ID, importedAliases(game))
		if err != nil {
			return nil, err
		}
		if added > 0 {
			updated = append(updated, EnrichFieldAliases)
		}
	}
	return updated, nil
}

// enrichmentTags selects high-quality external tags for enrichment, ordered
// by community usage, capped at maxImportedTags.
func enrichmentTags(game *provider.ExternalGame) []provider.ExternalTag {
	tags := make([]provider.ExternalTag, 0, len(game.Tags))
	seen := make(map[string]struct{}, len(game.Tags))
	for _, tag := range game.Tags {
		name := strings.TrimSpace(tag.Name)
		key := NormalizeGameTitle(name)
		if name == "" || key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		tag.Name = name
		tags = append(tags, tag)
	}
	sort.SliceStable(tags, func(i, j int) bool { return tags[i].Rating > tags[j].Rating })
	if len(tags) > maxImportedTags {
		tags = tags[:maxImportedTags]
	}
	return tags
}

// mergeTags adds galgame tags that are not linked yet; existing links are
// never removed. Returns the number of new links.
func mergeTags(ctx context.Context, tx *gorm.DB, galgameID uint, external []provider.ExternalTag) (int, error) {
	if len(external) == 0 {
		return 0, nil
	}
	var existing []galgameModel.GalgameTag
	if err := tx.WithContext(ctx).
		Where("galgame_id = ?", galgameID).
		Find(&existing).Error; err != nil {
		return 0, fmt.Errorf("load galgame tags for enrich: %w", err)
	}
	linked := make(map[uint]struct{}, len(existing))
	for _, item := range existing {
		linked[item.TagID] = struct{}{}
	}

	tagIDs, err := resolveTags(ctx, tx, external)
	if err != nil {
		return 0, err
	}
	added := 0
	for _, tagID := range tagIDs {
		if _, exists := linked[tagID]; exists {
			continue
		}
		if err := tx.WithContext(ctx).
			Create(&galgameModel.GalgameTag{GalgameID: galgameID, TagID: tagID}).Error; err != nil {
			return added, fmt.Errorf("link enriched tag: %w", err)
		}
		added++
	}
	return added, nil
}

// mergeAliases adds aliases that do not exist yet; existing aliases are never
// removed. Returns the number of new aliases.
func mergeAliases(ctx context.Context, tx *gorm.DB, galgameID uint, aliases []string) (int, error) {
	if len(aliases) == 0 {
		return 0, nil
	}
	var existing []galgameModel.Alias
	if err := tx.WithContext(ctx).
		Where("galgame_id = ?", galgameID).
		Find(&existing).Error; err != nil {
		return 0, fmt.Errorf("load galgame aliases for enrich: %w", err)
	}
	seen := make(map[string]struct{}, len(existing))
	for _, alias := range existing {
		seen[strings.ToLower(strings.TrimSpace(alias.Alias))] = struct{}{}
	}

	added := 0
	for _, alias := range aliases {
		key := strings.ToLower(strings.TrimSpace(alias))
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		if err := tx.WithContext(ctx).
			Create(&galgameModel.Alias{GalgameID: galgameID, Alias: alias}).Error; err != nil {
			return added, fmt.Errorf("create enriched alias: %w", err)
		}
		added++
	}
	return added, nil
}

// linkExternalSource upserts the (galgame, source, external_id) mapping and
// keeps a single row per source for the galgame.
func linkExternalSource(ctx context.Context, tx *gorm.DB, galgameID uint, game *provider.ExternalGame) error {
	now := time.Now()
	mapping := &importerModel.GalgameExternalSource{
		GalgameID:           galgameID,
		Source:              game.Source,
		ExternalID:          game.ExternalID,
		URL:                 externalSourceURL(game.Source, game.ExternalID),
		ExternalRating:      game.Rating,
		ExternalRatingCount: game.RatingCount,
		RawMetadata:         game.Raw,
		LastSyncedAt:        &now,
	}
	result := tx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "source"}, {Name: "external_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"galgame_id", "url", "external_rating", "external_rating_count",
				"raw_metadata", "last_synced_at", "updated_at",
			}),
		}).
		Create(mapping)
	if result.Error != nil {
		return fmt.Errorf("link external source: %w", result.Error)
	}
	if err := tx.WithContext(ctx).
		Where("galgame_id = ? AND source = ? AND external_id <> ?", galgameID, game.Source, game.ExternalID).
		Delete(&importerModel.GalgameExternalSource{}).Error; err != nil {
		return fmt.Errorf("replace stale external source: %w", err)
	}
	return nil
}

// ExternalCandidate is the API-facing match candidate for one galgame.
type ExternalCandidate struct {
	ExternalID    string     `json:"external_id"`
	Source        string     `json:"source"`
	Title         string     `json:"title"`
	OriginalTitle string     `json:"original_title"`
	CoverURL      string     `json:"cover_url"`
	ReleaseDate   *time.Time `json:"release_date"`
	Rating        *float64   `json:"rating"`
	RatingCount   *int       `json:"rating_count"`
	Confidence    float64    `json:"confidence"`
	Reasons       []string   `json:"reasons"`
	Linked        bool       `json:"linked"`
}

// SearchExternalCandidates searches the provider for the galgame and scores
// the results. It never persists anything.
func (s *Service) SearchExternalCandidates(
	ctx context.Context,
	galgameID uint,
	providerName string,
) ([]ExternalCandidate, error) {
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	selected, err := s.provider(providerName)
	if err != nil {
		return nil, err
	}
	input, err := s.buildMatchInput(ctx, galgameID)
	if err != nil {
		return nil, err
	}
	query := input.SearchQuery()
	if query == "" {
		return nil, ErrMatchInputEmpty
	}
	results, err := selected.Search(ctx, query, maxBangumiSearchResults)
	if err != nil {
		return nil, err
	}

	sources, err := s.repository.ListExternalSources(ctx, galgameID)
	if err != nil {
		return nil, err
	}
	linkedForGame := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		if source.Source == providerName {
			linkedForGame[source.ExternalID] = struct{}{}
		}
	}

	matches := MatchBangumiCandidates(*input, results)
	candidates := make([]ExternalCandidate, 0, len(matches))
	for _, match := range matches {
		_, isLinked := linkedForGame[match.Game.ExternalID]
		candidates = append(candidates, ExternalCandidate{
			ExternalID:    match.Game.ExternalID,
			Source:        match.Game.Source,
			Title:         match.Game.Title,
			OriginalTitle: match.Game.OriginalTitle,
			CoverURL:      match.Game.CoverURL,
			ReleaseDate:   match.Game.ReleaseDate,
			Rating:        match.Game.Rating,
			RatingCount:   match.Game.RatingCount,
			Confidence:    match.Confidence,
			Reasons:       match.Reasons,
			Linked:        isLinked,
		})
	}
	return candidates, nil
}

const maxBangumiSearchResults = 20

// buildMatchInput loads the galgame and converts it into matcher input.
func (s *Service) buildMatchInput(ctx context.Context, galgameID uint) (*MatchInput, error) {
	galgame, err := s.repository.FindGalgame(ctx, galgameID)
	if err != nil {
		return nil, err
	}
	if galgame == nil {
		return nil, ErrGalgameNotFound
	}
	input := &MatchInput{
		Title:         galgame.Title,
		OriginalTitle: galgame.OriginalTitle,
		RomajiTitle:   galgame.RomajiTitle,
		ReleaseDate:   galgame.ReleaseDate,
	}
	for _, alias := range galgame.Aliases {
		if strings.TrimSpace(alias.Alias) != "" {
			input.Aliases = append(input.Aliases, alias.Alias)
		}
	}
	if galgame.Developer != nil {
		input.Developer = galgame.Developer.Name
	}
	return input, nil
}

// candidatePreview is persisted on external_match_candidates for review UI.
type candidatePreview struct {
	ExternalID    string   `json:"external_id"`
	Title         string   `json:"title"`
	OriginalTitle string   `json:"original_title"`
	CoverURL      string   `json:"cover_url"`
	Rating        *float64 `json:"rating"`
	RatingCount   *int     `json:"rating_count"`
	URL           string   `json:"url"`
}

func newCandidatePreview(game provider.ExternalGame) candidatePreview {
	return candidatePreview{
		ExternalID:    game.ExternalID,
		Title:         game.Title,
		OriginalTitle: game.OriginalTitle,
		CoverURL:      game.CoverURL,
		Rating:        game.Rating,
		RatingCount:   game.RatingCount,
		URL:           externalSourceURL(game.Source, game.ExternalID),
	}
}

// SaveMatchCandidates upserts review-band candidates for the galgame.
// Previously rejected or approved candidates are left untouched.
func (s *Service) SaveMatchCandidates(ctx context.Context, galgameID uint, candidates []MatchCandidate) error {
	for _, candidate := range candidates {
		if candidate.Confidence < reviewThreshold || candidate.Confidence >= autoMatchThreshold {
			continue
		}
		reasons, err := json.Marshal(candidate.Reasons)
		if err != nil {
			return fmt.Errorf("encode match reasons: %w", err)
		}
		preview, err := json.Marshal(newCandidatePreview(candidate.Game))
		if err != nil {
			return fmt.Errorf("encode match preview: %w", err)
		}
		row := &importerModel.ExternalMatchCandidate{
			GalgameID:  galgameID,
			Provider:   candidate.Game.Source,
			ExternalID: candidate.Game.ExternalID,
			Confidence: candidate.Confidence,
			Reasons:    reasons,
			Preview:    preview,
		}
		if err := s.repository.UpsertMatchCandidate(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

// ApproveMatchCandidate links the candidate's external subject to its
// galgame and runs a default enrichment, then marks it approved.
func (s *Service) ApproveMatchCandidate(ctx context.Context, candidateID uint64, reviewer *uint) (*EnrichResult, error) {
	candidate, err := s.repository.FindMatchCandidate(ctx, candidateID)
	if err != nil {
		return nil, err
	}
	if candidate == nil {
		return nil, ErrMatchCandidateNotFound
	}
	if candidate.Status != importerModel.MatchCandidateStatusPending {
		return nil, ErrMatchCandidateReviewed
	}
	result, err := s.Enrich(ctx, candidate.GalgameID, candidate.Provider, candidate.ExternalID, DefaultEnrichOptions())
	if err != nil {
		return nil, err
	}
	if err := s.repository.ReviewMatchCandidate(ctx, candidateID, importerModel.MatchCandidateStatusApproved, reviewer); err != nil {
		return nil, err
	}
	return result, nil
}

// RejectMatchCandidate marks a pending candidate as rejected.
func (s *Service) RejectMatchCandidate(ctx context.Context, candidateID uint64, reviewer *uint) error {
	candidate, err := s.repository.FindMatchCandidate(ctx, candidateID)
	if err != nil {
		return err
	}
	if candidate == nil {
		return ErrMatchCandidateNotFound
	}
	if candidate.Status != importerModel.MatchCandidateStatusPending {
		return ErrMatchCandidateReviewed
	}
	return s.repository.ReviewMatchCandidate(ctx, candidateID, importerModel.MatchCandidateStatusRejected, reviewer)
}

// MatchCandidateBatchResult reports the review outcome of one candidate ID.
// Err is nil when the candidate was reviewed successfully.
type MatchCandidateBatchResult struct {
	ID  uint64
	Err error
}

// MatchCandidateBatchSummary aggregates per-candidate outcomes of one batch.
type MatchCandidateBatchSummary struct {
	Results       []MatchCandidateBatchResult
	SucceededCount int
	FailedCount    int
}

// MaxMatchCandidateBatchSize caps how many candidates one batch request may
// review; each approval calls the external provider, so batches stay bounded.
const MaxMatchCandidateBatchSize = 100

// BatchApproveMatchCandidates approves candidates sequentially. Every item is
// reported independently so a single failure never aborts the batch.
func (s *Service) BatchApproveMatchCandidates(ctx context.Context, ids []uint64, reviewer *uint) MatchCandidateBatchSummary {
	summary := MatchCandidateBatchSummary{Results: make([]MatchCandidateBatchResult, 0, len(ids))}
	for _, id := range ids {
		result := MatchCandidateBatchResult{ID: id}
		if _, err := s.ApproveMatchCandidate(ctx, id, reviewer); err != nil {
			result.Err = err
			summary.FailedCount++
		} else {
			summary.SucceededCount++
		}
		summary.Results = append(summary.Results, result)
	}
	return summary
}

// BatchRejectMatchCandidates rejects candidates sequentially with the same
// per-item reporting as BatchApproveMatchCandidates.
func (s *Service) BatchRejectMatchCandidates(ctx context.Context, ids []uint64, reviewer *uint) MatchCandidateBatchSummary {
	summary := MatchCandidateBatchSummary{Results: make([]MatchCandidateBatchResult, 0, len(ids))}
	for _, id := range ids {
		result := MatchCandidateBatchResult{ID: id}
		if err := s.RejectMatchCandidate(ctx, id, reviewer); err != nil {
			result.Err = err
			summary.FailedCount++
		} else {
			summary.SucceededCount++
		}
		summary.Results = append(summary.Results, result)
	}
	return summary
}

// ListMatchCandidates returns review candidates with galgame context.
func (s *Service) ListMatchCandidates(
	ctx context.Context,
	status *int16,
	page, limit int,
) ([]importerModel.ExternalMatchCandidate, int64, error) {
	return s.repository.ListMatchCandidates(ctx, status, page, limit)
}

// EnrichOverview summarizes enrichment coverage for the admin panel.
func (s *Service) EnrichOverview(ctx context.Context, providerName string) (importerRepository.EnrichOverview, error) {
	return s.repository.EnrichOverviewStats(ctx, providerName)
}
