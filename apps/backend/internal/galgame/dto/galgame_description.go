package dto

import "backend/internal/galgame/model"

// DescriptionSourceResponse is the structured provenance of a description.
type DescriptionSourceResponse struct {
	Type     string `json:"type" example:"vndb"`
	Name     string `json:"name" example:"VNDB"`
	URL      string `json:"url,omitempty" example:"https://vndb.org/v20424"`
	Official bool   `json:"official" example:"false"`
}

// GalgameDescriptionResponse is one language's description.
type GalgameDescriptionResponse struct {
	Language string                    `json:"language" example:"zh-CN"`
	Content  string                    `json:"content" example:"作品简介"`
	Source   DescriptionSourceResponse `json:"source"`
}

// GalgameDescriptionInput is one language's description payload for the
// admin upsert endpoints.
type GalgameDescriptionInput struct {
	Language   string `json:"language" binding:"required,oneof=zh-CN en-US ja-JP" example:"zh-CN"`
	Content    string `json:"content" example:"作品简介"`
	SourceType string `json:"source_type" binding:"omitempty,oneof=unknown vndb bangumi nextmoe official steam manual" example:"nextmoe"`
	SourceName string `json:"source_name" binding:"max=128" example:"NextMoe 资料库"`
	SourceURL  string `json:"source_url" example:"https://example.com"`
	IsOfficial bool   `json:"is_official" example:"false"`
}

type UpdateGalgameDescriptionsRequest struct {
	Descriptions []GalgameDescriptionInput `json:"descriptions" binding:"required,min=1,max=3,dive"`
}

type GalgameDescriptionsData struct {
	Descriptions map[string]GalgameDescriptionResponse `json:"descriptions"`
}

type GalgameDescriptionsResponse struct {
	Code int                    `json:"code" example:"0"`
	Data GalgameDescriptionsData `json:"data"`
	Msg  string                 `json:"msg" example:"success"`
}

// NewGalgameDescriptionResponse converts a model row, falling back to the
// default display name when source_name is empty.
func NewGalgameDescriptionResponse(description model.GalgameDescription) GalgameDescriptionResponse {
	name := description.SourceName
	if name == "" {
		name = model.DefaultDescriptionSourceName(description.SourceType)
	}
	return GalgameDescriptionResponse{
		Language: description.Language,
		Content:  description.Content,
		Source: DescriptionSourceResponse{
			Type:     description.SourceType,
			Name:     name,
			URL:      description.SourceURL,
			Official: description.IsOfficial,
		},
	}
}

// NewGalgameDescriptionResponses builds the language-keyed description map
// for the detail response.
func NewGalgameDescriptionResponses(descriptions []model.GalgameDescription) map[string]GalgameDescriptionResponse {
	items := make(map[string]GalgameDescriptionResponse, len(descriptions))
	for _, description := range descriptions {
		items[description.Language] = NewGalgameDescriptionResponse(description)
	}
	return items
}

// PrimaryDescription returns the zh-CN content used to keep the legacy
// single-description field backward compatible; it falls back to the legacy
// column when no zh-CN row exists yet.
func PrimaryDescription(descriptions []model.GalgameDescription, legacy string) string {
	for _, description := range descriptions {
		if description.Language == model.LanguageZhCN {
			return description.Content
		}
	}
	return legacy
}
