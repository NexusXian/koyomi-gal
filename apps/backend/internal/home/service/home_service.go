package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	articleModel "backend/internal/article/model"
	bannerModel "backend/internal/banner/model"
	communityModel "backend/internal/community/model"
	galgameModel "backend/internal/galgame/model"
	galgameRepo "backend/internal/galgame/repository"
	"backend/internal/home/dto"

	"github.com/redis/go-redis/v9"
)

const CacheKey = "koyomi:home:v2"

const cacheTTL = 3 * time.Minute

type bannerRepository interface {
	ListPublic(context.Context, int) ([]bannerModel.Banner, error)
}

type articleRepository interface {
	ListPublished(context.Context, string, int, int) ([]articleModel.Article, int64, error)
}

type galgamesRepository interface {
	ListPublished(context.Context, galgameRepo.GalgameListOptions) ([]galgameModel.Galgame, int64, error)
}

type postRepository interface {
	ListHome(context.Context, string, int) ([]communityModel.Post, error)
}

type HomeService struct {
	banners  bannerRepository
	articles articleRepository
	galgames galgamesRepository
	posts    postRepository
	cache    *redis.Client
}

func NewHomeService(
	banners bannerRepository,
	articles articleRepository,
	galgames galgamesRepository,
	posts postRepository,
	cache *redis.Client,
) *HomeService {
	return &HomeService{banners: banners, articles: articles, galgames: galgames, posts: posts, cache: cache}
}

func (s *HomeService) Get(ctx context.Context) (*dto.HomeData, error) {
	if cached := s.getCached(ctx); cached != nil {
		return cached, nil
	}
	banners, err := s.banners.ListPublic(ctx, 5)
	if err != nil {
		return nil, fmt.Errorf("list home banners: %w", err)
	}
	articles, _, err := s.articles.ListPublished(ctx, "", 1, 5)
	if err != nil {
		return nil, fmt.Errorf("list home articles: %w", err)
	}
	latestGalgames, _, err := s.galgames.ListPublished(ctx, galgameRepo.GalgameListOptions{Sort: "updated", Page: 1, Limit: 12})
	if err != nil {
		return nil, fmt.Errorf("list latest home galgames: %w", err)
	}
	popularGalgames, _, err := s.galgames.ListPublished(ctx, galgameRepo.GalgameListOptions{Sort: "popular", Page: 1, Limit: 12})
	if err != nil {
		return nil, fmt.Errorf("list popular home galgames: %w", err)
	}
	latestPosts, err := s.posts.ListHome(ctx, "latest", 10)
	if err != nil {
		return nil, fmt.Errorf("list latest home posts: %w", err)
	}
	popularPosts, err := s.posts.ListHome(ctx, "popular", 10)
	if err != nil {
		return nil, fmt.Errorf("list popular home posts: %w", err)
	}

	data := dto.NewHomeData(banners, articles, latestGalgames, popularGalgames, latestPosts, popularPosts)
	s.setCached(ctx, &data)
	return &data, nil
}

func (s *HomeService) getCached(ctx context.Context) *dto.HomeData {
	if s.cache == nil {
		return nil
	}
	value, err := s.cache.Get(ctx, CacheKey).Bytes()
	if err != nil {
		return nil
	}
	var data dto.HomeData
	if json.Unmarshal(value, &data) != nil {
		return nil
	}
	dto.EnsureSlices(&data)
	return &data
}

func (s *HomeService) setCached(ctx context.Context, data *dto.HomeData) {
	if s.cache == nil {
		return
	}
	value, err := json.Marshal(data)
	if err != nil {
		return
	}
	_ = s.cache.Set(ctx, CacheKey, value, cacheTTL).Err()
}
