package repository

import (
	"context"
	"testing"
	"time"

	"backend/internal/announcement/model"
	"backend/internal/testutil"
)

func TestListActivePlatformScheduleAndOrder(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	repo := NewAnnouncementRepository(db)
	ctx := context.Background()
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	values := []*model.Announcement{
		{Title: "all", Content: "all", Type: model.TypeNormal, DisplayMode: model.DisplayModeNormal, Target: model.TargetAll, Priority: 1, Published: true, Dismissible: true},
		{Title: "desktop", Content: "desktop", Type: model.TypeWarning, DisplayMode: model.DisplayModeBanner, Target: model.TargetDesktop, Priority: 10, Published: true, Dismissible: true, StartsAt: &past, EndsAt: &future},
		{Title: "android", Content: "android", Type: model.TypeUpdate, DisplayMode: model.DisplayModeModal, Target: model.TargetAndroid, Priority: 100, Published: true, Dismissible: true},
		{Title: "expired", Content: "expired", Type: model.TypeNormal, DisplayMode: model.DisplayModeNormal, Target: model.TargetAll, Priority: 100, Published: true, Dismissible: true, EndsAt: &past},
		{Title: "draft", Content: "draft", Type: model.TypeNormal, DisplayMode: model.DisplayModeNormal, Target: model.TargetAll, Priority: 100, Dismissible: true},
	}
	for _, value := range values {
		if err := repo.Create(ctx, value); err != nil {
			t.Fatalf("create announcement %s: %v", value.Title, err)
		}
	}

	items, err := repo.ListActive(ctx, "windows")
	if err != nil {
		t.Fatalf("list active announcements: %v", err)
	}
	if len(items) != 2 || items[0].Title != "desktop" || items[1].Title != "all" {
		t.Fatalf("unexpected announcements: %+v", items)
	}
}
