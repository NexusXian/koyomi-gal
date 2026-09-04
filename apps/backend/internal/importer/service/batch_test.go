package service

import (
	"context"
	"testing"
	"time"

	galgameModel "backend/internal/galgame/model"
	importerModel "backend/internal/importer/model"
	"backend/internal/importer/provider"
)

func TestRunImportJobPartialFailure(t *testing.T) {
	release := time.Date(2020, time.April, 1, 0, 0, 0, 0, time.UTC)

	created := testExternalGame("v1", "Batch Create", &release)

	duplicate := testExternalGame("v2", "Batch Duplicate", &release)

	broken := testExternalGame("v3", "", &release)
	broken.RomajiTitle = ""

	svc, db := newTestService(t, []provider.ExternalGame{created, duplicate, broken})
	svc.SetBatchEnqueuer(func(ctx context.Context, jobID int64) error { return nil })
	seedGalgame(t, db, "Batch Duplicate", &release)

	minRating := 0.0
	limit := 10
	job, err := svc.CreateBatchJob(context.Background(), "vndb", BatchParams{
		MinRating: &minRating,
		Limit:     limit,
	}, nil)
	if err != nil {
		t.Fatalf("create batch job: %v", err)
	}
	if job.Status != importerModel.ImportJobStatusPending {
		t.Fatalf("job status = %d, want pending", job.Status)
	}

	if err := svc.RunImportJob(context.Background(), int64(job.ID)); err != nil {
		t.Fatalf("run import job: %v", err)
	}

	finished, err := svc.GetImportJob(context.Background(), int64(job.ID))
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if finished.Status != importerModel.ImportJobStatusSucceeded {
		t.Fatalf("status = %d, want succeeded despite item failures", finished.Status)
	}
	if finished.TotalCount != 3 || finished.ProcessedCount != 3 {
		t.Errorf("total/processed = %d/%d, want 3/3", finished.TotalCount, finished.ProcessedCount)
	}
	if finished.CreatedCount != 1 {
		t.Errorf("created = %d, want 1", finished.CreatedCount)
	}
	if finished.SkippedCount != 1 {
		t.Errorf("skipped = %d, want 1 (possible duplicate)", finished.SkippedCount)
	}
	if finished.FailedCount != 1 {
		t.Errorf("failed = %d, want 1 (invalid item)", finished.FailedCount)
	}
	if got := int(countRows(t, db, &galgameModel.Galgame{})); got != 2 {
		t.Errorf("galgames = %d, want 2 (seeded + created)", got)
	}

	// A second run (Asynq retry) must be a no-op.
	if err := svc.RunImportJob(context.Background(), int64(job.ID)); err != nil {
		t.Fatalf("rerun: %v", err)
	}
	again, _ := svc.GetImportJob(context.Background(), int64(job.ID))
	if again.ProcessedCount != 3 {
		t.Errorf("rerun processed = %d, must not double-run", again.ProcessedCount)
	}
}

func TestRunImportJobRespectsLimit(t *testing.T) {
	games := make([]provider.ExternalGame, 0, 5)
	for i := 0; i < 5; i++ {
		games = append(games, testExternalGame("v"+string(rune('1'+i)), "Limit Game "+string(rune('1'+i)), nil))
	}
	svc, db := newTestService(t, games)
	svc.SetBatchEnqueuer(func(ctx context.Context, jobID int64) error { return nil })

	limit := 3
	job, err := svc.CreateBatchJob(context.Background(), "vndb", BatchParams{Limit: limit}, nil)
	if err != nil {
		t.Fatalf("create batch job: %v", err)
	}
	if err := svc.RunImportJob(context.Background(), int64(job.ID)); err != nil {
		t.Fatalf("run import job: %v", err)
	}
	finished, _ := svc.GetImportJob(context.Background(), int64(job.ID))
	if finished.ProcessedCount != 3 {
		t.Errorf("processed = %d, want limit 3", finished.ProcessedCount)
	}
	if got := int(countRows(t, db, &galgameModel.Galgame{})); got != 3 {
		t.Errorf("galgames = %d, want 3", got)
	}
}

func TestCreateBatchJobValidatesProvider(t *testing.T) {
	svc, _ := newTestService(t, nil)
	limit := 10
	if _, err := svc.CreateBatchJob(context.Background(), "bangumi", BatchParams{Limit: limit}, nil); err == nil {
		t.Fatal("expected error for provider without batch support")
	}
	if _, err := svc.CreateBatchJob(context.Background(), "unknown", BatchParams{Limit: limit}, nil); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestBatchDuplicateSkipsAlreadyImported(t *testing.T) {
	release := time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)
	game := testExternalGame("v77", "Repeated Import", &release)
	svc, db := newTestService(t, []provider.ExternalGame{game})
	svc.SetBatchEnqueuer(func(ctx context.Context, jobID int64) error { return nil })

	if _, err := svc.Import(context.Background(), ImportInput{Provider: "vndb", ExternalID: "v77"}); err != nil {
		t.Fatalf("seed import: %v", err)
	}
	limit := 10
	job, err := svc.CreateBatchJob(context.Background(), "vndb", BatchParams{Limit: limit}, nil)
	if err != nil {
		t.Fatalf("create batch job: %v", err)
	}
	if err := svc.RunImportJob(context.Background(), int64(job.ID)); err != nil {
		t.Fatalf("run import job: %v", err)
	}
	finished, _ := svc.GetImportJob(context.Background(), int64(job.ID))
	if finished.SkippedCount != 1 || finished.CreatedCount != 0 {
		t.Errorf("created/skipped = %d/%d, want 0/1", finished.CreatedCount, finished.SkippedCount)
	}
	if got := int(countRows(t, db, &galgameModel.Galgame{})); got != 1 {
		t.Errorf("galgames = %d, want 1", got)
	}
}
