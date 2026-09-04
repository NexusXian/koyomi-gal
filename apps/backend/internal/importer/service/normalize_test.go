package service

import "testing"

func TestNormalizeGameTitle(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"trims and lowercases", "  Summer Pockets ", "summer pockets"},
		{"collapses inner whitespace", "Summer   Pockets", "summer pockets"},
		{"full-width latin to half-width", "ＳＵＭＭＥＲ ＰＯＣＫＥＴＳ", "summer pockets"},
		{"full-width digits", "ＲＥ９９", "re99"},
		{"strips brackets", "「サクラノ詩」", "サクラノ詩"},
		{"strips double brackets", "『櫻の森の下を舞う』", "櫻の森の下を舞う"},
		{"strips colon variants", "CLANNAD：SIDE STORY", "clannad side story"},
		{"hyphen becomes separator", "サクラノ詩 -櫻の森の上を舞う-", "サクラノ詩 櫻の森の上を舞う"},
		{"wave dash variants", "Ever17 ～the out of infinity～", "ever17 the out of infinity"},
		{"tilde", "Fate/Stay Night ~Fate~", "fate stay night fate"},
		{"empty stays empty", "", ""},
	}
	for _, tc := range cases {
		if got := NormalizeGameTitle(tc.in); got != tc.want {
			t.Errorf("%s: NormalizeGameTitle(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestNormalizeGameTitleKeepsEditionWordsDistinct(t *testing.T) {
	base := NormalizeGameTitle("Summer Pockets")
	edition := NormalizeGameTitle("Summer Pockets REFLECTION BLUE")
	if base == edition {
		t.Fatalf("base %q must not equal edition %q", base, edition)
	}
	for _, pair := range [][2]string{
		{"CLANNAD", "CLANNAD AFTER STORY"},
		{"AIR", "AIR STANDARD EDITION"},
		{"reno", "reno2"},
	} {
		if NormalizeGameTitle(pair[0]) == NormalizeGameTitle(pair[1]) {
			t.Errorf("%q and %q must stay distinct", pair[0], pair[1])
		}
	}
}

func TestCasefoldValue(t *testing.T) {
	if casefoldValue("  Summer ") != "summer" {
		t.Errorf("casefoldValue must trim and lowercase")
	}
	if casefoldValue("サマポケ") != "サマポケ" {
		t.Errorf("casefoldValue must keep non-Latin scripts intact")
	}
}
