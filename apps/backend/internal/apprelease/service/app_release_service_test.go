package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/internal/apprelease/dto"
	"backend/internal/apprelease/model"
	githubInfrastructure "backend/internal/infrastructures/github"
)

func TestValidateAppRelease(t *testing.T) {
	checksum := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	upperChecksum := "0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF"
	negative := int64(-1)
	now := time.Now()
	valid := model.AppRelease{
		Platform: model.PlatformAndroid, VersionName: "1.0.0", VersionCode: 1,
		Title: "Release", Changelog: "Changes", DownloadURL: "https://example.com/app.apk",
		Status: model.StatusDraft,
	}
	tests := []struct {
		name  string
		value model.AppRelease
		err   error
	}{
		{name: "valid", value: valid},
		{name: "valid checksum", value: model.AppRelease{Platform: model.PlatformAndroid, VersionName: "1.0.0", VersionCode: 1, Title: "Release", Changelog: "Changes", DownloadURL: "http://example.com/app.apk", FileSHA256: &checksum, Status: model.StatusPublished, PublishedAt: &now}},
		{name: "relative url", value: model.AppRelease{Platform: model.PlatformAndroid, VersionName: "1.0.0", VersionCode: 1, Title: "Release", Changelog: "Changes", DownloadURL: "/app.apk", Status: model.StatusDraft}, err: ErrInvalidDownloadURL},
		{name: "uppercase checksum", value: model.AppRelease{Platform: model.PlatformAndroid, VersionName: "1.0.0", VersionCode: 1, Title: "Release", Changelog: "Changes", DownloadURL: "https://example.com/app.apk", FileSHA256: &upperChecksum, Status: model.StatusDraft}, err: ErrInvalidChecksum},
		{name: "negative file size", value: model.AppRelease{Platform: model.PlatformAndroid, VersionName: "1.0.0", VersionCode: 1, Title: "Release", Changelog: "Changes", DownloadURL: "https://example.com/app.apk", FileSize: &negative, Status: model.StatusDraft}, err: ErrInvalidAppRelease},
		{name: "minimum newer than release", value: model.AppRelease{Platform: model.PlatformAndroid, VersionName: "1.0.0", VersionCode: 1, Title: "Release", Changelog: "Changes", DownloadURL: "https://example.com/app.apk", MinimumVersionCode: 2, Status: model.StatusDraft}, err: ErrInvalidAppRelease},
		{name: "published without time", value: model.AppRelease{Platform: model.PlatformAndroid, VersionName: "1.0.0", VersionCode: 1, Title: "Release", Changelog: "Changes", DownloadURL: "https://example.com/app.apk", Status: model.StatusPublished}, err: ErrInvalidAppRelease},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateAppRelease(&test.value); !errors.Is(err, test.err) {
				t.Fatalf("expected %v, got %v", test.err, err)
			}
		})
	}
}

func TestReleaseFromRequestPreservesPublicationState(t *testing.T) {
	versionCode := int64(2)
	publishedAt := time.Now()
	announcementID := uint(3)
	current := &model.AppRelease{
		Status: model.StatusPublished, PublishedAt: &publishedAt, AnnouncementID: &announcementID,
	}
	value := releaseFromRequest(&dto.AppReleaseRequest{
		Platform: model.PlatformAndroid, VersionName: "1.0.1", VersionCode: &versionCode,
		Title: "Release", Changelog: "Changes", DownloadURL: "https://example.com/app.apk",
	}, current)
	if value.Status != model.StatusPublished || value.PublishedAt != current.PublishedAt ||
		value.AnnouncementID != current.AnnouncementID {
		t.Fatalf("publication state was not preserved: %+v", value)
	}
}

func TestListGitHubReleasesRequiresConfiguration(t *testing.T) {
	svc := NewAppReleaseService(nil, nil, nil)
	if _, err := svc.ListGitHubReleases(context.Background(), 1, 20); !errors.Is(err, ErrGitHubNotConfigured) {
		t.Fatalf("expected ErrGitHubNotConfigured, got %v", err)
	}
}

func TestListGitHubReleasesMapsAPKAssets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{
			"tag_name": "mobile-v1.2.0",
			"name": "Koyomi Gal Android 1.2.0 (build 42)",
			"body": "release notes",
			"draft": false,
			"prerelease": true,
			"created_at": "2026-09-01T10:00:00Z",
			"published_at": "2026-09-01T10:05:00Z",
			"assets": [{
				"name": "koyomi-gal-1.2.0-42-abcdef0.apk",
				"size": 12345678,
				"digest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				"browser_download_url": "https://github.com/owner/repo/releases/download/mobile-v1.2.0/koyomi-gal-1.2.0-42-abcdef0.apk",
				"updated_at": "2026-09-01T10:05:00Z"
			}]
		}, {
			"tag_name": "no-assets",
			"name": "No assets",
			"draft": false,
			"assets": []
		}]`))
	}))
	t.Cleanup(server.Close)

	svc := NewAppReleaseService(nil, nil,
		githubInfrastructure.NewClient(server.URL, "owner/repo", "", server.Client()))
	items, err := svc.ListGitHubReleases(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("list github releases: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	first := items[0]
	if first.Tag != "mobile-v1.2.0" || !first.Prerelease || first.Body != "release notes" {
		t.Fatalf("unexpected release data: %+v", first)
	}
	if first.APK == nil {
		t.Fatal("expected apk asset on first release")
	}
	if first.APK.DownloadURL == "" || first.APK.Size != 12345678 || first.APK.SHA256 == nil ||
		*first.APK.SHA256 != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected apk asset: %+v", first.APK)
	}
	if items[1].APK != nil {
		t.Fatalf("expected no apk asset on second release, got %+v", items[1].APK)
	}
}

func TestListGitHubReleasesWrapsUpstreamFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(server.Close)

	svc := NewAppReleaseService(nil, nil,
		githubInfrastructure.NewClient(server.URL, "owner/repo", "", server.Client()))
	if _, err := svc.ListGitHubReleases(context.Background(), 1, 20); !errors.Is(err, ErrGitHubUnavailable) {
		t.Fatalf("expected ErrGitHubUnavailable, got %v", err)
	}
}
