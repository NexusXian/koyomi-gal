package service

import (
	"context"
	"errors"
	"testing"

	notificationModel "backend/internal/notification/model"
	"backend/internal/novel/dto"
	"backend/internal/novel/model"
	"backend/internal/testutil"
)

func TestVolumeCreateAndValidation(t *testing.T) {
	env := newNovelTestServices(t)
	ctx := context.Background()
	creator := testutil.CreateUser(t, env.db, "volume-create-creator")

	novel := createTestNovel(t, env.novels, creator, "volume-host", model.NovelStatusPublished)

	volume, err := env.volumes.CreateVolume(ctx, creator, novel.ID, &dto.CreateVolumeRequest{
		VolumeNumber: intPtr(1),
		Title:        "青春猪头少年不会梦到兔女郎学姐",
		ISBN:         "978-4-04-865091-8",
		ReleaseDate:  "2014-04-10",
	})
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}
	if volume.SortOrder != 0 {
		t.Fatalf("first volume sort order: got %d want 0", volume.SortOrder)
	}
	if volume.Status != model.NovelStatusPending {
		t.Fatalf("volume should default to pending, got %d", volume.Status)
	}

	second, err := env.volumes.CreateVolume(ctx, creator, novel.ID, &dto.CreateVolumeRequest{
		VolumeNumber: intPtr(2),
		Title:        "第二卷",
		Status:       &publishedStatus,
	})
	if err != nil {
		t.Fatalf("create second volume: %v", err)
	}
	if second.SortOrder != 1 {
		t.Fatalf("second volume sort order: got %d want 1", second.SortOrder)
	}

	// a directly published volume credits its creator once
	assertContributionRows(t, env.db, novel.ID, 2) // novel create + published volume

	if _, err := env.volumes.CreateVolume(ctx, creator, 999999, &dto.CreateVolumeRequest{Title: "x"}); !errors.Is(err, ErrNovelNotFound) {
		t.Fatalf("expected ErrNovelNotFound for missing novel, got %v", err)
	}
	if _, err := env.volumes.CreateVolume(ctx, creator, novel.ID, &dto.CreateVolumeRequest{
		Title: "bad-isbn", ISBN: "not-an-isbn",
	}); !errors.Is(err, ErrInvalidISBN) {
		t.Fatalf("expected ErrInvalidISBN, got %v", err)
	}
	if _, err := env.volumes.CreateVolume(ctx, creator, novel.ID, &dto.CreateVolumeRequest{
		Title: "bad-url", CoverURL: "javascript:alert(1)",
	}); !errors.Is(err, ErrInvalidVolumeURL) {
		t.Fatalf("expected ErrInvalidVolumeURL, got %v", err)
	}
}

func TestVolumeUpdateAcrossNovelsRejected(t *testing.T) {
	env := newNovelTestServices(t)
	ctx := context.Background()
	creator := testutil.CreateUser(t, env.db, "volume-cross-creator")

	novelA := createTestNovel(t, env.novels, creator, "cross-a", model.NovelStatusPublished)
	novelB := createTestNovel(t, env.novels, creator, "cross-b", model.NovelStatusPublished)

	volume, err := env.volumes.CreateVolume(ctx, creator, novelA.ID, &dto.CreateVolumeRequest{
		VolumeNumber: intPtr(1),
		Title:        "A 的第一卷",
		Status:       &publishedStatus,
	})
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}

	// cross-novel update must fail (nested route guard)
	if _, err := env.volumes.UpdateVolume(ctx, creator, novelB.ID, volume.ID, &dto.UpdateVolumeRequest{
		Title:  "试图把 A 的卷改到 B 名下",
		Status: &publishedStatus,
	}); !errors.Is(err, ErrVolumeNotFound) {
		t.Fatalf("expected ErrVolumeNotFound for cross-novel update, got %v", err)
	}
	// cross-novel delete must fail too
	if err := env.volumes.DeleteVolume(ctx, novelB.ID, volume.ID); !errors.Is(err, ErrVolumeNotFound) {
		t.Fatalf("expected ErrVolumeNotFound for cross-novel delete, got %v", err)
	}
	// own-novel update works and records an update contribution
	if _, err := env.volumes.UpdateVolume(ctx, creator, novelA.ID, volume.ID, &dto.UpdateVolumeRequest{
		VolumeNumber: intPtr(1),
		Title:        "A 的第一卷（修订）",
		Status:       &publishedStatus,
	}); err != nil {
		t.Fatalf("update own volume: %v", err)
	}
	assertContributionRows(t, env.db, novelA.ID, 3) // novel create + volume create + volume update
}

func TestVolumeReviewFlow(t *testing.T) {
	env := newNovelTestServices(t)
	ctx := context.Background()
	creator := testutil.CreateUser(t, env.db, "volume-review-creator")
	reviewer := testutil.CreateUser(t, env.db, "volume-review-reviewer")
	env.assignReviewer(t, reviewer)

	novel := createTestNovel(t, env.novels, creator, "volume-review-host", model.NovelStatusPublished)
	volume, err := env.volumes.CreateVolume(ctx, creator, novel.ID, &dto.CreateVolumeRequest{
		Title: "待审核卷",
	})
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}
	assertContributionRows(t, env.db, novel.ID, 1) // only the novel create

	// reviewers got the submission notification
	assertNotificationCount(t, env.db, reviewer, notificationModel.TypeNovelVolumeSubmitted, 1)

	// reject stores the reason
	if _, err := env.volumes.ReviewVolume(ctx, reviewer, volume.ID, &dto.ReviewVolumeRequest{
		Status: model.NovelStatusRejected, Reason: "卷号缺失",
	}); err != nil {
		t.Fatalf("reject volume: %v", err)
	}
	assertNotificationCount(t, env.db, creator, notificationModel.TypeNovelVolumeRejected, 1)
	assertContributionRows(t, env.db, novel.ID, 1)

	// approve credits the volume creator
	if _, err := env.volumes.ReviewVolume(ctx, reviewer, volume.ID, &dto.ReviewVolumeRequest{
		Status: model.NovelStatusPublished,
	}); err != nil {
		t.Fatalf("approve volume: %v", err)
	}
	assertContributionRows(t, env.db, novel.ID, 2)
	assertNotificationCount(t, env.db, creator, notificationModel.TypeNovelVolumeApproved, 1)

	if _, err := env.volumes.ReviewVolume(ctx, reviewer, 999999, &dto.ReviewVolumeRequest{Status: 1}); !errors.Is(err, ErrVolumeNotFound) {
		t.Fatalf("expected ErrVolumeNotFound, got %v", err)
	}
}

func TestVolumeReorder(t *testing.T) {
	env := newNovelTestServices(t)
	ctx := context.Background()
	creator := testutil.CreateUser(t, env.db, "volume-reorder-creator")

	novel := createTestNovel(t, env.novels, creator, "reorder-host", model.NovelStatusPublished)
	first, err := env.volumes.CreateVolume(ctx, creator, novel.ID, &dto.CreateVolumeRequest{VolumeNumber: intPtr(1), Title: "一"})
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}
	second, err := env.volumes.CreateVolume(ctx, creator, novel.ID, &dto.CreateVolumeRequest{VolumeNumber: intPtr(2), Title: "二"})
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}
	third, err := env.volumes.CreateVolume(ctx, creator, novel.ID, &dto.CreateVolumeRequest{VolumeNumber: intPtr(3), Title: "三"})
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}

	if err := env.volumes.ReorderVolumes(ctx, novel.ID, &dto.ReorderVolumesRequest{
		IDs: []uint{third.ID, first.ID, second.ID},
	}); err != nil {
		t.Fatalf("reorder volumes: %v", err)
	}
	volumes, total, _, _, err := env.volumes.ListVolumes(ctx, novel.ID, false, 1, 20)
	if err != nil {
		t.Fatalf("list volumes: %v", err)
	}
	if total != 3 || len(volumes) != 3 {
		t.Fatalf("expected 3 volumes, got %d", len(volumes))
	}
	if volumes[0].ID != third.ID || volumes[2].ID != second.ID {
		t.Fatalf("reorder not applied: %+v", volumes)
	}

	// incomplete id sets are rejected
	if err := env.volumes.ReorderVolumes(ctx, novel.ID, &dto.ReorderVolumesRequest{
		IDs: []uint{first.ID, second.ID},
	}); !errors.Is(err, ErrInvalidVolumeReorder) {
		t.Fatalf("expected ErrInvalidVolumeReorder, got %v", err)
	}
}

func TestVolumeDeleteAndPublishVisibility(t *testing.T) {
	env := newNovelTestServices(t)
	ctx := context.Background()
	creator := testutil.CreateUser(t, env.db, "volume-delete-creator")

	novel := createTestNovel(t, env.novels, creator, "volume-visibility", model.NovelStatusPublished)
	volume, err := env.volumes.CreateVolume(ctx, creator, novel.ID, &dto.CreateVolumeRequest{Title: "内部卷"})
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}

	// pending volumes are not exposed publicly
	if _, err := env.volumes.GetVolume(ctx, novel.ID, volume.ID, true); !errors.Is(err, ErrVolumeNotFound) {
		t.Fatalf("expected pending volume hidden, got %v", err)
	}
	if err := env.volumes.DeleteVolume(ctx, novel.ID, volume.ID); err != nil {
		t.Fatalf("delete volume: %v", err)
	}
	if _, err := env.volumes.GetVolume(ctx, novel.ID, volume.ID, false); !errors.Is(err, ErrVolumeNotFound) {
		t.Fatalf("expected deleted volume gone, got %v", err)
	}
	if err := env.volumes.DeleteVolume(ctx, novel.ID, volume.ID); !errors.Is(err, ErrVolumeNotFound) {
		t.Fatalf("expected second delete to fail, got %v", err)
	}
}

func intPtr(value int) *int {
	return &value
}
