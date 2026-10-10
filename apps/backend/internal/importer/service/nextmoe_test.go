package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	galgameModel "backend/internal/galgame/model"
	galgameRepository "backend/internal/galgame/repository"
	"backend/internal/importer/provider"
	"backend/internal/infrastructures/nextmoe"

	"gorm.io/gorm"
)

func TestDecideZhDescription(t *testing.T) {
	incoming := zhIncoming{Content: "新简介", WorkID: "42", UpstreamSource: "bangumi"}
	automatic := zhDescriptionState{Content: "旧简介", SourceType: "nextmoe", SourceExternalID: "42", UpstreamSource: "bangumi", HasRow: true}
	manual := zhDescriptionState{Content: "人工简介", SourceType: "manual", HasRow: true}
	formNextMoe := zhDescriptionState{Content: "表单默认来源", SourceType: "nextmoe", HasRow: true}
	legacy := zhDescriptionState{Content: "旧列", SourceType: "unknown"}
	bangumi := zhDescriptionState{Content: "Bangumi 简介", SourceType: "bangumi", HasRow: true}

	cases := []struct {
		name      string
		current   zhDescriptionState
		incoming  zhIncoming
		opts      ZhDescriptionOptions
		wantWrite bool
		want      string
	}{
		{"fill empty", zhDescriptionState{}, incoming, ZhDescriptionOptions{}, true, ZhDescriptionImported},
		{"fill blank row", zhDescriptionState{Content: "  ", SourceType: "manual", HasRow: true}, incoming, ZhDescriptionOptions{}, true, ZhDescriptionImported},
		{"empty incoming", zhDescriptionState{}, zhIncoming{}, ZhDescriptionOptions{Force: true}, false, ZhDescriptionNoIntro},
		{"identical", zhDescriptionState{Content: "新简介", SourceType: "nextmoe", SourceExternalID: "42", UpstreamSource: "bangumi", HasRow: true}, incoming, ZhDescriptionOptions{Force: true}, false, ZhDescriptionUnchanged},
		{"identical manual", zhDescriptionState{Content: "新简介", SourceType: "manual", HasRow: true}, incoming, ZhDescriptionOptions{Force: true, OverwriteManual: true}, false, ZhDescriptionUnchanged},
		{"automatic kept", automatic, incoming, ZhDescriptionOptions{}, false, ZhDescriptionKeptExisting},
		{"automatic forced", automatic, incoming, ZhDescriptionOptions{Force: true}, true, ZhDescriptionImported},
		{"bangumi kept", bangumi, incoming, ZhDescriptionOptions{}, false, ZhDescriptionKeptExisting},
		{"bangumi forced", bangumi, incoming, ZhDescriptionOptions{Force: true}, true, ZhDescriptionImported},
		{"manual protected", manual, incoming, ZhDescriptionOptions{}, false, ZhDescriptionProtected},
		{"manual force only", manual, incoming, ZhDescriptionOptions{Force: true}, false, ZhDescriptionProtected},
		{"manual overwrite without force", manual, incoming, ZhDescriptionOptions{OverwriteManual: true}, false, ZhDescriptionProtected},
		{"manual overwrite", manual, incoming, ZhDescriptionOptions{Force: true, OverwriteManual: true}, true, ZhDescriptionImported},
		{"form nextmoe protected", formNextMoe, incoming, ZhDescriptionOptions{Force: true}, false, ZhDescriptionProtected},
		{"legacy protected", legacy, incoming, ZhDescriptionOptions{Force: true}, false, ZhDescriptionProtected},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			write, status := decideZhDescription(tc.current, tc.incoming, tc.opts)
			if write != tc.wantWrite || status != tc.want {
				t.Fatalf("got (%v, %q), want (%v, %q)", write, status, tc.wantWrite, tc.want)
			}
		})
	}
}

func TestBatchStatsRecord(t *testing.T) {
	var stats BatchStats
	for _, status := range []string{
		ZhDescriptionImported, ZhDescriptionUnchanged, ZhDescriptionKeptExisting, ZhDescriptionProtected,
		ZhDescriptionNotFound, ZhDescriptionAmbiguous, ZhDescriptionNoIntro, ZhDescriptionFailed,
		ZhDescriptionDisabled, ZhDescriptionSkipped, "",
	} {
		stats.record(status)
	}
	if stats != (BatchStats{ZhImported: 1, ZhKept: 3, ZhUnavailable: 3, ZhFailed: 1}) {
		t.Fatalf("unexpected stats %+v", stats)
	}
}

// fakeNextMoe serves the catalog endpoints from an in-memory VNDB ID to
// intro map. failing switches every response to 500.
type fakeNextMoe struct {
	intros  map[string]string
	failing atomic.Bool
	calls   atomic.Int32
}

func (f *fakeNextMoe) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer nmk_test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if f.failing.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if r.URL.Path != "/v2/catalog/works" {
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		type item struct {
			ID     string          `json:"id"`
			Refs   []nextmoe.Ref   `json:"refs"`
			Intros []nextmoe.Intro `json:"intros"`
		}
		items := []item{}
		for i, ref := range strings.Split(r.URL.Query().Get("refs"), ",") {
			id := strings.TrimPrefix(ref, "vndb:")
			content, ok := f.intros[id]
			if !ok {
				continue
			}
			items = append(items, item{
				ID:   string(rune('1' + i)),
				Refs: []nextmoe.Ref{{Source: "vndb", ExternalID: id}},
				Intros: []nextmoe.Intro{
					{Lang: "zh-Hant", Value: "繁體", Source: "x"},
					{Lang: "zh-Hans", Value: content, Source: "bangumi"},
				},
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "items": items})
	}
}

func newNextMoeTestService(t *testing.T, games []provider.ExternalGame, intros map[string]string) (*Service, *fakeNextMoe, *gorm.DB) {
	t.Helper()
	svc, db := newTestService(t, games)
	fake := &fakeNextMoe{intros: intros}
	server := httptest.NewServer(fake.handler(t))
	t.Cleanup(server.Close)
	svc.SetNextMoeClient(nextmoe.NewClient(server.URL+"/v2", "nmk_test", nextmoe.Options{
		HTTPClient:        server.Client(),
		RequestsPerMinute: 600000,
		MaxRetries:        1,
	}))
	return svc, fake, db
}

func TestImportNextMoeChineseDescription(t *testing.T) {
	game := testExternalGame("v17", "Ever17", nil)
	svc, _, db := newNextMoeTestService(t, []provider.ExternalGame{game}, map[string]string{"v17": " 中文简介\r\n第二行 "})
	ctx := context.Background()

	result, err := svc.Import(ctx, ImportInput{Provider: "vndb", ExternalID: "v17"})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if result.ZhDescriptionStatus != ZhDescriptionImported || result.GalgameID == nil {
		t.Fatalf("result = %+v, want imported zh", result)
	}
	rows := loadDescriptionRows(t, db, *result.GalgameID)
	zh := rows[galgameModel.LanguageZhCN]
	if zh.Content != "中文简介\n第二行" || zh.SourceType != galgameModel.DescriptionSourceNextMoe ||
		zh.SourceExternalID != "1" || zh.UpstreamSource != "bangumi" || zh.SourceName != "NextMoe 资料库" {
		t.Fatalf("zh row = %+v", zh)
	}
	if en := rows[galgameModel.LanguageEnUS]; en.Content != "description v17" || en.SourceType != galgameModel.DescriptionSourceVNDB {
		t.Fatalf("en row must stay VNDB, got %+v", en)
	}
	var stored galgameModel.Galgame
	if err := db.First(&stored, *result.GalgameID).Error; err != nil {
		t.Fatalf("load galgame: %v", err)
	}
	if stored.Description != "中文简介\n第二行" || stored.DescriptionSource != galgameModel.DescriptionSourceNextMoe {
		t.Fatalf("legacy mirror = %q/%q", stored.Description, stored.DescriptionSource)
	}

	again, err := svc.Import(ctx, ImportInput{Provider: "vndb", ExternalID: "v17"})
	if err != nil {
		t.Fatalf("reimport: %v", err)
	}
	if again.DuplicateStatus != DuplicateStatusAlreadyImported || again.ZhDescriptionStatus != ZhDescriptionUnchanged {
		t.Fatalf("reimport = %+v, want already_imported/unchanged", again)
	}
	after := loadDescriptionRows(t, db, *result.GalgameID)[galgameModel.LanguageZhCN]
	if !after.UpdatedAt.Equal(zh.UpdatedAt) {
		t.Fatalf("idempotent reimport must not rewrite the row")
	}
	if got := countRows(t, db, &galgameModel.GalgameDescription{}); got != 2 {
		t.Fatalf("description rows = %d, want 2", got)
	}
}

func TestImportNextMoeRespectsManualDescription(t *testing.T) {
	game := testExternalGame("v17", "Ever17", nil)
	svc, fake, db := newNextMoeTestService(t, []provider.ExternalGame{game}, map[string]string{"v17": "第一版"})
	ctx := context.Background()

	result, err := svc.Import(ctx, ImportInput{Provider: "vndb", ExternalID: "v17"})
	if err != nil || result.GalgameID == nil {
		t.Fatalf("import: %+v %v", result, err)
	}
	id := *result.GalgameID

	fake.intros["v17"] = "第二版"
	kept, err := svc.Import(ctx, ImportInput{Provider: "vndb", ExternalID: "v17"})
	if err != nil || kept.ZhDescriptionStatus != ZhDescriptionKeptExisting {
		t.Fatalf("unforced refresh = %+v %v, want kept_existing", kept, err)
	}
	forced, err := svc.Import(ctx, ImportInput{Provider: "vndb", ExternalID: "v17", ZhDescription: ZhDescriptionOptions{Force: true}})
	if err != nil || forced.ZhDescriptionStatus != ZhDescriptionImported {
		t.Fatalf("forced refresh = %+v %v, want imported", forced, err)
	}
	if zh := loadDescriptionRows(t, db, id)[galgameModel.LanguageZhCN]; zh.Content != "第二版" {
		t.Fatalf("forced refresh content = %q", zh.Content)
	}

	// A human edit through the regular description path clears provenance.
	repo := galgameRepository.NewGalgameDescriptionRepository(db)
	if err := repo.Upsert(ctx, &galgameModel.GalgameDescription{
		GalgameID: id, Language: galgameModel.LanguageZhCN, Content: "人工编辑",
		SourceType: galgameModel.DescriptionSourceNextMoe, SourceName: "NextMoe 资料库",
	}); err != nil {
		t.Fatalf("manual upsert: %v", err)
	}
	edited := loadDescriptionRows(t, db, id)[galgameModel.LanguageZhCN]
	if edited.SourceExternalID != "" || edited.UpstreamSource != "" {
		t.Fatalf("manual edit must clear provenance, got %+v", edited)
	}

	fake.intros["v17"] = "第三版"
	protected, err := svc.Import(ctx, ImportInput{Provider: "vndb", ExternalID: "v17", ZhDescription: ZhDescriptionOptions{Force: true}})
	if err != nil || protected.ZhDescriptionStatus != ZhDescriptionProtected {
		t.Fatalf("forced over manual = %+v %v, want protected", protected, err)
	}
	if zh := loadDescriptionRows(t, db, id)[galgameModel.LanguageZhCN]; zh.Content != "人工编辑" {
		t.Fatalf("manual content overwritten: %q", zh.Content)
	}
	overwritten, err := svc.Import(ctx, ImportInput{Provider: "vndb", ExternalID: "v17", ZhDescription: ZhDescriptionOptions{Force: true, OverwriteManual: true}})
	if err != nil || overwritten.ZhDescriptionStatus != ZhDescriptionImported {
		t.Fatalf("explicit overwrite = %+v %v, want imported", overwritten, err)
	}
	if zh := loadDescriptionRows(t, db, id)[galgameModel.LanguageZhCN]; zh.Content != "第三版" || zh.SourceExternalID == "" {
		t.Fatalf("explicit overwrite row = %+v", zh)
	}
}

func TestManualUpsertKeepsProvenanceWhenUnchanged(t *testing.T) {
	game := testExternalGame("v17", "Ever17", nil)
	svc, _, db := newNextMoeTestService(t, []provider.ExternalGame{game}, map[string]string{"v17": "导入简介"})
	ctx := context.Background()
	result, err := svc.Import(ctx, ImportInput{Provider: "vndb", ExternalID: "v17"})
	if err != nil || result.GalgameID == nil {
		t.Fatalf("import: %+v %v", result, err)
	}
	if err := galgameRepository.NewGalgameDescriptionRepository(db).Upsert(ctx, &galgameModel.GalgameDescription{
		GalgameID: *result.GalgameID, Language: galgameModel.LanguageZhCN, Content: "导入简介",
		SourceType: galgameModel.DescriptionSourceNextMoe, SourceName: "NextMoe 资料库",
	}); err != nil {
		t.Fatalf("resave: %v", err)
	}
	zh := loadDescriptionRows(t, db, *result.GalgameID)[galgameModel.LanguageZhCN]
	if zh.SourceExternalID != "1" || zh.UpstreamSource != "bangumi" {
		t.Fatalf("re-saving unchanged content must keep provenance, got %+v", zh)
	}
}

func TestImportNextMoeFailureKeepsVNDBData(t *testing.T) {
	game := testExternalGame("v17", "Ever17", nil)
	svc, fake, db := newNextMoeTestService(t, []provider.ExternalGame{game}, map[string]string{"v17": "中文"})
	fake.failing.Store(true)

	result, err := svc.Import(context.Background(), ImportInput{Provider: "vndb", ExternalID: "v17"})
	if err != nil {
		t.Fatalf("import must succeed despite NextMoe failure: %v", err)
	}
	if result.GalgameID == nil || result.ZhDescriptionStatus != ZhDescriptionFailed {
		t.Fatalf("result = %+v, want created with failed zh", result)
	}
	rows := loadDescriptionRows(t, db, *result.GalgameID)
	if _, ok := rows[galgameModel.LanguageZhCN]; ok {
		t.Fatal("no zh row expected after NextMoe failure")
	}
	if rows[galgameModel.LanguageEnUS].Content != "description v17" {
		t.Fatal("VNDB description must be kept")
	}
}

func TestImportNextMoeDisabled(t *testing.T) {
	svc, _ := newTestService(t, []provider.ExternalGame{testExternalGame("v17", "Ever17", nil)})
	result, err := svc.Import(context.Background(), ImportInput{Provider: "vndb", ExternalID: "v17"})
	if err != nil || result.ZhDescriptionStatus != ZhDescriptionDisabled || result.GalgameID == nil {
		t.Fatalf("disabled = %+v %v", result, err)
	}
}

func TestImportNextMoeNotFound(t *testing.T) {
	svc, _, _ := newNextMoeTestService(t, []provider.ExternalGame{testExternalGame("v18", "Ever18", nil)}, map[string]string{})
	result, err := svc.Import(context.Background(), ImportInput{Provider: "vndb", ExternalID: "v18"})
	if err != nil || result.ZhDescriptionStatus != ZhDescriptionNotFound || result.GalgameID == nil {
		t.Fatalf("not found = %+v %v", result, err)
	}
}

func TestPreviewIncludesNextMoeIntro(t *testing.T) {
	svc, _, _ := newNextMoeTestService(t, []provider.ExternalGame{testExternalGame("v17", "Ever17", nil)}, map[string]string{"v17": "预览简介"})
	preview, err := svc.Get(context.Background(), "vndb", "v17")
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.ZhDescription == nil || preview.ZhDescription.Status != ZhDescriptionAvailable ||
		preview.ZhDescription.Content != "预览简介" {
		t.Fatalf("preview zh = %+v", preview.ZhDescription)
	}
}

func TestBatchImportNextMoeStats(t *testing.T) {
	release := time.Date(2020, time.April, 1, 0, 0, 0, 0, time.UTC)
	games := []provider.ExternalGame{
		testExternalGame("v1", "Batch One", &release),
		testExternalGame("v2", "Batch Two", &release),
	}
	svc, fake, db := newNextMoeTestService(t, games, map[string]string{"v1": "批量中文"})
	svc.SetBatchEnqueuer(func(ctx context.Context, jobID int64) error { return nil })
	ctx := context.Background()

	run := func() BatchStats {
		job, err := svc.CreateBatchJob(ctx, "vndb", BatchParams{Limit: 10}, nil)
		if err != nil {
			t.Fatalf("create batch: %v", err)
		}
		if err := svc.RunImportJob(ctx, int64(job.ID)); err != nil {
			t.Fatalf("run batch: %v", err)
		}
		finished, err := svc.GetImportJob(ctx, int64(job.ID))
		if err != nil {
			t.Fatalf("get job: %v", err)
		}
		var stats BatchStats
		if err := json.Unmarshal(finished.Stats, &stats); err != nil {
			t.Fatalf("decode stats: %v", err)
		}
		return stats
	}

	first := run()
	if first != (BatchStats{ZhImported: 1, ZhUnavailable: 1}) {
		t.Fatalf("first batch stats = %+v", first)
	}
	if got := fake.calls.Load(); got != 1 {
		t.Fatalf("one page must cost one NextMoe request, got %d", got)
	}
	second := run()
	if second != (BatchStats{ZhKept: 1, ZhUnavailable: 1}) {
		t.Fatalf("rerun stats = %+v, want idempotent", second)
	}
	if got := countRows(t, db, &galgameModel.GalgameDescription{}); got != 3 {
		t.Fatalf("description rows = %d, want 2 en + 1 zh", got)
	}
}
