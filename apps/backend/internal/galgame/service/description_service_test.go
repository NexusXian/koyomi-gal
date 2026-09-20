package service

import (
	"context"
	"testing"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"
)

func newDescriptionTestService(t *testing.T) (*DescriptionService, *CatalogService, uint) {
	t.Helper()
	catalog, db, userID := newCatalogTestService(t)
	descriptions := NewDescriptionService(
		repository.NewGalgameRepository(db),
		repository.NewGalgameDescriptionRepository(db),
	)
	return descriptions, catalog, userID
}

func descriptionInputs() []dto.GalgameDescriptionInput {
	return []dto.GalgameDescriptionInput{
		{Language: "zh-CN", Content: "中文简介", SourceType: "nextmoe", SourceName: "NextMoe 资料库"},
		{Language: "en-US", Content: "English description", SourceType: "vndb", SourceName: "VNDB", SourceURL: "https://vndb.org/v20424"},
		{Language: "ja-JP", Content: "日本語紹介", SourceType: "official", SourceName: "游戏官网", SourceURL: "https://example.jp", IsOfficial: true},
	}
}

func TestUpsertGameDescriptionsCreatesAllLanguages(t *testing.T) {
	descriptions, catalog, actor := newDescriptionTestService(t)
	ctx := context.Background()
	galgame := createTestGalgame(t, catalog, actor, "desc-all-languages", nil, nil, "", 0, model.GalgameStatusPublished)

	rows, err := descriptions.UpsertGameDescriptions(ctx, galgame.ID, descriptionInputs(), actor)
	if err != nil {
		t.Fatalf("upsert descriptions: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows))
	}

	detail, err := catalog.GetGalgame(ctx, galgame.ID)
	if err != nil {
		t.Fatalf("get galgame: %v", err)
	}
	response := dto.NewGalgameResponse(detail)
	if len(response.Descriptions) != 3 {
		t.Fatalf("descriptions = %d, want 3", len(response.Descriptions))
	}
	zh := response.Descriptions["zh-CN"]
	if zh.Content != "中文简介" || zh.Source.Name != "NextMoe 资料库" || zh.Source.Type != "nextmoe" || zh.Source.Official {
		t.Errorf("zh-CN = %+v", zh)
	}
	en := response.Descriptions["en-US"]
	if en.Content != "English description" || en.Source.URL != "https://vndb.org/v20424" || en.Source.Official {
		t.Errorf("en-US = %+v", en)
	}
	ja := response.Descriptions["ja-JP"]
	if ja.Content != "日本語紹介" || !ja.Source.Official || ja.Source.Type != "official" {
		t.Errorf("ja-JP = %+v", ja)
	}
	// Legacy compatibility: the flat description mirrors zh-CN.
	if response.Description != "中文简介" {
		t.Errorf("legacy description = %q, want the zh-CN content", response.Description)
	}
}

func TestUpsertGameDescriptionsUpdatesSingleLanguageOnly(t *testing.T) {
	descriptions, catalog, actor := newDescriptionTestService(t)
	ctx := context.Background()
	galgame := createTestGalgame(t, catalog, actor, "desc-single-language", nil, nil, "", 0, model.GalgameStatusPublished)

	if _, err := descriptions.UpsertGameDescriptions(ctx, galgame.ID, descriptionInputs(), actor); err != nil {
		t.Fatalf("seed descriptions: %v", err)
	}
	updated := []dto.GalgameDescriptionInput{
		{Language: "en-US", Content: "Revised English", SourceType: "vndb", SourceName: "VNDB"},
	}
	if _, err := descriptions.UpsertGameDescriptions(ctx, galgame.ID, updated, actor); err != nil {
		t.Fatalf("update en-US: %v", err)
	}

	detail, err := catalog.GetGalgame(ctx, galgame.ID)
	if err != nil {
		t.Fatalf("get galgame: %v", err)
	}
	response := dto.NewGalgameResponse(detail)
	if response.Descriptions["en-US"].Content != "Revised English" {
		t.Errorf("en-US = %+v, want revised", response.Descriptions["en-US"])
	}
	if response.Descriptions["zh-CN"].Content != "中文简介" {
		t.Errorf("zh-CN must stay untouched, got %+v", response.Descriptions["zh-CN"])
	}
	if response.Descriptions["ja-JP"].Content != "日本語紹介" {
		t.Errorf("ja-JP must stay untouched, got %+v", response.Descriptions["ja-JP"])
	}
}

func TestUpsertGameDescriptionsKeepsRowsUnique(t *testing.T) {
	descriptions, catalog, actor := newDescriptionTestService(t)
	ctx := context.Background()
	galgame := createTestGalgame(t, catalog, actor, "desc-unique", nil, nil, "", 0, model.GalgameStatusPublished)

	for range 2 {
		if _, err := descriptions.UpsertGameDescriptions(ctx, galgame.ID, descriptionInputs(), actor); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}
	rows, err := descriptions.GetGameDescriptions(ctx, galgame.ID)
	if err != nil {
		t.Fatalf("get descriptions: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3 after repeated upserts", len(rows))
	}
}

func TestDeleteGameDescriptionRemovesOnlyOneLanguage(t *testing.T) {
	descriptions, catalog, actor := newDescriptionTestService(t)
	ctx := context.Background()
	galgame := createTestGalgame(t, catalog, actor, "desc-delete", nil, nil, "", 0, model.GalgameStatusPublished)

	if _, err := descriptions.UpsertGameDescriptions(ctx, galgame.ID, descriptionInputs(), actor); err != nil {
		t.Fatalf("seed descriptions: %v", err)
	}
	if err := descriptions.DeleteGameDescription(ctx, galgame.ID, model.LanguageJaJP, actor); err != nil {
		t.Fatalf("delete ja-JP: %v", err)
	}
	rows, err := descriptions.GetGameDescriptions(ctx, galgame.ID)
	if err != nil {
		t.Fatalf("get descriptions: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	for _, row := range rows {
		if row.Language == model.LanguageJaJP {
			t.Errorf("ja-JP row must be gone, got %+v", row)
		}
	}
	if err := descriptions.DeleteGameDescription(ctx, galgame.ID, model.LanguageJaJP, actor); err == nil {
		t.Error("deleting a missing language must fail")
	}
}

func TestUpsertGameDescriptionsRejectsInvalidLanguage(t *testing.T) {
	descriptions, catalog, actor := newDescriptionTestService(t)
	ctx := context.Background()
	galgame := createTestGalgame(t, catalog, actor, "desc-invalid-language", nil, nil, "", 0, model.GalgameStatusPublished)

	for _, language := range []string{"jp", "cn", "xxx", "zh", "zh_CN"} {
		input := []dto.GalgameDescriptionInput{{Language: language, Content: "text"}}
		if _, err := descriptions.UpsertGameDescriptions(ctx, galgame.ID, input, actor); err == nil {
			t.Errorf("language %q must be rejected", language)
		}
	}
}

func TestUpsertGameDescriptionsRejectsDangerousURL(t *testing.T) {
	descriptions, catalog, actor := newDescriptionTestService(t)
	ctx := context.Background()
	galgame := createTestGalgame(t, catalog, actor, "desc-invalid-url", nil, nil, "", 0, model.GalgameStatusPublished)

	for _, url := range []string{"javascript:alert(1)", "data:text/html,x", "file:///etc/passwd"} {
		input := []dto.GalgameDescriptionInput{{Language: "zh-CN", SourceURL: url}}
		if _, err := descriptions.UpsertGameDescriptions(ctx, galgame.ID, input, actor); err == nil {
			t.Errorf("source url %q must be rejected", url)
		}
	}
}

func TestUpsertGameDescriptionsAppliesDefaultsAndNameFallback(t *testing.T) {
	descriptions, catalog, actor := newDescriptionTestService(t)
	ctx := context.Background()
	galgame := createTestGalgame(t, catalog, actor, "desc-defaults", nil, nil, "", 0, model.GalgameStatusPublished)

	// Omitted source fields fall back to the language defaults.
	rows, err := descriptions.UpsertGameDescriptions(ctx, galgame.ID, []dto.GalgameDescriptionInput{
		{Language: "ja-JP", Content: "公式紹介文"},
	}, actor)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}
	if rows[0].SourceType != model.DescriptionSourceOfficial || !rows[0].IsOfficial {
		t.Errorf("ja-JP defaults = %+v, want official source", rows[0])
	}

	// An empty source_name falls back to the type's display name.
	rows, err = descriptions.UpsertGameDescriptions(ctx, galgame.ID, []dto.GalgameDescriptionInput{
		{Language: "en-US", Content: "text", SourceType: "steam"},
	}, actor)
	if err != nil {
		t.Fatalf("upsert en-US: %v", err)
	}
	if rows[0].SourceName != "Steam" {
		t.Errorf("source name = %q, want Steam", rows[0].SourceName)
	}
}

func TestUpdateGalgameDescriptionsArrayPath(t *testing.T) {
	_, catalog, actor := newDescriptionTestService(t)
	ctx := context.Background()
	galgame := createTestGalgame(t, catalog, actor, "desc-array-path", nil, nil, "", 0, model.GalgameStatusPublished)

	req := &dto.UpdateGalgameRequest{
		Title:          galgame.Title,
		OriginalTitle:  galgame.OriginalTitle,
		RomajiTitle:    galgame.RomajiTitle,
		Slug:           galgame.Slug,
		CoverURL:       galgame.CoverURL,
		BannerURL:      galgame.BannerURL,
		AgeRating:      &galgame.AgeRating,
		CoverSensitive: &galgame.CoverSensitive,
		Status:         &galgame.Status,
		Descriptions:   descriptionInputs(),
	}
	updated, err := catalog.UpdateGalgame(ctx, galgame.ID, req, actor)
	if err != nil {
		t.Fatalf("update galgame with descriptions: %v", err)
	}
	if len(updated.Descriptions) != 3 {
		t.Fatalf("descriptions = %d, want 3", len(updated.Descriptions))
	}
	if updated.Description != "中文简介" {
		t.Errorf("legacy description = %q, want zh-CN mirror", updated.Description)
	}
}

func TestUpdateGalgameLegacyDescriptionSeedsZhRow(t *testing.T) {
	_, catalog, actor := newDescriptionTestService(t)
	ctx := context.Background()
	galgame := createTestGalgame(t, catalog, actor, "desc-legacy-path", nil, nil, "", 0, model.GalgameStatusPublished)

	// Old clients send the flat description; it must land in the zh-CN row.
	current, err := catalog.GetGalgame(ctx, galgame.ID)
	if err != nil {
		t.Fatalf("get galgame: %v", err)
	}
	status := model.GalgameStatusPublished
	req := &dto.UpdateGalgameRequest{
		Title:          current.Title,
		OriginalTitle:  current.OriginalTitle,
		RomajiTitle:    current.RomajiTitle,
		Slug:           current.Slug,
		Description:    "旧客户端写入的中文",
		CoverURL:       current.CoverURL,
		BannerURL:      current.BannerURL,
		AgeRating:      &current.AgeRating,
		CoverSensitive: &current.CoverSensitive,
		Status:         &status,
	}
	updated, err := catalog.UpdateGalgame(ctx, galgame.ID, req, actor)
	if err != nil {
		t.Fatalf("update galgame legacy description: %v", err)
	}
	zh, ok := dto.NewGalgameResponse(updated).Descriptions["zh-CN"]
	if !ok || zh.Content != "旧客户端写入的中文" || zh.Source.Type != model.DescriptionSourceManual {
		t.Errorf("zh-CN = %+v ok=%v, want the manual legacy row", zh, ok)
	}
}

func TestUpsertGameDescriptionsMissingGalgame(t *testing.T) {
	descriptions, _, _ := newDescriptionTestService(t)
	missing := uint(999999)
	if _, err := descriptions.UpsertGameDescriptions(context.Background(), missing, descriptionInputs(), 0); err == nil {
		t.Error("upsert on a missing galgame must fail")
	}
}
