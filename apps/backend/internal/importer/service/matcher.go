package service

import (
	"math"
	"strings"
	"time"

	"backend/internal/importer/provider"
)

// Confidence thresholds: >= autoMatchThreshold links automatically,
// >= reviewThreshold queues a candidate for manual review, below is ignored.
const (
	autoMatchThreshold   = 0.85
	reviewThreshold      = 0.60
	matchCandidatesLimit = 10
)

// Match weights per matched signal. Signals are additive and capped at 1.
const (
	matchWeightOriginalTitle = 0.55
	matchWeightAlias         = 0.45
	matchWeightNormalized    = 0.40
	matchWeightReleaseYear   = 0.15
	matchWeightDeveloper     = 0.15
)

// Reason codes reported with every match candidate.
const (
	ReasonOriginalTitleMatch = "original_title_match"
	ReasonAliasMatch         = "alias_match"
	ReasonNormalizedMatch    = "normalized_title_match"
	ReasonReleaseYearMatch   = "release_year_match"
	ReasonDeveloperMatch     = "developer_match"
)

// MatchInput carries the local galgame identity used to score external
// search results.
type MatchInput struct {
	Title         string
	OriginalTitle string
	RomajiTitle   string
	Aliases       []string
	ReleaseDate   *time.Time
	Developer     string
}

// MatchCandidate pairs an external game with its match confidence.
type MatchCandidate struct {
	Game       provider.ExternalGame
	Confidence float64
	Reasons    []string
}

// SearchQuery returns the best keyword for external search.
func (in MatchInput) SearchQuery() string {
	for _, value := range []string{in.OriginalTitle, in.RomajiTitle, in.Title} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	if len(in.Aliases) > 0 {
		return strings.TrimSpace(in.Aliases[0])
	}
	return ""
}

// inputNames returns every identity string of the local galgame.
func (in MatchInput) inputNames() []string {
	values := make([]string, 0, len(in.Aliases)+3)
	values = append(values, in.Title, in.OriginalTitle, in.RomajiTitle)
	values = append(values, in.Aliases...)
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// MatchBangumiCandidates scores external search results against the local
// galgame. Only exact (raw or normalized) title identity contributes title
// confidence; fuzzy similarity is deliberately avoided so different editions
// are never merged.
func MatchBangumiCandidates(input MatchInput, results []provider.ExternalGame) []MatchCandidate {
	candidates := make([]MatchCandidate, 0, len(results))
	inputNames := input.inputNames()
	if len(inputNames) == 0 {
		return candidates
	}

	inputNormalized := make(map[string]struct{}, len(inputNames))
	for _, name := range inputNames {
		if normalized := NormalizeGameTitle(name); normalized != "" {
			inputNormalized[normalized] = struct{}{}
		}
	}

	for i := range results {
		game := results[i]
		confidence := 0.0
		reasons := make([]string, 0, 5)

		externalNames := game.Aliases
		primaryNames := []string{game.OriginalTitle}
		if strings.TrimSpace(game.Title) != "" {
			primaryNames = append(primaryNames, strings.TrimSpace(game.Title))
		}

		// Title identity signals are alternatives, never stacked, so a raw
		// original-title match can never outweigh edition differences.
		switch {
		case identityMatch(primaryNames, inputNames) || identityMatch(inputNames, primaryNames):
			confidence += matchWeightOriginalTitle
			reasons = append(reasons, ReasonOriginalTitleMatch)
		case identityMatch(externalNames, inputNames) || identityMatch(inputNames, externalNames):
			confidence += matchWeightAlias
			reasons = append(reasons, ReasonAliasMatch)
		case normalizedMatch(inputNormalized, primaryNames) || normalizedMatch(inputNormalized, externalNames):
			confidence += matchWeightNormalized
			reasons = append(reasons, ReasonNormalizedMatch)
		}
		if releaseYearEqual(input.ReleaseDate, game.ReleaseDate) {
			confidence += matchWeightReleaseYear
			reasons = append(reasons, ReasonReleaseYearMatch)
		}
		if developerMatch(input.Developer, game.Developer) {
			confidence += matchWeightDeveloper
			reasons = append(reasons, ReasonDeveloperMatch)
		}

		if confidence > 1 {
			confidence = 1
		}
		// Round to 4 decimals so threshold comparisons stay deterministic
		// regardless of weight summation order.
		confidence = math.Round(confidence*10000) / 10000
		if confidence >= reviewThreshold {
			candidates = append(candidates, MatchCandidate{
				Game:       game,
				Confidence: confidence,
				Reasons:    reasons,
			})
		}
	}

	// Stable sort keeps search rank order for equal confidences.
	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0 && candidates[j].Confidence > candidates[j-1].Confidence; j-- {
			candidates[j], candidates[j-1] = candidates[j-1], candidates[j]
		}
	}
	if len(candidates) > matchCandidatesLimit {
		candidates = candidates[:matchCandidatesLimit]
	}
	return candidates
}

// identityMatch reports whether any left name equals any right name after
// trim + casefold.
func identityMatch(left, right []string) bool {
	for _, l := range left {
		key := casefoldValue(l)
		if key == "" {
			continue
		}
		for _, r := range right {
			if key == casefoldValue(r) {
				return true
			}
		}
	}
	return false
}

func normalizedMatch(inputNormalized map[string]struct{}, externalNames []string) bool {
	for _, name := range externalNames {
		if _, exists := inputNormalized[NormalizeGameTitle(name)]; exists {
			return true
		}
	}
	return false
}

func releaseYearEqual(left, right *time.Time) bool {
	if left == nil || right == nil {
		return false
	}
	return left.UTC().Year() == right.UTC().Year()
}

func developerMatch(local string, external *provider.ExternalDeveloper) bool {
	if external == nil || strings.TrimSpace(local) == "" {
		return false
	}
	left := NormalizeGameTitle(local)
	right := NormalizeGameTitle(external.Name)
	if left == "" || right == "" {
		return false
	}
	return strings.Contains(left, right) || strings.Contains(right, left)
}
