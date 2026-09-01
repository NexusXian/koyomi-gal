package repository

import (
	"context"
	"testing"
	"time"

	"backend/internal/banner/model"
	"backend/internal/testutil"
)

func TestListPublicVisibilityAndOrder(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	repo := NewBannerRepository(db)
	ctx := context.Background()
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	values := []*model.Banner{
		{Title: "low", ImageURL: "/low", LinkType: model.LinkTypeNone, SortOrder: 1, IsActive: true},
		{Title: "high", ImageURL: "/high", LinkType: model.LinkTypeNone, SortOrder: 10, IsActive: true, StartAt: &past, EndAt: &future},
		{Title: "inactive", ImageURL: "/inactive", LinkType: model.LinkTypeNone, SortOrder: 100, IsActive: false},
		{Title: "future", ImageURL: "/future", LinkType: model.LinkTypeNone, SortOrder: 100, IsActive: true, StartAt: &future},
		{Title: "expired", ImageURL: "/expired", LinkType: model.LinkTypeNone, SortOrder: 100, IsActive: true, EndAt: &past},
	}
	for _, value := range values {
		if err := repo.Create(ctx, value); err != nil {
			t.Fatalf("create banner %s: %v", value.Title, err)
		}
	}

	banners, err := repo.ListPublic(ctx, 0)
	if err != nil {
		t.Fatalf("list public banners: %v", err)
	}
	if len(banners) != 2 || banners[0].Title != "high" || banners[1].Title != "low" {
		t.Fatalf("unexpected public banners: %+v", banners)
	}
}
