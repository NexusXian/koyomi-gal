package service

import (
	"context"
	"errors"
	"testing"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
)

func TestCatalogDetailEndpoints(t *testing.T) {
	svc, _, _ := newCatalogTestService(t)
	ctx := context.Background()

	developer := createTestDeveloper(t, svc, "detail-developer")
	found, err := svc.GetDeveloper(ctx, developer.ID)
	if err != nil || found.ID != developer.ID || found.Name != developer.Name {
		t.Fatalf("unexpected developer detail: %+v err=%v", found, err)
	}
	if _, err := svc.GetDeveloper(ctx, 999999); !errors.Is(err, ErrDeveloperNotFound) {
		t.Fatalf("expected ErrDeveloperNotFound, got %v", err)
	}

	tag := createTestTag(t, svc, "detail-tag")
	foundTag, err := svc.GetTag(ctx, tag.ID)
	if err != nil || foundTag.ID != tag.ID || foundTag.Name != tag.Name {
		t.Fatalf("unexpected tag detail: %+v err=%v", foundTag, err)
	}
	if _, err := svc.GetTag(ctx, 999999); !errors.Is(err, ErrTagNotFound) {
		t.Fatalf("expected ErrTagNotFound, got %v", err)
	}
}

func TestAdminGalgameQueries(t *testing.T) {
	svc, _, userID := newCatalogTestService(t)
	ctx := context.Background()

	pending := createTestGalgame(t, svc, userID, "admin-pending", nil, nil, "", 0,
		model.GalgameStatusPending)
	published := createTestGalgame(t, svc, userID, "admin-published", nil, nil, "", 0,
		model.GalgameStatusPublished)
	hidden := createTestGalgame(t, svc, userID, "admin-hidden", nil, nil, "", 0,
		model.GalgameStatusHidden)

	galgames, total, _, _, err := svc.ListAllGalgames(ctx, &dto.AdminGalgameQuery{})
	if err != nil || total != 3 || len(galgames) != 3 {
		t.Fatalf("expected all 3 galgames, got total=%d items=%d err=%v", total, len(galgames), err)
	}

	statusPending := model.GalgameStatusPending
	galgames, total, _, _, err = svc.ListAllGalgames(ctx, &dto.AdminGalgameQuery{Status: &statusPending})
	if err != nil || total != 1 || len(galgames) != 1 || galgames[0].ID != pending.ID {
		t.Fatalf("expected only pending galgame, got total=%d items=%d err=%v", total, len(galgames), err)
	}

	keyword := "admin-hidden"
	galgames, total, _, _, err = svc.ListAllGalgames(ctx, &dto.AdminGalgameQuery{Keyword: keyword})
	if err != nil || total != 1 || galgames[0].ID != hidden.ID {
		t.Fatalf("expected keyword match on hidden galgame, got total=%d err=%v", total, err)
	}

	detail, err := svc.GetGalgame(ctx, pending.ID)
	if err != nil || detail.ID != pending.ID || detail.Status != model.GalgameStatusPending {
		t.Fatalf("expected pending detail via admin query, got %+v err=%v", detail, err)
	}
	if _, err := svc.GetGalgame(ctx, 999999); !errors.Is(err, ErrGalgameNotFound) {
		t.Fatalf("expected ErrGalgameNotFound, got %v", err)
	}

	invalidStatus := int16(9)
	if _, _, _, _, err := svc.ListAllGalgames(ctx, &dto.AdminGalgameQuery{Status: &invalidStatus}); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
	if _, _, _, _, err := svc.ListAllGalgames(ctx, &dto.AdminGalgameQuery{Sort: "id"}); !errors.Is(err, ErrInvalidSort) {
		t.Fatalf("expected ErrInvalidSort, got %v", err)
	}

	publishedItems, total, _, _, err := svc.ListPublishedGalgames(ctx, &dto.GalgameQuery{})
	if err != nil || total != 1 || publishedItems[0].ID != published.ID {
		t.Fatalf("public listing must stay published-only, got total=%d err=%v", total, err)
	}
}
