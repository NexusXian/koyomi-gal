package service

import (
	"context"
	"errors"
	"testing"

	galgameDTO "backend/internal/galgame/dto"
	galgameRepo "backend/internal/galgame/repository"
	galgameService "backend/internal/galgame/service"
	notificationModel "backend/internal/notification/model"
	notificationRepo "backend/internal/notification/repository"
	notificationService "backend/internal/notification/service"
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
	contributionRepository := galgameRepo.NewContributionRepository(db, "https://img.example.com")
	contributionSvc := galgameService.NewContributionService(
		contributionRepository,
		galgameRepository,
		"https://img.example.com",
	)
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
	env.catalog.SetContributionService(contributionSvc)
	env.resources.SetContributionService(contributionSvc)
	env.reports = NewReportService(
		resourceRepo.NewReportRepository(db),
		resourceRepo.NewResourceRepository(db),
	)
	notificationSvc := notificationService.NewNotificationService(
		notificationRepo.NewNotificationRepository(db, "https://img.example.com"),
	)
	env.resources.SetNotificationDependencies(rbacSvc, notificationSvc)
	env.reports.SetNotificationDependencies(rbacSvc, notificationSvc)
	return env
}

func TestResourceContributionReviewLifecycle(t *testing.T) {
	env := newResourceTestEnv(t)
	ctx := context.Background()
	uploader := testutil.CreateUser(t, env.db, "resource-contribution-uploader")
	reviewer := testutil.CreateUser(t, env.db, "resource-contribution-reviewer")
	galgameID := env.createPublishedGalgame(t, uploader, "resource-contribution-game")
	pending := env.createResource(t, uploader, galgameID, "pending contribution", model.ResourceStatusPending, "https://example.com/pending")

	if countResourceContributions(t, env.db, pending.ID) != 0 {
		t.Fatal("pending resource created a contribution")
	}
	if _, err := env.resources.ReviewResource(ctx, pending.ID, &dto.ReviewResourceRequest{Status: model.ResourceStatusRejected}, reviewer); err != nil {
		t.Fatalf("reject resource: %v", err)
	}
	if countResourceContributions(t, env.db, pending.ID) != 0 {
		t.Fatal("rejected resource created a contribution")
	}

	approved := env.createResource(t, uploader, galgameID, "approved contribution", model.ResourceStatusPending, "https://example.com/approved")
	if _, err := env.resources.ReviewResource(ctx, approved.ID, &dto.ReviewResourceRequest{Status: model.ResourceStatusPublished}, reviewer); err != nil {
		t.Fatalf("approve resource: %v", err)
	}
	if _, err := env.resources.ReviewResource(ctx, approved.ID, &dto.ReviewResourceRequest{Status: model.ResourceStatusPublished}, reviewer); err != nil {
		t.Fatalf("repeat resource approval: %v", err)
	}
	if countResourceContributions(t, env.db, approved.ID) != 1 {
		t.Fatal("approved resource contribution was not idempotent")
	}
}

func countResourceContributions(t *testing.T, db *gorm.DB, resourceID uint) int64 {
	t.Helper()
	var count int64
	if err := db.Table("galgame_contributions").
		Where("source_type = ? AND source_id = ?", "resource", resourceID).
		Count(&count).Error; err != nil {
		t.Fatalf("count resource contributions: %v", err)
	}
	return count
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

func TestResourceAndReportNotifications(t *testing.T) {
	env := newResourceTestEnv(t)
	ctx := context.Background()
	uploader := testutil.CreateUser(t, env.db, "notification-uploader")
	reporter := testutil.CreateUser(t, env.db, "notification-reporter")
	reviewer := testutil.CreateUser(t, env.db, "notification-resource-reviewer")
	if err := env.rbac.AssignRoleByCode(ctx, reviewer, rbacService.RoleCodeAdmin); err != nil {
		t.Fatalf("assign reviewer role: %v", err)
	}
	galgameID := env.createPublishedGalgame(t, uploader, "notification-resource-game")

	published := env.createResource(t, uploader, galgameID, "publish resource", model.ResourceStatusPending, "https://example.com/publish")
	assertResourceNotificationCount(t, env.db, reviewer, notificationModel.TypeResourceSubmitted, 1)
	if _, err := env.resources.ReviewResource(ctx, published.ID, &dto.ReviewResourceRequest{Status: model.ResourceStatusPublished}, reviewer); err != nil {
		t.Fatalf("approve resource: %v", err)
	}
	assertResourceNotificationCount(t, env.db, uploader, notificationModel.TypeResourceApproved, 1)

	rejected := env.createResource(t, uploader, galgameID, "reject resource", model.ResourceStatusPending, "https://example.com/reject")
	if _, err := env.resources.ReviewResource(ctx, rejected.ID, &dto.ReviewResourceRequest{Status: model.ResourceStatusRejected}, reviewer); err != nil {
		t.Fatalf("reject resource: %v", err)
	}
	assertResourceNotificationCount(t, env.db, uploader, notificationModel.TypeResourceRejected, 1)

	hidden := env.createResource(t, uploader, galgameID, "hide resource", model.ResourceStatusPending, "https://example.com/hide")
	if _, err := env.resources.ReviewResource(ctx, hidden.ID, &dto.ReviewResourceRequest{Status: model.ResourceStatusHidden}, reviewer); err != nil {
		t.Fatalf("hide resource: %v", err)
	}
	assertResourceNotificationCount(t, env.db, uploader, notificationModel.TypeResourceHidden, 1)

	report, err := env.reports.Create(ctx, reporter, published.ID, &dto.CreateResourceReportRequest{Reason: model.ReportReasonInvalidLink})
	if err != nil {
		t.Fatalf("create report: %v", err)
	}
	assertResourceNotificationCount(t, env.db, reviewer, notificationModel.TypeResourceReported, 1)
	if _, err := env.reports.Handle(ctx, reviewer, report.ID, &dto.HandleResourceReportRequest{Status: model.ReportStatusResolved}); err != nil {
		t.Fatalf("resolve report: %v", err)
	}
	assertResourceNotificationCount(t, env.db, reporter, notificationModel.TypeReportResolved, 1)

	reportResource := env.createResource(t, uploader, galgameID, "reported rejected", model.ResourceStatusPublished, "https://example.com/report-reject")
	rejectedReport, err := env.reports.Create(ctx, reporter, reportResource.ID, &dto.CreateResourceReportRequest{Reason: model.ReportReasonOther})
	if err != nil {
		t.Fatalf("create rejected report: %v", err)
	}
	if _, err := env.reports.Handle(ctx, reviewer, rejectedReport.ID, &dto.HandleResourceReportRequest{Status: model.ReportStatusRejected}); err != nil {
		t.Fatalf("reject report: %v", err)
	}
	assertResourceNotificationCount(t, env.db, reporter, notificationModel.TypeReportRejected, 1)
}

func assertResourceNotificationCount(t *testing.T, db *gorm.DB, recipientID uint, notificationType notificationModel.NotificationType, want int64) {
	t.Helper()
	var count int64
	if err := db.Table("notifications").Where("recipient_id = ? AND type = ?", recipientID, notificationType).Count(&count).Error; err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if count != want {
		t.Fatalf("notification %s count: got %d want %d", notificationType, count, want)
	}
}

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

	items, total, page, limit, err := env.resources.ListPublishedByGalgame(ctx, galgameID, 0, 0)
	if err != nil {
		t.Fatalf("list published resources: %v", err)
	}
	if total != 1 || page != 1 || limit != 20 || len(items) != 1 || items[0].ID != published.ID || len(items[0].Links) != 1 {
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
	if _, _, _, _, err := env.resources.ListPublishedByGalgame(ctx, pendingGalgame.ID, 1, 20); !errors.Is(err, ErrGalgameNotFound) {
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
