package service

import (
	"testing"

	galgameModel "backend/internal/galgame/model"
)

func TestShouldReplaceDescription(t *testing.T) {
	cases := []struct {
		name           string
		current        string
		currentSource  string
		incoming       string
		incomingSource string
		force          bool
		want           bool
	}{
		{"empty current accepts bangumi", "", "", "中文简介", "bangumi", false, true},
		{"empty current accepts vndb", "", "", "English description", "vndb", false, true},
		{"bangumi outranks vndb", "English description", "vndb", "中文简介", "bangumi", false, true},
		{"vndb never outranks bangumi", "中文简介", "bangumi", "English description", "vndb", false, false},
		{"manual blocks bangumi", "管理员简介", "manual", "中文简介", "bangumi", false, false},
		{"manual blocks vndb", "管理员简介", "manual", "English description", "vndb", false, false},
		{"vndb beats unknown", "旧简介", "unknown", "English description", "vndb", false, true},
		{"bangumi beats unknown", "旧简介", "unknown", "中文简介", "bangumi", false, true},
		{"empty legacy source counts as unknown", "旧简介", "", "中文简介", "bangumi", false, true},
		{"empty incoming never replaces", "旧简介", "vndb", "", "bangumi", false, false},
		{"empty incoming never replaces even when current empty", "", "", "", "bangumi", false, false},
		{"force overwrites manual", "管理员简介", "manual", "中文简介", "bangumi", true, true},
		{"force overwrites same source", "中文简介", "bangumi", "新简介", "bangumi", true, true},
		{"same source without force stays", "中文简介", "bangumi", "新简介", "bangumi", false, false},
		{"vndb re-sync without force stays", "旧简介", "vndb", "新简介", "vndb", false, false},
		{"unknown incoming cannot replace vndb", "English description", "vndb", "新简介", "mystery", false, false},
	}
	for _, tc := range cases {
		if got := shouldReplaceDescription(
			tc.current, tc.currentSource, tc.incoming, tc.incomingSource, tc.force,
		); got != tc.want {
			t.Errorf("%s: shouldReplaceDescription(%q, %q, %q, %q, %v) = %v, want %v",
				tc.name, tc.current, tc.currentSource, tc.incoming, tc.incomingSource, tc.force, got, tc.want)
		}
	}
}

func TestNormalizeDescriptionSource(t *testing.T) {
	cases := map[string]string{
		"vndb":     galgameModel.DescriptionSourceVNDB,
		"bangumi":  galgameModel.DescriptionSourceBangumi,
		"manual":   galgameModel.DescriptionSourceManual,
		" VNDB ":   galgameModel.DescriptionSourceVNDB,
		"":         galgameModel.DescriptionSourceUnknown,
		"unknown":  galgameModel.DescriptionSourceUnknown,
		"eggplant": galgameModel.DescriptionSourceUnknown,
	}
	for input, want := range cases {
		if got := normalizeDescriptionSource(input); got != want {
			t.Errorf("normalizeDescriptionSource(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDescriptionSourceForImport(t *testing.T) {
	cases := []struct {
		source      string
		description string
		want        string
	}{
		{"vndb", "English description", galgameModel.DescriptionSourceVNDB},
		{"bangumi", "中文简介", galgameModel.DescriptionSourceBangumi},
		{"vndb", "", galgameModel.DescriptionSourceUnknown},
		{"bangumi", "   \r\n", galgameModel.DescriptionSourceUnknown},
		{"mystery", "some text", galgameModel.DescriptionSourceUnknown},
	}
	for _, tc := range cases {
		if got := descriptionSourceForImport(tc.source, tc.description); got != tc.want {
			t.Errorf("descriptionSourceForImport(%q, %q) = %q, want %q",
				tc.source, tc.description, got, tc.want)
		}
	}
}

func TestNormalizeDescription(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"  text  ", "text"},
		{"line1\r\nline2", "line1\nline2"},
		{"\r\n\r\n", ""},
		{"already\nfine", "already\nfine"},
	}
	for _, tc := range cases {
		if got := normalizeDescription(tc.in); got != tc.want {
			t.Errorf("normalizeDescription(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
