package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	galgameDTO "backend/internal/galgame/dto"
	galgameRepo "backend/internal/galgame/repository"
	"backend/internal/middleware"
	novelRepo "backend/internal/novel/repository"
	rbacService "backend/internal/rbac/service"
	"backend/internal/resource/dto"
	resourceRepo "backend/internal/resource/repository"
	resourceService "backend/internal/resource/service"
	"backend/internal/testutil"

	"github.com/gin-gonic/gin"
)

func newReportTestRouter(t *testing.T) (*gin.Engine, *resourceHandlerEnv) {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	env := newResourceHandlerEnv(t)

	reportHandler := NewReportHandler(resourceService.NewReportService(
		resourceRepo.NewReportRepository(env.db),
		resourceRepo.NewResourceRepository(env.db),
	))
	auth := middleware.Auth(testAuthSecret)
	requirePermission := middleware.RequirePermission(env.rbac)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	protected := router.Group("/api/v1", auth)
	{
		protected.POST("/resources/:id/reports", reportHandler.CreateReport)
	}
	admin := router.Group("/api/v1/admin", auth)
	{
		admin.GET("/resource-reports", requirePermission("resource_report:list"), reportHandler.ListReports)
		admin.PUT("/resource-reports/:id/handle", requirePermission("resource_report:handle"), reportHandler.HandleReport)
	}
	return router, env
}

func TestReportEndpointsFlow(t *testing.T) {
	router, env := newReportTestRouter(t)
	ctx := context.Background()
	uploader := testutil.CreateUser(t, env.db, "report-http-uploader")
	reporter := testutil.CreateUser(t, env.db, "report-http-reporter")
	admin := testutil.CreateUser(t, env.db, "report-http-admin")
	if err := env.rbac.AssignRoleByCode(ctx, admin, rbacService.RoleCodeAdmin); err != nil {
		t.Fatalf("assign admin role: %v", err)
	}
	galgame, err := env.catalog.CreateGalgame(ctx, uploader, &galgameDTO.CreateGalgameRequest{
		Title: "report-http-game", Slug: "report-http-game", Status: 1,
	})
	if err != nil {
		t.Fatalf("create galgame: %v", err)
	}
	resource, err := resourceService.NewResourceService(
		resourceRepo.NewResourceRepository(env.db),
		galgameRepo.NewGalgameRepository(env.db),
		novelRepo.NewNovelRepository(env.db),
		env.rbac,
	).CreateResource(ctx, uploader, &dto.CreateResourceRequest{
		TargetType: "galgame",
		TargetID:   galgame.ID,
		Title:      "report-http-resource",
		Type:       1,
		Status:     func() *int16 { published := int16(1); return &published }(),
		Links:      []string{"https://example.com/report"},
	})
	if err != nil {
		t.Fatalf("create resource: %v", err)
	}

	reporterToken := accessTokenFor(t, reporter)
	adminToken := accessTokenFor(t, admin)

	res := doResourceRequest(router, http.MethodPost,
		"/api/v1/resources/"+itoa(resource.ID)+"/reports", "", map[string]any{
			"reason": 0,
		})
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("create report without token: expected 401, got %d", res.Code)
	}

	res = doResourceRequest(router, http.MethodPost,
		"/api/v1/resources/"+itoa(resource.ID)+"/reports", reporterToken, map[string]any{
			"reason":      0,
			"description": "link broken",
		})
	if res.Code != http.StatusOK {
		t.Fatalf("create report: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var created struct {
		Code int                    `json:"code"`
		Data dto.ResourceReportData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode report response: %v", err)
	}
	if created.Data.UserID != reporter || created.Data.Status != 0 {
		t.Fatalf("unexpected report data: %+v", created.Data)
	}

	res = doResourceRequest(router, http.MethodPost,
		"/api/v1/resources/"+itoa(resource.ID)+"/reports", reporterToken, map[string]any{"reason": 6})
	if res.Code != http.StatusConflict {
		t.Fatalf("duplicate report: expected 409, got %d", res.Code)
	}

	res = doResourceRequest(router, http.MethodGet, "/api/v1/admin/resource-reports", "", nil)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("admin report list without token: expected 401, got %d", res.Code)
	}
	res = doResourceRequest(router, http.MethodGet, "/api/v1/admin/resource-reports", reporterToken, nil)
	if res.Code != http.StatusForbidden {
		t.Fatalf("admin report list without permission: expected 403, got %d", res.Code)
	}

	res = doResourceRequest(router, http.MethodGet,
		"/api/v1/admin/resource-reports?status=0", adminToken, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("admin report list: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var listed struct {
		Code int                        `json:"code"`
		Data dto.ResourceReportListData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode report list response: %v", err)
	}
	if listed.Data.Total != 1 || len(listed.Data.Items) != 1 {
		t.Fatalf("expected 1 pending report, got %+v", listed.Data)
	}
	if listed.Data.Items[0].Resource == nil || listed.Data.Items[0].Resource.Title != "report-http-resource" {
		t.Fatalf("expected resource summary preloaded, got %+v", listed.Data.Items[0].Resource)
	}
	reportID := listed.Data.Items[0].ID

	res = doResourceRequest(router, http.MethodPut,
		"/api/v1/admin/resource-reports/"+itoa(reportID)+"/handle", adminToken, map[string]any{"status": 0})
	if res.Code != http.StatusBadRequest {
		t.Fatalf("invalid handle status: expected 400, got %d", res.Code)
	}

	res = doResourceRequest(router, http.MethodPut,
		"/api/v1/admin/resource-reports/"+itoa(reportID)+"/handle", adminToken, map[string]any{"status": 1})
	if res.Code != http.StatusOK {
		t.Fatalf("handle report: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var handled struct {
		Code int                    `json:"code"`
		Data dto.ResourceReportData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &handled); err != nil {
		t.Fatalf("decode handled report response: %v", err)
	}
	if handled.Data.Status != 1 || handled.Data.HandledBy == nil || *handled.Data.HandledBy != admin {
		t.Fatalf("unexpected handled report: %+v", handled.Data)
	}

	res = doResourceRequest(router, http.MethodPut,
		"/api/v1/admin/resource-reports/999999/handle", adminToken, map[string]any{"status": 1})
	if res.Code != http.StatusNotFound {
		t.Fatalf("handle unknown report: expected 404, got %d", res.Code)
	}
}

func itoa(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}
