package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"backend/internal/importer/provider"
)

// VNDBLookupProvider is the provider surface the VNDB tool needs; the importer
// module's VNDBProvider implements it.
type VNDBLookupProvider interface {
	Get(ctx context.Context, externalID string) (*provider.ExternalGame, error)
	Search(ctx context.Context, query string, limit int) ([]provider.ExternalGame, error)
}

// VNDBTool answers VNDB queries for the agent. VNDB is evidence only; the
// model may not treat a VNDB listing as the final verdict.
type VNDBTool struct {
	provider VNDBLookupProvider
}

type VNDBToolInput struct {
	VNDBID string `json:"vndb_id"`
	Title  string `json:"title"`
}

type VNDBLookupResult struct {
	Found       bool     `json:"found"`
	VNDBID      string   `json:"vndb_id"`
	Title       string   `json:"title"`
	Aliases     []string `json:"aliases"`
	Developer   string   `json:"developer"`
	ReleaseDate string   `json:"release_date"`
	Tags        []string `json:"tags"`
	Hint        string   `json:"hint"`
}

func NewVNDBTool(provider VNDBLookupProvider) *VNDBTool {
	return &VNDBTool{provider: provider}
}

func (t *VNDBTool) Lookup(ctx context.Context, input VNDBToolInput) (string, error) {
	if t.provider == nil {
		return "", errors.New("vndb lookup is not configured")
	}
	if input.VNDBID != "" && input.Title != "" {
		return "", errors.New("provide either vndb_id or title, not both")
	}

	var game *provider.ExternalGame
	var err error
	if input.VNDBID != "" {
		game, err = t.provider.Get(ctx, strings.TrimSpace(input.VNDBID))
	} else if strings.TrimSpace(input.Title) != "" {
		games, searchErr := t.provider.Search(ctx, strings.TrimSpace(input.Title), 3)
		if searchErr != nil {
			return "", searchErr
		}
		if len(games) > 0 {
			game = &games[0]
		}
	} else {
		return "", errors.New("provide either vndb_id or title")
	}
	if err != nil {
		return "", fmt.Errorf("vndb lookup failed: %w", err)
	}
	if game == nil {
		return marshalVNDBResult(VNDBLookupResult{Found: false}, nil), nil
	}

	developer := ""
	if game.Developer != nil {
		developer = game.Developer.Name
	}
	tags := make([]string, 0, len(game.Tags))
	for _, tag := range game.Tags {
		tags = append(tags, tag.Name)
	}
	result := VNDBLookupResult{
		Found:       true,
		VNDBID:      game.ExternalID,
		Title:       game.Title,
		Aliases:     game.Aliases,
		Developer:   developer,
		ReleaseDate: formatDateHint(game.ReleaseDate),
		Tags:        tags,
		Hint:        vndbHint(game),
	}
	return marshalVNDBResult(result, nil), nil
}

// vndbHint surfaces content tags that are often confused with R18. It never
// claims R18 by itself.
func vndbHint(game *provider.ExternalGame) string {
	lowerTags := make([]string, 0, len(game.Tags))
	for _, tag := range game.Tags {
		lowerTags = append(lowerTags, strings.ToLower(tag.Name))
	}
	var hints []string
	for _, sensitive := range []string{"sexual content", "nudity", "mature"} {
		for _, tag := range lowerTags {
			if strings.Contains(tag, sensitive) {
				hints = append(hints, sensitive)
				break
			}
		}
	}
	if len(hints) == 0 {
		return ""
	}
	return "VNDB content tags include: " + strings.Join(hints, ", ") +
		". These tags alone do not prove an R18 release."
}

func marshalVNDBResult(result VNDBLookupResult, _ any) string {
	payload, err := json.Marshal(result)
	if err != nil {
		return `{"found":false}`
	}
	return string(payload)
}
