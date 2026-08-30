package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	galgameDTO "backend/internal/galgame/dto"
	galgameRepo "backend/internal/galgame/repository"
	galgameService "backend/internal/galgame/service"
	"backend/internal/middleware"
	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	"backend/internal/resource/dto"
	resourceRepo "backend/internal/resource/repository"
	resourceService "backend/internal/resource/service"
	"backend/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const testAuthSecret = "resource-test-secret"

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

type resourceHandlerEnv struct {
	router  *gin.Engine
	db      *gorm.DB
	catalog *galgameService.CatalogService
	rbac    *rbacService.RBACService
}

func newResourceHandlerEnv(t *testing.T) *resourceHandlerEnv {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	galgameRepository := galgameRepo.NewGalgameRepository(db)
	rbacSvc := rbacService.NewRBACService(rbacRepo.NewRBACRepository(db))
	if err := rbacSvc.SeedDefaults(context.Background()); err != nil {
		t.Fatalf("seed rbac defaults: %v", err)
	}
	resources := resourceRepo.NewResourceRepository(db)
	resourceSvc := resourceService.NewResourceService(resources, galgameRepository, rbacSvc)
	handler := NewResourceHandler(resourceSvc)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/galgames/:id/resources", handler.ListGalgameResources)
	router.GET("/api/v1/resources/:id", handler.GetResource)

	protected := router.Group("/api/v1/resources", middleware.Auth(testAuthSecret))
	{
		protected.POST("", handler.CreateResource)
		protected.PUT("/:id", handler.UpdateResource)
		protected.DELETE("/:id", handler.DeleteResource)
	}

	return &resourceHandlerEnv{
		router: router,
		db:     db,
		catalog: galgameService.NewCatalogService(
			galgameRepository,
			galgameRepo.NewDeveloperRepository(db),
			galgameRepo.NewTagRepository(db),
		),
		rbac: rbacSvc,
	}
}

func doResourceRequest(router *gin.Engine, method, path, token string, body any) *httptest.ResponseRecorder {
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}

func TestResourceEndpointsAuthBoundaries(t *testing.T) {
	env := newResourceHandlerEnv(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, env.db, "resource-http-user")
	galgame, err := env.catalog.CreateGalgame(ctx, user, &galgameDTO.CreateGalgameRequest{
		Title:  "resource-http-game",
		Slug:   "resource-http-game",
		Status: 1,
	})
	if err != nil {
		t.Fatalf("create galgame: %v", err)
	}

	res := doResourceRequest(env.router, http.MethodPost, "/api/v1/resources", "", map[string]any{
		"galgame_id": galgame.ID,
		"title":      "no auth",
		"links":      []string{"https://example.com/a"},
	})
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("create without token: expected 401, got %d", res.Code)
	}
	res = doResourceRequest(env.router, http.MethodPut, "/api/v1/resources/1", "", nil)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("update without token: expected 401, got %d", res.Code)
	}
	res = doResourceRequest(env.router, http.MethodDelete, "/api/v1/resources/1", "", nil)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("delete without token: expected 401, got %d", res.Code)
	}

	res = doResourceRequest(env.router, http.MethodGet,
		"/api/v1/galgames/"+strconv.FormatUint(uint64(galgame.ID), 10)+"/resources", "", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("public list without token: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	res = doResourceRequest(env.router, http.MethodGet, "/api/v1/resources/999999", "", nil)
	if res.Code != http.StatusNotFound {
		t.Fatalf("public detail of unknown resource: expected 404, got %d", res.Code)
	}
}

func TestResourceEndpointsOwnerFlow(t *testing.T) {
	env := newResourceHandlerEnv(t)
	ctx := context.Background()
	owner := testutil.CreateUser(t, env.db, "resource-http-owner")
	stranger := testutil.CreateUser(t, env.db, "resource-http-stranger")
	admin := testutil.CreateUser(t, env.db, "resource-http-admin")
	if err := env.rbac.AssignRoleByCode(ctx, admin, rbacService.RoleCodeAdmin); err != nil {
		t.Fatalf("assign admin role: %v", err)
	}
	galgame, err := env.catalog.CreateGalgame(ctx, owner, &galgameDTO.CreateGalgameRequest{
		Title:  "resource-http-flow-game",
		Slug:   "resource-http-flow-game",
		Status: 1,
	})
	if err != nil {
		t.Fatalf("create galgame: %v", err)
	}
	ownerToken := accessTokenFor(t, owner)
	strangerToken := accessTokenFor(t, stranger)
	adminToken := accessTokenFor(t, admin)
	galgamePath := "/api/v1/galgames/" + strconv.FormatUint(uint64(galgame.ID), 10)

	res := doResourceRequest(env.router, http.MethodPost, "/api/v1/resources", ownerToken, map[string]any{
		"galgame_id": galgame.ID,
		"title":      "http resource",
		"type":       2,
		"status":     1,
		"links":      []string{"https://example.com/a", "https://example.com/b"},
	})
	if res.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var created struct {
		Code int              `json:"code"`
		Data dto.ResourceData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Data.Title != "http resource" || len(created.Data.Links) != 2 || created.Data.Status != 1 {
		t.Fatalf("unexpected created data: %+v", created.Data)
	}
	resourcePath := "/api/v1/resources/" + strconv.FormatUint(uint64(created.Data.ID), 10)

	res = doResourceRequest(env.router, http.MethodGet, galgamePath+"/resources", "", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("public list: expected 200, got %d", res.Code)
	}
	var listed struct {
		Code int                `json:"code"`
		Data []dto.ResourceData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listed.Data) != 1 || listed.Data[0].ID != created.Data.ID {
		t.Fatalf("expected 1 published resource, got %+v", listed.Data)
	}

	res = doResourceRequest(env.router, http.MethodPut, resourcePath, strangerToken, map[string]any{
		"title":  "hijack",
		"type":   0,
		"status": 1,
		"links":  []string{"https://example.com/evil"},
	})
	if res.Code != http.StatusForbidden {
		t.Fatalf("stranger update: expected 403, got %d", res.Code)
	}
	res = doResourceRequest(env.router, http.MethodDelete, resourcePath, strangerToken, nil)
	if res.Code != http.StatusForbidden {
		t.Fatalf("stranger delete: expected 403, got %d", res.Code)
	}

	res = doResourceRequest(env.router, http.MethodPut, resourcePath, ownerToken, map[string]any{
		"title":  "http resource v2",
		"type":   3,
		"status": 1,
		"links":  []string{"https://example.com/c"},
	})
	if res.Code != http.StatusOK {
		t.Fatalf("owner update: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var updated struct {
		Code int              `json:"code"`
		Data dto.ResourceData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.Data.Title != "http resource v2" || len(updated.Data.Links) != 1 {
		t.Fatalf("unexpected updated data: %+v", updated.Data)
	}

	res = doResourceRequest(env.router, http.MethodDelete, resourcePath, adminToken, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("admin delete: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	res = doResourceRequest(env.router, http.MethodGet, resourcePath, "", nil)
	if res.Code != http.StatusNotFound {
		t.Fatalf("detail after delete: expected 404, got %d", res.Code)
	}

	var count int64
	if err := env.db.Raw(
		"SELECT resource_count FROM galgames WHERE id = ?", galgame.ID,
	).Scan(&count).Error; err != nil || count != 0 {
		t.Fatalf("expected resource_count 0 after delete, got %d err=%v", count, err)
	}
}
