package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"
	"backend/internal/galgame/service"
	"backend/internal/testutil"

	"github.com/gin-gonic/gin"
)

func TestCharacterHandlerInvalidSpoilerQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/galgames/:id/characters", NewCharacterHandler(nil).ListGalgameCharacters)
	for _, query := range []string{"", "1", "0", "TRUE", "False", "yes", "invalid", "%20true"} {
		t.Run("spoiler="+query, func(t *testing.T) {
			res := httptest.NewRecorder()
			router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/galgames/1/characters?spoiler="+query, nil))
			if res.Code != http.StatusBadRequest {
				t.Fatalf("invalid spoiler query: status=%d body=%s", res.Code, res.Body.String())
			}
			if res.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatalf("response must not be cached: %v", res.Header())
			}
		})
	}
}

func TestCharacterHandlerSpoilerQuery(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	svc := service.NewCharacterService(repository.NewGalgameRepository(db), repository.NewCharacterRepository(db))
	game := &model.Galgame{Title: "Character query", Slug: "character-query", Status: model.GalgameStatusPublished}
	if err := db.Create(game).Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	var relationIDs []uint
	var characterIDs []uint
	for i, hidden := range []bool{true, false} {
		character, err := svc.CreateCharacter(ctx, &dto.CharacterRequest{
			Name: fmt.Sprintf("Character %d", i), ImageURL: "https://example.com/character.png", Source: "vndb", SourceID: fmt.Sprintf("c%d", i),
		})
		if err != nil {
			t.Fatal(err)
		}
		relation, err := svc.BindGalgameCharacter(ctx, game.ID, &dto.GalgameCharacterRequest{
			CharacterID: character.ID,
			UpdateGalgameCharacterRequest: dto.UpdateGalgameCharacterRequest{
				Role: "main", AppearanceSpoiler: hidden, Description: "Public description", SpoilerDescription: "Secret",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		relationIDs = append(relationIDs, relation.ID)
		characterIDs = append(characterIDs, character.ID)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/galgames/:id/characters", NewCharacterHandler(svc).ListGalgameCharacters)
	for _, query := range []string{"", "?spoiler=false", "?spoiler=true"} {
		t.Run("query="+query, func(t *testing.T) {
			res := httptest.NewRecorder()
			router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/galgames/%d/characters%s", game.ID, query), nil))
			if res.Code != http.StatusOK || res.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatalf("public list: status=%d headers=%v body=%s", res.Code, res.Header(), res.Body.String())
			}
			var body struct {
				Code int `json:"code"`
				Data struct {
					Items []map[string]any `json:"items"`
				} `json:"data"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Code != 0 || len(body.Data.Items) != 2 {
				t.Fatalf("unexpected response envelope: %s", res.Body.String())
			}
			reveal := query == "?spoiler=true"
			if !reveal {
				want := map[string]any{
					"id": float64(relationIDs[0]), "role": "main", "appearance_spoiler": true,
					"has_spoiler": true, "sort_order": float64(0),
				}
				if !reflect.DeepEqual(body.Data.Items[0], want) {
					t.Fatalf("default/false must conceal all identity: %s", res.Body.String())
				}
			}
			for i, item := range body.Data.Items {
				if value, exists := item["spoiler_description"]; exists != reveal || (reveal && value != "Secret") {
					t.Fatalf("spoiler text must require explicit true: %s", res.Body.String())
				}
				if reveal || i == 1 {
					if item["character_id"] != float64(characterIDs[i]) || item["name"] != fmt.Sprintf("Character %d", i) ||
						item["image_url"] != "https://example.com/character.png" || item["description"] != "Public description" {
						t.Fatalf("visible character metadata missing: %s", res.Body.String())
					}
				}
			}
		})
	}
}
