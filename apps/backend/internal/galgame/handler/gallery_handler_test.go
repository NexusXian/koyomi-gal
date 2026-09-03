package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"backend/internal/galgame/dto"
	galgameRepo "backend/internal/galgame/repository"
	galgameService "backend/internal/galgame/service"
	"backend/internal/middleware"
	imageRepo "backend/internal/image/repository"
	imageService "backend/internal/image/service"
	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	"backend/internal/testutil"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func newGalleryTestRouter(t *testing.T) (*gin.Engine, *gorm.DB, *galgameService.GalleryService, *rbacService.RBACService) {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	galgameRepository := galgameRepo.NewGalgameRepository(db)
	rbacSvc := rbacService.NewRBACService(rbacRepo.NewRBACRepository(db))
	if err := rbacSvc.SeedDefaults(context.Background()); err != nil {
		t.Fatalf("seed rbac defaults: %v", err)
	}
	images := imageService.NewImageAssetService(
		imageRepo.NewImageAssetRepository(db),
		nil,
		nil,
		nil,
		"https://img.example.com",
	)
	galleryService := galgameService.NewGalleryService(
		galgameRepository,
		galgameRepo.NewGalleryRepository(db),
		images,
	)
	galleryHandler := NewGalleryHandler(galleryService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/galgames/:id/gallery", galleryHandler.ListGalgameGallery)

	admin := router.Group("/api/v1/admin",
		middleware.Auth(testAuthSecret),
		middleware.RequirePermission(rbacSvc)("galgame_gallery:manage"),
	)
	{
		admin.GET("/galgames/:id/gallery", galleryHandler.ListAdminGalgameGallery)
		admin.POST("/galgames/:id/gallery", galleryHandler.CreateGalgameGalleryImage)
		admin.PUT("/galgames/:id/gallery/order", galleryHandler.ReorderGalgameGallery)
		admin.PATCH("/galgames/:id/gallery/:galleryId", galleryHandler.UpdateGalgameGalleryImage)
		admin.DELETE("/galgames/:id/gallery/:galleryId", galleryHandler.DeleteGalgameGalleryImage)
	}
	return router, db, galleryService, rbacSvc
}

func TestGalleryEndpoints(t *testing.T) {
	router, db, _, rbacSvc := newGalleryTestRouter(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, db, "gallery-endpoint-user")
	admin := testutil.CreateUser(t, db, "gallery-endpoint-admin")
	if err := rbacSvc.AssignRoleByCode(ctx, admin, rbacService.RoleCodeAdmin); err != nil {
		t.Fatalf("assign admin role: %v", err)
	}

	catalog := galgameService.NewCatalogService(
		galgameRepo.NewGalgameRepository(db),
		galgameRepo.NewDeveloperRepository(db),
		galgameRepo.NewTagRepository(db),
	)
	published, err := catalog.CreateGalgame(ctx, user, &dto.CreateGalgameRequest{
		Title: "gallery-endpoint-published", Slug: "gallery-endpoint-published", Status: 1,
	})
	if err != nil {
		t.Fatalf("create published galgame: %v", err)
	}
	pending, err := catalog.CreateGalgame(ctx, user, &dto.CreateGalgameRequest{
		Title: "gallery-endpoint-pending", Slug: "gallery-endpoint-pending", Status: 0,
	})
	if err != nil {
		t.Fatalf("create pending galgame: %v", err)
	}

	var assetID uint
	if err := db.Raw(`
INSERT INTO image_assets (object_key, original_name, mime_type, extension, size, category, status, created_at, updated_at)
VALUES ('galgames/10001/2026/09/gallery-endpoint.webp', 'cg.webp', 'image/webp', 'webp', 1024, 'galgames', 1, NOW(), NOW())
RETURNING id
`).Scan(&assetID).Error; err != nil {
		t.Fatalf("create image asset: %v", err)
	}

	base := "/api/v1/admin/galgames/" + strconv.FormatUint(uint64(published.ID), 10)

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, base+"/gallery", strings.NewReader(
		`{"asset_id":`+strconv.FormatUint(uint64(assetID), 10)+`,"title":"标题","image_type":1,"is_spoiler":true}`,
	))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("create without token: expected 401, got %d", res.Code)
	}

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, base+"/gallery", strings.NewReader(
		`{"asset_id":`+strconv.FormatUint(uint64(assetID), 10)+`}`,
	))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessTokenFor(t, user))
	router.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("create without permission: expected 403, got %d", res.Code)
	}

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, base+"/gallery", strings.NewReader(
		`{"asset_id":`+strconv.FormatUint(uint64(assetID), 10)+`,"title":"标题","image_type":1,"is_spoiler":true}`,
	))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessTokenFor(t, admin))
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var created struct {
		Code int                   `json:"code"`
		Data dto.GalleryImageData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Data.ID == 0 || !created.Data.IsSpoiler || created.Data.URL == "" {
		t.Fatalf("unexpected create payload: %+v", created.Data)
	}

	res = httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet,
		"/api/v1/galgames/"+strconv.FormatUint(uint64(published.ID), 10)+"/gallery", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("public list: expected 200, got %d", res.Code)
	}
	var listed struct {
		Code int                `json:"code"`
		Data dto.GalleryListData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listed.Data.Total != 1 || len(listed.Data.Items) != 1 || listed.Data.Items[0].IsSpoiler != true {
		t.Fatalf("unexpected list payload: %+v", listed.Data)
	}

	res = httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet,
		"/api/v1/galgames/"+strconv.FormatUint(uint64(pending.ID), 10)+"/gallery", nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("public list of pending galgame: expected 404, got %d", res.Code)
	}

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet,
		"/api/v1/admin/galgames/"+strconv.FormatUint(uint64(pending.ID), 10)+"/gallery", nil)
	req.Header.Set("Authorization", "Bearer "+accessTokenFor(t, admin))
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("admin list of pending galgame: expected 200, got %d", res.Code)
	}

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch,
		base+"/gallery/"+strconv.FormatUint(uint64(created.Data.ID), 10),
		strings.NewReader(`{"is_spoiler":false}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessTokenFor(t, admin))
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d body=%s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, base+"/gallery/order",
		strings.NewReader(`{"ids":[`+strconv.FormatUint(uint64(created.Data.ID), 10)+`]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessTokenFor(t, admin))
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("reorder: expected 200, got %d body=%s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, base+"/gallery/order",
		strings.NewReader(`{"ids":[999999]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessTokenFor(t, admin))
	router.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("reorder with unknown id: expected 400, got %d", res.Code)
	}

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete,
		"/api/v1/admin/galgames/"+strconv.FormatUint(uint64(pending.ID), 10)+
			"/gallery/"+strconv.FormatUint(uint64(created.Data.ID), 10), nil)
	req.Header.Set("Authorization", "Bearer "+accessTokenFor(t, admin))
	router.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("cross-galgame delete: expected 404, got %d", res.Code)
	}

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete,
		base+"/gallery/"+strconv.FormatUint(uint64(created.Data.ID), 10), nil)
	req.Header.Set("Authorization", "Bearer "+accessTokenFor(t, admin))
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
}
