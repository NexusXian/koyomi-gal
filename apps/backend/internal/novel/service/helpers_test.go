package service

import (
	"context"
	"strings"
	"testing"

	contributionRepo "backend/internal/contribution/repository"
	contributionService "backend/internal/contribution/service"
	galgameDTO "backend/internal/galgame/dto"
	galgameModel "backend/internal/galgame/model"
	galgameRepo "backend/internal/galgame/repository"
	galgameService "backend/internal/galgame/service"
	notificationModel "backend/internal/notification/model"
	notificationRepo "backend/internal/notification/repository"
	notificationService "backend/internal/notification/service"
	"backend/internal/novel/dto"
	"backend/internal/novel/model"
	"backend/internal/novel/repository"
	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	relationRepo "backend/internal/relation/repository"
	"backend/internal/testutil"

	"gorm.io/gorm"
)

const testPublicURL = "https://img.example.com"

var publishedStatus = int16(model.NovelStatusPublished)

type novelTestServices struct {
	db          *gorm.DB
	novels      *NovelService
	volumes     *VolumeService
	relations   *RelationService
	catalog     *galgameService.CatalogService
	rbac        *rbacService.RBACService
	contributor *contributionService.ContributionService
}

func newNovelTestServices(t *testing.T) *novelTestServices {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	galgameRepository := galgameRepo.NewGalgameRepository(db)
	novelRepository := repository.NewNovelRepository(db)
	volumeRepository := repository.NewVolumeRepository(db)
	relationRepository := relationRepo.NewRelationRepository(db)
	tagRepository := galgameRepo.NewTagRepository(db)

	rbacSvc := rbacService.NewRBACService(rbacRepo.NewRBACRepository(db))
	if err := rbacSvc.SeedDefaults(context.Background()); err != nil {
		t.Fatalf("seed rbac defaults: %v", err)
	}
	notificationSvc := notificationService.NewNotificationService(
		notificationRepo.NewNotificationRepository(db, testPublicURL),
	)
	contributorSvc := contributionService.NewContributionService(
		contributionRepo.NewContributionRepository(db, testPublicURL),
		galgameRepository,
		testPublicURL,
	)

	novels := NewNovelService(novelRepository, volumeRepository, tagRepository, galgameRepository, relationRepository)
	novels.SetContributionService(contributorSvc)
	novels.SetNotificationDependencies(rbacSvc, notificationSvc)

	volumes := NewVolumeService(volumeRepository, novelRepository)
	volumes.SetContributionService(contributorSvc)
	volumes.SetNotificationDependencies(rbacSvc, notificationSvc)

	relations := NewRelationService(relationRepository, novelRepository, galgameRepository)
	relations.SetContributionService(contributorSvc)

	catalog := galgameService.NewCatalogService(
		galgameRepository,
		galgameRepo.NewDeveloperRepository(db),
		tagRepository,
	)
	catalog.SetContributionService(contributorSvc)

	return &novelTestServices{
		db:          db,
		novels:      novels,
		volumes:     volumes,
		relations:   relations,
		catalog:     catalog,
		rbac:        rbacSvc,
		contributor: contributorSvc,
	}
}

// assignReviewer gives the user the admin role so they hold the seeded
// novel:review permission, mirroring real seeding.
func (env *novelTestServices) assignReviewer(t *testing.T, userID uint) {
	t.Helper()
	if err := env.rbac.AssignRoleByCode(context.Background(), userID, rbacService.RoleCodeAdmin); err != nil {
		t.Fatalf("assign admin role: %v", err)
	}
}

func createTestNovel(t *testing.T, svc *NovelService, userID uint, slug string, status int16) *model.Novel {
	t.Helper()
	novel, err := svc.CreateNovel(context.Background(), userID, &dto.CreateNovelRequest{
		Title:         strings.ToUpper(slug[:1]) + slug[1:],
		OriginalTitle: "原名-" + slug,
		Slug:          slug,
		Summary:       "测试简介",
		Author:        "测试作者",
		Illustrator:   "测试插画师",
		Publisher:     "测试出版社",
		Label:         "测试文库",
		Language:      "ja",
		Region:        "JP",
		ReleaseStatus: model.ReleaseStatusOngoing,
		AgeRating:     galgameModel.AgeRatingAll,
		Status:        status,
	})
	if err != nil {
		t.Fatalf("create novel %s: %v", slug, err)
	}
	return novel
}

func createTestGalgameForNovel(t *testing.T, catalog *galgameService.CatalogService, userID uint, slug string) uint {
	t.Helper()
	galgame, err := catalog.CreateGalgame(context.Background(), userID, &galgameDTO.CreateGalgameRequest{
		Title:         slug,
		OriginalTitle: slug,
		RomajiTitle:   slug,
		Slug:          slug,
		AgeRating:     galgameModel.AgeRatingAll,
		Status:        galgameModel.GalgameStatusPublished,
	})
	if err != nil {
		t.Fatalf("create galgame %s: %v", slug, err)
	}
	return galgame.ID
}

func assertNotificationCount(t *testing.T, db *gorm.DB, recipientID uint, notificationType notificationModel.NotificationType, want int64) {
	t.Helper()
	var count int64
	if err := db.Table("notifications").Where("recipient_id = ? AND type = ?", recipientID, notificationType).Count(&count).Error; err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if count != want {
		t.Fatalf("notification %s count: got %d want %d", notificationType, count, want)
	}
}

func assertContributionRows(t *testing.T, db *gorm.DB, novelID uint, want int64) {
	t.Helper()
	var count int64
	if err := db.Table("work_contributions").
		Where("target_type = 'novel' AND target_id = ?", novelID).
		Count(&count).Error; err != nil {
		t.Fatalf("count novel contributions: %v", err)
	}
	if count != want {
		t.Fatalf("novel contribution rows: got %d want %d", count, want)
	}
}
