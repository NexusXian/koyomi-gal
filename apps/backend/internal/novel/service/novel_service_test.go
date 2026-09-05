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

func TestNovelCreateValidation(t *testing.T) {
	env := newNovelTestServices(t)
	ctx := context.Background()
	creator := testutil.CreateUser(t, env.db, "novel-create-user")

	if _, err := env.novels.CreateNovel(ctx, creator, &dto.CreateNovelRequest{
		Title: "", Slug: "empty-title", AgeRating: 0, Status: 0,
	}); !errors.Is(err, ErrInvalidNovelInput) {
		t.Fatalf("expected ErrInvalidNovelInput for empty title, got %v", err)
	}
	if _, err := env.novels.CreateNovel(ctx, creator, &dto.CreateNovelRequest{
		Title: "x", Slug: "bad-release", ReleaseStatus: "wrong", AgeRating: 0, Status: 0,
	}); !errors.Is(err, ErrInvalidReleaseState) {
		t.Fatalf("expected ErrInvalidReleaseState, got %v", err)
	}
	if _, err := env.novels.CreateNovel(ctx, creator, &dto.CreateNovelRequest{
		Title: "x", Slug: "bad-url", AgeRating: 0, Status: 0, CoverURL: "ftp://not-allowed",
	}); !errors.Is(err, ErrInvalidNovelURL) {
		t.Fatalf("expected ErrInvalidNovelURL, got %v", err)
	}
	if _, err := env.novels.CreateNovel(ctx, creator, &dto.CreateNovelRequest{
		Title: "x", Slug: "bad-age", AgeRating: 99, Status: 0,
	}); !errors.Is(err, ErrInvalidAgeRating) {
		t.Fatalf("expected ErrInvalidAgeRating, got %v", err)
	}
	if _, err := env.novels.CreateNovel(ctx, creator, &dto.CreateNovelRequest{
		Title: "x", Slug: "bad-status", AgeRating: 0, Status: 3,
	}); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus for direct hidden creation, got %v", err)
	}
}

func TestNovelSlugUniqueness(t *testing.T) {
	env := newNovelTestServices(t)
	ctx := context.Background()
	creator := testutil.CreateUser(t, env.db, "novel-slug-user")

	createTestNovel(t, env.novels, creator, "duplicate-slug", model.NovelStatusPublished)
	if _, err := env.novels.CreateNovel(ctx, creator, &dto.CreateNovelRequest{
		Title: "另一个标题", Slug: "duplicate-slug", AgeRating: 0, Status: 0,
	}); !errors.Is(err, ErrNovelSlugExists) {
		t.Fatalf("expected ErrNovelSlugExists, got %v", err)
	}
}

func TestNovelCreateAndReviewLifecycle(t *testing.T) {
	env := newNovelTestServices(t)
	ctx := context.Background()
	creator := testutil.CreateUser(t, env.db, "novel-life-creator")
	reviewer := testutil.CreateUser(t, env.db, "novel-life-reviewer")
	env.assignReviewer(t, reviewer)

	// pending creation notifies reviewers; no contribution until approval
	novel := createTestNovel(t, env.novels, creator, "lifecycle-pending", model.NovelStatusPending)
	assertContributionRows(t, env.db, novel.ID, 0)
	assertNotificationCount(t, env.db, reviewer, notificationModel.TypeNovelSubmitted, 1)

	// rejection stores reason and notifies the creator
	if _, err := env.novels.ReviewNovel(ctx, reviewer, novel.ID, &dto.ReviewNovelRequest{
		Status: model.NovelStatusRejected, Reason: "资料不完整",
	}); err != nil {
		t.Fatalf("reject novel: %v", err)
	}
	assertNotificationCount(t, env.db, creator, notificationModel.TypeNovelRejected, 1)
	assertContributionRows(t, env.db, novel.ID, 0)

	rejected, err := env.novels.GetNovel(ctx, novel.ID)
	if err != nil {
		t.Fatalf("get rejected novel: %v", err)
	}
	if rejected.Status != model.NovelStatusRejected || rejected.RejectReason != "资料不完整" {
		t.Fatalf("reject reason not persisted: %+v", rejected)
	}

	// approve grants the creator a contribution and notifies
	if _, err := env.novels.ReviewNovel(ctx, reviewer, novel.ID, &dto.ReviewNovelRequest{
		Status: model.NovelStatusPublished,
	}); err != nil {
		t.Fatalf("approve novel: %v", err)
	}
	assertContributionRows(t, env.db, novel.ID, 1)
	assertNotificationCount(t, env.db, creator, notificationModel.TypeNovelApproved, 1)

	// idempotent re-review changes nothing
	if _, err := env.novels.ReviewNovel(ctx, reviewer, novel.ID, &dto.ReviewNovelRequest{
		Status: model.NovelStatusPublished,
	}); err != nil {
		t.Fatalf("repeat approval: %v", err)
	}
	assertContributionRows(t, env.db, novel.ID, 1)
	assertNotificationCount(t, env.db, creator, notificationModel.TypeNovelApproved, 1)

	// unknown novel and invalid transitions fail cleanly
	if _, err := env.novels.ReviewNovel(ctx, reviewer, 999999, &dto.ReviewNovelRequest{Status: 1}); !errors.Is(err, ErrNovelNotFound) {
		t.Fatalf("expected ErrNovelNotFound, got %v", err)
	}
	if _, err := env.novels.ReviewNovel(ctx, reviewer, novel.ID, &dto.ReviewNovelRequest{Status: 3}); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}

	// after approval the novel is publicly visible with cleared reject reason
	published, err := env.novels.GetPublishedNovel(ctx, novel.ID)
	if err != nil {
		t.Fatalf("expected approved novel to be public: %v", err)
	}
	if published.RejectReason != "" {
		t.Fatalf("expected reject reason cleared after approval, got %q", published.RejectReason)
	}
}

func TestNovelDirectPublishCreditsCreator(t *testing.T) {
	env := newNovelTestServices(t)
	ctx := context.Background()
	creator := testutil.CreateUser(t, env.db, "novel-direct-creator")

	novel := createTestNovel(t, env.novels, creator, "direct-published", model.NovelStatusPublished)
	assertContributionRows(t, env.db, novel.ID, 1)
	if _, err := env.novels.GetPublishedNovel(ctx, novel.ID); err != nil {
		t.Fatalf("published novel should be visible: %v", err)
	}
}

func TestNovelUpdateAndContributions(t *testing.T) {
	env := newNovelTestServices(t)
	ctx := context.Background()
	creator := testutil.CreateUser(t, env.db, "novel-update-creator")
	editor := testutil.CreateUser(t, env.db, "novel-update-editor")

	novel := createTestNovel(t, env.novels, creator, "update-me", model.NovelStatusPublished)
	assertContributionRows(t, env.db, novel.ID, 1)

	// editing metadata of a published novel records an edit contribution
	updated, err := env.novels.UpdateNovel(ctx, editor, novel.ID, &dto.UpdateNovelRequest{
		Title:            novel.Title + "改",
		OriginalTitle:    novel.OriginalTitle,
		Slug:             novel.Slug,
		Summary:          novel.Summary,
		Author:           novel.Author,
		Illustrator:      novel.Illustrator,
		Publisher:        novel.Publisher,
		Label:            novel.Label,
		Language:         novel.Language,
		Region:           novel.Region,
		ReleaseStatus:    novel.ReleaseStatus,
		AgeRating:        &novel.AgeRating,
		IsCoverSensitive: &novel.IsCoverSensitive,
		Status:           &novel.Status,
		TagIDs:           []uint{},
	})
	if err != nil {
		t.Fatalf("update novel: %v", err)
	}
	assertContributionRows(t, env.db, novel.ID, 2)
	if updated.Title != novel.Title+"改" {
		t.Fatalf("updated title mismatch: %s", updated.Title)
	}

	// updating a nonexistent novel and a duplicated slug fail cleanly
	if _, err := env.novels.UpdateNovel(ctx, editor, 999999, &dto.UpdateNovelRequest{
		Title: "x", Slug: "missing-novel", ReleaseStatus: "ongoing",
		AgeRating: &novel.AgeRating, IsCoverSensitive: &novel.IsCoverSensitive,
		Status: &novel.Status,
	}); !errors.Is(err, ErrNovelNotFound) {
		t.Fatalf("expected ErrNovelNotFound, got %v", err)
	}
	createTestNovel(t, env.novels, creator, "slug-taken", model.NovelStatusPublished)
	if _, err := env.novels.UpdateNovel(ctx, editor, novel.ID, &dto.UpdateNovelRequest{
		Title: "x", Slug: "slug-taken", ReleaseStatus: "ongoing",
		AgeRating: &novel.AgeRating, IsCoverSensitive: &novel.IsCoverSensitive,
		Status: &novel.Status,
	}); !errors.Is(err, ErrNovelSlugExists) {
		t.Fatalf("expected ErrNovelSlugExists, got %v", err)
	}
}

func TestNovelDeleteCleansAssociations(t *testing.T) {
	env := newNovelTestServices(t)
	ctx := context.Background()
	creator := testutil.CreateUser(t, env.db, "novel-delete-creator")

	novel := createTestNovel(t, env.novels, creator, "delete-me", model.NovelStatusPublished)
	volume, err := env.volumes.CreateVolume(ctx, creator, novel.ID, &dto.CreateVolumeRequest{
		Title:  "Vol 1",
		Status: &publishedStatus,
	})
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}
	if err := env.novels.DeleteNovel(ctx, novel.ID); err != nil {
		t.Fatalf("delete novel: %v", err)
	}
	if _, err := env.novels.GetNovel(ctx, novel.ID); !errors.Is(err, ErrNovelNotFound) {
		t.Fatalf("expected deleted novel to disappear, got %v", err)
	}
	var count int64
	if err := env.db.Table("novel_volumes").Where("id = ? AND deleted_at IS NULL", volume.ID).Count(&count).Error; err != nil {
		t.Fatalf("count volumes: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected volume to be soft deleted, got %d rows", count)
	}
	if err := env.novels.DeleteNovel(ctx, 999999); !errors.Is(err, ErrNovelNotFound) {
		t.Fatalf("expected ErrNovelNotFound, got %v", err)
	}
}
