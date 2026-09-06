package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"
	"backend/internal/testutil"

	"gorm.io/gorm"
)

func TestProjectGalgameCharacterJSON(t *testing.T) {
	created := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	for _, tc := range []struct {
		name       string
		appearance bool
		spoiler    bool
	}{
		{"hidden", true, false},
		{"visible without spoilers", false, false},
		{"reveal hidden", true, true},
		{"reveal visible", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			relation := model.GalgameCharacter{
				ID: 91, GalgameID: 72, CharacterID: 53, Role: model.CharacterRoleMain,
				SpoilerLevel: model.SpoilerLevelMajor, AppearanceSpoiler: tc.appearance,
				Description: "Game-specific description", SpoilerDescription: "Secret identity",
				SortOrder: -2, CreatedAt: created, UpdatedAt: updated,
				Character: &model.Character{
					ID: 53, Name: "Character name", OriginalName: "Original name",
					Description: "Shared public biography", ImageURL: "https://example.com/character.png",
					Gender: "female", Birthday: "03-14", BloodType: "AB", Height: 165,
					Source: "vndb", SourceID: "c123", CreatedAt: created.Add(-time.Hour), UpdatedAt: created,
				},
			}
			encoded, err := json.Marshal(ProjectGalgameCharacter(&relation, tc.spoiler))
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(encoded, &got); err != nil {
				t.Fatal(err)
			}
			want := map[string]any{
				"id": float64(91), "role": "main", "appearance_spoiler": tc.appearance,
				"has_spoiler": true, "sort_order": float64(-2),
			}
			if !tc.appearance || tc.spoiler {
				for key, value := range map[string]any{
					"character_id": float64(53), "name": "Character name", "original_name": "Original name",
					"image_url": "https://example.com/character.png", "description": "Game-specific description",
					"public_description": "Shared public biography", "gender": "female", "birthday": "03-14",
					"blood_type": "AB", "height": float64(165), "source": "vndb", "source_id": "c123",
					"spoiler_level": "major", "created_at": created.Format(time.RFC3339), "updated_at": updated.Format(time.RFC3339),
				} {
					want[key] = value
				}
				if tc.spoiler {
					want["spoiler_description"] = "Secret identity"
				}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("JSON whitelist mismatch:\ngot:  %s\nwant: %+v", encoded, want)
			}
		})
	}
}

func TestProjectGalgameCharacterHasSpoiler(t *testing.T) {
	for _, tc := range []struct {
		name        string
		appearance  bool
		level       model.SpoilerLevel
		description string
		want        bool
	}{
		{"none", false, model.SpoilerLevelNone, "", false},
		{"appearance only", true, model.SpoilerLevelNone, "", true},
		{"minor only", false, model.SpoilerLevelMinor, "", true},
		{"major only", false, model.SpoilerLevelMajor, "", true},
		{"text only", false, model.SpoilerLevelNone, "Secret", true},
		{"all", true, model.SpoilerLevelMajor, "Secret", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, spoiler := range []bool{false, true} {
				relation := model.GalgameCharacter{
					AppearanceSpoiler: tc.appearance, SpoilerLevel: tc.level, SpoilerDescription: tc.description,
					Description: "Non-spoiler text", Character: &model.Character{Name: "Name", Description: "Public biography"},
				}
				encoded, err := json.Marshal(ProjectGalgameCharacter(&relation, spoiler))
				if err != nil {
					t.Fatal(err)
				}
				var got map[string]any
				if err := json.Unmarshal(encoded, &got); err != nil {
					t.Fatal(err)
				}
				if got["has_spoiler"] != tc.want {
					t.Fatalf("spoiler=%t: has_spoiler=%v want=%t", spoiler, got["has_spoiler"], tc.want)
				}
				if _, exists := got["spoiler_description"]; exists != spoiler {
					t.Fatalf("spoiler=%t: unexpected spoiler_description presence: %s", spoiler, encoded)
				}
			}
		})
	}
}

func TestProjectGalgameCharacterStringEnumsAndZeroValues(t *testing.T) {
	for role, roleName := range map[model.CharacterRole]string{
		model.CharacterRoleOther: "other", model.CharacterRoleProtagonist: "protagonist",
		model.CharacterRoleMain: "main", model.CharacterRoleSupporting: "supporting", model.CharacterRoleGuest: "guest",
	} {
		for level, levelName := range map[model.SpoilerLevel]string{
			model.SpoilerLevelNone: "none", model.SpoilerLevelMinor: "minor", model.SpoilerLevelMajor: "major",
		} {
			t.Run(roleName+"/"+levelName, func(t *testing.T) {
				encoded, err := json.Marshal(ProjectGalgameCharacter(&model.GalgameCharacter{
					Role: role, SpoilerLevel: level, Character: &model.Character{},
				}, true))
				if err != nil {
					t.Fatal(err)
				}
				var got map[string]any
				if err := json.Unmarshal(encoded, &got); err != nil {
					t.Fatal(err)
				}
				if got["role"] != roleName || got["spoiler_level"] != levelName {
					t.Fatalf("enums must serialize as strings: %s", encoded)
				}
				for _, key := range []string{"name", "original_name", "image_url", "description", "public_description", "gender", "birthday", "blood_type", "source", "source_id", "spoiler_description"} {
					if value, exists := got[key]; !exists || value != "" {
						t.Fatalf("revealed empty %s must remain present: %s", key, encoded)
					}
				}
				if got["height"] != float64(0) || got["character_id"] != float64(0) {
					t.Fatalf("revealed zero values must remain present: %s", encoded)
				}
			})
		}
	}
}

func TestCharacterServicePublishedScopingAndOrdering(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	svc := NewCharacterService(repository.NewGalgameRepository(db), repository.NewCharacterRepository(db))
	ctx := context.Background()
	published := createGalleryGalgame(t, db, "character-published", model.GalgameStatusPublished)
	var relations []uint
	for i, order := range []int{5, -1, 5, 0} {
		character, err := svc.CreateCharacter(ctx, &dto.CharacterRequest{Name: fmt.Sprintf("Character %d", i)})
		if err != nil {
			t.Fatal(err)
		}
		relation, err := svc.BindGalgameCharacter(ctx, published, &dto.GalgameCharacterRequest{
			CharacterID:                   character.ID,
			UpdateGalgameCharacterRequest: dto.UpdateGalgameCharacterRequest{SortOrder: order, AppearanceSpoiler: i == 0},
		})
		if err != nil {
			t.Fatal(err)
		}
		relations = append(relations, relation.ID)
	}
	wantOrder := []uint{relations[1], relations[3], relations[0], relations[2]}
	for _, spoiler := range []bool{false, true} {
		for range 3 {
			data, err := svc.ListPublishedCharacters(ctx, published, spoiler)
			if err != nil || len(data.Items) != len(wantOrder) {
				t.Fatalf("list published: %+v err=%v", data, err)
			}
			for i, item := range data.Items {
				if item.ID != wantOrder[i] {
					t.Fatalf("sort_order ASC, id ASC: got=%+v want=%v", data.Items, wantOrder)
				}
			}
			if (data.Items[2].CharacterID != nil) != spoiler {
				t.Fatalf("appearance identity visibility must follow spoiler=%t: %+v", spoiler, data.Items[2])
			}
		}
	}
	for _, status := range []int16{model.GalgameStatusPending, model.GalgameStatusRejected, model.GalgameStatusHidden} {
		id := createGalleryGalgame(t, db, fmt.Sprintf("character-status-%d", status), status)
		character, err := svc.CreateCharacter(ctx, &dto.CharacterRequest{Name: "Unpublished character"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.BindGalgameCharacter(ctx, id, &dto.GalgameCharacterRequest{CharacterID: character.ID}); err != nil {
			t.Fatal(err)
		}
		for _, spoiler := range []bool{false, true} {
			if _, err := svc.ListPublishedCharacters(ctx, id, spoiler); !errors.Is(err, ErrCharacterGalgameNotFound) {
				t.Fatalf("status=%d spoiler=%t must reject public listing: %v", status, spoiler, err)
			}
		}
		data, err := svc.ListAdminCharacters(ctx, id)
		if err != nil || len(data.Items) != 1 || data.Items[0].CharacterID == nil || *data.Items[0].CharacterID != character.ID {
			t.Fatalf("admin must list only this unpublished game's characters: %+v err=%v", data, err)
		}
	}
	if _, err := svc.ListPublishedCharacters(ctx, 999999, true); !errors.Is(err, ErrCharacterGalgameNotFound) {
		t.Fatalf("missing game must reject public listing: %v", err)
	}
}

func TestCharacterServiceScopedReplacementAndUnbind(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	repo := repository.NewCharacterRepository(db)
	svc := NewCharacterService(repository.NewGalgameRepository(db), repo)
	ctx := context.Background()
	first := createGalleryGalgame(t, db, "character-first", model.GalgameStatusPublished)
	second := createGalleryGalgame(t, db, "character-second", model.GalgameStatusPending)
	unbound := createGalleryGalgame(t, db, "character-unbound", model.GalgameStatusPublished)
	character, err := svc.CreateCharacter(ctx, &dto.CharacterRequest{
		Name: "Shared", OriginalName: "Original", Description: "Public", ImageURL: "https://example.com/character.png",
		Gender: "female", Birthday: "03-14", BloodType: "AB", Height: 165, Source: "vndb", SourceID: "c123",
	})
	if err != nil {
		t.Fatal(err)
	}
	request := &dto.GalgameCharacterRequest{
		CharacterID: character.ID,
		UpdateGalgameCharacterRequest: dto.UpdateGalgameCharacterRequest{
			Role: "main", SpoilerLevel: "major", AppearanceSpoiler: true,
			Description: "Game description", SpoilerDescription: "Secret", SortOrder: 9,
		},
	}
	for _, id := range []uint{first, second} {
		if _, err := svc.BindGalgameCharacter(ctx, id, request); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.BindGalgameCharacter(ctx, first, request); !errors.Is(err, ErrCharacterConflict) {
		t.Fatalf("duplicate binding must map to conflict: %v", err)
	}
	before, err := repo.FindRelation(ctx, second, character.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateGalgameCharacter(ctx, unbound, character.ID, &dto.UpdateGalgameCharacterRequest{}); !errors.Is(err, ErrCharacterNotFound) {
		t.Fatalf("cross-game update must not find another game's relation: %v", err)
	}
	if err := svc.UnbindGalgameCharacter(ctx, unbound, character.ID); !errors.Is(err, ErrCharacterNotFound) {
		t.Fatalf("cross-game unbind must not find another game's relation: %v", err)
	}
	updated, err := svc.UpdateGalgameCharacter(ctx, first, character.ID, &dto.UpdateGalgameCharacterRequest{})
	if err != nil || updated == nil || updated.Role != "other" || updated.SpoilerLevel == nil || *updated.SpoilerLevel != "none" ||
		updated.AppearanceSpoiler || updated.HasSpoiler || updated.SortOrder != 0 ||
		updated.Description == nil || *updated.Description != "" || updated.SpoilerDescription == nil || *updated.SpoilerDescription != "" {
		t.Fatalf("replacement must return zero values: %+v err=%v", updated, err)
	}
	persisted, err := repo.FindRelation(ctx, first, character.ID)
	if err != nil || persisted == nil || persisted.Role != model.CharacterRoleOther || persisted.SpoilerLevel != model.SpoilerLevelNone ||
		persisted.AppearanceSpoiler || persisted.SortOrder != 0 || persisted.Description != "" || persisted.SpoilerDescription != "" ||
		persisted.GalgameID != first || persisted.CharacterID != character.ID || persisted.ID != updated.ID {
		t.Fatalf("replacement must persist zero values without changing identity: %+v err=%v", persisted, err)
	}
	after, err := repo.FindRelation(ctx, second, character.ID)
	if err != nil || !reflect.DeepEqual(after, before) {
		t.Fatalf("scoped update changed another relation or shared metadata: got=%+v want=%+v err=%v", after, before, err)
	}
	if err := svc.UnbindGalgameCharacter(ctx, first, character.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.FindRelation(ctx, first, character.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("unbound relation must be absent: %v", err)
	}
	after, err = repo.FindRelation(ctx, second, character.ID)
	if err != nil || !reflect.DeepEqual(after, before) {
		t.Fatalf("unbind changed another relation or shared character: got=%+v want=%+v err=%v", after, before, err)
	}
	if _, err := svc.UpdateCharacter(ctx, character.ID, &dto.CharacterRequest{Name: "Renamed"}); err != nil {
		t.Fatal(err)
	}
	shared, err := repo.FindByID(ctx, character.ID)
	if err != nil || shared == nil || shared.Name != "Renamed" || shared.OriginalName != "" || shared.Description != "" ||
		shared.ImageURL != "" || shared.Gender != "" || shared.Birthday != "" || shared.BloodType != "" ||
		shared.Height != 0 || shared.Source != "" || shared.SourceID != "" {
		t.Fatalf("shared replacement must persist empty metadata: %+v err=%v", shared, err)
	}
}
