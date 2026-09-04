package service

import (
	"testing"
	"time"

	"backend/internal/importer/provider"
)

func date(y int, m time.Month, d int) *time.Time {
	value := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return &value
}

func bangumiGame(id, name, nameCN string, release *time.Time, developer string, aliases ...string) provider.ExternalGame {
	ratingCount := 5819
	game := provider.ExternalGame{
		Source:        "bangumi",
		ExternalID:    id,
		Title:         nameCN,
		OriginalTitle: name,
		Aliases:       append([]string{name, nameCN}, aliases...),
		ReleaseDate:   release,
		RatingCount:   &ratingCount,
	}
	if developer != "" {
		game.Developer = &provider.ExternalDeveloper{Name: developer}
	}
	return game
}

func hasReason(match MatchCandidate, reason string) bool {
	for _, got := range match.Reasons {
		if got == reason {
			return true
		}
	}
	return false
}

func almostEqual(a, b float64) bool {
	return a-b < 1e-9 && b-a < 1e-9
}

func TestMatchHighConfidenceAutoBand(t *testing.T) {
	input := MatchInput{
		Title:         "Summer Pockets",
		OriginalTitle: "サマーポケッツ",
		RomajiTitle:   "Summer Pockets",
		Aliases:       []string{"サマポケ", "SP"},
		ReleaseDate:   date(2018, time.June, 29),
		Developer:     "Key",
	}
	results := []provider.ExternalGame{
		bangumiGame("200763", "Summer Pockets", "夏日口袋", date(2018, time.June, 29), "VISUAL ARTS / Key", "サマーポケッツ", "サマポケ"),
	}

	matches := MatchBangumiCandidates(input, results)
	if len(matches) != 1 {
		t.Fatalf("matches = %d, want 1", len(matches))
	}
	best := matches[0]
	want := matchWeightOriginalTitle + matchWeightReleaseYear + matchWeightDeveloper
	if best.Confidence != want {
		t.Errorf("confidence = %v, want %v (original + year + developer)", best.Confidence, want)
	}
	if best.Confidence < autoMatchThreshold {
		t.Errorf("confidence %v must reach the auto-match threshold", best.Confidence)
	}
	for _, reason := range []string{ReasonOriginalTitleMatch, ReasonReleaseYearMatch, ReasonDeveloperMatch} {
		if !hasReason(best, reason) {
			t.Errorf("reasons missing %q: %v", reason, best.Reasons)
		}
	}
}

func TestMatchAliasOnlyReviewBand(t *testing.T) {
	// Bangumi primary names differ from every local name; only an infobox
	// alias links the two entries.
	input := MatchInput{
		Title:         "樱之诗",
		OriginalTitle: "サクラノ詩 -櫻の森の上を舞う-",
		RomajiTitle:   "Sakura no Uta",
		ReleaseDate:   date(2015, time.November, 27),
		Developer:     "Makura",
	}
	results := []provider.ExternalGame{
		bangumiGame("183082", "サクラノ詩", "", date(2015, time.November, 27), "枕", "樱之诗"),
	}

	matches := MatchBangumiCandidates(input, results)
	if len(matches) != 1 {
		t.Fatalf("matches = %d, want 1", len(matches))
	}
	best := matches[0]
	if best.Confidence != matchWeightAlias+matchWeightReleaseYear {
		t.Errorf("confidence = %v, want %v (alias + year)", best.Confidence, matchWeightAlias+matchWeightReleaseYear)
	}
	if !hasReason(best, ReasonAliasMatch) || hasReason(best, ReasonOriginalTitleMatch) {
		t.Errorf("reasons = %v, want alias signal without primary-name signal", best.Reasons)
	}
	if best.Confidence < reviewThreshold || best.Confidence >= autoMatchThreshold {
		t.Errorf("confidence %v must sit in the review band", best.Confidence)
	}
}

func TestMatchNormalizedTitleReviewBand(t *testing.T) {
	// The wave-dash and hyphen variants only agree after normalization.
	input := MatchInput{
		Title:       "Ever17 ～the out of infinity～",
		ReleaseDate: date(2002, time.August, 29),
		Developer:   "KID",
	}
	results := []provider.ExternalGame{
		bangumiGame("12", "Ever17 -the out of infinity-", "时光的羁绊", date(2002, time.August, 29), "KID"),
	}
	matches := MatchBangumiCandidates(input, results)
	if len(matches) != 1 {
		t.Fatalf("matches = %d, want 1 via normalized identity", len(matches))
	}
	if matches[0].Confidence != matchWeightNormalized+matchWeightReleaseYear+matchWeightDeveloper {
		t.Errorf("confidence = %v, want normalized + year + developer", matches[0].Confidence)
	}
	if !hasReason(matches[0], ReasonNormalizedMatch) {
		t.Errorf("reasons = %v, want normalized signal", matches[0].Reasons)
	}
}

func TestMatchIgnoresBelowReviewThreshold(t *testing.T) {
	input := MatchInput{
		Title:       "Summer Pockets",
		ReleaseDate: date(2018, time.June, 29),
	}
	results := []provider.ExternalGame{
		// Only the release year matches.
		bangumiGame("301972", "Summer！", "", date(2018, time.June, 29), ""),
	}
	if matches := MatchBangumiCandidates(input, results); len(matches) != 0 {
		t.Fatalf("matches = %+v, want none below threshold", matches)
	}
}

func TestMatchDifferentEditionNotMerged(t *testing.T) {
	input := MatchInput{
		Title:         "Summer Pockets",
		OriginalTitle: "サマーポケッツ",
		RomajiTitle:   "Summer Pockets",
		Aliases:       []string{"SP"},
		ReleaseDate:   date(2018, time.June, 29),
		Developer:     "Key",
	}
	results := []provider.ExternalGame{
		bangumiGame("295957", "Summer Pockets REFLECTION BLUE", "夏日口袋 REFLECTION BLUE", date(2020, time.June, 26), "VISUAL ARTS / Key", "サマーポケッツRB", "SPRB"),
	}
	if matches := MatchBangumiCandidates(input, results); len(matches) != 0 {
		t.Fatalf("matches = %+v, want none: REFLECTION BLUE is a different edition", matches)
	}
}

func TestMatchSameTitleDifferentYearStaysBelowAuto(t *testing.T) {
	input := MatchInput{
		Title:       "CLANNAD",
		ReleaseDate: date(2018, time.June, 29),
		Developer:   "Key",
	}
	results := []provider.ExternalGame{
		bangumiGame("7", "CLANNAD", "CLANNAD 光见守的坂道上", date(2004, time.April, 28), "Key"),
	}
	matches := MatchBangumiCandidates(input, results)
	if len(matches) != 1 {
		t.Fatalf("matches = %d, want 1 in review band", len(matches))
	}
	if matches[0].Confidence >= autoMatchThreshold {
		t.Errorf("confidence %v must stay below auto threshold when the year differs", matches[0].Confidence)
	}
}

func TestMatchOrdersByConfidence(t *testing.T) {
	input := MatchInput{
		Title:       "Summer Pockets",
		Aliases:     []string{"SP"},
		ReleaseDate: date(2018, time.June, 29),
		Developer:   "Key",
	}
	results := []provider.ExternalGame{
		// Port edition shares an alias plus year/developer but not the title.
		bangumiGame("300001", "Summer Pockets SP", "", date(2018, time.June, 29), "Key", "SP"),
		bangumiGame("200763", "Summer Pockets", "夏日口袋", date(2018, time.June, 29), "VISUAL ARTS / Key"),
	}
	matches := MatchBangumiCandidates(input, results)
	if len(matches) != 2 {
		t.Fatalf("matches = %d, want 2", len(matches))
	}
	if matches[0].Game.ExternalID != "200763" {
		t.Errorf("first match = %s, want the exact-title candidate", matches[0].Game.ExternalID)
	}
	if matches[0].Confidence <= matches[1].Confidence {
		t.Errorf("matches must be sorted by confidence desc: %v", matches)
	}
}

func TestMatchDeveloperContainment(t *testing.T) {
	input := MatchInput{
		Title:     "Unknown Game",
		Developer: "Key",
	}
	results := []provider.ExternalGame{
		bangumiGame("1", "Unknown Game", "", nil, "VISUAL ARTS / Key"),
	}
	matches := MatchBangumiCandidates(input, results)
	if len(matches) != 1 {
		t.Fatalf("matches = %d, want title + developer containment", len(matches))
	}
	if matches[0].Confidence != matchWeightOriginalTitle+matchWeightDeveloper {
		t.Errorf("confidence = %v, want %v (original + developer)",
			matches[0].Confidence, matchWeightOriginalTitle+matchWeightDeveloper)
	}
}

func TestMatchEmptyInputNames(t *testing.T) {
	matches := MatchBangumiCandidates(MatchInput{}, []provider.ExternalGame{
		bangumiGame("1", "Game", "", nil, ""),
	})
	if len(matches) != 0 {
		t.Fatalf("matches = %+v, want none without input identity", matches)
	}
}

func TestMatchSearchQuery(t *testing.T) {
	if got := (MatchInput{OriginalTitle: "サマーポケッツ"}).SearchQuery(); got != "サマーポケッツ" {
		t.Errorf("search query = %q, want original title first", got)
	}
	if got := (MatchInput{RomajiTitle: "Summer Pockets"}).SearchQuery(); got != "Summer Pockets" {
		t.Errorf("search query = %q, want romaji fallback", got)
	}
	if got := (MatchInput{Title: "夏日口袋"}).SearchQuery(); got != "夏日口袋" {
		t.Errorf("search query = %q, want title fallback", got)
	}
	if got := (MatchInput{}).SearchQuery(); got != "" {
		t.Errorf("search query = %q, want empty", got)
	}
}
