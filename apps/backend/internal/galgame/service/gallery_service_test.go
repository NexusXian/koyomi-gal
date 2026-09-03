package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"
	imageModel "backend/internal/image/model"
	imageRepo "backend/internal/image/repository"
	imageService "backend/internal/image/service"
	"backend/internal/testutil"

	"gorm.io/gorm"
)

func newGalleryTestService(t *testing.T) (*GalleryService, *gorm.DB, uint) {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	galleryRepo := repository.NewGalleryRepository(db)
	svc := NewGalleryService(
		repository.NewGalgameRepository(db),
		galleryRepo,
		imageService.NewImageAssetService(
			imageRepo.NewImageAssetRepository(db),
			nil,
			nil,
			nil,
			"https://img.example.com",
		),
	)
	return svc, db, testutil.CreateUser(t, db, "gallery-user")
}

func createGalleryAsset(t *testing.T, db *gorm.DB, mime string) uint {
	t.Helper()
	var id uint
	err := db.Raw(`
INSERT INTO image_assets (object_key, original_name, mime_type, extension, size, category, status, created_at, updated_at)
VALUES (?, 'test.webp', ?, 'webp', 1024, 'galgames', 1, NOW(), NOW())
RETURNING id
`, fmt.Sprintf("galgames/10001/2026/09/%d.webp", testAssetSeq()), mime).Scan(&id).Error
	if err != nil {
		t.Fatalf("create test image asset: %v", err)
	}
	return id
}

var galleryAssetSeq int

func testAssetSeq() int {
	galleryAssetSeq++
	return galleryAssetSeq
}

func createGalleryGalgame(t *testing.T, db *gorm.DB, slug string, status int16) uint {
	t.Helper()
	var id uint
	err := db.Raw(`
INSERT INTO galgames (title, original_title, romaji_title, slug, description, cover_url, banner_url, age_rating, status, rating_average, rating_count, favorite_count, resource_count, post_count, created_at, updated_at)
VALUES (?, '', '', ?, '', '', '', 0, ?, 0, 0, 0, 0, 0, NOW(), NOW())
RETURNING id
`, slug, slug, status).Scan(&id).Error
	if err != nil {
		t.Fatalf("create test galgame %s: %v", slug, err)
	}
	return id
}

func TestGalleryListScoping(t *testing.T) {
	svc, db, actor := newGalleryTestService(t)
	ctx := context.Background()

	published := createGalleryGalgame(t, db, "gallery-published", model.GalgameStatusPublished)
	pending := createGalleryGalgame(t, db, "gallery-pending", model.GalgameStatusPending)

	asset := createGalleryAsset(t, db, "image/webp")
	if _, err := svc.CreateGalleryImage(ctx, published, actor, &dto.CreateGalleryImageRequest{
		AssetID: asset, Title: "截图一", ImageType: model.GalleryImageTypeScreenshot,
	}); err != nil {
		t.Fatalf("create gallery image: %v", err)
	}

	data, err := svc.ListPublishedGallery(ctx, published)
	if err != nil || data.Total != 1 || len(data.Items) != 1 {
		t.Fatalf("expected 1 published gallery item, got total=%d items=%d err=%v", data.Total, len(data.Items), err)
	}
	if !strings.HasPrefix(data.Items[0].URL, "https://img.example.com/galgames/") {
		t.Fatalf("unexpected url: %s", data.Items[0].URL)
	}
	if data.Items[0].AssetID != asset || data.Items[0].SortOrder != 1 || data.Items[0].Title != "截图一" {
		t.Fatalf("unexpected item payload: %+v", data.Items[0])
	}

	if _, err := svc.ListPublishedGallery(ctx, pending); !errors.Is(err, ErrGalleryGalgameNotFound) {
		t.Fatalf("pending galgame must 404 on public gallery, got %v", err)
	}
	if _, err := svc.ListPublishedGallery(ctx, 999999); !errors.Is(err, ErrGalleryGalgameNotFound) {
		t.Fatalf("expected ErrGalleryGalgameNotFound, got %v", err)
	}

	adminData, err := svc.ListAdminGallery(ctx, pending)
	if err != nil || adminData.Total != 0 {
		t.Fatalf("admin listing must work for pending galgame, got %+v err=%v", adminData, err)
	}
	if _, err := svc.ListAdminGallery(ctx, 999999); !errors.Is(err, ErrGalleryGalgameNotFound) {
		t.Fatalf("expected ErrGalleryGalgameNotFound for admin listing, got %v", err)
	}
}

func TestGalleryCreateValidation(t *testing.T) {
	svc, db, actor := newGalleryTestService(t)
	ctx := context.Background()
	galgameID := createGalleryGalgame(t, db, "gallery-create", model.GalgameStatusPublished)

	if _, err := svc.CreateGalleryImage(ctx, 999999, actor, &dto.CreateGalleryImageRequest{AssetID: 1}); !errors.Is(err, ErrGalleryGalgameNotFound) {
		t.Fatalf("expected ErrGalleryGalgameNotFound, got %v", err)
	}
	if _, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{AssetID: 999999}); !errors.Is(err, ErrGalleryAssetNotFound) {
		t.Fatalf("expected ErrGalleryAssetNotFound, got %v", err)
	}

	pdfAsset := createGalleryAsset(t, db, "application/pdf")
	if _, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{AssetID: pdfAsset}); !errors.Is(err, ErrGalleryAssetNotFound) {
		t.Fatalf("expected ErrGalleryAssetNotFound for non-image asset, got %v", err)
	}

	pendingAsset := createGalleryAsset(t, db, "image/webp")
	if err := db.Exec("UPDATE image_assets SET status = ? WHERE id = ?", imageModel.ImageStatusPending, pendingAsset).Error; err != nil {
		t.Fatalf("mark asset pending: %v", err)
	}
	if _, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{AssetID: pendingAsset}); !errors.Is(err, ErrGalleryAssetNotFound) {
		t.Fatalf("expected ErrGalleryAssetNotFound for inactive asset, got %v", err)
	}

	asset := createGalleryAsset(t, db, "image/webp")
	created, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{
		AssetID: asset, Title: "第一张", ImageType: model.GalleryImageTypeCG, IsSpoiler: true,
	})
	if err != nil || created.ID == 0 || created.SortOrder != 1 || !created.IsSpoiler || created.ImageType != model.GalleryImageTypeCG {
		t.Fatalf("unexpected create result: %+v err=%v", created, err)
	}

	if _, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{AssetID: asset}); !errors.Is(err, ErrGalleryAssetDuplicate) {
		t.Fatalf("expected ErrGalleryAssetDuplicate, got %v", err)
	}
}

func TestGalleryLimitAndAppendOrder(t *testing.T) {
	svc, db, actor := newGalleryTestService(t)
	ctx := context.Background()
	galgameID := createGalleryGalgame(t, db, "gallery-limit", model.GalgameStatusPublished)

	for i := 0; i < MaxGalleryImages; i++ {
		asset := createGalleryAsset(t, db, "image/webp")
		if _, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{AssetID: asset}); err != nil {
			t.Fatalf("create gallery image %d: %v", i, err)
		}
	}

	extra := createGalleryAsset(t, db, "image/webp")
	if _, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{AssetID: extra}); !errors.Is(err, ErrGalleryLimitExceeded) {
		t.Fatalf("expected ErrGalleryLimitExceeded, got %v", err)
	}

	data, err := svc.ListAdminGallery(ctx, galgameID)
	if err != nil || data.Total != MaxGalleryImages {
		t.Fatalf("expected %d images, got %d err=%v", MaxGalleryImages, data.Total, err)
	}
	for i, item := range data.Items {
		if item.SortOrder != i+1 {
			t.Fatalf("expected append order %d at index %d, got %d", i+1, i, item.SortOrder)
		}
	}
}

func TestGalleryUpdateDelete(t *testing.T) {
	svc, db, actor := newGalleryTestService(t)
	ctx := context.Background()
	galgameID := createGalleryGalgame(t, db, "gallery-update", model.GalgameStatusPublished)
	otherID := createGalleryGalgame(t, db, "gallery-update-other", model.GalgameStatusPublished)

	asset := createGalleryAsset(t, db, "image/webp")
	created, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{AssetID: asset})
	if err != nil {
		t.Fatalf("create gallery image: %v", err)
	}

	newTitle := "新标题"
	newSpoiler := true
	updated, err := svc.UpdateGalleryImage(ctx, galgameID, created.ID, &dto.UpdateGalleryImageRequest{
		Title: &newTitle, IsSpoiler: &newSpoiler, ImageType: &[]int16{model.GalleryImageTypePromotional}[0],
	})
	if err != nil || updated.Title != newTitle || !updated.IsSpoiler || updated.ImageType != model.GalleryImageTypePromotional {
		t.Fatalf("unexpected update result: %+v err=%v", updated, err)
	}

	if _, err := svc.UpdateGalleryImage(ctx, otherID, created.ID, &dto.UpdateGalleryImageRequest{Title: &newTitle}); !errors.Is(err, ErrGalleryImageNotFound) {
		t.Fatalf("cross-galgame update must fail, got %v", err)
	}
	if _, err := svc.UpdateGalleryImage(ctx, galgameID, 999999, &dto.UpdateGalleryImageRequest{Title: &newTitle}); !errors.Is(err, ErrGalleryImageNotFound) {
		t.Fatalf("expected ErrGalleryImageNotFound, got %v", err)
	}

	if err := svc.DeleteGalleryImage(ctx, otherID, created.ID); !errors.Is(err, ErrGalleryImageNotFound) {
		t.Fatalf("cross-galgame delete must fail, got %v", err)
	}
	if err := svc.DeleteGalleryImage(ctx, galgameID, created.ID); err != nil {
		t.Fatalf("delete gallery image: %v", err)
	}
	data, err := svc.ListAdminGallery(ctx, galgameID)
	if err != nil || data.Total != 0 {
		t.Fatalf("expected empty gallery after delete, got %d err=%v", data.Total, err)
	}

	// Deleting the relation must not touch the asset itself.
	var count int64
	if err := db.Table("image_assets").Where("id = ?", asset).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("asset must survive gallery deletion, count=%d err=%v", count, err)
	}
}

func TestGalleryReorder(t *testing.T) {
	svc, db, actor := newGalleryTestService(t)
	ctx := context.Background()
	galgameID := createGalleryGalgame(t, db, "gallery-reorder", model.GalgameStatusPublished)
	otherID := createGalleryGalgame(t, db, "gallery-reorder-other", model.GalgameStatusPublished)

	var ids []uint
	for i := 0; i < 3; i++ {
		asset := createGalleryAsset(t, db, "image/webp")
		created, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{AssetID: asset})
		if err != nil {
			t.Fatalf("create gallery image %d: %v", i, err)
		}
		ids = append(ids, created.ID)
	}

	otherAsset := createGalleryAsset(t, db, "image/webp")
	otherCreated, err := svc.CreateGalleryImage(ctx, otherID, actor, &dto.CreateGalleryImageRequest{AssetID: otherAsset})
	if err != nil {
		t.Fatalf("create other gallery image: %v", err)
	}

	if err := svc.ReorderGallery(ctx, galgameID, &dto.ReorderGalleryRequest{IDs: []uint{ids[2], ids[0], ids[1]}}); err != nil {
		t.Fatalf("reorder gallery: %v", err)
	}
	data, err := svc.ListAdminGallery(ctx, galgameID)
	if err != nil {
		t.Fatalf("list reordered gallery: %v", err)
	}
	for i, item := range data.Items {
		if item.ID != []uint{ids[2], ids[0], ids[1]}[i] || item.SortOrder != i {
			t.Fatalf("unexpected order at %d: %+v", i, item)
		}
	}

	if err := svc.ReorderGallery(ctx, galgameID, &dto.ReorderGalleryRequest{IDs: []uint{ids[0], ids[1]}}); !errors.Is(err, ErrGalleryInvalidReorder) {
		t.Fatalf("partial id set must fail, got %v", err)
	}
	if err := svc.ReorderGallery(ctx, galgameID, &dto.ReorderGalleryRequest{IDs: []uint{ids[0], ids[1], otherCreated.ID}}); !errors.Is(err, ErrGalleryInvalidReorder) {
		t.Fatalf("cross-galgame id must fail, got %v", err)
	}
	if err := svc.ReorderGallery(ctx, galgameID, &dto.ReorderGalleryRequest{IDs: []uint{ids[0], ids[1], 999999}}); !errors.Is(err, ErrGalleryInvalidReorder) {
		t.Fatalf("unknown id must fail, got %v", err)
	}
	if err := svc.ReorderGallery(ctx, galgameID, &dto.ReorderGalleryRequest{IDs: []uint{ids[0], ids[0], ids[1]}}); !errors.Is(err, ErrGalleryInvalidReorder) {
		t.Fatalf("duplicate id must fail, got %v", err)
	}
	if err := svc.ReorderGallery(ctx, 999999, &dto.ReorderGalleryRequest{IDs: ids}); !errors.Is(err, ErrGalleryGalgameNotFound) {
		t.Fatalf("expected ErrGalleryGalgameNotFound, got %v", err)
	}
}
