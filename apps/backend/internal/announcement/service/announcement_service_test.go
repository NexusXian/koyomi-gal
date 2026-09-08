package service

import (
	"errors"
	"testing"
	"time"

	"backend/internal/announcement/model"
)

func TestValidateAnnouncement(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)
	valid := model.Announcement{
		Title: "Notice", Content: "Content", Type: model.TypeNormal,
		DisplayMode: model.DisplayModeNormal, Target: model.TargetAll,
	}
	tests := []struct {
		name  string
		value model.Announcement
		err   error
	}{
		{name: "valid", value: valid},
		{name: "equal schedule", value: model.Announcement{Title: "Notice", Content: "Content", Type: model.TypeNormal, DisplayMode: model.DisplayModeNormal, Target: model.TargetAll, StartsAt: &now, EndsAt: &now}},
		{name: "empty title", value: model.Announcement{Content: "Content", Type: model.TypeNormal, DisplayMode: model.DisplayModeNormal, Target: model.TargetAll}, err: ErrInvalidAnnouncement},
		{name: "invalid type", value: model.Announcement{Title: "Notice", Content: "Content", Type: "other", DisplayMode: model.DisplayModeNormal, Target: model.TargetAll}, err: ErrInvalidAnnouncement},
		{name: "invalid schedule", value: model.Announcement{Title: "Notice", Content: "Content", Type: model.TypeNormal, DisplayMode: model.DisplayModeNormal, Target: model.TargetAll, StartsAt: &later, EndsAt: &now}, err: ErrInvalidSchedule},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateAnnouncement(&test.value); !errors.Is(err, test.err) {
				t.Fatalf("expected %v, got %v", test.err, err)
			}
		})
	}
}

func TestValidQueryPlatform(t *testing.T) {
	for _, platform := range []string{"web", "android", "ios", "windows", "macos", "linux", "desktop"} {
		if !validQueryPlatform(platform) {
			t.Fatalf("expected %s to be valid", platform)
		}
	}
	if validQueryPlatform("all") {
		t.Fatal("all must not be accepted as a query platform")
	}
}
