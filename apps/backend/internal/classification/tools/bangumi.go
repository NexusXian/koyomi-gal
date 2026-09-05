package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/importer/provider"
)

// BangumiLookupProvider is the provider surface the Bangumi tool needs; the
// importer module's BangumiProvider implements it.
type BangumiLookupProvider interface {
	Get(ctx context.Context, externalID string) (*provider.ExternalGame, error)
	Search(ctx context.Context, query string, limit int) ([]provider.ExternalGame, error)
}

// BangumiTool answers Bangumi queries for the agent. Bangumi is an auxiliary
// source and never the final verdict.
type BangumiTool struct {
	provider BangumiLookupProvider
}

type BangumiToolInput struct {
	SubjectID string `json:"subject_id"`
	Title     string `json:"title"`
}

type BangumiLookupResult struct {
	Found         bool     `json:"found"`
	SubjectID     string   `json:"subject_id"`
	Title         string   `json:"title"`
	OriginalTitle string   `json:"original_title"`
	Aliases       []string `json:"aliases"`
	Developer     string   `json:"developer"`
	ReleaseDate   string   `json:"release_date"`
	Tags          []string `json:"tags"`
}

func NewBangumiTool(provider BangumiLookupProvider) *BangumiTool {
	return &BangumiTool{provider: provider}
}

func (t *BangumiTool) Lookup(ctx context.Context, input BangumiToolInput) (string, error) {
	if t.provider == nil {
		return "", errors.New("bangumi lookup is not configured")
	}
	if input.SubjectID != "" && input.Title != "" {
		return "", errors.New("provide either subject_id or title, not both")
	}

	var game *provider.ExternalGame
	var err error
	if input.SubjectID != "" {
		game, err = t.provider.Get(ctx, strings.TrimSpace(input.SubjectID))
	} else if strings.TrimSpace(input.Title) != "" {
		games, searchErr := t.provider.Search(ctx, strings.TrimSpace(input.Title), 3)
		if searchErr != nil {
			return "", searchErr
		}
		if len(games) > 0 {
			game = &games[0]
		}
	} else {
		return "", errors.New("provide either subject_id or title")
	}
	if err != nil {
		return "", fmt.Errorf("bangumi lookup failed: %w", err)
	}
	if game == nil {
		return marshalBangumiResult(BangumiLookupResult{Found: false}), nil
	}

	developer := ""
	if game.Developer != nil {
		developer = game.Developer.Name
	}
	tags := make([]string, 0, len(game.Tags))
	for _, tag := range game.Tags {
		tags = append(tags, tag.Name)
	}
	result := BangumiLookupResult{
		Found:         true,
		SubjectID:     game.ExternalID,
		Title:         game.Title,
		OriginalTitle: game.OriginalTitle,
		Aliases:       game.Aliases,
		Developer:     developer,
		ReleaseDate:   formatDateHint(game.ReleaseDate),
		Tags:          tags,
	}
	return marshalBangumiResult(result), nil
}

func marshalBangumiResult(result BangumiLookupResult) string {
	payload, err := json.Marshal(result)
	if err != nil {
		return `{"found":false}`
	}
	return string(payload)
}

func formatDateHint(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02")
}
