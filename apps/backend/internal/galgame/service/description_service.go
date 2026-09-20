package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	contributionModel "backend/internal/contribution/model"
	contributionService "backend/internal/contribution/service"
	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"
	relationModel "backend/internal/relation/model"
	"backend/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const maxDescriptionContentLength = 100000

var (
	ErrInvalidDescriptionLanguage = errors.New("description language is invalid")
	ErrInvalidDescriptionSource   = errors.New("description source type is invalid")
	ErrInvalidDescriptionURL      = errors.New("description source url is invalid")
	ErrDuplicateDescriptionLang   = errors.New("descriptions contain duplicate language")
	ErrDescriptionTooLong         = errors.New("description content is too long")
	ErrDescriptionNameTooLong     = errors.New("description source name is too long")
)

// DescriptionService owns the per-language galgame descriptions and their
// structured source metadata.
type DescriptionService struct {
	galgames      *repository.GalgameRepository
	descriptions  *repository.GalgameDescriptionRepository
	contributions *contributionService.ContributionService
}

func NewDescriptionService(
	galgames *repository.GalgameRepository,
	descriptions *repository.GalgameDescriptionRepository,
) *DescriptionService {
	return &DescriptionService{galgames: galgames, descriptions: descriptions}
}

func (s *DescriptionService) SetContributionService(contributions *contributionService.ContributionService) {
	s.contributions = contributions
}

func (s *DescriptionService) GetGameDescriptions(
	ctx context.Context,
	galgameID uint,
) ([]model.GalgameDescription, error) {
	galgame, err := s.galgames.FindByID(ctx, galgameID)
	if err != nil {
		logger.Error("find galgame for descriptions", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return nil, err
	}
	if galgame == nil {
		return nil, ErrGalgameNotFound
	}
	return s.descriptions.FindByGalgameID(ctx, galgameID)
}

// UpsertGameDescriptions validates and writes the submitted languages in one
// transaction and records an edit contribution for published games.
func (s *DescriptionService) UpsertGameDescriptions(
	ctx context.Context,
	galgameID uint,
	inputs []dto.GalgameDescriptionInput,
	actorID uint,
) ([]model.GalgameDescription, error) {
	rows, err := ValidateDescriptionInputs(inputs)
	if err != nil {
		return nil, err
	}
	galgame, err := s.galgames.FindByID(ctx, galgameID)
	if err != nil {
		logger.Error("find galgame for description upsert", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return nil, err
	}
	if galgame == nil {
		return nil, ErrGalgameNotFound
	}
	existing, err := s.descriptions.FindByGalgameID(ctx, galgameID)
	if err != nil {
		return nil, err
	}

	write := func(tx *gorm.DB) error {
		repo := repository.NewGalgameDescriptionRepository(tx)
		for i := range rows {
			rows[i].GalgameID = galgameID
			if err := repo.Upsert(ctx, &rows[i]); err != nil {
				return err
			}
		}
		if descriptionsChanged(existing, rows) && galgame.Status == model.GalgameStatusPublished &&
			s.contributions != nil && actorID != 0 {
			return s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
				TargetType: relationModel.WorkTypeGalgame,
				TargetID:   galgameID,
				UserID:     actorID,
				Action:     contributionModel.ContributionActionEdit,
			}, tx)
		}
		return nil
	}
	if s.contributions != nil {
		err = s.contributions.Transaction(ctx, write)
	} else {
		err = s.descriptionsUpsertTransaction(ctx, write)
	}
	if err != nil {
		logger.Error("upsert galgame descriptions", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return nil, err
	}
	return s.descriptions.FindByGalgameID(ctx, galgameID)
}

func (s *DescriptionService) descriptionsUpsertTransaction(
	ctx context.Context,
	write func(tx *gorm.DB) error,
) error {
	return s.descriptions.Transaction(ctx, write)
}

// DeleteGameDescription removes one language's description row entirely.
func (s *DescriptionService) DeleteGameDescription(
	ctx context.Context,
	galgameID uint,
	language string,
	actorID uint,
) error {
	if !model.ValidDescriptionLanguage(language) {
		return ErrInvalidDescriptionLanguage
	}
	galgame, err := s.galgames.FindByID(ctx, galgameID)
	if err != nil {
		logger.Error("find galgame for description delete", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return err
	}
	if galgame == nil {
		return ErrGalgameNotFound
	}
	existing, err := s.descriptions.FindByGalgameIDAndLanguage(ctx, galgameID, language)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrGalgameNotFound
	}
	delete := func(tx *gorm.DB) error {
		if err := repository.NewGalgameDescriptionRepository(tx).Delete(ctx, galgameID, language); err != nil {
			return err
		}
		if galgame.Status == model.GalgameStatusPublished && s.contributions != nil && actorID != 0 {
			return s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
				TargetType: relationModel.WorkTypeGalgame,
				TargetID:   galgameID,
				UserID:     actorID,
				Action:     contributionModel.ContributionActionEdit,
			}, tx)
		}
		return nil
	}
	if s.contributions != nil {
		err = s.contributions.Transaction(ctx, delete)
	} else {
		err = s.descriptionsUpsertTransaction(ctx, delete)
	}
	if err != nil {
		logger.Error("delete galgame description", zap.Uint("galgame_id", galgameID), zap.String("language", language), zap.Error(err))
		return err
	}
	return nil
}

// DefaultDescriptionSource returns the conventional source for a language,
// used when a payload omits source fields. Saved values always come from
// the database; this only seeds defaults.
func DefaultDescriptionSource(language string) (sourceType, sourceName string, isOfficial bool) {
	switch language {
	case model.LanguageZhCN:
		return model.DescriptionSourceNextMoe, "NextMoe 资料库", false
	case model.LanguageEnUS:
		return model.DescriptionSourceVNDB, "VNDB", false
	case model.LanguageJaJP:
		return model.DescriptionSourceOfficial, "游戏官网", true
	default:
		return model.DescriptionSourceUnknown, "", false
	}
}

// ValidateDescriptionInputs normalizes and validates submitted descriptions,
// returning ready-to-insert rows (GalgameID must be set by the caller).
func ValidateDescriptionInputs(inputs []dto.GalgameDescriptionInput) ([]model.GalgameDescription, error) {
	if len(inputs) == 0 {
		return nil, ErrInvalidCatalogInput
	}
	seen := make(map[string]struct{}, len(inputs))
	rows := make([]model.GalgameDescription, 0, len(inputs))
	for _, input := range inputs {
		language := strings.TrimSpace(input.Language)
		if !model.ValidDescriptionLanguage(language) {
			return nil, ErrInvalidDescriptionLanguage
		}
		if _, duplicate := seen[language]; duplicate {
			return nil, ErrDuplicateDescriptionLang
		}
		seen[language] = struct{}{}

		sourceType := strings.ToLower(strings.TrimSpace(input.SourceType))
		isOfficial := input.IsOfficial
		if sourceType == "" {
			sourceType, _, isOfficial = DefaultDescriptionSource(language)
		} else if !model.ValidDescriptionSourceType(sourceType) {
			return nil, ErrInvalidDescriptionSource
		}

		sourceName := strings.TrimSpace(input.SourceName)
		if utf8.RuneCountInString(sourceName) > 128 {
			return nil, ErrDescriptionNameTooLong
		}
		if sourceName == "" {
			sourceName = model.DefaultDescriptionSourceName(sourceType)
		}

		sourceURL := strings.TrimSpace(input.SourceURL)
		if sourceURL != "" {
			parsed, err := url.Parse(sourceURL)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
				return nil, ErrInvalidDescriptionURL
			}
		}

		content := strings.TrimSpace(strings.ReplaceAll(input.Content, "\r\n", "\n"))
		if utf8.RuneCountInString(content) > maxDescriptionContentLength {
			return nil, ErrDescriptionTooLong
		}

		rows = append(rows, model.GalgameDescription{
			Language:   language,
			Content:    content,
			SourceType: sourceType,
			SourceName: sourceName,
			SourceURL:  sourceURL,
			IsOfficial: isOfficial,
		})
	}
	return rows, nil
}

// descriptionsChanged reports whether any upserted row differs from the
// stored state, so unrelated edits do not record contributions.
func descriptionsChanged(existing []model.GalgameDescription, rows []model.GalgameDescription) bool {
	current := make(map[string]model.GalgameDescription, len(existing))
	for _, item := range existing {
		current[item.Language] = item
	}
	for _, row := range rows {
		stored, ok := current[row.Language]
		if !ok {
			return true
		}
		if stored.Content != row.Content ||
			stored.SourceType != row.SourceType ||
			stored.SourceName != row.SourceName ||
			stored.SourceURL != row.SourceURL ||
			stored.IsOfficial != row.IsOfficial {
			return true
		}
	}
	return false
}

// upsertDescriptionsInTx writes validated rows inside an existing
// transaction; used by the catalog service create/update flows.
func upsertDescriptionsInTx(
	ctx context.Context,
	tx *gorm.DB,
	galgameID uint,
	rows []model.GalgameDescription,
) error {
	repo := repository.NewGalgameDescriptionRepository(tx)
	for i := range rows {
		rows[i].GalgameID = galgameID
		if err := repo.Upsert(ctx, &rows[i]); err != nil {
			return fmt.Errorf("upsert galgame descriptions: %w", err)
		}
	}
	return nil
}
