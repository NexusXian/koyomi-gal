package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"backend/internal/importer/provider"
)

type fakeGameLookup struct {
	game *provider.ExternalGame
	ids  []string
}

func (f *fakeGameLookup) Get(ctx context.Context, externalID string) (*provider.ExternalGame, error) {
	f.ids = append(f.ids, externalID)
	if f.game == nil {
		return nil, nil
	}
	copy := *f.game
	copy.ExternalID = externalID
	return &copy, nil
}

func (f *fakeGameLookup) Search(ctx context.Context, query string, limit int) ([]provider.ExternalGame, error) {
	if f.game == nil {
		return nil, nil
	}
	return []provider.ExternalGame{*f.game}, nil
}

func TestVNDBToolReturnsSummaryAndHint(t *testing.T) {
	date := time.Date(2019, 7, 26, 0, 0, 0, 0, time.UTC)
	fake := &fakeGameLookup{game: &provider.ExternalGame{
		ExternalID:  "v20431",
		Title:       "Sakura no Toki",
		Aliases:     []string{"サクラノ刻"},
		ReleaseDate: &date,
		Developer:   &provider.ExternalDeveloper{Name: "Makura"},
		Tags: []provider.ExternalTag{
			{Name: "Sexual Content", Rating: 0.9},
			{Name: "Nudity", Rating: 0.8},
		},
	}}
	tool := NewVNDBTool(fake)
	output, err := tool.Lookup(context.Background(), VNDBToolInput{VNDBID: "v20431"})
	if err != nil {
		t.Fatalf("vndb lookup: %v", err)
	}
	if !strings.Contains(output, "Makura") {
		t.Fatalf("developer missing from output: %s", output)
	}
	if !strings.Contains(output, "sexual content") {
		t.Fatalf("hint missing from output: %s", output)
	}
	if !strings.Contains(output, "do not prove") {
		t.Fatalf("hint must warn about false positives: %s", output)
	}
}

func TestVNDBToolRequiresEitherIDOrTitle(t *testing.T) {
	fake := &fakeGameLookup{game: &provider.ExternalGame{Title: "x"}}
	tool := NewVNDBTool(fake)
	if _, err := tool.Lookup(context.Background(), VNDBToolInput{}); err == nil {
		t.Fatal("expected error without id or title")
	}
	if _, err := tool.Lookup(context.Background(), VNDBToolInput{VNDBID: "v1", Title: "x"}); err == nil {
		t.Fatal("expected error when both id and title are set")
	}
}

func TestVNDBToolNotFound(t *testing.T) {
	tool := NewVNDBTool(&fakeGameLookup{game: nil})
	output, err := tool.Lookup(context.Background(), VNDBToolInput{VNDBID: "v99999"})
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if !strings.Contains(output, `"found":false`) {
		t.Fatalf("expected found:false, got %s", output)
	}
}

func TestBangumiToolLookupByTitle(t *testing.T) {
	fake := &fakeGameLookup{game: &provider.ExternalGame{
		ExternalID:    "12345",
		Title:         "D.C.5",
		OriginalTitle: "D.C.5 ～ダ・カーポ5～",
		Developer:     &provider.ExternalDeveloper{Name: "CIRCUS"},
	}}
	tool := NewBangumiTool(fake)
	output, err := tool.Lookup(context.Background(), BangumiToolInput{Title: "D.C.5"})
	if err != nil {
		t.Fatalf("bangumi lookup: %v", err)
	}
	for _, want := range []string{"D.C.5", "CIRCUS", `"found":true`} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in output: %s", want, output)
		}
	}
}
