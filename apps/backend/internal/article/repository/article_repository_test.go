package repository

import (
	"context"
	"testing"
	"time"

	"backend/internal/article/model"
	"backend/internal/testutil"
)

func TestPublishedVisibilityOrderAndViewIncrement(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	repo := NewArticleRepository(db)
	ctx := context.Background()
	now := time.Now()
	old := now.Add(-2 * time.Hour)
	recent := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	values := []*model.Article{
		{Title: "recent", Content: "content", Type: model.TypeNews, IsPublished: true, PublishedAt: &recent},
		{Title: "pinned", Content: "content", Type: model.TypeAnnouncement, IsPinned: true, IsPublished: true, PublishedAt: &old},
		{Title: "future", Content: "content", Type: model.TypeNews, IsPublished: true, PublishedAt: &future},
		{Title: "draft", Content: "content", Type: model.TypeNews},
	}
	for _, value := range values {
		if err := repo.Create(ctx, value); err != nil {
			t.Fatalf("create article %s: %v", value.Title, err)
		}
	}

	articles, total, err := repo.ListPublished(ctx, "", 1, 20)
	if err != nil {
		t.Fatalf("list published articles: %v", err)
	}
	if total != 2 || len(articles) != 2 || articles[0].Title != "pinned" || articles[1].Title != "recent" {
		t.Fatalf("unexpected published articles: total=%d values=%+v", total, articles)
	}

	detail, err := repo.FindPublishedAndIncrementView(ctx, values[0].ID)
	if err != nil || detail == nil || detail.ViewCount != 1 {
		t.Fatalf("increment published view: detail=%+v err=%v", detail, err)
	}
	hidden, err := repo.FindPublishedAndIncrementView(ctx, values[2].ID)
	if err != nil || hidden != nil {
		t.Fatalf("future article should be hidden: detail=%+v err=%v", hidden, err)
	}
}
