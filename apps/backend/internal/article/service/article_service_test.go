package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend/internal/article/dto"
	"backend/internal/article/model"
	articleRepo "backend/internal/article/repository"
	rbacDTO "backend/internal/rbac/dto"
	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	"backend/internal/testutil"
)

func TestUpdateArticlePermissionSplit(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	ctx := context.Background()
	rbacRepository := rbacRepo.NewRBACRepository(db)
	rbac := rbacService.NewRBACService(rbacRepository)
	if err := rbac.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed permissions: %v", err)
	}

	updater := testutil.CreateUser(t, db, "article-updater")
	publisher := testutil.CreateUser(t, db, "article-publisher")
	nobody := testutil.CreateUser(t, db, "article-nobody")
	grantPermission(t, ctx, rbac, rbacRepository, updater, PermissionArticleUpdate)
	grantPermission(t, ctx, rbac, rbacRepository, publisher, PermissionArticlePublish)

	articles := articleRepo.NewArticleRepository(db)
	svc := NewArticleService(articles, rbac, nil)
	article := &model.Article{Title: "title", Content: "content", Type: model.TypeNews}
	if err := articles.Create(ctx, article); err != nil {
		t.Fatalf("create draft: %v", err)
	}

	published := true
	updated, err := svc.Update(ctx, publisher, article.ID, &dto.UpdateArticleRequest{
		Title: "title", Content: "content", Type: model.TypeNews, IsPublished: &published,
	})
	if err != nil || updated.PublishedAt == nil {
		t.Fatalf("publisher should publish unchanged content: article=%+v err=%v", updated, err)
	}
	originalPublishedAt := updated.PublishedAt
	rescheduledAt := originalPublishedAt.Add(-time.Hour)
	_, err = svc.Update(ctx, updater, article.ID, &dto.UpdateArticleRequest{
		Title: "title", Content: "content", Type: model.TypeNews,
		IsPublished: &published, PublishedAt: &rescheduledAt,
	})
	if !errors.Is(err, ErrArticleForbidden) {
		t.Fatalf("updater should not change publication time, got %v", err)
	}

	updated, err = svc.Update(ctx, updater, article.ID, &dto.UpdateArticleRequest{
		Title: "title", Content: "updated content", Type: model.TypeNews, IsPublished: &published,
	})
	if err != nil || updated.PublishedAt == nil {
		t.Fatalf("updater should edit without re-publish permission: article=%+v err=%v", updated, err)
	}

	_, err = svc.Update(ctx, nobody, article.ID, &dto.UpdateArticleRequest{
		Title: "title", Content: "updated content", Type: model.TypeNews,
		IsPublished: &published, PublishedAt: updated.PublishedAt,
	})
	if !errors.Is(err, ErrArticleForbidden) {
		t.Fatalf("no-op update without permission should be forbidden, got %v", err)
	}

	draft := false
	updated, err = svc.Update(ctx, publisher, article.ID, &dto.UpdateArticleRequest{
		Title: "title", Content: "updated content", Type: model.TypeNews, IsPublished: &draft,
	})
	if err != nil || !sameTime(updated.PublishedAt, originalPublishedAt) {
		t.Fatalf("publisher should unpublish without losing publication time: article=%+v err=%v", updated, err)
	}

	_, err = svc.Update(ctx, publisher, article.ID, &dto.UpdateArticleRequest{
		Title: "changed", Content: "updated content", Type: model.TypeNews, IsPublished: &draft,
	})
	if !errors.Is(err, ErrArticleForbidden) {
		t.Fatalf("publisher should not edit content, got %v", err)
	}
}

func TestPublicationTime(t *testing.T) {
	original := time.Date(2026, time.August, 31, 10, 15, 42, 0, time.UTC)
	if got := publicationTime(false, nil, &original); !sameTime(got, &original) {
		t.Fatalf("unpublish should preserve publication time: got=%v", got)
	}
	if got := publicationTime(true, nil, &original); !sameTime(got, &original) {
		t.Fatalf("republish should reuse publication time: got=%v", got)
	}
}

func grantPermission(
	t *testing.T,
	ctx context.Context,
	rbac *rbacService.RBACService,
	repository *rbacRepo.RBACRepository,
	userID uint,
	permissionCode string,
) {
	t.Helper()
	permission, err := repository.FindPermissionByCode(ctx, permissionCode)
	if err != nil || permission == nil {
		t.Fatalf("find permission %s: permission=%+v err=%v", permissionCode, permission, err)
	}
	role, err := rbac.CreateRole(ctx, &rbacDTO.CreateRoleRequest{Name: permissionCode, Code: "test_" + permission.Action})
	if err != nil {
		t.Fatalf("create role for %s: %v", permissionCode, err)
	}
	if err := rbac.SetRolePermissions(ctx, role.ID, []int64{permission.ID}); err != nil {
		t.Fatalf("grant permission %s: %v", permissionCode, err)
	}
	if err := rbac.AssignRoleToUser(ctx, userID, role.ID); err != nil {
		t.Fatalf("assign role for %s: %v", permissionCode, err)
	}
}
