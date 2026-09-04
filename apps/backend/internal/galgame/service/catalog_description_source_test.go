package service

import (
	"context"
	"testing"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
)

func updateRequestFor(galgame *model.Galgame, description string) *dto.UpdateGalgameRequest {
	ageRating := galgame.AgeRating
	coverSensitive := galgame.CoverSensitive
	status := galgame.Status
	return &dto.UpdateGalgameRequest{
		Title:          galgame.Title,
		OriginalTitle:  galgame.OriginalTitle,
		RomajiTitle:    galgame.RomajiTitle,
		Slug:           galgame.Slug,
		Description:    description,
		CoverURL:       galgame.CoverURL,
		BannerURL:      galgame.BannerURL,
		AgeRating:      &ageRating,
		CoverSensitive: &coverSensitive,
		Status:         &status,
	}
}

func TestUpdateGalgameMarksDescriptionSource(t *testing.T) {
	svc, db, userID := newCatalogTestService(t)
	ctx := context.Background()
	galgame, err := svc.CreateGalgame(ctx, userID, &dto.CreateGalgameRequest{
		Title:       "source-manual",
		Slug:        "source-manual",
		Status:      model.GalgameStatusPublished,
		AgeRating:   model.AgeRatingAll,
		Description: "人工填写简介",
	})
	if err != nil {
		t.Fatalf("create galgame: %v", err)
	}

	reload := func() model.Galgame {
		var game model.Galgame
		if err := db.First(&game, galgame.ID).Error; err != nil {
			t.Fatalf("reload galgame: %v", err)
		}
		return game
	}
	load := func() model.Galgame {
		game, err := svc.GetGalgame(ctx, galgame.ID)
		if err != nil {
			t.Fatalf("get galgame: %v", err)
		}
		return *game
	}

	if got := load(); got.DescriptionSource != model.DescriptionSourceManual {
		t.Fatalf("create with description: source = %q, want manual", got.DescriptionSource)
	}

	// Editing fields other than the description must not flip the source to
	// manual when it already is manual.
	current := reload()
	titleOnly := updateRequestFor(&current, current.Description)
	titleOnly.Title = "source-manual-renamed"
	updated, err := svc.UpdateGalgame(ctx, galgame.ID, titleOnly, userID)
	if err != nil {
		t.Fatalf("update title only: %v", err)
	}
	if updated.DescriptionSource != model.DescriptionSourceManual {
		t.Fatalf("title-only update: source = %q, want manual kept", updated.DescriptionSource)
	}

	// An imported (vndb) description that is not edited by a human must keep
	// its source even when other fields change.
	if err := db.Model(&model.Galgame{}).Where("id = ?", galgame.ID).Updates(map[string]any{
		"description":        "English VNDB text",
		"description_source": model.DescriptionSourceVNDB,
	}).Error; err != nil {
		t.Fatalf("seed vndb source: %v", err)
	}
	current = reload()
	otherFields := updateRequestFor(&current, current.Description)
	otherFields.CoverURL = "https://example.com/other.jpg"
	if _, err := svc.UpdateGalgame(ctx, galgame.ID, otherFields, userID); err != nil {
		t.Fatalf("update non-description fields: %v", err)
	}
	if got := load(); got.DescriptionSource != model.DescriptionSourceVNDB {
		t.Fatalf("non-description update: source = %q, want vndb kept", got.DescriptionSource)
	}

	// Editing the description marks the row manual.
	current = reload()
	edit := updateRequestFor(&current, "管理员重写的中文简介")
	edited, err := svc.UpdateGalgame(ctx, galgame.ID, edit, userID)
	if err != nil {
		t.Fatalf("update description: %v", err)
	}
	if edited.Description != "管理员重写的中文简介" || edited.DescriptionSource != model.DescriptionSourceManual {
		t.Fatalf("description edit: got %q/%q, want rewritten text marked manual",
			edited.Description, edited.DescriptionSource)
	}

	// Clearing the description resets the source to unknown instead of manual.
	current = reload()
	cleared := updateRequestFor(&current, "")
	afterClear, err := svc.UpdateGalgame(ctx, galgame.ID, cleared, userID)
	if err != nil {
		t.Fatalf("clear description: %v", err)
	}
	if afterClear.Description != "" || afterClear.DescriptionSource != model.DescriptionSourceUnknown {
		t.Fatalf("description clear: got %q/%q, want empty text with unknown source",
			afterClear.Description, afterClear.DescriptionSource)
	}
}
