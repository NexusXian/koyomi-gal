package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	articleModel "backend/internal/article/model"
	bannerModel "backend/internal/banner/model"
	communityModel "backend/internal/community/model"
	galgameModel "backend/internal/galgame/model"
	galgameRepo "backend/internal/galgame/repository"
)

type fakeBanners struct{}

func (fakeBanners) ListPublic(context.Context, int) ([]bannerModel.Banner, error) {
	return nil, nil
}

type fakeArticles struct{}

func (fakeArticles) ListPublished(context.Context, string, int, int) ([]articleModel.Article, int64, error) {
	return nil, 0, nil
}

type fakeGalgames struct {
	sorts []string
}

func (f *fakeGalgames) ListPublished(_ context.Context, options galgameRepo.GalgameListOptions) ([]galgameModel.Galgame, int64, error) {
	f.sorts = append(f.sorts, options.Sort)
	return nil, 0, nil
}

type fakePosts struct {
	sorts []string
}

func (f *fakePosts) ListHome(_ context.Context, sort string, _ int) ([]communityModel.Post, error) {
	f.sorts = append(f.sorts, sort)
	return nil, nil
}

func TestGetReturnsNonNullCollectionsAndRequestedSorts(t *testing.T) {
	galgames := &fakeGalgames{}
	posts := &fakePosts{}
	svc := NewHomeService(fakeBanners{}, fakeArticles{}, galgames, posts, nil)
	data, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("get home: %v", err)
	}
	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal home: %v", err)
	}
	if strings.Contains(string(payload), "null") {
		t.Fatalf("home collections must serialize as arrays: %s", payload)
	}
	if len(galgames.sorts) != 2 || galgames.sorts[0] != "updated" || galgames.sorts[1] != "popular" {
		t.Fatalf("unexpected galgame sorts: %v", galgames.sorts)
	}
	if len(posts.sorts) != 2 || posts.sorts[0] != "latest" || posts.sorts[1] != "popular" {
		t.Fatalf("unexpected post sorts: %v", posts.sorts)
	}
}
