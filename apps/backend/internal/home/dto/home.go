package dto

import (
	"time"

	articleModel "backend/internal/article/model"
	bannerModel "backend/internal/banner/model"
	communityModel "backend/internal/community/model"
	galgameModel "backend/internal/galgame/model"
)

type Banner struct {
	ID        uint   `json:"id" example:"1"`
	Title     string `json:"title" example:"夏日新作专题"`
	Subtitle  string `json:"subtitle" example:"发现本月值得关注的新作"`
	ImageURL  string `json:"image_url" example:"https://example.com/banner.jpg"`
	LinkType  string `json:"link_type" example:"galgame"`
	LinkValue string `json:"link_value" example:"1"`
}

type Announcement struct {
	ID          uint       `json:"id" example:"1"`
	Title       string     `json:"title" example:"站点更新公告"`
	Summary     string     `json:"summary" example:"本次更新内容概览"`
	CoverURL    string     `json:"cover_url" example:"https://example.com/cover.jpg"`
	Type        string     `json:"type" example:"announcement"`
	IsPinned    bool       `json:"is_pinned" example:"true"`
	PublishedAt *time.Time `json:"published_at"`
}

type Developer struct {
	ID   uint   `json:"id" example:"1"`
	Name string `json:"name" example:"YUZUSOFT"`
}

type Galgame struct {
	ID             uint       `json:"id" example:"1"`
	Title          string     `json:"title" example:"千恋＊万花"`
	CoverURL       string     `json:"cover_url" example:"https://example.com/cover.jpg"`
	CoverSensitive bool       `json:"cover_sensitive" example:"false"`
	Developer      *Developer `json:"developer"`
	RatingAverage  float64    `json:"rating_average" example:"8.72"`
	FavoriteCount  int64      `json:"favorite_count" example:"300"`
	ReleaseDate    *time.Time `json:"release_date"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Author struct {
	ID       uint   `json:"id" example:"1"`
	Username string `json:"username" example:"koyomi"`
	Avatar   string `json:"avatar" example:"https://example.com/avatar.jpg"`
}

type PostGalgame struct {
	ID       uint   `json:"id" example:"1"`
	Title    string `json:"title" example:"千恋＊万花"`
	CoverURL string `json:"cover_url" example:"https://example.com/cover.jpg"`
}

type Post struct {
	ID            uint         `json:"id" example:"1"`
	Title         string       `json:"title" example:"千恋＊万花通关感想"`
	Author        *Author      `json:"author"`
	Galgame       *PostGalgame `json:"galgame"`
	LikeCount     int64        `json:"like_count" example:"10"`
	CommentCount  int64        `json:"comment_count" example:"3"`
	FavoriteCount int64        `json:"favorite_count" example:"2"`
	CreatedAt     time.Time    `json:"created_at"`
}

type HomeData struct {
	Banners         []Banner       `json:"banners"`
	Announcements   []Announcement `json:"announcements"`
	LatestGalgames  []Galgame      `json:"latest_galgames"`
	PopularGalgames []Galgame      `json:"popular_galgames"`
	LatestPosts     []Post         `json:"latest_posts"`
	PopularPosts    []Post         `json:"popular_posts"`
}

type HomeResponse struct {
	Code int      `json:"code" example:"0"`
	Data HomeData `json:"data"`
	Msg  string   `json:"msg" example:"success"`
}

func NewHomeData(
	banners []bannerModel.Banner,
	articles []articleModel.Article,
	latestGalgames, popularGalgames []galgameModel.Galgame,
	latestPosts, popularPosts []communityModel.Post,
) HomeData {
	return HomeData{
		Banners: newBanners(banners), Announcements: newAnnouncements(articles),
		LatestGalgames: newGalgames(latestGalgames), PopularGalgames: newGalgames(popularGalgames),
		LatestPosts: newPosts(latestPosts), PopularPosts: newPosts(popularPosts),
	}
}

func EnsureSlices(data *HomeData) {
	if data.Banners == nil {
		data.Banners = []Banner{}
	}
	if data.Announcements == nil {
		data.Announcements = []Announcement{}
	}
	if data.LatestGalgames == nil {
		data.LatestGalgames = []Galgame{}
	}
	if data.PopularGalgames == nil {
		data.PopularGalgames = []Galgame{}
	}
	if data.LatestPosts == nil {
		data.LatestPosts = []Post{}
	}
	if data.PopularPosts == nil {
		data.PopularPosts = []Post{}
	}
}

func newBanners(values []bannerModel.Banner) []Banner {
	items := make([]Banner, 0, len(values))
	for _, value := range values {
		items = append(items, Banner{ID: value.ID, Title: value.Title, Subtitle: value.Subtitle, ImageURL: value.ImageURL,
			LinkType: value.LinkType, LinkValue: value.LinkValue})
	}
	return items
}

func newAnnouncements(values []articleModel.Article) []Announcement {
	items := make([]Announcement, 0, len(values))
	for _, value := range values {
		items = append(items, Announcement{ID: value.ID, Title: value.Title, Summary: value.Summary,
			CoverURL: value.CoverURL, Type: value.Type, IsPinned: value.IsPinned, PublishedAt: value.PublishedAt})
	}
	return items
}

func newGalgames(values []galgameModel.Galgame) []Galgame {
	items := make([]Galgame, 0, len(values))
	for _, value := range values {
		var developer *Developer
		if value.Developer != nil {
			developer = &Developer{ID: value.Developer.ID, Name: value.Developer.Name}
		}
		items = append(items, Galgame{ID: value.ID, Title: value.Title, CoverURL: value.CoverURL,
			CoverSensitive: value.CoverSensitive,
			Developer: developer, RatingAverage: value.RatingAverage, FavoriteCount: value.FavoriteCount,
			ReleaseDate: value.ReleaseDate, UpdatedAt: value.UpdatedAt})
	}
	return items
}

func newPosts(values []communityModel.Post) []Post {
	items := make([]Post, 0, len(values))
	for _, value := range values {
		var author *Author
		if value.AuthorID != nil {
			author = &Author{ID: *value.AuthorID, Username: value.AuthorName, Avatar: value.AuthorAvatar}
		}
		var galgame *PostGalgame
		if value.GalgameID != nil {
			galgame = &PostGalgame{ID: *value.GalgameID, Title: value.GalgameTitle, CoverURL: value.GalgameCoverURL}
		}
		items = append(items, Post{ID: value.ID, Title: value.Title, Author: author, Galgame: galgame,
			LikeCount: value.LikeCount, CommentCount: value.CommentCount,
			FavoriteCount: value.FavoriteCount, CreatedAt: value.CreatedAt})
	}
	return items
}
