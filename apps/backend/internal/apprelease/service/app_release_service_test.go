package service

import (
	"errors"
	"testing"
	"time"

	"backend/internal/apprelease/dto"
	"backend/internal/apprelease/model"
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
