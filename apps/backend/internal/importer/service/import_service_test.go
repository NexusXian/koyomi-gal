package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	galgameModel "backend/internal/galgame/model"
	importerModel "backend/internal/importer/model"
	"backend/internal/importer/provider"
	importerRepository "backend/internal/importer/repository"
	"backend/internal/testutil"

	"gorm.io/gorm"
)

type stubProvider struct {
	games []provider.ExternalGame
}

func (s *stubProvider) Search(ctx context.Context, query string, limit int) ([]provider.ExternalGame, error) {
	results := make([]provider.ExternalGame, 0, len(s.games))
	for _, game := range s.games {
		if strings.Contains(strings.ToLower(game.Title), strings.ToLower(query)) {
			results = append(results, game)
		}
	}
	return results, nil
}

func (s *stubProvider) Get(ctx context.Context, externalID string) (*provider.ExternalGame, error) {
	for i := range s.games {
		if s.games[i].ExternalID == externalID {
			copy := s.games[i]
			return &copy, nil
		}
	}
	return nil, nil
}

func (s *stubProvider) ListBatch(
	ctx context.Context,
	filters provider.BatchFilters,
	page, results int,
	withCount bool,
) (*provider.BatchListResult, error) {
	if page < 1 {
		page = 1
	}
	start := (page - 1) * results
	if start >= len(s.games) {
		return &provider.BatchListResult{Games: []provider.ExternalGame{}, More: false, Count: int64(len(s.games))}, nil
	}
	end := start + results
	if end > len(s.games) {
		end = len(s.games)
	}
	return &provider.BatchListResult{
		Games: s.games[start:end],
		More:  end < len(s.games),
		Count: int64(len(s.games)),
	}, nil
}

func testExternalGame(id, title string, releaseDate *time.Time) provider.ExternalGame {
	rating := 8.5
	ratingCount := 1234
	length := 1200
	return provider.ExternalGame{
		Source:           "vndb",
		ExternalID:       id,
		Title:            title,
		OriginalTitle:    "オリジナル " + id,
		RomajiTitle:      title,
		Aliases:          []string{"alias-" + id},
		Description:      "description " + id,
		CoverURL:         "https://t.vndb.org/cv/" + id + ".jpg",
		ReleaseDate:      releaseDate,
		OriginalLanguage: "ja",
		LengthMinutes:    &length,
		Rating:           &rating,
		RatingCount:      &ratingCount,
		Developer:        &provider.ExternalDeveloper{ExternalID: "p1", Name: "Test Developer"},
		Tags: []provider.ExternalTag{
			{Name: "Romance", Spoiler: 0, Rating: 3.0},
			{Name: "Hidden Spoiler", Spoiler: 2, Rating: 3.0},
		},
		Raw: json.RawMessage(`{"id":"` + id + `"}`),
	}
}

func newTestService(t *testing.T, games []provider.ExternalGame) (*Service, *gorm.DB) {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	repo := importerRepository.NewRepository(db)
	return NewService(repo, map[string]provider.Provider{"vndb": &stubProvider{games: games}}, nil), db
}

func seedGalgame(t *testing.T, db *gorm.DB, title string, releaseDate *time.Time) uint {
	t.Helper()
	game := &galgameModel.Galgame{
		Title:       title,
		Slug:        strings.ToLower(strings.ReplaceAll(title, " ", "-")),
		Status:      galgameModel.GalgameStatusPublished,
		ReleaseDate: releaseDate,
	}
	if err := db.Create(game).Error; err != nil {
		t.Fatalf("seed galgame: %v", err)
	}
	return game.ID
}

func countRows(t *testing.T, db *gorm.DB, model any) int64 {
	t.Helper()
	var count int64
	if err := db.Model(model).Count(&count).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	return count
}

func TestImportNewGame(t *testing.T) {
	release := time.Date(2018, time.June, 29, 0, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, []provider.ExternalGame{testExternalGame("v20424", "Summer Pockets", &release)})

	result, err := svc.Import(context.Background(), ImportInput{
		Provider:        "vndb",
		ExternalID:      "v20424",
		DuplicateAction: DuplicateActionError,
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if result.DuplicateStatus != DuplicateStatusNone || result.GalgameID == nil {
		t.Fatalf("result = %+v, want fresh create", result)
	}

	var game galgameModel.Galgame
	if err := db.Preload("Aliases").Preload("Tags").First(&game, *result.GalgameID).Error; err != nil {
		t.Fatalf("load imported galgame: %v", err)
	}
	if game.Status != galgameModel.GalgameStatusPending {
		t.Errorf("status = %d, want pending: external imports require review", game.Status)
	}
	if game.SourceType != galgameModel.GalgameSourceVNDB {
		t.Errorf("source type = %d, want VNDB", game.SourceType)
	}
	if game.DescriptionSource != galgameModel.DescriptionSourceVNDB {
		t.Errorf("description source = %q, want vndb", game.DescriptionSource)
	}
	if game.OriginalLanguage != "ja" || game.LengthMinutes == nil || *game.LengthMinutes != 1200 {
		t.Errorf("language/length = %q/%v", game.OriginalLanguage, game.LengthMinutes)
	}
	if game.RatingAverage != 0 || game.RatingCount != 0 {
		t.Errorf("site rating must stay untouched, got %f/%d", game.RatingAverage, game.RatingCount)
	}
	if len(game.Aliases) == 0 {
		t.Error("expected aliases to be imported")
	}
	if len(game.Tags) != 1 {
		t.Fatalf("tags = %v, want only non-spoiler Romance", game.Tags)
	}
	var developer galgameModel.Developer
	if err := db.First(&developer, game.DeveloperID).Error; err != nil {
		t.Fatalf("developer: %v", err)
	}
	if developer.Name != "Test Developer" || developer.Slug != "test developer" {
		t.Errorf("developer = %q/%q", developer.Name, developer.Slug)
	}

	source, err := svc.repository.FindExternalSource(context.Background(), "vndb", "v20424")
	if err != nil || source == nil {
		t.Fatalf("external source: %v %+v", err, source)
	}
	if source.GalgameID != game.ID || source.ExternalRating == nil || *source.ExternalRating != 8.5 {
		t.Errorf("external source = %+v", source)
	}
	if source.ExternalRatingCount == nil || *source.ExternalRatingCount != 1234 {
		t.Errorf("external rating count = %v", source.ExternalRatingCount)
	}
	if source.LastSyncedAt == nil {
		t.Error("last_synced_at should be set")
	}
}

func TestImportAlreadyImported(t *testing.T) {
	svc, db := newTestService(t, []provider.ExternalGame{testExternalGame("v17", "Ever17", nil)})

	first, err := svc.Import(context.Background(), ImportInput{Provider: "vndb", ExternalID: "v17"})
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	second, err := svc.Import(context.Background(), ImportInput{Provider: "vndb", ExternalID: "v17"})
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if second.DuplicateStatus != DuplicateStatusAlreadyImported {
		t.Fatalf("status = %q, want already_imported", second.DuplicateStatus)
	}
	if second.ExistingGalgameID == nil || *second.ExistingGalgameID != *first.GalgameID {
		t.Fatalf("existing id = %v, want %v", second.ExistingGalgameID, first.GalgameID)
	}
	if got := countRows(t, db, &galgameModel.Galgame{}); got != 1 {
		t.Errorf("galgames = %d, want 1", got)
	}
}

func TestImportPossibleDuplicateBlocksAndResolves(t *testing.T) {
	release := time.Date(2018, time.June, 29, 0, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, []provider.ExternalGame{testExternalGame("v20424", "Summer Pockets", &release)})
	seeded := seedGalgame(t, db, "Summer Pockets", &release)

	blocked, err := svc.Import(context.Background(), ImportInput{Provider: "vndb", ExternalID: "v20424"})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if blocked.DuplicateStatus != DuplicateStatusPossible {
		t.Fatalf("status = %q, want possible", blocked.DuplicateStatus)
	}
	if len(blocked.Candidates) != 1 || blocked.Candidates[0].ID != seeded {
		t.Fatalf("candidates = %+v, want seeded id %d", blocked.Candidates, seeded)
	}
	if got := countRows(t, db, &galgameModel.Galgame{}); got != 1 {
		t.Errorf("possible duplicate must not auto-merge or create, galgames = %d", got)
	}
	if got := countRows(t, db, &importerModel.GalgameExternalSource{}); got != 0 {
		t.Errorf("no mapping should exist yet, got %d", got)
	}

	forced, err := svc.Import(context.Background(), ImportInput{
		Provider:        "vndb",
		ExternalID:      "v20424",
		DuplicateAction: DuplicateActionCreateNew,
	})
	if err != nil {
		t.Fatalf("create_new import: %v", err)
	}
	if forced.DuplicateStatus != DuplicateStatusNone || forced.GalgameID == nil || *forced.GalgameID == seeded {
		t.Fatalf("create_new result = %+v", forced)
	}
}

func TestImportLinkExisting(t *testing.T) {
	release := time.Date(2018, time.June, 29, 0, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, []provider.ExternalGame{testExternalGame("v20424", "Summer Pockets", &release)})
	seeded := seedGalgame(t, db, "夏日口袋", &release)

	linked, err := svc.Import(context.Background(), ImportInput{
		Provider:          "vndb",
		ExternalID:        "v20424",
		DuplicateAction:   DuplicateActionLinkExisting,
		ExistingGalgameID: &seeded,
	})
	if err != nil {
		t.Fatalf("link import: %v", err)
	}
	if linked.DuplicateStatus != DuplicateStatusNone || linked.GalgameID == nil || *linked.GalgameID != seeded {
		t.Fatalf("link result = %+v", linked)
	}

	var game galgameModel.Galgame
	if err := db.First(&game, seeded).Error; err != nil {
		t.Fatalf("load seeded: %v", err)
	}
	if game.Title != "夏日口袋" {
		t.Errorf("title = %q, must not be overwritten without force", game.Title)
	}
	if game.SourceType != galgameModel.GalgameSourceVNDB {
		t.Errorf("source type = %d, want VNDB after linking", game.SourceType)
	}
	if game.MetadataUpdatedAt == nil {
		t.Error("metadata_updated_at should be set after linking")
	}
	source, err := svc.repository.FindExternalSource(context.Background(), "vndb", "v20424")
	if err != nil || source == nil || source.GalgameID != seeded {
		t.Fatalf("mapping after link: %v %+v", err, source)
	}

	forced, err := svc.Import(context.Background(), ImportInput{
		Provider:            "vndb",
		ExternalID:          "v20424",
		DuplicateAction:     DuplicateActionLinkExisting,
		ExistingGalgameID:   &seeded,
		ForceMetadataUpdate: true,
	})
	if err != nil {
		t.Fatalf("link import should return already_imported for existing mapping: %v", err)
	}
	if forced.DuplicateStatus != DuplicateStatusAlreadyImported {
		t.Fatalf("relink status = %q, want already_imported", forced.DuplicateStatus)
	}
}

func TestExternalSourceUniqueConstraint(t *testing.T) {
	svc, db := newTestService(t, nil)
	ctx := context.Background()
	first := seedGalgame(t, db, "First Game", nil)
	second := seedGalgame(t, db, "Second Game", nil)
	if err := svc.repository.Transaction(ctx, func(tx *gorm.DB) error {
		return createExternalSource(ctx, tx, first, &provider.ExternalGame{Source: "vndb", ExternalID: "v17"})
	}); err != nil {
		t.Fatalf("first mapping insert: %v", err)
	}
	err := svc.repository.Transaction(ctx, func(tx *gorm.DB) error {
		return createExternalSource(ctx, tx, second, &provider.ExternalGame{Source: "vndb", ExternalID: "v17"})
	})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate mapping err = %v, want conflict", err)
	}
	if got := countRows(t, db, &importerModel.GalgameExternalSource{}); got != 1 {
		t.Errorf("mappings = %d, want 1", got)
	}
}

func TestImportRollsBackOnFailure(t *testing.T) {
	game := testExternalGame("v1", "Rollback Game", nil)
	// Tag name longer than VARCHAR(100) forces the tag insert inside the
	// import transaction to fail, so nothing may be persisted.
	longTag := strings.Repeat("x", 200)
	game.Tags = []provider.ExternalTag{{Name: longTag, Spoiler: 0, Rating: 3.0}}
	svc, db := newTestService(t, []provider.ExternalGame{game})

	if _, err := svc.Import(context.Background(), ImportInput{Provider: "vndb", ExternalID: "v1"}); err == nil {
		t.Fatal("expected import failure")
	}
	if got := countRows(t, db, &galgameModel.Galgame{}); got != 0 {
		t.Errorf("galgames = %d, want full rollback to 0", got)
	}
	if got := countRows(t, db, &importerModel.GalgameExternalSource{}); got != 0 {
		t.Errorf("mappings = %d, want 0", got)
	}
	if got := countRows(t, db, &galgameModel.Developer{}); got != 0 {
		t.Errorf("developers = %d, want 0", got)
	}
	if got := countRows(t, db, &galgameModel.Alias{}); got != 0 {
		t.Errorf("aliases = %d, want 0", got)
	}
}

func TestImportRecordsDescriptionSourceAndPendingStatus(t *testing.T) {
	release := time.Date(2018, time.June, 29, 0, 0, 0, 0, time.UTC)

	emptyDescription := testExternalGame("v-empty", "No Description Game", &release)
	emptyDescription.Description = "  \r\n"
	vndbGame := testExternalGame("v20424", "Summer Pockets", &release)
	bangumiGame := testExternalGame("b200763", "Summer Pockets", &release)
	bangumiGame.Source = "bangumi"
	bangumiGame.Description = "中文简介"

	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	svc := NewService(importerRepository.NewRepository(db), map[string]provider.Provider{
		"vndb":    &stubProvider{games: []provider.ExternalGame{vndbGame, emptyDescription}},
		"bangumi": &stubProvider{games: []provider.ExternalGame{bangumiGame}},
	}, nil)

	for _, tc := range []struct {
		provider   string
		externalID string
		wantSource string
	}{
		{"vndb", "v20424", galgameModel.DescriptionSourceVNDB},
		{"bangumi", "b200763", galgameModel.DescriptionSourceBangumi},
		{"vndb", "v-empty", galgameModel.DescriptionSourceUnknown},
	} {
		result, err := svc.Import(context.Background(), ImportInput{
			Provider:        tc.provider,
			ExternalID:      tc.externalID,
			DuplicateAction: DuplicateActionCreateNew,
		})
		if err != nil {
			t.Fatalf("import %s/%s: %v", tc.provider, tc.externalID, err)
		}
		if result.GalgameID == nil {
			t.Fatalf("import %s/%s: no galgame id: %+v", tc.provider, tc.externalID, result)
		}
		var game galgameModel.Galgame
		if err := db.First(&game, *result.GalgameID).Error; err != nil {
			t.Fatalf("load %s/%s: %v", tc.provider, tc.externalID, err)
		}
		if game.Status != galgameModel.GalgameStatusPending {
			t.Errorf("%s/%s: status = %d, want pending", tc.provider, tc.externalID, game.Status)
		}
		if game.DescriptionSource != tc.wantSource {
			t.Errorf("%s/%s: description source = %q, want %q",
				tc.provider, tc.externalID, game.DescriptionSource, tc.wantSource)
		}
		if tc.wantSource == galgameModel.DescriptionSourceUnknown && game.Description != "" {
			t.Errorf("%s/%s: description = %q, want blank after normalization",
				tc.provider, tc.externalID, game.Description)
		}
	}
}

func TestImportLinkMissingGalgame(t *testing.T) {
	svc, db := newTestService(t, []provider.ExternalGame{testExternalGame("v9", "Ghost Link", nil)})
	missing := uint(999999)
	_, err := svc.Import(context.Background(), ImportInput{
		Provider:          "vndb",
		ExternalID:        "v9",
		DuplicateAction:   DuplicateActionLinkExisting,
		ExistingGalgameID: &missing,
	})
	if err == nil {
		t.Fatal("expected error for missing target galgame")
	}
	if got := countRows(t, db, &importerModel.GalgameExternalSource{}); got != 0 {
		t.Errorf("mappings = %d, want 0", got)
	}
}

func TestImportVndbUsesBangumiChineseSummary(t *testing.T) {
	release := time.Date(2018, time.June, 29, 0, 0, 0, 0, time.UTC)
	vndbGame := testExternalGame("v20424", "Summer Pockets", &release)
	vndbGame.OriginalTitle = "" // keep the search query on the latin title
	vndbGame.Description = "English description from VNDB"
	bangumiGame := testExternalGame("200763", "Summer Pockets", &release)
	bangumiGame.Source = "bangumi"
	bangumiGame.OriginalTitle = ""
	bangumiGame.Description = "Bangumi 中文简介"

	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	svc := NewService(importerRepository.NewRepository(db), map[string]provider.Provider{
		"vndb":    &stubProvider{games: []provider.ExternalGame{vndbGame}},
		"bangumi": &stubProvider{games: []provider.ExternalGame{bangumiGame}},
	}, nil)

	result, err := svc.Import(context.Background(), ImportInput{
		Provider:        "vndb",
		ExternalID:      "v20424",
		DuplicateAction: DuplicateActionCreateNew,
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if result.GalgameID == nil {
		t.Fatalf("no galgame id: %+v", result)
	}
	var game galgameModel.Galgame
	if err := db.First(&game, *result.GalgameID).Error; err != nil {
		t.Fatalf("load imported galgame: %v", err)
	}
	if game.Description != "Bangumi 中文简介" {
		t.Errorf("description = %q, want the Bangumi Chinese summary", game.Description)
	}
	if game.DescriptionSource != galgameModel.DescriptionSourceBangumi {
		t.Errorf("description source = %q, want bangumi", game.DescriptionSource)
	}
	if game.Slug != "vndb-v20424" || game.SourceType != galgameModel.GalgameSourceVNDB {
		t.Errorf("identity must stay vndb, got slug %q / source type %d", game.Slug, game.SourceType)
	}
	source, err := svc.repository.FindExternalSource(context.Background(), "vndb", "v20424")
	if err != nil || source == nil || source.GalgameID != game.ID {
		t.Fatalf("vndb external source: %v %+v", err, source)
	}
}

func TestImportVndbFallsBackWithoutBangumiMatch(t *testing.T) {
	release := time.Date(2018, time.June, 29, 0, 0, 0, 0, time.UTC)
	vndbGame := testExternalGame("v20424", "Summer Pockets", &release)
	vndbGame.OriginalTitle = ""
	vndbGame.Description = "English description from VNDB"

	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	svc := NewService(importerRepository.NewRepository(db), map[string]provider.Provider{
		"vndb":    &stubProvider{games: []provider.ExternalGame{vndbGame}},
		"bangumi": &stubProvider{games: []provider.ExternalGame{}},
	}, nil)

	result, err := svc.Import(context.Background(), ImportInput{
		Provider:        "vndb",
		ExternalID:      "v20424",
		DuplicateAction: DuplicateActionCreateNew,
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if result.GalgameID == nil {
		t.Fatalf("no galgame id: %+v", result)
	}
	var game galgameModel.Galgame
	if err := db.First(&game, *result.GalgameID).Error; err != nil {
		t.Fatalf("load imported galgame: %v", err)
	}
	if game.Description != "English description from VNDB" {
		t.Errorf("description = %q, want the VNDB fallback", game.Description)
	}
	if game.DescriptionSource != galgameModel.DescriptionSourceVNDB {
		t.Errorf("description source = %q, want vndb", game.DescriptionSource)
	}
}

func TestImportForceLinkKeepsHigherPriorityDescription(t *testing.T) {
	release := time.Date(2018, time.June, 29, 0, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, []provider.ExternalGame{testExternalGame("v20424", "Summer Pockets", &release)})
	seeded := seedGalgame(t, db, "Summer Pockets", &release)
	if err := db.Model(&galgameModel.Galgame{}).Where("id = ?", seeded).
		Updates(map[string]any{
			"description":        "Bangumi 中文简介",
			"description_source": galgameModel.DescriptionSourceBangumi,
		}).Error; err != nil {
		t.Fatalf("seed bangumi description: %v", err)
	}

	result, err := svc.Import(context.Background(), ImportInput{
		Provider:            "vndb",
		ExternalID:          "v20424",
		DuplicateAction:     DuplicateActionLinkExisting,
		ExistingGalgameID:   &seeded,
		ForceMetadataUpdate: true,
	})
	if err != nil {
		t.Fatalf("force link import: %v", err)
	}
	if result.DuplicateStatus != DuplicateStatusNone || result.GalgameID == nil || *result.GalgameID != seeded {
		t.Fatalf("force link result = %+v, want seeded %d", result, seeded)
	}
	var game galgameModel.Galgame
	if err := db.First(&game, seeded).Error; err != nil {
		t.Fatalf("load seeded galgame: %v", err)
	}
	if game.Description != "Bangumi 中文简介" || game.DescriptionSource != galgameModel.DescriptionSourceBangumi {
		t.Errorf("description = %q/%q, force sync must not downgrade the Bangumi summary",
			game.Description, game.DescriptionSource)
	}
	source, err := svc.repository.FindExternalSource(context.Background(), "vndb", "v20424")
	if err != nil || source == nil || source.GalgameID != seeded {
		t.Fatalf("vndb external source after force link: %v %+v", err, source)
	}
}
