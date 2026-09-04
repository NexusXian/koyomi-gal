package service

import (
	"context"
	"encoding/json"
	"errors"
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

func bangumiExternalGame(id, name, nameCN string, release *time.Time) provider.ExternalGame {
	rating := 8.2
	ratingCount := 5819
	game := provider.ExternalGame{
		Source:        "bangumi",
		ExternalID:    id,
		Title:         nameCN,
		OriginalTitle: name,
		Aliases:       []string{name, nameCN, "别名-" + id},
		Description:   "中文简介 " + id,
		CoverURL:      "https://lain.bgm.tv/pic/cover/l/" + id + ".jpg",
		ReleaseDate:   release,
		Rating:        &rating,
		RatingCount:   &ratingCount,
		Developer:     &provider.ExternalDeveloper{Name: "VISUAL ARTS / Key"},
		Raw:           json.RawMessage(`{"id":` + id + `}`),
	}
	game.Tags = []provider.ExternalTag{
		{Name: "催泪", Rating: 328},
		{Name: "Key", Rating: 2362},
	}
	return game
}

// stubSearchProvider stubs a provider whose Search returns fixed results.
type stubSearchProvider struct {
	games   []provider.ExternalGame
	failing bool
}

func (s *stubSearchProvider) Search(ctx context.Context, query string, limit int) ([]provider.ExternalGame, error) {
	if s.failing {
		return nil, context.DeadlineExceeded
	}
	results := make([]provider.ExternalGame, 0, len(s.games))
	for i, game := range s.games {
		if i >= limit {
			break
		}
		results = append(results, game)
	}
	return results, nil
}

func (s *stubSearchProvider) Get(ctx context.Context, externalID string) (*provider.ExternalGame, error) {
	for i := range s.games {
		if s.games[i].ExternalID == externalID {
			copy := s.games[i]
			return &copy, nil
		}
	}
	return nil, nil
}

func newEnrichTestService(t *testing.T, bangumi provider.Provider) (*Service, *gorm.DB) {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	repo := importerRepository.NewRepository(db)
	providers := map[string]provider.Provider{
		"vndb":    &stubProvider{},
		"bangumi": bangumi,
	}
	return NewService(repo, providers, nil), db
}

func seedVndbGalgame(t *testing.T, db *gorm.DB, title, originalTitle, romajiTitle string, release *time.Time, developer string) uint {
	t.Helper()
	game := &galgameModel.Galgame{
		Title:         title,
		OriginalTitle: originalTitle,
		RomajiTitle:   romajiTitle,
		Slug:          "vndb-test-" + strings.ToLower(strings.ReplaceAll(originalTitle, " ", "-")),
		Description:   "",
		ReleaseDate:   release,
		Status:        galgameModel.GalgameStatusPublished,
		SourceType:    galgameModel.GalgameSourceVNDB,
	}
	if developer != "" {
		var existing galgameModel.Developer
		err := db.Where("slug = ?", developer).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			created := &galgameModel.Developer{Name: developer, Slug: developer}
			if err := db.Create(created).Error; err != nil {
				t.Fatalf("seed developer: %v", err)
			}
			game.DeveloperID = &created.ID
		} else if err != nil {
			t.Fatalf("find developer: %v", err)
		} else {
			game.DeveloperID = &existing.ID
		}
	}
	if err := db.Create(game).Error; err != nil {
		t.Fatalf("seed galgame: %v", err)
	}
	source := &importerModel.GalgameExternalSource{
		GalgameID:  game.ID,
		Source:     "vndb",
		ExternalID: "v" + time.Now().Format("150405.000000000") + "-" + string(rune('a'+int(game.ID%26))),
		URL:        "https://vndb.org/v1",
	}
	if err := db.Create(source).Error; err != nil {
		t.Fatalf("seed vndb source: %v", err)
	}
	return game.ID
}

func TestEnrichFillsOnlyEmptyFields(t *testing.T) {
	release := date(2018, time.June, 29)
	external := bangumiExternalGame("200763", "Summer Pockets", "夏日口袋", release)
	svc, db := newEnrichTestService(t, &stubSearchProvider{games: []provider.ExternalGame{external}})
	gameID := seedVndbGalgame(t, db, "Summer Pockets", "サマーポケッツ", "Summer Pockets", release, "Key")

	// Pre-existing user data that must survive.
	if err := db.Create(&galgameModel.Alias{GalgameID: gameID, Alias: "既存别名"}).Error; err != nil {
		t.Fatalf("seed alias: %v", err)
	}
	if err := db.Model(&galgameModel.Galgame{}).Where("id = ?", gameID).
		Update("description", "用户维护的简介").Error; err != nil {
		t.Fatalf("seed description: %v", err)
	}

	result, err := svc.Enrich(context.Background(), gameID, "bangumi", "200763", DefaultEnrichOptions())
	if err != nil {
		t.Fatalf("enrich: %v", err)
	}
	if result.GalgameID != gameID {
		t.Errorf("result galgame id = %d", result.GalgameID)
	}
	for _, field := range result.UpdatedFields {
		if field == EnrichFieldTitle || field == EnrichFieldDescription {
			t.Errorf("field %q must not be touched when already maintained", field)
		}
	}

	var game galgameModel.Galgame
	if err := db.Preload("Aliases").Preload("Tags").First(&game, gameID).Error; err != nil {
		t.Fatalf("load enriched galgame: %v", err)
	}
	if game.Title != "Summer Pockets" {
		t.Errorf("title = %q, must not be overwritten by the Bangumi Chinese name", game.Title)
	}
	if game.Description != "用户维护的简介" {
		t.Errorf("description = %q, must not be overwritten", game.Description)
	}
	if game.CoverURL != external.CoverURL {
		t.Errorf("cover url = %q, want the empty cover filled", game.CoverURL)
	}
	if game.SourceType != galgameModel.GalgameSourceMixed {
		t.Errorf("source type = %d, want mixed after bangumi enrichment", game.SourceType)
	}
	aliasSet := map[string]bool{}
	for _, alias := range game.Aliases {
		aliasSet[alias.Alias] = true
	}
	if !aliasSet["既存别名"] {
		t.Errorf("existing alias removed: %v", game.Aliases)
	}
	if !aliasSet["夏日口袋"] || !aliasSet["别名-200763"] {
		t.Errorf("bangumi aliases missing: %v", game.Aliases)
	}
	if len(game.Tags) != 2 {
		t.Errorf("tags = %d, want both bangumi tags added", len(game.Tags))
	}
}

func TestEnrichFillsEmptyTitleAndDescription(t *testing.T) {
	release := date(2018, time.June, 29)
	external := bangumiExternalGame("200763", "Summer Pockets", "夏日口袋", release)
	svc, db := newEnrichTestService(t, &stubSearchProvider{games: []provider.ExternalGame{external}})
	gameID := seedVndbGalgame(t, db, "", "サマーポケッツ", "Summer Pockets", release, "")

	result, err := svc.Enrich(context.Background(), gameID, "bangumi", "200763", DefaultEnrichOptions())
	if err != nil {
		t.Fatalf("enrich: %v", err)
	}
	assertContainsField(t, result.UpdatedFields, EnrichFieldTitle)
	assertContainsField(t, result.UpdatedFields, EnrichFieldDescription)

	var game galgameModel.Galgame
	if err := db.First(&game, gameID).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	if game.Title != "夏日口袋" {
		t.Errorf("title = %q, want Chinese title filled", game.Title)
	}
	if game.Description != "中文简介 200763" {
		t.Errorf("description = %q, want Chinese summary filled", game.Description)
	}
}

func assertContainsField(t *testing.T, fields []string, want string) {
	t.Helper()
	for _, field := range fields {
		if field == want {
			return
		}
	}
	t.Errorf("updated fields %v missing %q", fields, want)
}

func TestEnrichForceOverwritesMaintainedFields(t *testing.T) {
	release := date(2018, time.June, 29)
	external := bangumiExternalGame("200763", "Summer Pockets", "夏日口袋", release)
	svc, db := newEnrichTestService(t, &stubSearchProvider{games: []provider.ExternalGame{external}})
	gameID := seedVndbGalgame(t, db, "本地标题", "サマーポケッツ", "Summer Pockets", release, "Key")

	opts := DefaultEnrichOptions()
	opts.Force = true
	if _, err := svc.Enrich(context.Background(), gameID, "bangumi", "200763", opts); err != nil {
		t.Fatalf("enrich: %v", err)
	}
	var game galgameModel.Galgame
	if err := db.First(&game, gameID).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	if game.Title != "夏日口袋" {
		t.Errorf("title = %q, want forced overwrite", game.Title)
	}
	if game.Description != "中文简介 200763" {
		t.Errorf("description = %q, want forced overwrite", game.Description)
	}
}

func TestEnrichRespectsFieldSelection(t *testing.T) {
	release := date(2018, time.June, 29)
	external := bangumiExternalGame("200763", "Summer Pockets", "夏日口袋", release)
	svc, db := newEnrichTestService(t, &stubSearchProvider{games: []provider.ExternalGame{external}})
	gameID := seedVndbGalgame(t, db, "", "サマーポケッツ", "Summer Pockets", release, "")

	opts, err := ParseEnrichFields([]string{EnrichFieldTitle}, false)
	if err != nil {
		t.Fatalf("parse fields: %v", err)
	}
	if _, err := svc.Enrich(context.Background(), gameID, "bangumi", "200763", opts); err != nil {
		t.Fatalf("enrich: %v", err)
	}
	var game galgameModel.Galgame
	if err := db.First(&game, gameID).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	if game.Title != "夏日口袋" {
		t.Errorf("title = %q, want selected fill", game.Title)
	}
	if game.Description != "" || game.CoverURL != "" {
		t.Errorf("unselected fields must stay empty: %q/%q", game.Description, game.CoverURL)
	}
	var aliases []galgameModel.Alias
	if err := db.Where("galgame_id = ?", gameID).Find(&aliases).Error; err != nil {
		t.Fatalf("aliases: %v", err)
	}
	if len(aliases) != 0 {
		t.Errorf("aliases = %v, want none (field not selected)", aliases)
	}
}

func TestParseEnrichFieldsRejectsUnknown(t *testing.T) {
	if _, err := ParseEnrichFields([]string{"title", "hacker"}, false); err == nil {
		t.Fatal("expected error for unknown field")
	}
	opts, err := ParseEnrichFields(nil, false)
	if err != nil {
		t.Fatalf("default fields: %v", err)
	}
	if !opts.FillTitle || !opts.FillTags || opts.Force {
		t.Errorf("default options = %+v", opts)
	}
}

func TestEnrichLinksExternalSourceOnce(t *testing.T) {
	release := date(2018, time.June, 29)
	external := bangumiExternalGame("200763", "Summer Pockets", "夏日口袋", release)
	other := bangumiExternalGame("999999", "Summer Pockets Old", "旧条目", release)
	svc, db := newEnrichTestService(t, &stubSearchProvider{games: []provider.ExternalGame{external, other}})
	gameID := seedVndbGalgame(t, db, "Summer Pockets", "サマーポケッツ", "Summer Pockets", release, "Key")

	if _, err := svc.Enrich(context.Background(), gameID, "bangumi", "999999", DefaultEnrichOptions()); err != nil {
		t.Fatalf("first enrich: %v", err)
	}
	result, err := svc.Enrich(context.Background(), gameID, "bangumi", "200763", DefaultEnrichOptions())
	if err != nil {
		t.Fatalf("second enrich: %v", err)
	}
	if result.UpdatedFields == nil {
		result.UpdatedFields = []string{}
	}

	var sources []importerModel.GalgameExternalSource
	if err := db.Where("galgame_id = ? AND source = 'bangumi'", gameID).Find(&sources).Error; err != nil {
		t.Fatalf("load sources: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("bangumi sources = %d, want exactly one row per galgame", len(sources))
	}
	if sources[0].ExternalID != "200763" || sources[0].ExternalRating == nil || *sources[0].ExternalRating != 8.2 {
		t.Errorf("source = %+v, want upserted mapping with rating", sources[0])
	}
	if sources[0].URL != "https://bgm.tv/subject/200763" {
		t.Errorf("url = %q", sources[0].URL)
	}
	if sources[0].LastSyncedAt == nil {
		t.Error("last_synced_at must be set")
	}
}

func TestEnrichMissingGalgame(t *testing.T) {
	external := bangumiExternalGame("200763", "Summer Pockets", "夏日口袋", nil)
	svc, _ := newEnrichTestService(t, &stubSearchProvider{games: []provider.ExternalGame{external}})
	missing := uint(999999)
	if _, err := svc.Enrich(context.Background(), missing, "bangumi", "200763", DefaultEnrichOptions()); err == nil {
		t.Fatal("expected error for missing galgame")
	}
}

func TestEnrichMissingExternalSubject(t *testing.T) {
	svc, db := newEnrichTestService(t, &stubSearchProvider{})
	gameID := seedVndbGalgame(t, db, "Game", "ゲーム", "Game", nil, "")
	if _, err := svc.Enrich(context.Background(), gameID, "bangumi", "404404", DefaultEnrichOptions()); err == nil {
		t.Fatal("expected error for missing external subject")
	}
}

func TestEnrichOverviewStats(t *testing.T) {
	release := date(2018, time.June, 29)
	svc, db := newEnrichTestService(t, &stubSearchProvider{})
	linked := seedVndbGalgame(t, db, "Linked", "リンク", "Linked", release, "")
	unlinked := seedVndbGalgame(t, db, "Unlinked", "未链接", "Unlinked", release, "")
	manual := seedVndbGalgame(t, db, "Manual", "手動", "Manual", release, "")
	if err := db.Create(&importerModel.GalgameExternalSource{
		GalgameID: linked, Source: "bangumi", ExternalID: "1", URL: "",
	}).Error; err != nil {
		t.Fatalf("seed bangumi source: %v", err)
	}
	// Remove the vndb source of the manual game; it must not count.
	if err := db.Where("galgame_id = ?", manual).Delete(&importerModel.GalgameExternalSource{}).Error; err != nil {
		t.Fatalf("delete manual source: %v", err)
	}

	overview, err := svc.EnrichOverview(context.Background(), "bangumi")
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if overview.VndbCount != 2 {
		t.Errorf("vndb count = %d, want 2", overview.VndbCount)
	}
	if overview.LinkedCount != 1 {
		t.Errorf("linked count = %d, want 1", overview.LinkedCount)
	}
	if overview.UnlinkedCount != 1 {
		t.Errorf("unlinked count = %d, want 1 (galgame %d)", overview.UnlinkedCount, unlinked)
	}
	if overview.PendingCandidates != 0 {
		t.Errorf("pending candidates = %d, want 0", overview.PendingCandidates)
	}
}

func TestListGalgamesForEnrichment(t *testing.T) {
	release := date(2018, time.June, 29)
	svc, db := newEnrichTestService(t, &stubSearchProvider{})
	eligible := seedVndbGalgame(t, db, "Eligible", "対象", "Eligible", release, "")
	seedGalgameAlias(t, db, eligible, "エリジブル")
	already := seedVndbGalgame(t, db, "Already", "済み", "Already", release, "")
	noVndb := seedVndbGalgame(t, db, "NoVndb", "無し", "NoVndb", release, "")
	if err := db.Create(&importerModel.GalgameExternalSource{
		GalgameID: already, Source: "bangumi", ExternalID: "2", URL: "",
	}).Error; err != nil {
		t.Fatalf("seed bangumi source: %v", err)
	}
	// Strip the vndb source from the third game; it must not be eligible.
	if err := db.Where("galgame_id = ?", noVndb).Delete(&importerModel.GalgameExternalSource{}).Error; err != nil {
		t.Fatalf("strip vndb source: %v", err)
	}

	games, err := svc.repository.ListGalgamesForEnrichment(context.Background(), enrichRequireSource, "bangumi", 0, 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(games) != 1 || games[0].ID != eligible {
		t.Fatalf("games = %+v, want only %d", games, eligible)
	}
	if len(games[0].Aliases) == 0 {
		t.Error("aliases must be preloaded for matching")
	}
}

func TestSaveAndReviewMatchCandidates(t *testing.T) {
	release := date(2015, time.November, 27)
	external := bangumiExternalGame("183082", "サクラノ詩", "樱之诗", release)
	svc, db := newEnrichTestService(t, &stubSearchProvider{games: []provider.ExternalGame{external}})
	gameID := seedVndbGalgame(t, db, "樱之诗", "サクラノ詩 -櫻の森の上を舞う-", "Sakura no Uta", release, "Makura")

	// Alias-only + year: review band.
	input := MatchInput{
		Title:         "樱之诗",
		OriginalTitle: "サクラノ詩 -櫻の森の上を舞う-",
		RomajiTitle:   "Sakura no Uta",
		ReleaseDate:   release,
	}
	matches := MatchBangumiCandidates(input, []provider.ExternalGame{external})
	if len(matches) != 1 || matches[0].Confidence >= autoMatchThreshold {
		t.Fatalf("matches = %+v, want a review-band candidate", matches)
	}
	if err := svc.SaveMatchCandidates(context.Background(), gameID, matches); err != nil {
		t.Fatalf("save candidates: %v", err)
	}
	// Re-saving refreshes the pending row instead of duplicating it.
	if err := svc.SaveMatchCandidates(context.Background(), gameID, matches); err != nil {
		t.Fatalf("re-save candidates: %v", err)
	}

	pending, total, err := svc.ListMatchCandidates(context.Background(), nil, 1, 20)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(pending) != 1 {
		t.Fatalf("candidates = %d/%d, want 1", len(pending), total)
	}
	candidate := pending[0]
	if candidate.GalgameID != gameID || candidate.ExternalID != "183082" {
		t.Fatalf("candidate = %+v", candidate)
	}
	if candidate.Status != importerModel.MatchCandidateStatusPending {
		t.Errorf("status = %d, want pending", candidate.Status)
	}

	// Rejecting then re-saving must not resurrect the rejected candidate.
	if err := svc.RejectMatchCandidate(context.Background(), candidate.ID, nil); err != nil {
		t.Fatalf("reject: %v", err)
	}
	if err := svc.RejectMatchCandidate(context.Background(), candidate.ID, nil); err == nil {
		t.Fatal("double reject must fail")
	}
	if err := svc.SaveMatchCandidates(context.Background(), gameID, matches); err != nil {
		t.Fatalf("save after reject: %v", err)
	}
	_, total, _ = svc.ListMatchCandidates(context.Background(), nil, 1, 20)
	if total != 1 {
		t.Fatalf("candidates total = %d, want rejected row preserved without duplicates", total)
	}
	var reloaded importerModel.ExternalMatchCandidate
	if err := db.First(&reloaded, candidate.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Status != importerModel.MatchCandidateStatusRejected {
		t.Errorf("status = %d, want rejected", reloaded.Status)
	}
}

func TestApproveMatchCandidateEnriches(t *testing.T) {
	release := date(2015, time.November, 27)
	external := bangumiExternalGame("183082", "サクラノ詩", "樱之诗", release)
	svc, db := newEnrichTestService(t, &stubSearchProvider{games: []provider.ExternalGame{external}})
	gameID := seedVndbGalgame(t, db, "樱之诗", "サクラノ詩 -櫻の森の上を舞う-", "Sakura no Uta", release, "Makura")

	matches := MatchBangumiCandidates(MatchInput{
		Title: "樱之诗", OriginalTitle: "サクラノ詩 -櫻の森の上を舞う-", ReleaseDate: release,
	}, []provider.ExternalGame{external})
	if err := svc.SaveMatchCandidates(context.Background(), gameID, matches); err != nil {
		t.Fatalf("save: %v", err)
	}
	pending, _, err := svc.ListMatchCandidates(context.Background(), nil, 1, 20)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending: %v %+v", err, pending)
	}

	reviewer := testutil.CreateUser(t, db, "enrich-reviewer")
	result, err := svc.ApproveMatchCandidate(context.Background(), pending[0].ID, &reviewer)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if result.GalgameID != gameID || result.ExternalID != "183082" {
		t.Fatalf("result = %+v", result)
	}

	var approved importerModel.ExternalMatchCandidate
	if err := db.First(&approved, pending[0].ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if approved.Status != importerModel.MatchCandidateStatusApproved || approved.ReviewedBy == nil || *approved.ReviewedBy != reviewer {
		t.Errorf("candidate = %+v, want approved by %d", approved, reviewer)
	}
	var sources []importerModel.GalgameExternalSource
	if err := db.Where("galgame_id = ? AND source = 'bangumi'", gameID).Find(&sources).Error; err != nil {
		t.Fatalf("sources: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("bangumi sources = %d, want the approved link", len(sources))
	}
	if _, err := svc.ApproveMatchCandidate(context.Background(), pending[0].ID, &reviewer); err == nil {
		t.Fatal("double approve must fail")
	}
}

func TestApproveMatchCandidateMissingSubject(t *testing.T) {
	release := date(2015, time.November, 27)
	external := bangumiExternalGame("183082", "サクラノ詩", "樱之诗", release)
	svc, db := newEnrichTestService(t, &stubSearchProvider{games: []provider.ExternalGame{external}})
	gameID := seedVndbGalgame(t, db, "樱之诗", "サクラノ詩", "Sakura no Uta", release, "")

	// Save a candidate pointing at an external subject the stub cannot fetch.
	row := &importerModel.ExternalMatchCandidate{
		GalgameID:  gameID,
		Provider:   "bangumi",
		ExternalID: "404404",
		Confidence: 0.72,
		Reasons:    json.RawMessage(`["alias_match"]`),
	}
	if err := svc.repository.UpsertMatchCandidate(context.Background(), row); err != nil {
		t.Fatalf("seed candidate: %v", err)
	}
	pending, _, _ := svc.ListMatchCandidates(context.Background(), nil, 1, 20)
	if len(pending) != 1 {
		t.Fatalf("pending = %+v", pending)
	}
	if _, err := svc.ApproveMatchCandidate(context.Background(), pending[0].ID, nil); err == nil {
		t.Fatal("approve must fail when the external subject is gone")
	}
	// The failed approve must leave the candidate pending for retry.
	var reloaded importerModel.ExternalMatchCandidate
	if err := db.First(&reloaded, pending[0].ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Status != importerModel.MatchCandidateStatusPending {
		t.Errorf("status = %d, want still pending after failed approve", reloaded.Status)
	}
}
