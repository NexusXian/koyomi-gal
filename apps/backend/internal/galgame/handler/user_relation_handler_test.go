package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"backend/internal/galgame/dto"
	galgameRepo "backend/internal/galgame/repository"
	galgameService "backend/internal/galgame/service"
	"backend/internal/middleware"
	"backend/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const testAuthSecret = "relation-test-secret"

type testTokenClaims struct {
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

func accessTokenFor(t *testing.T, userID uint) string {
	t.Helper()
	claims := testTokenClaims{
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "koyomi-gal",
			Subject:   strconv.FormatUint(uint64(userID), 10),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testAuthSecret))
	if err != nil {
		t.Fatalf("sign access token: %v", err)
	}
	return token
}

func newRelationTestRouter(t *testing.T) (*gin.Engine, *gorm.DB, *galgameService.CatalogService) {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	galgames := galgameRepo.NewGalgameRepository(db)
	relations := galgameRepo.NewUserRelationRepository(db)
	catalog := galgameService.NewCatalogService(
		galgames,
		galgameRepo.NewDeveloperRepository(db),
		galgameRepo.NewTagRepository(db),
	)
	relationHandler := NewUserRelationHandler(
		galgameService.NewRatingService(galgames, relations),
		galgameService.NewFavoriteService(galgames, relations),
		galgameService.NewUserStateService(galgames, relations),
		galgameService.NewUserRelationService(galgames, relations),
	)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/galgames/:id", middleware.Auth(testAuthSecret))
	{
		group.PUT("/rating", relationHandler.UpsertRating)
		group.DELETE("/rating", relationHandler.DeleteRating)
		group.POST("/favorite", relationHandler.AddFavorite)
		group.DELETE("/favorite", relationHandler.RemoveFavorite)
		group.PUT("/state", relationHandler.UpsertState)
		group.DELETE("/state", relationHandler.DeleteState)
		group.GET("/me", relationHandler.GetMyRelation)
	}
	return router, db, catalog
}

func doRelationRequest(
	router *gin.Engine,
	method, path, token string,
	body any,
) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		payload, _ := json.Marshal(body)
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}

func TestUserRelationEndpointsRequireAuth(t *testing.T) {
	router, _, _ := newRelationTestRouter(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPut, "/api/v1/galgames/1/rating"},
		{http.MethodDelete, "/api/v1/galgames/1/rating"},
		{http.MethodPost, "/api/v1/galgames/1/favorite"},
		{http.MethodDelete, "/api/v1/galgames/1/favorite"},
		{http.MethodPut, "/api/v1/galgames/1/state"},
		{http.MethodDelete, "/api/v1/galgames/1/state"},
		{http.MethodGet, "/api/v1/galgames/1/me"},
	}
	for _, endpoint := range endpoints {
		res := doRelationRequest(router, endpoint.method, endpoint.path, "", nil)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without token: expected 401, got %d", endpoint.method, endpoint.path, res.Code)
		}
		var body struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %s %s response: %v", endpoint.method, endpoint.path, err)
		}
		if body.Code != 205 {
			t.Fatalf("%s %s without token: expected code 205, got %d", endpoint.method, endpoint.path, body.Code)
		}
	}

	res := doRelationRequest(router, http.MethodPut, "/api/v1/galgames/1/rating", "invalid-token", nil)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("invalid token: expected 401, got %d", res.Code)
	}
}

func TestUserRelationEndpointsFlow(t *testing.T) {
	router, db, catalog := newRelationTestRouter(t)
	ctx := t.Context()
	user := testutil.CreateUser(t, db, "relation-http-user")
	galgame, err := catalog.CreateGalgame(ctx, user, &dto.CreateGalgameRequest{
		Title:  "relation-http-game",
		Slug:   "relation-http-game",
		Status: 1,
	})
	if err != nil {
		t.Fatalf("create galgame: %v", err)
	}
	token := accessTokenFor(t, user)
	base := "/api/v1/galgames/" + strconv.FormatUint(uint64(galgame.ID), 10)

	res := doRelationRequest(router, http.MethodPut, base+"/rating", token, map[string]any{"score": 8})
	if res.Code != http.StatusOK {
		t.Fatalf("put rating: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var ratingRes struct {
		Code int            `json:"code"`
		Data dto.RatingData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &ratingRes); err != nil {
		t.Fatalf("decode rating response: %v", err)
	}
	if ratingRes.Data.Score != 8 {
		t.Fatalf("expected score 8, got %+v", ratingRes.Data)
	}

	res = doRelationRequest(router, http.MethodPut, base+"/rating", token, map[string]any{"score": 12})
	if res.Code != http.StatusBadRequest {
		t.Fatalf("invalid score: expected 400, got %d", res.Code)
	}

	res = doRelationRequest(router, http.MethodPost, base+"/favorite", token, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("post favorite: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	res = doRelationRequest(router, http.MethodPost, base+"/favorite", token, nil)
	if res.Code != http.StatusConflict {
		t.Fatalf("duplicate favorite: expected 409, got %d", res.Code)
	}

	res = doRelationRequest(router, http.MethodPut, base+"/state", token,
		map[string]any{"state": 2, "play_time_minutes": 90})
	if res.Code != http.StatusOK {
		t.Fatalf("put state: expected 200, got %d body=%s", res.Code, res.Body.String())
	}

	res = doRelationRequest(router, http.MethodGet, base+"/me", token, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("get me: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var relationRes struct {
		Code int                         `json:"code"`
		Data dto.GalgameUserRelationData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &relationRes); err != nil {
		t.Fatalf("decode relation response: %v", err)
	}
	if relationRes.Data.GalgameID != galgame.ID {
		t.Fatalf("expected galgame_id %d, got %d", galgame.ID, relationRes.Data.GalgameID)
	}
	if relationRes.Data.Rating == nil || relationRes.Data.Rating.Score != 8 {
		t.Fatalf("expected rating score 8, got %+v", relationRes.Data.Rating)
	}
	if relationRes.Data.Favorite == nil || !relationRes.Data.Favorite.Favorited {
		t.Fatalf("expected favorited, got %+v", relationRes.Data.Favorite)
	}
	if relationRes.Data.State == nil || relationRes.Data.State.State != 2 || relationRes.Data.State.PlayTimeMinutes != 90 {
		t.Fatalf("unexpected state, got %+v", relationRes.Data.State)
	}

	res = doRelationRequest(router, http.MethodDelete, base+"/rating", token, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("delete rating: expected 200, got %d", res.Code)
	}
	res = doRelationRequest(router, http.MethodDelete, base+"/rating", token, nil)
	if res.Code != http.StatusNotFound {
		t.Fatalf("delete missing rating: expected 404, got %d", res.Code)
	}

	res = doRelationRequest(router, http.MethodDelete, base+"/state", token, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("delete state: expected 200, got %d", res.Code)
	}

	res = doRelationRequest(router, http.MethodGet, base+"/me", token, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("get me after cleanup: expected 200, got %d", res.Code)
	}
	if err := json.Unmarshal(res.Body.Bytes(), &relationRes); err != nil {
		t.Fatalf("decode relation response: %v", err)
	}
	if relationRes.Data.Rating != nil || relationRes.Data.State != nil {
		t.Fatalf("expected rating and state removed, got %+v", relationRes.Data)
	}
	if relationRes.Data.Favorite == nil {
		t.Fatal("expected favorite to remain")
	}

	res = doRelationRequest(router, http.MethodPut, "/api/v1/galgames/999999/rating", token,
		map[string]any{"score": 5})
	if res.Code != http.StatusNotFound {
		t.Fatalf("unknown galgame: expected 404, got %d", res.Code)
	}
	res = doRelationRequest(router, http.MethodPut, "/api/v1/galgames/not-a-number/rating", token,
		map[string]any{"score": 5})
	if res.Code != http.StatusBadRequest {
		t.Fatalf("invalid galgame id: expected 400, got %d", res.Code)
	}
}
