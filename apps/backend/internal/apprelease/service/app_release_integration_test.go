package service

import (
	"context"
	"testing"
	"time"

	announcementModel "backend/internal/announcement/model"
	announcementRepo "backend/internal/announcement/repository"
	"backend/internal/apprelease/dto"
	"backend/internal/apprelease/repository"
	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	"backend/internal/testutil"
)

func TestManagedAnnouncementFollowsReleaseLifecycle(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	ctx := context.Background()
	actorID := testutil.CreateUser(t, db, "release-publisher")
	rbac := rbacService.NewRBACService(rbacRepo.NewRBACRepository(db))
	if err := rbac.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed permissions: %v", err)
	}
	if err := rbac.AssignRoleByCode(ctx, actorID, rbacService.RoleCodeAdmin); err != nil {
		t.Fatalf("assign admin role: %v", err)
	}
	svc := NewAppReleaseService(repository.NewAppReleaseRepository(db), rbac, nil)
	announcements := announcementRepo.NewAnnouncementRepository(db)
	versionCode := int64(12)
	release, err := svc.Create(ctx, actorID, &dto.AppReleaseRequest{
		Platform: "android", VersionName: "1.0.0", VersionCode: &versionCode,
		Title: "Release", Changelog: "Changes", DownloadURL: "https://example.com/app.apk",
	})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}

	release, err = svc.Publish(ctx, release.ID, actorID, true)
	if err != nil || release.AnnouncementID == nil || !release.AnnouncementManaged {
		t.Fatalf("publish release: release=%+v err=%v", release, err)
	}
	announcementID := *release.AnnouncementID
	changedPublishedAt := time.Now().Add(time.Hour)
	release, err = svc.Update(ctx, actorID, release.ID, &dto.AppReleaseRequest{
		Platform: "android", VersionName: "1.0.0", VersionCode: &versionCode,
		Title: "Updated release", Changelog: "Updated changes", DownloadURL: "https://example.com/app.apk",
		MinimumVersionCode: 10, ForceUpdate: true, PublishedAt: &changedPublishedAt,
	})
	if err != nil {
		t.Fatalf("update published release: %v", err)
	}
	announcement, err := announcements.FindByID(ctx, announcementID)
	if err != nil || announcement == nil || announcement.Title != release.Title ||
		announcement.Content != release.Changelog || announcement.StartsAt == nil ||
		!announcement.StartsAt.Equal(changedPublishedAt) || announcement.Dismissible || !announcement.Published {
		t.Fatalf("generated announcement was not synchronized: announcement=%+v err=%v", announcement, err)
	}

	release, err = svc.Disable(ctx, release.ID)
	if err != nil {
		t.Fatalf("disable release: %v", err)
	}
	assertAnnouncementPublished(t, ctx, announcements, announcementID, false)
	release, err = svc.Publish(ctx, release.ID, actorID, false)
	if err != nil || release.AnnouncementID == nil || *release.AnnouncementID != announcementID {
		t.Fatalf("republish release: release=%+v err=%v", release, err)
	}
	assertAnnouncementPublished(t, ctx, announcements, announcementID, true)
	release, err = svc.Update(ctx, actorID, release.ID, &dto.AppReleaseRequest{
		Platform: "android", VersionName: "1.0.0", VersionCode: &versionCode,
		Title: release.Title, Changelog: release.Changelog, DownloadURL: release.DownloadURL,
		MinimumVersionCode: release.MinimumVersionCode, ForceUpdate: release.ForceUpdate,
		Status: "disabled",
	})
	if err != nil {
		t.Fatalf("move release to disabled via update: %v", err)
	}
	assertAnnouncementPublished(t, ctx, announcements, announcementID, false)
	release, err = svc.Publish(ctx, release.ID, actorID, false)
	if err != nil {
		t.Fatalf("republish updated release: %v", err)
	}
	assertAnnouncementPublished(t, ctx, announcements, announcementID, true)

	release, err = svc.Update(ctx, actorID, release.ID, &dto.AppReleaseRequest{
		Platform: "android", VersionName: "1.0.0", VersionCode: &versionCode,
		Title: release.Title, Changelog: release.Changelog, DownloadURL: release.DownloadURL,
		MinimumVersionCode: release.MinimumVersionCode, ForceUpdate: release.ForceUpdate,
		Status: "draft",
	})
	if err != nil {
		t.Fatalf("move release to draft: %v", err)
	}
	assertAnnouncementPublished(t, ctx, announcements, announcementID, false)
	if _, err := svc.Publish(ctx, release.ID, actorID, false); err != nil {
		t.Fatalf("publish release before deletion: %v", err)
	}
	if err := svc.Delete(ctx, release.ID); err != nil {
		t.Fatalf("delete release: %v", err)
	}
	assertAnnouncementPublished(t, ctx, announcements, announcementID, false)

	var count int64
	if err := db.Table("announcements").Where("id = ?", announcementID).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("expected one linked announcement, count=%d err=%v", count, err)
	}
}

func TestArbitraryLinkedAnnouncementIsNotManaged(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	ctx := context.Background()
	actorID := testutil.CreateUser(t, db, "manual-link-publisher")
	announcements := announcementRepo.NewAnnouncementRepository(db)
	announcement := &announcementModel.Announcement{
		Title: "Manual", Content: "Manual content", Type: announcementModel.TypeNormal,
		DisplayMode: announcementModel.DisplayModeNormal, Target: announcementModel.TargetAll,
		Dismissible: true, Published: true, CreatedBy: &actorID,
	}
	if err := announcements.Create(ctx, announcement); err != nil {
		t.Fatalf("create arbitrary announcement: %v", err)
	}
	replacement := &announcementModel.Announcement{
		Title: "Replacement", Content: "Replacement content", Type: announcementModel.TypeNormal,
		DisplayMode: announcementModel.DisplayModeNormal, Target: announcementModel.TargetAll,
		Dismissible: true, Published: true, CreatedBy: &actorID,
	}
	if err := announcements.Create(ctx, replacement); err != nil {
		t.Fatalf("create replacement announcement: %v", err)
	}
	svc := NewAppReleaseService(repository.NewAppReleaseRepository(db), nil, nil)
	versionCode := int64(20)
	release, err := svc.Create(ctx, actorID, &dto.AppReleaseRequest{
		Platform: "android", VersionName: "2.0.0", VersionCode: &versionCode,
		Title: "Release", Changelog: "Changes", DownloadURL: "https://example.com/app.apk",
		AnnouncementID: &announcement.ID,
	})
	if err != nil || release.AnnouncementManaged {
		t.Fatalf("create manually linked release: release=%+v err=%v", release, err)
	}
	release, err = svc.Update(ctx, actorID, release.ID, &dto.AppReleaseRequest{
		Platform: "android", VersionName: "2.0.0", VersionCode: &versionCode,
		Title: "Release", Changelog: "Changes", DownloadURL: "https://example.com/app.apk",
		AnnouncementID: &replacement.ID,
	})
	if err != nil || release.AnnouncementManaged {
		t.Fatalf("replace arbitrary announcement link: release=%+v err=%v", release, err)
	}
	assertAnnouncementPublished(t, ctx, announcements, announcement.ID, true)
	if _, err := svc.Publish(ctx, release.ID, actorID, false); err != nil {
		t.Fatalf("publish manually linked release: %v", err)
	}
	if _, err := svc.Disable(ctx, release.ID); err != nil {
		t.Fatalf("disable manually linked release: %v", err)
	}
	if err := svc.Delete(ctx, release.ID); err != nil {
		t.Fatalf("delete manually linked release: %v", err)
	}
	assertAnnouncementPublished(t, ctx, announcements, replacement.ID, true)
	stored, err := announcements.FindByID(ctx, announcement.ID)
	if err != nil || stored == nil || !stored.Published || stored.Title != "Manual" || stored.Content != "Manual content" {
		t.Fatalf("arbitrary announcement was changed: announcement=%+v err=%v", stored, err)
	}
}

func assertAnnouncementPublished(
	t *testing.T,
	ctx context.Context,
	repository *announcementRepo.AnnouncementRepository,
	id uint,
	published bool,
) {
	t.Helper()
	announcement, err := repository.FindByID(ctx, id)
	if err != nil || announcement == nil || announcement.Published != published {
		t.Fatalf("expected announcement published=%v, announcement=%+v err=%v", published, announcement, err)
	}
}
