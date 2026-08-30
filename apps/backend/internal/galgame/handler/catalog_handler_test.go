package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"backend/internal/galgame/dto"
	galgameRepo "backend/internal/galgame/repository"
	galgameService "backend/internal/galgame/service"
	"backend/internal/middleware"
	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	"backend/internal/testutil"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func newCatalogAdminTestRouter(t *testing.T) (*gin.Engine, *gorm.DB, *galgameService.CatalogService, *rbacService.RBACService) {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	galgameRepository := galgameRepo.NewGalgameRepository(db)
	rbacSvc := rbacService.NewRBACService(rbacRepo.NewRBACRepository(db))
	if err := rbacSvc.SeedDefaults(context.Background()); err != nil {
		t.Fatalf("seed rbac defaults: %v", err)
	}
	catalog := galgameService.NewCatalogService(
		galgameRepository,
		galgameRepo.NewDeveloperRepository(db),
		galgameRepo.NewTagRepository(db),
	)
	catalogHandler := NewCatalogHandler(catalog)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/developers/:id", catalogHandler.GetDeveloper)
	router.GET("/api/v1/tags/:id", catalogHandler.GetTag)

	admin := router.Group("/api/v1/admin",
		middleware.Auth(testAuthSecret),
		middleware.RequirePermission(rbacSvc)("galgame:review"),
	)
	{
		admin.GET("/galgames", catalogHandler.ListAdminGalgames)
		admin.GET("/galgames/:id", catalogHandler.GetAdminGalgame)
	}
	return router, db, catalog, rbacSvc
}

func TestCatalogDetailAndAdminEndpoints(t *testing.T) {
	router, db, catalog, rbacSvc := newCatalogAdminTestRouter(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, db, "catalog-admin-user")
	admin := testutil.CreateUser(t, db, "catalog-admin-admin")
	if err := rbacSvc.AssignRoleByCode(ctx, admin, rbacService.RoleCodeAdmin); err != nil {
		t.Fatalf("assign admin role: %v", err)
	}

	pending, err := catalog.CreateGalgame(ctx, user, &dto.CreateGalgameRequest{
		Title: "catalog-admin-pending", Slug: "catalog-admin-pending", Status: 0,
	})
	if err != nil {
		t.Fatalf("create pending galgame: %v", err)
	}
	published, err := catalog.CreateGalgame(ctx, user, &dto.CreateGalgameRequest{
		Title: "catalog-admin-published", Slug: "catalog-admin-published", Status: 1,
	})
	if err != nil {
		t.Fatalf("create published galgame: %v", err)
	}
	developer, err := catalog.CreateDeveloper(ctx, &dto.CreateDeveloperRequest{Name: "detail-dev", Slug: "detail-dev"})
	if err != nil {
		t.Fatalf("create developer: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/developers/"+strconv.FormatUint(uint64(developer.ID), 10), nil)
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("developer detail: expected 200, got %d body=%s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/developers/999999", nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("unknown developer: expected 404, got %d", res.Code)
	}

	res = httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/admin/galgames", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("admin list without token: expected 401, got %d", res.Code)
	}

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/galgames", nil)
	req.Header.Set("Authorization", "Bearer "+accessTokenFor(t, user))
	router.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("admin list without permission: expected 403, got %d", res.Code)
	}

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/galgames", nil)
	req.Header.Set("Authorization", "Bearer "+accessTokenFor(t, admin))
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("admin list: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var listed struct {
		Code int                 `json:"code"`
		Data dto.GalgameListData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode admin list response: %v", err)
	}
	if listed.Data.Total != 2 || len(listed.Data.Items) != 2 {
		t.Fatalf("expected 2 galgames of all statuses, got %+v", listed.Data)
	}

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet,
		"/api/v1/admin/galgames/"+strconv.FormatUint(uint64(pending.ID), 10), nil)
	req.Header.Set("Authorization", "Bearer "+accessTokenFor(t, admin))
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("admin detail of pending: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var detail struct {
		Code int                 `json:"code"`
		Data dto.GalgameResponse `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode admin detail response: %v", err)
	}
	if detail.Data.ID != pending.ID || detail.Data.Status != 0 {
		t.Fatalf("expected pending galgame detail, got %+v", detail.Data)
	}

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet,
		"/api/v1/admin/galgames/"+strconv.FormatUint(uint64(published.ID), 10), nil)
	req.Header.Set("Authorization", "Bearer "+accessTokenFor(t, admin))
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK || detail.Data.ID == 0 {
		t.Fatalf("unexpected admin detail response: %d", res.Code)
	}

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/galgames/999999", nil)
	req.Header.Set("Authorization", "Bearer "+accessTokenFor(t, admin))
	router.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("unknown galgame admin detail: expected 404, got %d", res.Code)
	}
}
