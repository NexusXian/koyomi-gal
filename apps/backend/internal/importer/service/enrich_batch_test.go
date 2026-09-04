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

	"gorm.io/gorm"
)

// querySearchProvider routes searches by keyword so each galgame sees only
// its own candidates, mimicking the real provider.
type querySearchProvider struct {
	byKeyword map[string][]provider.ExternalGame
	failKeys  map[string]bool
}

func (q *querySearchProvider) Search(ctx context.Context, keyword string, limit int) ([]provider.ExternalGame, error) {
	if q.failKeys != nil && q.failKeys[keyword] {
		return nil, context.DeadlineExceeded
	}
	games := q.byKeyword[keyword]
	if len(games) > limit {
		games = games[:limit]
	}
	return games, nil
}

func (q *querySearchProvider) Get(ctx context.Context, externalID string) (*provider.ExternalGame, error) {
	for _, games := range q.byKeyword {
		for i := range games {
			if games[i].ExternalID == externalID {
				found := games[i]
				return &found, nil
			}
		}
	}
	return nil, nil
}

func seedGalgameAlias(t *testing.T, db *gorm.DB, galgameID uint, alias string) {
	t.Helper()
	if err := db.Create(&galgameModel.Alias{GalgameID: galgameID, Alias: alias}).Error; err != nil {
		t.Fatalf("seed alias: %v", err)
	}
}

func TestRunEnrichJobBandsAndPartialFailure(t *testing.T) {
	release := date(2018, time.June, 29)

	// High-confidence game: exact title + year + developer.
	autoGame := bangumiExternalGame("200763", "Summer Pockets", "夏日口袋", release)
	autoGame.Developer = &provider.ExternalDeveloper{Name: "Key"}

	// Review-band game: linked only through an alias and the release year.
	reviewRelease := date(2015, time.November, 27)
	reviewGame := bangumiExternalGame("183082", "サクラノ詩", "", reviewRelease)
	reviewGame.Aliases = []string{"サクラノ詩", "樱之诗"}

	bgm := &querySearchProvider{
		byKeyword: map[string][]provider.ExternalGame{
			"サマーポケッツ": {autoGame},
			"レビューゲーム": {reviewGame},
			"ノーヒット":   {},
		},
		failKeys: map[string]bool{"フェイルゲーム": true},
	}
	svc, db := newEnrichTestService(t, bgm)
	svc.SetEnrichEnqueuer(func(ctx context.Context, jobID int64) error { return nil })

	autoID := seedVndbGalgame(t, db, "Summer Pockets", "サマーポケッツ", "Summer Pockets", release, "Key")
	reviewID := seedVndbGalgame(t, db, "レビュー 佳作", "レビューゲーム", "Review Game", reviewRelease, "Makura")
	seedGalgameAlias(t, db, reviewID, "樱之诗")
	notFoundID := seedVndbGalgame(t, db, "Not Found", "ノーヒット", "No Hit", release, "")
	failID := seedVndbGalgame(t, db, "Failing", "フェイルゲーム", "Failing", release, "")
	// Already linked games are skipped entirely.
	linkedID := seedVndbGalgame(t, db, "Linked", "リンク済み", "Linked", release, "")
	if err := db.Create(&importerModel.GalgameExternalSource{
		GalgameID: linkedID, Source: "bangumi", ExternalID: "1", URL: "",
	}).Error; err != nil {
		t.Fatalf("seed bangumi source: %v", err)
	}

	limit := 10
	job, err := svc.CreateEnrichJob(context.Background(), "bangumi", limit, nil)
	if err != nil {
		t.Fatalf("create enrich job: %v", err)
	}
	if job.JobType != importerModel.ImportJobTypeEnrich {
		t.Fatalf("job type = %q, want enrich", job.JobType)
	}
	if err := svc.RunImportJob(context.Background(), int64(job.ID)); err != nil {
		t.Fatalf("run enrich job: %v", err)
	}

	finished, err := svc.GetImportJob(context.Background(), int64(job.ID))
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if finished.Status != importerModel.ImportJobStatusSucceeded {
		t.Fatalf("status = %d, want succeeded despite item failures", finished.Status)
	}
	if finished.TotalCount != 4 || finished.ProcessedCount != 4 {
		t.Errorf("total/processed = %d/%d, want 4/4 (linked game skipped)", finished.TotalCount, finished.ProcessedCount)
	}
	if finished.CreatedCount != 1 {
		t.Errorf("matched = %d, want 1 auto-linked", finished.CreatedCount)
	}
	if finished.SkippedCount != 1 {
		t.Errorf("not_found = %d, want 1", finished.SkippedCount)
	}
	if finished.FailedCount != 1 {
		t.Errorf("failed = %d, want 1 (search failure)", finished.FailedCount)
	}

	var stats EnrichStats
	if err := json.Unmarshal(finished.Stats, &stats); err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Matched != 1 || stats.Review != 1 || stats.NotFound != 1 || stats.Failed != 1 {
		t.Errorf("stats = %+v, want matched/review/not_found/failed = 1/1/1/1", stats)
	}

	// Auto-linked game received the bangumi source and enrichment.
	var autoSources []importerModel.GalgameExternalSource
	if err := db.Where("galgame_id = ? AND source = 'bangumi'", autoID).Find(&autoSources).Error; err != nil {
		t.Fatalf("auto sources: %v", err)
	}
	if len(autoSources) != 1 {
		t.Fatalf("auto-linked galgame must have a bangumi source, got %d", len(autoSources))
	}
	var autoGameRow galgameModel.Galgame
	if err := db.First(&autoGameRow, autoID).Error; err != nil {
		t.Fatalf("load auto game: %v", err)
	}
	if autoGameRow.CoverURL == "" {
		t.Error("auto enrichment must fill the empty cover")
	}

	// Review-band game got a pending candidate, no source link.
	var candidates []importerModel.ExternalMatchCandidate
	if err := db.Where("galgame_id = ?", reviewID).Find(&candidates).Error; err != nil {
		t.Fatalf("review candidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].Status != importerModel.MatchCandidateStatusPending {
		t.Fatalf("candidates = %+v, want one pending", candidates)
	}
	var reviewSources []importerModel.GalgameExternalSource
	if err := db.Where("galgame_id = ? AND source = 'bangumi'", reviewID).Find(&reviewSources).Error; err != nil {
		t.Fatalf("review sources: %v", err)
	}
	if len(reviewSources) != 0 {
		t.Errorf("review-band game must not be linked automatically")
	}

	// Not-found and failing games have neither candidates nor links.
	for _, id := range []uint{notFoundID, failID} {
		var count int64
		if err := db.Model(&importerModel.GalgameExternalSource{}).
			Where("galgame_id = ? AND source = 'bangumi'", id).Count(&count).Error; err != nil {
			t.Fatalf("count sources: %v", err)
		}
		if count != 0 {
			t.Errorf("galgame %d must stay unlinked", id)
		}
	}

	// A second run (Asynq retry) must be a no-op.
	if err := svc.RunImportJob(context.Background(), int64(job.ID)); err != nil {
		t.Fatalf("rerun: %v", err)
	}
	again, _ := svc.GetImportJob(context.Background(), int64(job.ID))
	if again.ProcessedCount != 4 {
		t.Errorf("rerun processed = %d, must not double-run", again.ProcessedCount)
	}
}

func TestCreateEnrichJobValidatesProvider(t *testing.T) {
	// No enqueuer wired: provider validation fails first, then the missing
	// queue fails without leaving a pending job behind.
	svc, db := newEnrichTestService(t, &stubSearchProvider{})
	if _, err := svc.CreateEnrichJob(context.Background(), "unknown", 10, nil); err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if _, err := svc.CreateEnrichJob(context.Background(), "bangumi", 10, nil); err == nil {
		t.Fatal("expected error when the enrich queue is not configured")
	}
	var jobs int64
	if err := db.Model(&importerModel.ImportJob{}).Count(&jobs).Error; err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if jobs != 1 {
		t.Fatalf("jobs = %d, want 1 (failed enqueue marks the job failed)", jobs)
	}
	var failed importerModel.ImportJob
	if err := db.First(&failed).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	if failed.Status != importerModel.ImportJobStatusFailed {
		t.Errorf("status = %d, want failed", failed.Status)
	}
}

func TestRunEnrichJobRespectsLimit(t *testing.T) {
	release := date(2018, time.June, 29)
	first := bangumiExternalGame("200763", "Game One", "游戏一", release)
	first.Developer = &provider.ExternalDeveloper{Name: "Key"}
	second := bangumiExternalGame("200764", "Game Two", "游戏二", release)
	second.Developer = &provider.ExternalDeveloper{Name: "Key"}
	third := bangumiExternalGame("200765", "Game Three", "游戏三", release)
	third.Developer = &provider.ExternalDeveloper{Name: "Key"}
	bgm := &querySearchProvider{
		byKeyword: map[string][]provider.ExternalGame{
			"サマーポケッツ":  {first},
			"サマーポケッツ２": {second},
			"サマーポケッツ３": {third},
		},
	}
	svc, db := newEnrichTestService(t, bgm)
	svc.SetEnrichEnqueuer(func(ctx context.Context, jobID int64) error { return nil })

	seedVndbGalgame(t, db, "Game One", "サマーポケッツ", "Game One", release, "Key")
	seedVndbGalgame(t, db, "Game Two", "サマーポケッツ２", "Game Two", release, "Key")
	seedVndbGalgame(t, db, "Game Three", "サマーポケッツ３", "Game Three", release, "Key")

	job, err := svc.CreateEnrichJob(context.Background(), "bangumi", 2, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.RunImportJob(context.Background(), int64(job.ID)); err != nil {
		t.Fatalf("run: %v", err)
	}
	finished, _ := svc.GetImportJob(context.Background(), int64(job.ID))
	if finished.ProcessedCount != 2 || finished.TotalCount != 2 {
		t.Errorf("total/processed = %d/%d, want limit 2", finished.TotalCount, finished.ProcessedCount)
	}
	var linked int64
	if err := db.Model(&importerModel.GalgameExternalSource{}).
		Where("source = 'bangumi'").Count(&linked).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if linked != 2 {
		t.Errorf("linked rows = %d, want 2", linked)
	}
}

func TestSearchExternalCandidatesReturnsScoredList(t *testing.T) {
	release := date(2018, time.June, 29)
	autoGame := bangumiExternalGame("200763", "Summer Pockets", "夏日口袋", release)
	autoGame.Developer = &provider.ExternalDeveloper{Name: "Key"}
	edition := bangumiExternalGame("295957", "Summer Pockets REFLECTION BLUE", "夏日口袋 REFLECTION BLUE", date(2020, time.June, 26))
	edition.Developer = &provider.ExternalDeveloper{Name: "Key"}

	svc, db := newEnrichTestService(t, &querySearchProvider{
		byKeyword: map[string][]provider.ExternalGame{
			"サマーポケッツ": {autoGame, edition},
		},
	})
	gameID := seedVndbGalgame(t, db, "Summer Pockets", "サマーポケッツ", "Summer Pockets", release, "Key")

	candidates, err := svc.SearchExternalCandidates(context.Background(), gameID, "bangumi")
	if err != nil {
		t.Fatalf("search candidates: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates = %+v, want only the matching edition", candidates)
	}
	if candidates[0].ExternalID != "200763" {
		t.Fatalf("candidate = %+v", candidates[0])
	}
	if !strings.Contains(strings.Join(candidates[0].Reasons, ","), "original_title_match") {
		t.Errorf("reasons = %v", candidates[0].Reasons)
	}
	if candidates[0].Linked {
		t.Error("candidate must not be linked yet")
	}

	// After enrichment the candidate is flagged as linked.
	if _, err := svc.Enrich(context.Background(), gameID, "bangumi", "200763", DefaultEnrichOptions()); err != nil {
		t.Fatalf("enrich: %v", err)
	}
	candidates, err = svc.SearchExternalCandidates(context.Background(), gameID, "bangumi")
	if err != nil {
		t.Fatalf("search candidates again: %v", err)
	}
	if len(candidates) != 1 || !candidates[0].Linked {
		t.Fatalf("candidates = %+v, want linked flag", candidates)
	}
}

func TestSearchExternalCandidatesMissingGalgame(t *testing.T) {
	svc, _ := newEnrichTestService(t, &stubSearchProvider{})
	missing := uint(999999)
	if _, err := svc.SearchExternalCandidates(context.Background(), missing, "bangumi"); err == nil {
		t.Fatal("expected error for missing galgame")
	}
}
