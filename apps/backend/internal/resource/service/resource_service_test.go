package service

import (
	"context"
	"errors"
	"testing"

	galgameDTO "backend/internal/galgame/dto"
	galgameRepo "backend/internal/galgame/repository"
	galgameService "backend/internal/galgame/service"
	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	"backend/internal/resource/dto"
	"backend/internal/resource/model"
	resourceRepo "backend/internal/resource/repository"
	"backend/internal/testutil"

	"gorm.io/gorm"
)

type resourceTestEnv struct {
	resources *ResourceService
	reports   *ReportService
	catalog   *galgameService.CatalogService
	rbac      *rbacService.RBACService
	db        *gorm.DB
}

func newResourceTestEnv(t *testing.T) *resourceTestEnv {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	galgameRepository := galgameRepo.NewGalgameRepository(db)
	rbacSvc := rbacService.NewRBACService(rbacRepo.NewRBACRepository(db))
	if err := rbacSvc.SeedDefaults(context.Background()); err != nil {
		t.Fatalf("seed rbac defaults: %v", err)
	}
	env := &resourceTestEnv{
		catalog: galgameService.NewCatalogService(
			galgameRepository,
			galgameRepo.NewDeveloperRepository(db),
			galgameRepo.NewTagRepository(db),
		),
		rbac: rbacSvc,
		db:   db,
	}
	env.resources = NewResourceService(
		resourceRepo.NewResourceRepository(db),
		galgameRepository,
		rbacSvc,
	)
	env.reports = NewReportService(
		resourceRepo.NewReportRepository(db),
		resourceRepo.NewResourceRepository(db),
	)
	return env
}

func (e *resourceTestEnv) createPublishedGalgame(t *testing.T, userID uint, title string) uint {
	t.Helper()
	galgame, err := e.catalog.CreateGalgame(context.Background(), userID, &galgameDTO.CreateGalgameRequest{
		Title:  title,
		Slug:   title,
		Status: 1,
	})
	if err != nil {
		t.Fatalf("create galgame %s: %v", title, err)
	}
	return galgame.ID
}

func (e *resourceTestEnv) createResource(
	t *testing.T,
	uploaderID, galgameID uint,
	title string,
	status int16,
	links ...string,
) *model.Resource {
	t.Helper()
	requestStatus := status
	resource, err := e.resources.CreateResource(context.Background(), uploaderID, &dto.CreateResourceRequest{
		GalgameID: galgameID,
		Title:     title,
		Type:      model.ResourceTypeGame,
		Status:    &requestStatus,
		Links:     links,
	})
	if err != nil {
		t.Fatalf("create resource %s: %v", title, err)
	}
	return resource
}

func (e *resourceTestEnv) resourceCount(t *testing.T, galgameID uint) int64 {
	t.Helper()
	var count int64
	if err := e.db.Raw(
		"SELECT resource_count FROM galgames WHERE id = ?", galgameID,
	).Scan(&count).Error; err != nil {
		t.Fatalf("read resource count: %v", err)
	}
	return count
}

func statusPtr(status int16) *int16 { return &status }

func TestResourceCreateIncrementsCount(t *testing.T) {
	env := newResourceTestEnv(t)
	ctx := context.Background()
	uploader := testutil.CreateUser(t, env.db, "resource-uploader")
	galgameID := env.createPublishedGalgame(t, uploader, "resource-create-game")

	first := env.createResource(t, uploader, galgameID, "patch", model.ResourceStatusPublished,
		"https://example.com/a", "https://example.com/b")
	if first.GalgameID != galgameID || first.UploaderID == nil || *first.UploaderID != uploader {
		t.Fatalf("unexpected created resource: %+v", first)
	}
	if len(first.Links) != 2 || first.Links[0].URL != "https://example.com/a" {
		t.Fatalf("unexpected links: %+v", first.Links)
	}
	if count := env.resourceCount(t, galgameID); count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}

	env.createResource(t, uploader, galgameID, "save", model.ResourceStatusPending,
		"https://example.com/c")
	if count := env.resourceCount(t, galgameID); count != 2 {
		t.Fatalf("pending resource must also count, expected 2, got %d", count)
	}

	if _, err := env.resources.CreateResource(ctx, uploader, &dto.CreateResourceRequest{
		GalgameID: galgameID,
		Title:     " ",
		Links:     []string{"https://example.com/x"},
	}); !errors.Is(err, ErrInvalidResourceInput) {
		t.Fatalf("expected ErrInvalidResourceInput, got %v", err)
	}
	if _, err := env.resources.CreateResource(ctx, uploader, &dto.CreateResourceRequest{
		GalgameID: galgameID,
		Title:     "bad type",
		Type:      7,
		Links:     []string{"https://example.com/x"},
	}); !errors.Is(err, ErrInvalidResourceType) {
		t.Fatalf("expected ErrInvalidResourceType, got %v", err)
	}
	if _, err := env.resources.CreateResource(ctx, uploader, &dto.CreateResourceRequest{
		GalgameID: galgameID,
		Title:     "bad status",
		Status:    statusPtr(4),
		Links:     []string{"https://example.com/x"},
	}); !errors.Is(err, ErrInvalidResourceStatus) {
		t.Fatalf("expected ErrInvalidResourceStatus, got %v", err)
	}
	if _, err := env.resources.CreateResource(ctx, uploader, &dto.CreateResourceRequest{
		GalgameID: galgameID,
		Title:     "no links",
		Links:     []string{"  "},
	}); !errors.Is(err, ErrEmptyResourceLinks) {
		t.Fatalf("expected ErrEmptyResourceLinks, got %v", err)
	}
	if _, err := env.resources.CreateResource(ctx, uploader, &dto.CreateResourceRequest{
		GalgameID: 999999,
		Title:     "unknown galgame",
		Links:     []string{"https://example.com/x"},
	}); !errors.Is(err, ErrGalgameNotFound) {
		t.Fatalf("expected ErrGalgameNotFound, got %v", err)
	}
	if count := env.resourceCount(t, galgameID); count != 2 {
		t.Fatalf("failed creates must not change count, got %d", count)
	}
}

func TestResourcePublicQueriesOnlyPublished(t *testing.T) {
	env := newResourceTestEnv(t)
	ctx := context.Background()
	uploader := testutil.CreateUser(t, env.db, "resource-reader")
	galgameID := env.createPublishedGalgame(t, uploader, "resource-public-game")

	published := env.createResource(t, uploader, galgameID, "published", model.ResourceStatusPublished,
		"https://example.com/p")
	pending := env.createResource(t, uploader, galgameID, "pending", model.ResourceStatusPending,
		"https://example.com/q")
	env.createResource(t, uploader, galgameID, "hidden", model.ResourceStatusHidden,
		"https://example.com/r")

	items, err := env.resources.ListPublishedByGalgame(ctx, galgameID)
	if err != nil {
		t.Fatalf("list published resources: %v", err)
	}
	if len(items) != 1 || items[0].ID != published.ID || len(items[0].Links) != 1 {
		t.Fatalf("expected only the published resource with links, got %+v", items)
	}

	detail, err := env.resources.GetPublishedResource(ctx, published.ID)
	if err != nil || detail.Title != "published" {
		t.Fatalf("expected published detail, got %+v err=%v", detail, err)
	}
	if _, err := env.resources.GetPublishedResource(ctx, pending.ID); !errors.Is(err, ErrResourceNotFound) {
		t.Fatalf("expected ErrResourceNotFound for pending detail, got %v", err)
	}

	pendingGalgame, err := env.catalog.CreateGalgame(ctx, uploader, &galgameDTO.CreateGalgameRequest{
		Title:  "resource-pending-game",
		Slug:   "resource-pending-game",
		Status: 0,
	})
	if err != nil {
		t.Fatalf("create pending galgame: %v", err)
	}
	if _, err := env.resources.ListPublishedByGalgame(ctx, pendingGalgame.ID); !errors.Is(err, ErrGalgameNotFound) {
		t.Fatalf("expected ErrGalgameNotFound for pending galgame, got %v", err)
	}
}

func TestResourceOwnershipAndPermission(t *testing.T) {
	env := newResourceTestEnv(t)
	ctx := context.Background()
	owner := testutil.CreateUser(t, env.db, "resource-owner")
	stranger := testutil.CreateUser(t, env.db, "resource-stranger")
	admin := testutil.CreateUser(t, env.db, "resource-admin")
	if err := env.rbac.AssignRoleByCode(ctx, admin, rbacService.RoleCodeAdmin); err != nil {
		t.Fatalf("assign admin role: %v", err)
	}
	galgameID := env.createPublishedGalgame(t, owner, "resource-permission-game")
	resource := env.createResource(t, owner, galgameID, "owned", model.ResourceStatusPublished,
		"https://example.com/1")

	updated, err := env.resources.UpdateResource(ctx, owner, resource.ID, &dto.UpdateResourceRequest{
		Title:  "owned-v2",
		Type:   model.ResourceTypePatch,
		Status: statusPtr(model.ResourceStatusPublished),
		Links:  []string{"https://example.com/2", "https://example.com/3"},
	})
	if err != nil {
		t.Fatalf("owner update: %v", err)
	}
	if updated.Title != "owned-v2" || updated.Type != model.ResourceTypePatch || len(updated.Links) != 2 {
		t.Fatalf("unexpected updated resource: %+v", updated)
	}
	if count := env.resourceCount(t, galgameID); count != 1 {
		t.Fatalf("update must not change count, got %d", count)
	}

	if _, err := env.resources.UpdateResource(ctx, stranger, resource.ID, &dto.UpdateResourceRequest{
		Title:  "hijack",
		Type:   model.ResourceTypeGame,
		Status: statusPtr(model.ResourceStatusPublished),
		Links:  []string{"https://example.com/evil"},
	}); !errors.Is(err, ErrForbiddenResource) {
		t.Fatalf("expected ErrForbiddenResource for stranger update, got %v", err)
	}
	if err := env.resources.DeleteResource(ctx, stranger, resource.ID); !errors.Is(err, ErrForbiddenResource) {
		t.Fatalf("expected ErrForbiddenResource for stranger delete, got %v", err)
	}

	if _, err := env.resources.UpdateResource(ctx, admin, resource.ID, &dto.UpdateResourceRequest{
		Title:  "admin-edit",
		Type:   model.ResourceTypeGuide,
		Status: statusPtr(model.ResourceStatusHidden),
		Links:  []string{"https://example.com/admin"},
	}); err != nil {
		t.Fatalf("admin update: %v", err)
	}

	if err := env.resources.DeleteResource(ctx, owner, resource.ID); err != nil {
		t.Fatalf("owner delete: %v", err)
	}
	if count := env.resourceCount(t, galgameID); count != 0 {
		t.Fatalf("expected count 0 after delete, got %d", count)
	}
	var linkCount int64
	if err := env.db.Raw(
		"SELECT COUNT(*) FROM resource_links WHERE resource_id = ?", resource.ID,
	).Scan(&linkCount).Error; err != nil || linkCount != 0 {
		t.Fatalf("expected links to cascade, got %d err=%v", linkCount, err)
	}
	if err := env.resources.DeleteResource(ctx, owner, resource.ID); !errors.Is(err, ErrResourceNotFound) {
		t.Fatalf("expected ErrResourceNotFound on second delete, got %v", err)
	}
	if count := env.resourceCount(t, galgameID); count != 0 {
		t.Fatalf("expected count to stay 0, got %d", count)
	}
}

func TestResourceWritesRollbackOnError(t *testing.T) {
	env := newResourceTestEnv(t)
	ctx := context.Background()
	uploader := testutil.CreateUser(t, env.db, "resource-rollback")
	galgameID := env.createPublishedGalgame(t, uploader, "resource-rollback-game")

	injected := errors.New("injected failure")
	err := env.db.Transaction(func(tx *gorm.DB) error {
		resources := resourceRepo.NewResourceRepository(tx)
		resource := &model.Resource{
			GalgameID:  galgameID,
			UploaderID: &uploader,
			Title:      "rollback",
			Type:       model.ResourceTypeGame,
			Status:     model.ResourceStatusPublished,
		}
		if err := resources.Create(ctx, resource); err != nil {
			return err
		}
		if err := resources.CreateLinks(ctx, resource.ID, []string{"https://example.com/r"}); err != nil {
			return err
		}
		if err := resources.IncrementResourceCount(ctx, galgameID); err != nil {
			return err
		}
		return injected
	})
	if !errors.Is(err, injected) {
		t.Fatalf("expected injected failure, got %v", err)
	}
	if count := env.resourceCount(t, galgameID); count != 0 {
		t.Fatalf("expected count rolled back to 0, got %d", count)
	}
	var resourceCount int64
	if err := env.db.Raw("SELECT COUNT(*) FROM resources WHERE galgame_id = ?", galgameID).
		Scan(&resourceCount).Error; err != nil || resourceCount != 0 {
		t.Fatalf("expected resource rolled back, got %d err=%v", resourceCount, err)
	}
}
