package service

import (
	"testing"

	"backend/internal/importer/provider"
)

func TestNormalizeName(t *testing.T) {
	cases := map[string]string{
		"Summer Pockets":  "summerpockets",
		"サマーポケッツ":         "サマーポケッツ",
		" Ever17!! ":      "ever17",
		"":                "",
		"!!!":             "",
		"夏日口袋 REFLECTION": "夏日口袋reflection",
	}
	for input, want := range cases {
		if got := normalizeName(input); got != want {
			t.Errorf("normalizeName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestImportedAliasesDeduplicates(t *testing.T) {
	game := &provider.ExternalGame{
		Aliases:       []string{"Summer Pockets", "summer pockets ", "", "  ", "SUMMER POCKETS"},
		OriginalTitle: "サマーポケッツ",
		RomajiTitle:   "Summer Pockets",
	}
	aliases := importedAliases(game)
	if len(aliases) != 2 {
		t.Fatalf("aliases = %v, want 2 unique values", aliases)
	}
	if aliases[0] != "Summer Pockets" {
		t.Errorf("first alias = %q", aliases[0])
	}
	if aliases[1] != "サマーポケッツ" {
		t.Errorf("second alias = %q, want original title", aliases[1])
	}
}

func TestImportedAliasesWithoutDuplicates(t *testing.T) {
	game := &provider.ExternalGame{
		Aliases:       []string{"Alias 1"},
		OriginalTitle: "オリジナル",
		RomajiTitle:   "Romaji",
	}
	aliases := importedAliases(game)
	if len(aliases) != 3 {
		t.Fatalf("aliases = %v, want 3 entries", aliases)
	}
}

func TestImportedTagsFiltersAndCaps(t *testing.T) {
	tags := make([]provider.ExternalTag, 0, 30)
	for i := 0; i < 20; i++ {
		rating := 1.0
		if i%2 == 0 {
			rating = 2.0
		}
		tags = append(tags, provider.ExternalTag{
			ExternalID: "g1",
			Name:       string(rune('a'+i%26)) + "-tag",
			Spoiler:    0,
			Rating:     rating,
		})
	}
	tags = append(tags,
		provider.ExternalTag{Name: "spoiler-tag", Spoiler: 1, Rating: 3.0},
		provider.ExternalTag{Name: "low-rating", Spoiler: 0, Rating: 1.0},
	)
	game := &provider.ExternalGame{Tags: tags}

	imported := importedTags(game)
	if len(imported) != 10 {
		t.Fatalf("imported tags = %d, want 10 (spoiler/low-rating filtered, cap 15)", len(imported))
	}
	for i := 1; i < len(imported); i++ {
		if imported[i-1].Rating < imported[i].Rating {
			t.Errorf("tags not sorted by rating desc: %v", imported)
		}
	}
}

func TestImportedTagsCapsAtFifteen(t *testing.T) {
	tags := make([]provider.ExternalTag, 0, 20)
	for i := 0; i < 20; i++ {
		tags = append(tags, provider.ExternalTag{
			Name:    "tag-" + string(rune('a'+i)),
			Spoiler: 0,
			Rating:  2.0,
		})
	}
	imported := importedTags(&provider.ExternalGame{Tags: tags})
	if len(imported) != 15 {
		t.Fatalf("imported tags = %d, want 15", len(imported))
	}
}

func TestImportedTagsDeduplicatesByName(t *testing.T) {
	game := &provider.ExternalGame{Tags: []provider.ExternalTag{
		{Name: "Romance", Spoiler: 0, Rating: 3.0},
		{Name: "romance ", Spoiler: 0, Rating: 2.5},
		{Name: "Drama", Spoiler: 0, Rating: 2.0},
	}}
	imported := importedTags(game)
	if len(imported) != 2 {
		t.Fatalf("imported tags = %d, want 2 after dedup", len(imported))
	}
	if imported[0].Name != "Romance" {
		t.Errorf("higher rated duplicate should win, got %q", imported[0].Name)
	}
}
