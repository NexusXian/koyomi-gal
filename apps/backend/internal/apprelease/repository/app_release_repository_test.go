package repository

import (
	"context"
	"testing"
	"time"

	"backend/internal/apprelease/model"
	"backend/internal/testutil"
)

func TestFindLatestPublished(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	repo := NewAppReleaseRepository(db)
	ctx := context.Background()
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	values := []*model.AppRelease{
		{Platform: model.PlatformAndroid, VersionName: "1.0.1", VersionCode: 11, Title: "11", Changelog: "11", DownloadURL: "https://example.com/11.apk", Status: model.StatusPublished, PublishedAt: &past},
		{Platform: model.PlatformAndroid, VersionName: "1.0.2", VersionCode: 12, Title: "12", Changelog: "12", DownloadURL: "https://example.com/12.apk", Status: model.StatusPublished, PublishedAt: &past},
		{Platform: model.PlatformAndroid, VersionName: "1.0.3", VersionCode: 13, Title: "future", Changelog: "future", DownloadURL: "https://example.com/13.apk", Status: model.StatusPublished, PublishedAt: &future},
		{Platform: model.PlatformAndroid, VersionName: "1.0.4", VersionCode: 14, Title: "disabled", Changelog: "disabled", DownloadURL: "https://example.com/14.apk", Status: model.StatusDisabled},
	}
	for _, value := range values {
		if err := repo.Create(ctx, value); err != nil {
			t.Fatalf("create release %s: %v", value.VersionName, err)
		}
	}

	latest, err := repo.FindLatestPublished(ctx, model.PlatformAndroid, 10)
	if err != nil || latest == nil || latest.VersionCode != 12 {
		t.Fatalf("expected version 12, release=%+v err=%v", latest, err)
	}
	latest, err = repo.FindLatestPublished(ctx, model.PlatformAndroid, 12)
	if err != nil || latest != nil {
		t.Fatalf("expected no update, release=%+v err=%v", latest, err)
	}
}

func TestCreateRejectsMinimumVersionNewerThanRelease(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	repo := NewAppReleaseRepository(db)
	err := repo.Create(context.Background(), &model.AppRelease{
		Platform: model.PlatformAndroid, VersionName: "1.0.0", VersionCode: 1,
		Title: "Release", Changelog: "Changes", DownloadURL: "https://example.com/app.apk",
		MinimumVersionCode: 2, Status: model.StatusDraft,
	})
	if err == nil {
		t.Fatal("expected database check constraint to reject minimum_version_code > version_code")
	}
}
