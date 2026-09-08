package dto

import (
	"time"

	"backend/internal/apprelease/model"
)

type LatestReleaseQuery struct {
	Platform    string `form:"platform" binding:"required,oneof=android ios windows macos linux"`
	VersionCode *int64 `form:"versionCode" binding:"required,gte=0"`
	VersionName string `form:"versionName"`
}

type AdminReleaseQuery struct {
	Page  int `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

type GitHubReleaseAssetData struct {
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	SHA256      *string   `json:"sha256"`
	DownloadURL string    `json:"downloadUrl"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type GitHubReleaseData struct {
	Tag         string                  `json:"tag"`
	Name        string                  `json:"name"`
	Body        string                  `json:"body"`
	Prerelease  bool                    `json:"prerelease"`
	CreatedAt   time.Time               `json:"createdAt"`
	PublishedAt time.Time               `json:"publishedAt"`
	APK         *GitHubReleaseAssetData `json:"apk"`
}

type GitHubReleaseListData struct {
	Items []GitHubReleaseData `json:"items"`
}

type GitHubReleaseListResponse struct {
	Code int                   `json:"code"`
	Data GitHubReleaseListData `json:"data"`
	Msg  string                `json:"msg"`
}

type AppReleaseRequest struct {
	Platform           string     `json:"platform" binding:"required,oneof=android ios windows macos linux"`
	VersionName        string     `json:"versionName" binding:"required,max=64"`
	VersionCode        *int64     `json:"versionCode" binding:"required,gte=0"`
	Title              string     `json:"title" binding:"required,max=255"`
	Changelog          string     `json:"changelog" binding:"required"`
	DownloadURL        string     `json:"downloadUrl" binding:"required,max=2048"`
	FileSize           *int64     `json:"fileSize" binding:"omitempty,gte=0"`
	SHA256             *string    `json:"sha256"`
	MinimumVersionCode int64      `json:"minimumVersionCode" binding:"gte=0"`
	ForceUpdate        bool       `json:"forceUpdate"`
	Status             string     `json:"status" binding:"omitempty,oneof=draft published disabled"`
	PublishedAt        *time.Time `json:"publishedAt"`
	AnnouncementID     *uint      `json:"announcementId"`
	CreateAnnouncement bool       `json:"createAnnouncement"`
}

type PublishReleaseRequest struct {
	CreateAnnouncement bool `json:"createAnnouncement"`
}

type VersionData struct {
	VersionName string `json:"versionName"`
	VersionCode int64  `json:"versionCode"`
}

type NoUpdateData struct {
	HasUpdate bool `json:"hasUpdate"`
}

type UpdateData struct {
	HasUpdate     bool        `json:"hasUpdate"`
	ForceUpdate   bool        `json:"forceUpdate"`
	LatestVersion VersionData `json:"latestVersion"`
	Title         string      `json:"title"`
	Changelog     string      `json:"changelog"`
	DownloadURL   string      `json:"downloadUrl"`
	FileSize      *int64      `json:"fileSize"`
	SHA256        *string     `json:"sha256"`
	PublishedAt   time.Time   `json:"publishedAt"`
}

type LatestReleaseResponse struct {
	Code int        `json:"code"`
	Data UpdateData `json:"data"`
	Msg  string     `json:"msg"`
}

type AppReleaseData struct {
	ID                 uint       `json:"id"`
	Platform           string     `json:"platform"`
	VersionName        string     `json:"versionName"`
	VersionCode        int64      `json:"versionCode"`
	Title              string     `json:"title"`
	Changelog          string     `json:"changelog"`
	DownloadURL        string     `json:"downloadUrl"`
	FileSize           *int64     `json:"fileSize"`
	SHA256             *string    `json:"sha256"`
	MinimumVersionCode int64      `json:"minimumVersionCode"`
	ForceUpdate        bool       `json:"forceUpdate"`
	Status             string     `json:"status"`
	PublishedAt        *time.Time `json:"publishedAt"`
	AnnouncementID     *uint      `json:"announcementId"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

type AppReleaseListData struct {
	Items []AppReleaseData `json:"items"`
	Total int64            `json:"total"`
	Page  int              `json:"page"`
	Limit int              `json:"limit"`
}

type AdminReleaseListResponse struct {
	Code int                `json:"code"`
	Data AppReleaseListData `json:"data"`
	Msg  string             `json:"msg"`
}

type AppReleaseDataResponse struct {
	Code int            `json:"code"`
	Data AppReleaseData `json:"data"`
	Msg  string         `json:"msg"`
}

func NewUpdateData(value *model.AppRelease, clientVersionCode int64) UpdateData {
	return UpdateData{
		HasUpdate:     true,
		ForceUpdate:   value.ForceUpdate || clientVersionCode < value.MinimumVersionCode,
		LatestVersion: VersionData{VersionName: value.VersionName, VersionCode: value.VersionCode},
		Title:         value.Title, Changelog: value.Changelog, DownloadURL: value.DownloadURL,
		FileSize: value.FileSize, SHA256: value.FileSHA256, PublishedAt: *value.PublishedAt,
	}
}

func NewAppReleaseData(value *model.AppRelease) AppReleaseData {
	return AppReleaseData{
		ID: value.ID, Platform: value.Platform, VersionName: value.VersionName,
		VersionCode: value.VersionCode, Title: value.Title, Changelog: value.Changelog,
		DownloadURL: value.DownloadURL, FileSize: value.FileSize, SHA256: value.FileSHA256,
		MinimumVersionCode: value.MinimumVersionCode, ForceUpdate: value.ForceUpdate,
		Status: value.Status, PublishedAt: value.PublishedAt, AnnouncementID: value.AnnouncementID,
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func NewAppReleaseList(values []model.AppRelease) []AppReleaseData {
	items := make([]AppReleaseData, 0, len(values))
	for i := range values {
		items = append(items, NewAppReleaseData(&values[i]))
	}
	return items
}
