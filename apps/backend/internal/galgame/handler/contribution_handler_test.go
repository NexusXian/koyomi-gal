package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"
	"backend/internal/galgame/service"
	"backend/internal/testutil"

	"github.com/gin-gonic/gin"
)

func TestListGalgameContributorsAPI(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	ctx := context.Background()
	galgames := repository.NewGalgameRepository(db)
	contributionRepository := repository.NewContributionRepository(db, "https://img.example.com")
	contributions := service.NewContributionService(contributionRepository, galgames, "https://img.example.com")
	first := testutil.CreateUser(t, db, "api-contributor-first")
	second := testutil.CreateUser(t, db, "api-contributor-second")
	game := &model.Galgame{Title: "api-contributors", Slug: "api-contributors", Status: model.GalgameStatusPublished}
	if err := galgames.Create(ctx, game); err != nil {
		t.Fatalf("create galgame: %v", err)
	}
	for range 2 {
		if err := contributions.RecordContribution(ctx, service.RecordContributionInput{
			GalgameID: game.ID,
			UserID:    first,
			Action:    model.ContributionActionEdit,
		}); err != nil {
			t.Fatalf("record first contributor: %v", err)
		}
	}
	if err := contributions.RecordContribution(ctx, service.RecordContributionInput{
		GalgameID: game.ID,
		UserID:    second,
		Action:    model.ContributionActionGallery,
	}); err != nil {
		t.Fatalf("record second contributor: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/galgames/:id/contributors", NewContributionHandler(contributions).ListGalgameContributors)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/galgames/"+strconv.FormatUint(uint64(game.ID), 10)+"/contributors?page=1&page_size=20", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status: got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Code int                     `json:"code"`
		Data dto.ContributorListData `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Code != 0 || response.Data.Total != 2 || len(response.Data.Items) != 2 {
		t.Fatalf("response: %+v", response)
	}
	if response.Data.Items[0].UserID != first || response.Data.Items[0].ContributionCount != 2 {
		t.Fatalf("grouped contributor order/count: %+v", response.Data.Items)
	}
}
