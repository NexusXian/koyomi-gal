package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	galgameModel "backend/internal/galgame/model"
	galgameService "backend/internal/galgame/service"
	importerModel "backend/internal/importer/model"
	"backend/internal/importer/provider"
	importerRepository "backend/internal/importer/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	DuplicateActionError        = "error"
	DuplicateActionCreateNew    = "create_new"
	DuplicateActionLinkExisting = "link_existing"

	DuplicateStatusNone            = "none"
	DuplicateStatusPossible        = "possible"
	DuplicateStatusAlreadyImported = "already_imported"
)

var (
	ErrProviderNotFound        = errors.New("import provider not found")
	ErrExternalGameNotFound    = errors.New("external game not found")
	ErrInvalidDuplicateAction  = errors.New("invalid duplicate action")
	ErrExistingGalgameRequired = errors.New("existing galgame ID is required")
	ErrExistingGalgameNotFound = errors.New("existing galgame not found")
	errExternalSourceConflict  = errors.New("external source already exists")
)

type DuplicateCandidate struct {
	ID            uint       `json:"id"`
	Title         string     `json:"title"`
	OriginalTitle string     `json:"original_title"`
	ReleaseDate   *time.Time `json:"release_date"`
}

type PreviewResult struct {
	Game            *provider.ExternalGame `json:"game"`
	DuplicateStatus string                 `json:"duplicate_status"`
	Candidates      []DuplicateCandidate   `json:"candidates"`
}

type ImportInput struct {
	Provider            string
	ExternalID          string
	DuplicateAction     string
	ExistingGalgameID   *uint
	ForceMetadataUpdate bool
	CreatedBy           *uint
	RecordContribution  bool
}

type ImportResult struct {
	DuplicateStatus   string               `json:"duplicate_status"`
	GalgameID         *uint                `json:"galgame_id,omitempty"`
	ExistingGalgameID *uint                `json:"existing_galgame_id,omitempty"`
	Candidates        []DuplicateCandidate `json:"candidates"`
}

type Service struct {
	repository    *importerRepository.Repository
	providers     map[string]provider.Provider
	contributions *galgameService.ContributionService
	batchEnqueuer func(ctx context.Context, jobID int64) error
}

func NewService(
	repository *importerRepository.Repository,
	providers map[string]provider.Provider,
	contributions *galgameService.ContributionService,
) *Service {
	registered := make(map[string]provider.Provider, len(providers))
	for name, item := range providers {
		registered[strings.ToLower(strings.TrimSpace(name))] = item
	}
	return &Service{repository: repository, providers: registered, contributions: contributions}
}

func (s *Service) Providers() []string {
	providers := make([]string, 0, len(s.providers))
	for name := range s.providers {
		providers = append(providers, name)
	}
	return providers
}

func (s *Service) Search(
	ctx context.Context,
	providerName, query string,
	limit int,
) ([]PreviewResult, error) {
	selected, err := s.provider(providerName)
	if err != nil {
		return nil, err
	}
	games, err := selected.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	results := make([]PreviewResult, 0, len(games))
	for i := range games {
		preview, err := s.preview(ctx, &games[i])
		if err != nil {
			return nil, err
		}
		results = append(results, *preview)
	}
	return results, nil
}

func (s *Service) Get(
	ctx context.Context,
	providerName, externalID string,
) (*PreviewResult, error) {
	selected, err := s.provider(providerName)
	if err != nil {
		return nil, err
	}
	game, err := selected.Get(ctx, strings.TrimSpace(externalID))
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, ErrExternalGameNotFound
	}
	return s.preview(ctx, game)
}

func (s *Service) Import(ctx context.Context, input ImportInput) (*ImportResult, error) {
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	input.ExternalID = strings.TrimSpace(input.ExternalID)
	if input.DuplicateAction == "" {
		input.DuplicateAction = DuplicateActionError
	}
	if input.DuplicateAction != DuplicateActionError &&
		input.DuplicateAction != DuplicateActionCreateNew &&
		input.DuplicateAction != DuplicateActionLinkExisting {
		return nil, ErrInvalidDuplicateAction
	}
	if input.DuplicateAction == DuplicateActionLinkExisting &&
		(input.ExistingGalgameID == nil || *input.ExistingGalgameID == 0) {
		return nil, ErrExistingGalgameRequired
	}

	selected, err := s.provider(input.Provider)
	if err != nil {
		return nil, err
	}
	existingSource, err := s.repository.FindExternalSource(ctx, input.Provider, input.ExternalID)
	if err != nil {
		return nil, err
	}
	if existingSource != nil {
		return alreadyImportedResult(existingSource.GalgameID), nil
	}
	game, err := selected.Get(ctx, input.ExternalID)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, ErrExternalGameNotFound
	}
	return s.importFetched(ctx, game, input)
}

// importFetched imports an already-fetched external game. Batch imports call
// it directly with page data to avoid one provider request per item.
func (s *Service) importFetched(
	ctx context.Context,
	game *provider.ExternalGame,
	input ImportInput,
) (*ImportResult, error) {
	if input.DuplicateAction == "" {
		input.DuplicateAction = DuplicateActionError
	}
	preview, err := s.preview(ctx, game)
	if err != nil {
		return nil, err
	}
	if preview.DuplicateStatus == DuplicateStatusAlreadyImported && len(preview.Candidates) > 0 {
		return alreadyImportedResult(preview.Candidates[0].ID), nil
	}
	if input.DuplicateAction == DuplicateActionError && preview.DuplicateStatus == DuplicateStatusPossible {
		return &ImportResult{
			DuplicateStatus: DuplicateStatusPossible,
			Candidates:      preview.Candidates,
		}, nil
	}

	var galgameID uint
	err = s.repository.Transaction(ctx, func(tx *gorm.DB) error {
		if input.DuplicateAction == DuplicateActionLinkExisting {
			existing, err := findGalgameWithRelations(ctx, tx, *input.ExistingGalgameID)
			if err != nil {
				return err
			}
			if existing == nil {
				return ErrExistingGalgameNotFound
			}
			galgameID = existing.ID
			if input.ForceMetadataUpdate {
				if err := updateGalgameMetadata(ctx, tx, existing, game); err != nil {
					return err
				}
			} else {
				now := time.Now()
				if err := tx.WithContext(ctx).Model(&galgameModel.Galgame{}).
					Where("id = ?", existing.ID).
					Updates(map[string]any{
						"source_type":         mergedSourceType(existing.SourceType, game.Source),
						"metadata_updated_at": now,
						"updated_at":          now,
					}).Error; err != nil {
					return fmt.Errorf("update linked galgame metadata timestamp: %w", err)
				}
			}
		} else {
			created, err := createGalgame(ctx, tx, game, input.CreatedBy)
			if err != nil {
				return err
			}
			galgameID = created.ID
			if input.RecordContribution && input.CreatedBy != nil && *input.CreatedBy != 0 && s.contributions != nil {
				sourceType := string(galgameModel.ContributionSourceGalgameCreate)
				sourceID := created.ID
				if err := s.contributions.RecordContribution(ctx, galgameService.RecordContributionInput{
					GalgameID:  created.ID,
					UserID:     *input.CreatedBy,
					Action:     galgameModel.ContributionActionCreate,
					SourceType: &sourceType,
					SourceID:   &sourceID,
				}, tx); err != nil {
					return err
				}
			}
		}
		return createExternalSource(ctx, tx, galgameID, game)
	})
	if errors.Is(err, errExternalSourceConflict) {
		existing, lookupErr := s.repository.FindExternalSource(ctx, game.Source, game.ExternalID)
		if lookupErr != nil {
			return nil, lookupErr
		}
		if existing != nil {
			return alreadyImportedResult(existing.GalgameID), nil
		}
	}
	if err != nil {
		return nil, err
	}
	return &ImportResult{DuplicateStatus: DuplicateStatusNone, GalgameID: &galgameID, Candidates: []DuplicateCandidate{}}, nil
}

func (s *Service) preview(ctx context.Context, game *provider.ExternalGame) (*PreviewResult, error) {
	existing, err := s.repository.FindExternalSource(ctx, game.Source, game.ExternalID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return &PreviewResult{
			Game:            game,
			DuplicateStatus: DuplicateStatusAlreadyImported,
			Candidates:      []DuplicateCandidate{{ID: existing.GalgameID}},
		}, nil
	}
	candidates, err := s.possibleDuplicates(ctx, game)
	if err != nil {
		return nil, err
	}
	status := DuplicateStatusNone
	if len(candidates) > 0 {
		status = DuplicateStatusPossible
	}
	return &PreviewResult{Game: game, DuplicateStatus: status, Candidates: candidates}, nil
}

func (s *Service) possibleDuplicates(
	ctx context.Context,
	game *provider.ExternalGame,
) ([]DuplicateCandidate, error) {
	rows, err := s.repository.FindDuplicateCandidates(ctx, game.ReleaseDate, 200)
	if err != nil {
		return nil, err
	}
	externalNames := map[string]struct{}{}
	for _, name := range []string{game.Title, game.OriginalTitle, game.RomajiTitle} {
		if normalized := normalizeName(name); normalized != "" {
			externalNames[normalized] = struct{}{}
		}
	}
	candidates := make([]DuplicateCandidate, 0)
	for _, row := range rows {
		matched := false
		for _, name := range []string{row.Title, row.OriginalTitle, row.RomajiTitle} {
			if _, exists := externalNames[normalizeName(name)]; exists {
				matched = true
				break
			}
		}
		if matched {
			candidates = append(candidates, DuplicateCandidate{
				ID: row.ID, Title: row.Title, OriginalTitle: row.OriginalTitle, ReleaseDate: row.ReleaseDate,
			})
		}
	}
	return candidates, nil
}

func (s *Service) provider(name string) (provider.Provider, error) {
	selected := s.providers[strings.ToLower(strings.TrimSpace(name))]
	if selected == nil {
		return nil, ErrProviderNotFound
	}
	return selected, nil
}

func alreadyImportedResult(galgameID uint) *ImportResult {
	return &ImportResult{
		DuplicateStatus:   DuplicateStatusAlreadyImported,
		ExistingGalgameID: &galgameID,
		Candidates:        []DuplicateCandidate{},
	}
}

func findGalgameWithRelations(ctx context.Context, tx *gorm.DB, id uint) (*galgameModel.Galgame, error) {
	var game galgameModel.Galgame
	err := tx.WithContext(ctx).Preload("Aliases").Preload("Tags").First(&game, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find existing galgame: %w", err)
	}
	return &game, nil
}

func createGalgame(
	ctx context.Context,
	tx *gorm.DB,
	game *provider.ExternalGame,
	createdBy *uint,
) (*galgameModel.Galgame, error) {
	developerID, err := resolveDeveloper(ctx, tx, game.Developer)
	if err != nil {
		return nil, err
	}
	tagIDs, err := resolveTags(ctx, tx, importedTags(game))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	created := &galgameModel.Galgame{
		Title:             strings.TrimSpace(game.Title),
		OriginalTitle:     strings.TrimSpace(game.OriginalTitle),
		RomajiTitle:       strings.TrimSpace(game.RomajiTitle),
		Slug:              game.Source + "-" + game.ExternalID,
		Description:       strings.TrimSpace(game.Description),
		CoverURL:          strings.TrimSpace(game.CoverURL),
		DeveloperID:       developerID,
		ReleaseDate:       game.ReleaseDate,
		OriginalLanguage:  strings.TrimSpace(game.OriginalLanguage),
		LengthMinutes:     game.LengthMinutes,
		SourceType:        sourceTypeForProvider(game.Source),
		MetadataUpdatedAt: &now,
		Status:            galgameModel.GalgameStatusPublished,
		CreatedBy:         createdBy,
	}
	if created.Title == "" {
		created.Title = created.RomajiTitle
	}
	if created.Title == "" {
		return nil, errors.New("external game title is empty")
	}
	if err := tx.WithContext(ctx).Create(created).Error; err != nil {
		return nil, fmt.Errorf("create imported galgame: %w", err)
	}
	if err := replaceAliases(ctx, tx, created.ID, importedAliases(game)); err != nil {
		return nil, err
	}
	if err := replaceTags(ctx, tx, created.ID, tagIDs); err != nil {
		return nil, err
	}
	return created, nil
}

func updateGalgameMetadata(
	ctx context.Context,
	tx *gorm.DB,
	existing *galgameModel.Galgame,
	game *provider.ExternalGame,
) error {
	developerID, err := resolveDeveloper(ctx, tx, game.Developer)
	if err != nil {
		return err
	}
	tagIDs, err := resolveTags(ctx, tx, importedTags(game))
	if err != nil {
		return err
	}
	now := time.Now()
	updates := map[string]any{
		"title":               strings.TrimSpace(game.Title),
		"original_title":      strings.TrimSpace(game.OriginalTitle),
		"romaji_title":        strings.TrimSpace(game.RomajiTitle),
		"description":         strings.TrimSpace(game.Description),
		"cover_url":           strings.TrimSpace(game.CoverURL),
		"developer_id":        developerID,
		"release_date":        game.ReleaseDate,
		"original_language":   strings.TrimSpace(game.OriginalLanguage),
		"length_minutes":      game.LengthMinutes,
		"source_type":         mergedSourceType(existing.SourceType, game.Source),
		"metadata_updated_at": now,
		"updated_at":          now,
	}
	if strings.TrimSpace(game.Title) == "" {
		updates["title"] = existing.Title
	}
	if err := tx.WithContext(ctx).Model(&galgameModel.Galgame{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("force update galgame metadata: %w", err)
	}
	if err := replaceAliases(ctx, tx, existing.ID, importedAliases(game)); err != nil {
		return err
	}
	return replaceTags(ctx, tx, existing.ID, tagIDs)
}

func resolveDeveloper(
	ctx context.Context,
	tx *gorm.DB,
	external *provider.ExternalDeveloper,
) (*uint, error) {
	if external == nil || strings.TrimSpace(external.Name) == "" {
		return nil, nil
	}
	name := strings.TrimSpace(external.Name)
	slug := normalizeSlug(name)
	var developer galgameModel.Developer
	err := tx.WithContext(ctx).
		Where("LOWER(BTRIM(name)) = ? OR slug = ?", strings.ToLower(name), slug).
		Order("id").First(&developer).Error
	if err == nil {
		return &developer.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("find imported developer: %w", err)
	}
	developer = galgameModel.Developer{Name: name, Slug: slug}
	result := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&developer)
	if result.Error != nil {
		return nil, fmt.Errorf("create imported developer: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		if err := tx.WithContext(ctx).Where("slug = ?", slug).First(&developer).Error; err != nil {
			return nil, fmt.Errorf("find concurrently created developer: %w", err)
		}
	}
	return &developer.ID, nil
}

func resolveTags(ctx context.Context, tx *gorm.DB, external []provider.ExternalTag) ([]uint, error) {
	ids := make([]uint, 0, len(external))
	for _, item := range external {
		slug := normalizeSlug(item.Name)
		var tag galgameModel.Tag
		err := tx.WithContext(ctx).
			Where("LOWER(BTRIM(name)) = ? OR slug = ?", strings.ToLower(item.Name), slug).
			Order("id").First(&tag).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tag = galgameModel.Tag{Name: item.Name, Slug: slug}
			result := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&tag)
			if result.Error != nil {
				return nil, fmt.Errorf("create imported tag: %w", result.Error)
			}
			if result.RowsAffected == 0 {
				if err := tx.WithContext(ctx).Where("name = ? OR slug = ?", item.Name, slug).First(&tag).Error; err != nil {
					return nil, fmt.Errorf("find concurrently created tag: %w", err)
				}
			}
		} else if err != nil {
			return nil, fmt.Errorf("find imported tag: %w", err)
		}
		ids = append(ids, tag.ID)
	}
	return ids, nil
}

func replaceAliases(ctx context.Context, tx *gorm.DB, galgameID uint, aliases []string) error {
	if err := tx.WithContext(ctx).Where("galgame_id = ?", galgameID).Delete(&galgameModel.Alias{}).Error; err != nil {
		return fmt.Errorf("delete imported aliases: %w", err)
	}
	if len(aliases) == 0 {
		return nil
	}
	rows := make([]galgameModel.Alias, 0, len(aliases))
	for _, alias := range aliases {
		rows = append(rows, galgameModel.Alias{GalgameID: galgameID, Alias: alias})
	}
	if err := tx.WithContext(ctx).Create(&rows).Error; err != nil {
		return fmt.Errorf("create imported aliases: %w", err)
	}
	return nil
}

func replaceTags(ctx context.Context, tx *gorm.DB, galgameID uint, tagIDs []uint) error {
	if err := tx.WithContext(ctx).Where("galgame_id = ?", galgameID).Delete(&galgameModel.GalgameTag{}).Error; err != nil {
		return fmt.Errorf("delete imported tags: %w", err)
	}
	for _, tagID := range tagIDs {
		if err := tx.WithContext(ctx).Create(&galgameModel.GalgameTag{GalgameID: galgameID, TagID: tagID}).Error; err != nil {
			return fmt.Errorf("create imported galgame tag: %w", err)
		}
	}
	return nil
}

func createExternalSource(ctx context.Context, tx *gorm.DB, galgameID uint, game *provider.ExternalGame) error {
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
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "source"}, {Name: "external_id"}}, DoNothing: true}).
		Create(mapping)
	if result.Error != nil {
		return fmt.Errorf("create external source: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errExternalSourceConflict
	}
	return nil
}

func externalSourceURL(source, externalID string) string {
	if source == "vndb" {
		return "https://vndb.org/" + externalID
	}
	return ""
}

func sourceTypeForProvider(name string) int16 {
	switch name {
	case "vndb":
		return galgameModel.GalgameSourceVNDB
	case "bangumi":
		return galgameModel.GalgameSourceBangumi
	default:
		return galgameModel.GalgameSourceManual
	}
}

func mergedSourceType(current int16, providerName string) int16 {
	providerType := sourceTypeForProvider(providerName)
	switch {
	case current == providerType:
		return current
	case current == galgameModel.GalgameSourceManual:
		return providerType
	default:
		return galgameModel.GalgameSourceMixed
	}
}
