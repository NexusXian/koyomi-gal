package service

import (
	"sort"
	"strings"
	"unicode"

	"backend/internal/importer/provider"
)

const (
	maxImportedTags      = 15
	minimumVNDBTagRating = 1.5
)

func normalizeName(value string) string {
	return strings.Map(func(char rune) rune {
		if unicode.IsLetter(char) || unicode.IsNumber(char) {
			return unicode.ToLower(char)
		}
		return -1
	}, strings.TrimSpace(value))
}

func normalizeSlug(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func importedAliases(game *provider.ExternalGame) []string {
	values := make([]string, 0, len(game.Aliases)+2)
	values = append(values, game.Aliases...)
	values = append(values, game.OriginalTitle, game.RomajiTitle)
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		key := strings.ToLower(value)
		if value == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func importedTags(game *provider.ExternalGame) []provider.ExternalTag {
	tags := make([]provider.ExternalTag, 0, len(game.Tags))
	seen := make(map[string]struct{}, len(game.Tags))
	for _, tag := range game.Tags {
		name := strings.TrimSpace(tag.Name)
		key := normalizeName(name)
		if name == "" || key == "" || tag.Spoiler != 0 || tag.Rating < minimumVNDBTagRating {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		tag.Name = name
		tags = append(tags, tag)
	}
	sort.SliceStable(tags, func(i, j int) bool {
		return tags[i].Rating > tags[j].Rating
	})
	if len(tags) > maxImportedTags {
		tags = tags[:maxImportedTags]
	}
	return tags
}
