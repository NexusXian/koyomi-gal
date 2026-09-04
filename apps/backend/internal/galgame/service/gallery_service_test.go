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

func assetPtr(id uint) *uint { return &id }

func TestGalleryListScoping(t *testing.T) {
	svc, db, actor := newGalleryTestService(t)
	ctx := context.Background()

	published := createGalleryGalgame(t, db, "gallery-published", model.GalgameStatusPublished)
	pending := createGalleryGalgame(t, db, "gallery-pending", model.GalgameStatusPending)

	asset := createGalleryAsset(t, db, "image/webp")
	created, err := svc.CreateGalleryImage(ctx, published, actor, &dto.CreateGalleryImageRequest{
		AssetID: assetPtr(asset), Title: "截图一", ImageType: model.GalleryImageTypeScreenshot,
	})
	if err != nil {
		t.Fatalf("create gallery image: %v", err)
	}
	if created.Status != model.GalleryImageStatusPending {
		t.Fatalf("new gallery image must start pending, got %d", created.Status)
	}

	// Pending images are invisible on the public gallery.
	data, err := svc.ListPublishedGallery(ctx, published)
	if err != nil || data.Total != 0 {
		t.Fatalf("pending image must not be publicly listed, got total=%d err=%v", data.Total, err)
	}

	adminData, err := svc.ListAdminGallery(ctx, published)
	if err != nil || adminData.Total != 1 || len(adminData.Items) != 1 {
		t.Fatalf("admin listing must show pending image, got total=%d err=%v", adminData.Total, err)
	}

	if _, err := svc.ReviewGalleryImages(ctx, ReviewGalleryImagesInput{
		IDs: []uint{created.ID}, Approve: true, AdminID: actor,
	}); err != nil {
		t.Fatalf("approve gallery image: %v", err)
	}

	data, err = svc.ListPublishedGallery(ctx, published)
	if err != nil || data.Total != 1 || len(data.Items) != 1 {
		t.Fatalf("expected 1 published gallery item, got total=%d items=%d err=%v", data.Total, len(data.Items), err)
	}
	if !strings.HasPrefix(data.Items[0].URL, "https://img.example.com/galgames/") {
		t.Fatalf("unexpected url: %s", data.Items[0].URL)
	}
	if data.Items[0].AssetID == nil || *data.Items[0].AssetID != asset ||
		data.Items[0].SortOrder != 1 || data.Items[0].Title != "截图一" {
		t.Fatalf("unexpected item payload: %+v", data.Items[0])
	}

	if _, err := svc.ListPublishedGallery(ctx, pending); !errors.Is(err, ErrGalleryGalgameNotFound) {
		t.Fatalf("pending galgame must 404 on public gallery, got %v", err)
	}
	if _, err := svc.ListPublishedGallery(ctx, 999999); !errors.Is(err, ErrGalleryGalgameNotFound) {
		t.Fatalf("expected ErrGalleryGalgameNotFound, got %v", err)
	}

	adminPending, err := svc.ListAdminGallery(ctx, pending)
	if err != nil || adminPending.Total != 0 {
		t.Fatalf("admin listing must work for pending galgame, got %+v err=%v", adminPending, err)
	}
	if _, err := svc.ListAdminGallery(ctx, 999999); !errors.Is(err, ErrGalleryGalgameNotFound) {
		t.Fatalf("expected ErrGalleryGalgameNotFound for admin listing, got %v", err)
	}
}

func TestGalleryCreateValidation(t *testing.T) {
	svc, db, actor := newGalleryTestService(t)
	ctx := context.Background()
	galgameID := createGalleryGalgame(t, db, "gallery-create", model.GalgameStatusPublished)

	if _, err := svc.CreateGalleryImage(ctx, 999999, actor, &dto.CreateGalleryImageRequest{AssetID: assetPtr(1)}); !errors.Is(err, ErrGalleryGalgameNotFound) {
		t.Fatalf("expected ErrGalleryGalgameNotFound, got %v", err)
	}
	if _, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{}); !errors.Is(err, ErrGalleryInvalidSource) {
		t.Fatalf("expected ErrGalleryInvalidSource for empty source, got %v", err)
	}
	if _, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{
		AssetID: assetPtr(1), ExternalURL: "https://example.com/a.jpg",
	}); !errors.Is(err, ErrGalleryInvalidSource) {
		t.Fatalf("expected ErrGalleryInvalidSource for both sources, got %v", err)
	}
	if _, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{AssetID: assetPtr(999999)}); !errors.Is(err, ErrGalleryAssetNotFound) {
		t.Fatalf("expected ErrGalleryAssetNotFound, got %v", err)
	}

	pdfAsset := createGalleryAsset(t, db, "application/pdf")
	if _, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{AssetID: assetPtr(pdfAsset)}); !errors.Is(err, ErrGalleryAssetNotFound) {
		t.Fatalf("expected ErrGalleryAssetNotFound for non-image asset, got %v", err)
	}

	pendingAsset := createGalleryAsset(t, db, "image/webp")
	if err := db.Exec("UPDATE image_assets SET status = ? WHERE id = ?", imageModel.ImageStatusPending, pendingAsset).Error; err != nil {
		t.Fatalf("mark asset pending: %v", err)
	}
	if _, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{AssetID: assetPtr(pendingAsset)}); !errors.Is(err, ErrGalleryAssetNotFound) {
		t.Fatalf("expected ErrGalleryAssetNotFound for inactive asset, got %v", err)
	}

	for _, invalid := range []string{
		"javascript:alert(1)",
		"data:image/png;base64,AAAA",
		"ftp://example.com/a.jpg",
		"example.com/a.jpg",
		"http://user:pass@example.com/a.jpg",
		"https://example.com/with space.jpg",
		"http://",
		"https://" + strings.Repeat("a", 2100),
	} {
		if _, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{
			ExternalURL: invalid,
		}); !errors.Is(err, ErrGalleryInvalidURL) {
			t.Fatalf("expected ErrGalleryInvalidURL for %q, got %v", invalid, err)
		}
	}

	external, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{
		ExternalURL: "https://source-a.com/images/123.jpg",
	})
	if err != nil || external.ID == 0 || external.Status != model.GalleryImageStatusPending {
		t.Fatalf("unexpected external create result: %+v err=%v", external, err)
	}
	if external.URL != "https://source-a.com/images/123.jpg" || external.SourceType != model.GallerySourceExternal {
		t.Fatalf("unexpected external payload: %+v", external)
	}
	if external.Width != nil || external.Height != nil {
		t.Fatalf("external image must not carry dimensions: %+v", external)
	}
	if _, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{
		ExternalURL: "https://source-a.com/images/123.jpg",
	}); !errors.Is(err, ErrGalleryURLDuplicate) {
		t.Fatalf("expected ErrGalleryURLDuplicate, got %v", err)
	}

	asset := createGalleryAsset(t, db, "image/webp")
	created, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{
		AssetID: assetPtr(asset), Title: "第一张", ImageType: model.GalleryImageTypeCG, IsSpoiler: true,
	})
	if err != nil || created.ID == 0 || created.SortOrder != 2 || !created.IsSpoiler || created.ImageType != model.GalleryImageTypeCG {
		t.Fatalf("unexpected create result: %+v err=%v", created, err)
	}

	if _, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{AssetID: assetPtr(asset)}); !errors.Is(err, ErrGalleryAssetDuplicate) {
		t.Fatalf("expected ErrGalleryAssetDuplicate, got %v", err)
	}
}

func TestGalleryLimitAndAppendOrder(t *testing.T) {
	svc, db, actor := newGalleryTestService(t)
	ctx := context.Background()
	galgameID := createGalleryGalgame(t, db, "gallery-limit", model.GalgameStatusPublished)

	var ids []uint
	for i := 0; i < MaxGalleryImages; i++ {
		created, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{
			ExternalURL: fmt.Sprintf("https://example.com/limit/%d.jpg", i),
		})
		if err != nil {
			t.Fatalf("create gallery image %d: %v", i, err)
		}
		ids = append(ids, created.ID)
	}

	extra := createGalleryAsset(t, db, "image/webp")
	if _, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{AssetID: assetPtr(extra)}); !errors.Is(err, ErrGalleryLimitExceeded) {
		t.Fatalf("expected ErrGalleryLimitExceeded, got %v", err)
	}
	if _, err := svc.BatchCreateGalleryImages(ctx, galgameID, actor, &dto.BatchCreateGalleryRequest{
		Items: []dto.BatchGalleryImageItem{{ExternalURL: "https://example.com/over-limit.jpg"}},
	}); !errors.Is(err, ErrGalleryLimitExceeded) {
		t.Fatalf("expected ErrGalleryLimitExceeded for batch, got %v", err)
	}

	// Rejected images no longer consume slots.
	if _, err := svc.ReviewGalleryImages(ctx, ReviewGalleryImagesInput{
		IDs: ids, Approve: false, Reason: "重置名额", AdminID: actor,
	}); err != nil {
		t.Fatalf("reject gallery images: %v", err)
	}
	if _, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{AssetID: assetPtr(extra)}); err != nil {
		t.Fatalf("create after rejecting all images: %v", err)
	}

	data, err := svc.ListAdminGallery(ctx, galgameID)
	if err != nil || data.Total != MaxGalleryImages+1 {
		t.Fatalf("expected %d images, got %d err=%v", MaxGalleryImages+1, data.Total, err)
	}
	for i, item := range data.Items[:MaxGalleryImages] {
		if item.SortOrder != i+1 {
			t.Fatalf("expected append order %d at index %d, got %d", i+1, i, item.SortOrder)
		}
	}
}

func TestGalleryBatchCreate(t *testing.T) {
	svc, db, actor := newGalleryTestService(t)
	ctx := context.Background()
	galgameID := createGalleryGalgame(t, db, "gallery-batch", model.GalgameStatusPublished)

	existing, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{
		ExternalURL: "https://source-a.com/existing.webp",
	})
	if err != nil {
		t.Fatalf("seed existing image: %v", err)
	}

	result, err := svc.BatchCreateGalleryImages(ctx, galgameID, actor, &dto.BatchCreateGalleryRequest{
		Items: []dto.BatchGalleryImageItem{
			{ExternalURL: "https://source-a.com/1.webp", ImageType: model.GalleryImageTypeScreenshot},
			{ExternalURL: "https://source-a.com/2.webp", ImageType: model.GalleryImageTypeCG, IsSpoiler: true},
			{ExternalURL: " https://source-a.com/1.webp "}, // trims to a duplicate
			{ExternalURL: "https://source-a.com/existing.webp"},
			{ExternalURL: "javascript:alert(1)"},
			{ExternalURL: ""},
		},
	})
	if err != nil {
		t.Fatalf("batch create: %v", err)
	}
	if result.Created != 2 || result.Skipped != 2 || result.Failed != 2 {
		t.Fatalf("unexpected batch result: %+v", result)
	}

	// Re-running the same import is idempotent.
	result, err = svc.BatchCreateGalleryImages(ctx, galgameID, actor, &dto.BatchCreateGalleryRequest{
		Items: []dto.BatchGalleryImageItem{
			{ExternalURL: "https://source-a.com/1.webp"},
			{ExternalURL: "https://source-a.com/2.webp"},
		},
	})
	if err != nil || result.Created != 0 || result.Skipped != 2 {
		t.Fatalf("expected idempotent rerun, got %+v err=%v", result, err)
	}

	data, err := svc.ListAdminGallery(ctx, galgameID)
	if err != nil || data.Total != 3 {
		t.Fatalf("expected 3 gallery images, got %d err=%v", data.Total, err)
	}
	for _, item := range data.Items {
		if item.Status != model.GalleryImageStatusPending {
			t.Fatalf("batch import must create pending images, got %+v", item)
		}
	}
	if existing.ID == 0 {
		t.Fatalf("existing image id missing")
	}
}

func TestGalleryReviewFlow(t *testing.T) {
	svc, db, actor := newGalleryTestService(t)
	ctx := context.Background()
	galgameID := createGalleryGalgame(t, db, "gallery-review", model.GalgameStatusPublished)
	reviewer := testutil.CreateUser(t, db, "gallery-reviewer")

	approveMe, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{
		ExternalURL: "https://source-a.com/approve.webp",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	rejectMe, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{
		ExternalURL: "https://source-a.com/reject.webp",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	batchOne, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{
		ExternalURL: "https://source-a.com/batch1.webp",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	batchTwo, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{
		ExternalURL: "https://source-a.com/batch2.webp",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	reviewed, err := svc.ReviewGalleryImages(ctx, ReviewGalleryImagesInput{
		IDs: []uint{approveMe.ID}, Approve: true, AdminID: reviewer,
	})
	if err != nil || reviewed != 1 {
		t.Fatalf("approve: reviewed=%d err=%v", reviewed, err)
	}

	reviewed, err = svc.ReviewGalleryImages(ctx, ReviewGalleryImagesInput{
		IDs: []uint{rejectMe.ID}, Approve: false, Reason: "图片与游戏无关", AdminID: reviewer,
	})
	if err != nil || reviewed != 1 {
		t.Fatalf("reject: reviewed=%d err=%v", reviewed, err)
	}

	reviewed, err = svc.ReviewGalleryImages(ctx, ReviewGalleryImagesInput{
		IDs: []uint{batchOne.ID, batchTwo.ID, 999999}, Approve: true, AdminID: reviewer,
	})
	if err != nil || reviewed != 2 {
		t.Fatalf("batch approve: reviewed=%d err=%v", reviewed, err)
	}

	var row model.GalleryImage
	if err := db.First(&row, rejectMe.ID).Error; err != nil {
		t.Fatalf("load rejected image: %v", err)
	}
	if row.Status != model.GalleryImageStatusRejected || row.RejectReason != "图片与游戏无关" ||
		row.ReviewedBy == nil || *row.ReviewedBy != reviewer || row.ReviewedAt == nil {
		t.Fatalf("unexpected rejected row: %+v", row)
	}

	// Only published items reach the public gallery.
	public, err := svc.ListPublishedGallery(ctx, galgameID)
	if err != nil || public.Total != 3 {
		t.Fatalf("expected 3 published images, got %d err=%v", public.Total, err)
	}

	// Review queue lists pending images with galgame + submitter context.
	pending := model.GalleryImageStatusPending
	queue, err := svc.ListGalleryReviews(ctx, GalleryReviewQuery{Status: &pending})
	if err != nil || queue.Total != 0 {
		t.Fatalf("expected empty pending queue after review, got %d err=%v", queue.Total, err)
	}
	rejectedStatus := model.GalleryImageStatusRejected
	queue, err = svc.ListGalleryReviews(ctx, GalleryReviewQuery{Status: &rejectedStatus})
	if err != nil || queue.Total != 1 {
		t.Fatalf("expected 1 rejected in queue, got %d err=%v", queue.Total, err)
	}
	if len(queue.Items) != 1 || queue.Items[0].GalgameID != galgameID ||
		queue.Items[0].GalgameTitle != "gallery-review" || queue.Items[0].CreatedByUsername == "" ||
		queue.Items[0].ReviewedByUsername == "" || queue.Items[0].RejectReason != "图片与游戏无关" {
		t.Fatalf("unexpected review item: %+v", queue.Items)
	}

	externalFilter := model.GallerySourceExternal
	queue, err = svc.ListGalleryReviews(ctx, GalleryReviewQuery{SourceType: &externalFilter})
	if err != nil || queue.Total != 4 {
		t.Fatalf("expected 4 external images in queue, got %d err=%v", queue.Total, err)
	}
	queue, err = svc.ListGalleryReviews(ctx, GalleryReviewQuery{GalgameID: 999999})
	if err != nil || queue.Total != 0 {
		t.Fatalf("expected empty queue for unknown galgame, got %d err=%v", queue.Total, err)
	}
}

func TestGalleryUpdateDelete(t *testing.T) {
	svc, db, actor := newGalleryTestService(t)
	ctx := context.Background()
	galgameID := createGalleryGalgame(t, db, "gallery-update", model.GalgameStatusPublished)
	otherID := createGalleryGalgame(t, db, "gallery-update-other", model.GalgameStatusPublished)

	asset := createGalleryAsset(t, db, "image/webp")
	created, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{AssetID: assetPtr(asset)})
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
		created, err := svc.CreateGalleryImage(ctx, galgameID, actor, &dto.CreateGalleryImageRequest{
			AssetID: assetPtr(createGalleryAsset(t, db, "image/webp")),
		})
		if err != nil {
			t.Fatalf("create gallery image %d: %v", i, err)
		}
		ids = append(ids, created.ID)
	}

	otherCreated, err := svc.CreateGalleryImage(ctx, otherID, actor, &dto.CreateGalleryImageRequest{
		AssetID: assetPtr(createGalleryAsset(t, db, "image/webp")),
	})
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
