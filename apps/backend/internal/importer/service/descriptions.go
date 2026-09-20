package service

import (
	"context"
	"fmt"

	galgameModel "backend/internal/galgame/model"
	galgameRepository "backend/internal/galgame/repository"
	"backend/internal/importer/provider"

	"gorm.io/gorm"
)

// providerDescription is one language's description prepared for storage,
// with structured provenance instead of free-form source text.
type providerDescription struct {
	Language   string
	Content    string
	SourceURL  string
}

// providerDescriptionLanguage maps an import provider onto the language its
// descriptions are written in. VNDB descriptions are English, Bangumi
// summaries are Chinese.
func providerDescriptionLanguage(source string) string {
	switch source {
	case galgameModel.DescriptionSourceVNDB:
		return galgameModel.LanguageEnUS
	case galgameModel.DescriptionSourceBangumi:
		return galgameModel.LanguageZhCN
	default:
		return ""
	}
}

// newProviderDescription builds the provider's own description row.
func newProviderDescription(game *provider.ExternalGame) *providerDescription {
	language := providerDescriptionLanguage(game.Source)
	content := normalizeDescription(game.Description)
	if language == "" || content == "" {
		return nil
	}
	return &providerDescription{
		Language:  language,
		Content:   content,
		SourceURL: externalSourceURL(game.Source, game.ExternalID),
	}
}

// descriptionSourceFromProvider maps a provider name onto the stored source
// type constant for description rows.
func descriptionSourceFromProvider(source string) string {
	switch source {
	case galgameModel.DescriptionSourceVNDB:
		return galgameModel.DescriptionSourceVNDB
	case galgameModel.DescriptionSourceBangumi:
		return galgameModel.DescriptionSourceBangumi
	default:
		return galgameModel.DescriptionSourceUnknown
	}
}

// effectiveDescriptionState returns the current content and source for a
// language, treating the legacy single-description column as fallback
// state: a vndb-sourced column holds English, anything else is Chinese.
func effectiveDescriptionState(
	ctx context.Context,
	tx *gorm.DB,
	galgame *galgameModel.Galgame,
	language string,
) (string, string) {
	var row galgameModel.GalgameDescription
	err := tx.WithContext(ctx).
		Where("galgame_id = ? AND language = ?", galgame.ID, language).
		First(&row).Error
	if err == nil {
		return row.Content, row.SourceType
	}
	if err != gorm.ErrRecordNotFound {
		return "", galgameModel.DescriptionSourceUnknown
	}
	if galgame.Description == "" {
		return "", galgameModel.DescriptionSourceUnknown
	}
	// The legacy column already stores a normalized description source.
	columnSource := normalizeDescriptionSource(galgame.DescriptionSource)
	if language == galgameModel.LanguageEnUS && columnSource == galgameModel.DescriptionSourceVNDB {
		return galgame.Description, galgameModel.DescriptionSourceVNDB
	}
	if language == galgameModel.LanguageZhCN && columnSource != galgameModel.DescriptionSourceVNDB {
		return galgame.Description, columnSource
	}
	return "", galgameModel.DescriptionSourceUnknown
}

// upsertDescriptionRow writes one per-language description row inside the
// caller's transaction.
func upsertDescriptionRow(
	ctx context.Context,
	tx *gorm.DB,
	galgameID uint,
	language, content, sourceType, sourceURL string,
) error {
	row := galgameModel.GalgameDescription{
		GalgameID:  galgameID,
		Language:   language,
		Content:    content,
		SourceType: sourceType,
		SourceName: galgameModel.DefaultDescriptionSourceName(sourceType),
		SourceURL:  sourceURL,
	}
	if err := galgameRepository.NewGalgameDescriptionRepository(tx).Upsert(ctx, &row); err != nil {
		return fmt.Errorf("upsert imported description: %w", err)
	}
	return nil
}

// applyProviderDescription syncs the provider's own description into its
// language row, honouring source priority so maintained content is never
// downgraded. It returns whether a row was written.
func applyProviderDescription(
	ctx context.Context,
	tx *gorm.DB,
	galgame *galgameModel.Galgame,
	game *provider.ExternalGame,
	force bool,
) (bool, error) {
	incoming := newProviderDescription(game)
	if incoming == nil {
		return false, nil
	}
	currentContent, currentSource := effectiveDescriptionState(ctx, tx, galgame, incoming.Language)
	if incoming.Content == currentContent ||
		!shouldReplaceDescription(currentContent, currentSource, incoming.Content, game.Source, force) {
		return false, nil
	}
	sourceType := descriptionSourceFromProvider(game.Source)
	if err := upsertDescriptionRow(
		ctx, tx, galgame.ID, incoming.Language, incoming.Content, sourceType, incoming.SourceURL,
	); err != nil {
		return false, err
	}
	return true, nil
}
