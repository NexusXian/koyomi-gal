package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"
	notificationModel "backend/internal/notification/model"
	notificationRepo "backend/internal/notification/repository"
	notificationService "backend/internal/notification/service"
	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	"backend/internal/testutil"

	"gorm.io/gorm"
)

func newCatalogTestService(t *testing.T) (*CatalogService, *gorm.DB, uint) {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	svc := NewCatalogService(
		repository.NewGalgameRepository(db),
		repository.NewDeveloperRepository(db),
		repository.NewTagRepository(db),
	)
	rbacSvc := rbacService.NewRBACService(rbacRepo.NewRBACRepository(db))
	if err := rbacSvc.SeedDefaults(context.Background()); err != nil {
		t.Fatalf("seed rbac defaults: %v", err)
	}
	svc.SetNotificationDependencies(rbacSvc, notificationService.NewNotificationService(
		notificationRepo.NewNotificationRepository(db, "https://img.example.com"),
	))
	return svc, db, testutil.CreateUser(t, db, "catalog-user")
}

func TestGalgameNotifications(t *testing.T) {
	svc, db, creator := newCatalogTestService(t)
	ctx := context.Background()
	reviewer := testutil.CreateUser(t, db, "catalog-reviewer")
	rbacSvc := rbacService.NewRBACService(rbacRepo.NewRBACRepository(db))
	if err := rbacSvc.AssignRoleByCode(ctx, reviewer, rbacService.RoleCodeAdmin); err != nil {
		t.Fatalf("assign reviewer role: %v", err)
	}

	pending := createTestGalgame(t, svc, creator, "notify-pending", nil, nil, "", 0, model.GalgameStatusPending)
	assertNotificationCount(t, db, reviewer, notificationModel.TypeGalgameSubmitted, 1)
	if _, err := svc.ReviewGalgame(ctx, reviewer, pending.ID, &dto.ReviewGalgameRequest{Status: model.GalgameStatusPublished}); err != nil {
		t.Fatalf("approve galgame: %v", err)
	}
	assertNotificationCount(t, db, creator, notificationModel.TypeGalgameApproved, 1)
	if _, err := svc.ReviewGalgame(ctx, reviewer, pending.ID, &dto.ReviewGalgameRequest{Status: model.GalgameStatusPublished}); err != nil {
		t.Fatalf("repeat approval: %v", err)
	}
	assertNotificationCount(t, db, creator, notificationModel.TypeGalgameApproved, 1)

	rejected := createTestGalgame(t, svc, creator, "notify-rejected", nil, nil, "", 0, model.GalgameStatusPending)
	if _, err := svc.ReviewGalgame(ctx, reviewer, rejected.ID, &dto.ReviewGalgameRequest{Status: model.GalgameStatusRejected}); err != nil {
		t.Fatalf("reject galgame: %v", err)
	}
	assertNotificationCount(t, db, creator, notificationModel.TypeGalgameRejected, 1)
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

func createTestDeveloper(t *testing.T, svc *CatalogService, name string) *model.Developer {
	t.Helper()
	developer, err := svc.CreateDeveloper(context.Background(), &dto.CreateDeveloperRequest{
		Name: name,
		Slug: name,
	})
	if err != nil {
		t.Fatalf("create developer %s: %v", name, err)
	}
	return developer
}

func createTestTag(t *testing.T, svc *CatalogService, name string) *model.Tag {
	t.Helper()
	tag, err := svc.CreateTag(context.Background(), &dto.CreateTagRequest{
		Name: name,
		Slug: name,
	})
	if err != nil {
		t.Fatalf("create tag %s: %v", name, err)
	}
	return tag
}

func createTestGalgame(
	t *testing.T,
	svc *CatalogService,
	userID uint,
	title string,
	developerID *uint,
	tagIDs []uint,
	releaseDate string,
	ageRating int16,
	status int16,
	aliases ...string,
) *model.Galgame {
	t.Helper()
	galgame, err := svc.CreateGalgame(context.Background(), userID, &dto.CreateGalgameRequest{
		Title:         title,
		OriginalTitle: title,
		RomajiTitle:   title,
		Slug:          title,
		DeveloperID:   developerID,
		TagIDs:        tagIDs,
		ReleaseDate:   releaseDate,
		AgeRating:     ageRating,
		Status:        status,
		Aliases:       aliases,
	})
	if err != nil {
		t.Fatalf("create galgame %s: %v", title, err)
	}
	return galgame
}

func TestCatalogCreateAndDetail(t *testing.T) {
	svc, _, userID := newCatalogTestService(t)
	ctx := context.Background()

	developer := createTestDeveloper(t, svc, "yuzusoft")
	tagA := createTestTag(t, svc, "pure-love")
	tagB := createTestTag(t, svc, "school")
	galgame := createTestGalgame(
		t,
		svc,
		userID,
		"senren-banka",
		&developer.ID,
		[]uint{tagA.ID, tagB.ID},
		"2016-07-29",
		model.AgeRatingR18,
		model.GalgameStatusPublished,
		"千恋＊万花",
		"千恋万花",
	)

	detail, err := svc.GetPublishedGalgame(ctx, galgame.ID)
	if err != nil {
		t.Fatalf("get galgame detail: %v", err)
	}
	if detail.Developer == nil || detail.Developer.ID != developer.ID {
		t.Fatalf("expected developer %d, got %+v", developer.ID, detail.Developer)
	}
	if len(detail.Tags) != 2 || len(detail.Aliases) != 2 {
		t.Fatalf("expected 2 tags and aliases, got tags=%d aliases=%d", len(detail.Tags), len(detail.Aliases))
	}

	_, err = svc.CreateDeveloper(ctx, &dto.CreateDeveloperRequest{Name: "duplicate", Slug: "YUZUSOFT"})
	if !errors.Is(err, ErrDeveloperSlugExists) {
		t.Fatalf("expected duplicate developer slug, got %v", err)
	}
	_, err = svc.CreateTag(ctx, &dto.CreateTagRequest{Name: tagA.Name, Slug: "new-slug"})
	if !errors.Is(err, ErrTagNameExists) {
		t.Fatalf("expected duplicate tag name, got %v", err)
	}
	_, err = svc.CreateGalgame(ctx, userID, &dto.CreateGalgameRequest{
		Title:  "duplicate",
		Slug:   "SENREN-BANKA",
		Status: model.GalgameStatusPublished,
	})
	if !errors.Is(err, ErrGalgameSlugExists) {
		t.Fatalf("expected duplicate galgame slug, got %v", err)
	}
}

func TestGalgameSearchAndFilters(t *testing.T) {
	svc, _, userID := newCatalogTestService(t)
	ctx := context.Background()
	developerA := createTestDeveloper(t, svc, "developer-a")
	developerB := createTestDeveloper(t, svc, "developer-b")
	tagA := createTestTag(t, svc, "tag-a")
	tagB := createTestTag(t, svc, "tag-b")
	tagC := createTestTag(t, svc, "tag-c")

	gameA := createTestGalgame(t, svc, userID, "千恋＊万花", &developerA.ID,
		[]uint{tagA.ID, tagB.ID}, "2016-07-29", model.AgeRatingR18,
		model.GalgameStatusPublished, "Senren Banka")
	gameB := createTestGalgame(t, svc, userID, "game-b", &developerA.ID,
		[]uint{tagA.ID}, "2020-01-01", model.AgeRatingAll,
		model.GalgameStatusPublished)
	pending := createTestGalgame(t, svc, userID, "pending-game", &developerB.ID,
		[]uint{tagA.ID, tagB.ID}, "2018-01-01", model.AgeRatingR18,
		model.GalgameStatusPending, "hidden alias")
	gameD := createTestGalgame(t, svc, userID, "game-d", &developerB.ID,
		[]uint{tagB.ID, tagC.ID}, "2010-01-01", model.AgeRatingR15,
		model.GalgameStatusPublished)

	tests := []struct {
		name     string
		query    dto.GalgameQuery
		expected []uint
	}{
		{name: "keyword title", query: dto.GalgameQuery{Keyword: "千恋"}, expected: []uint{gameA.ID}},
		{name: "keyword alias", query: dto.GalgameQuery{Keyword: "senren"}, expected: []uint{gameA.ID}},
		{name: "developer", query: dto.GalgameQuery{DeveloperID: &developerB.ID}, expected: []uint{gameD.ID}},
		{name: "single tag", query: dto.GalgameQuery{TagIDs: []uint{tagA.ID}}, expected: []uint{gameB.ID, gameA.ID}},
		{name: "multiple tag AND", query: dto.GalgameQuery{TagIDs: []uint{tagA.ID, tagB.ID}}, expected: []uint{gameA.ID}},
		{name: "release year", query: dto.GalgameQuery{ReleaseFrom: intPtr(2015), ReleaseTo: intPtr(2019)}, expected: []uint{gameA.ID}},
		{name: "age rating", query: dto.GalgameQuery{AgeRating: int16Ptr(model.AgeRatingR18)}, expected: []uint{gameA.ID}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items, total, _, _, err := svc.ListPublishedGalgames(ctx, &test.query)
			if err != nil {
				t.Fatalf("list galgames: %v", err)
			}
			if total != int64(len(test.expected)) || len(items) != len(test.expected) {
				t.Fatalf("expected %d results, got total=%d items=%d", len(test.expected), total, len(items))
			}
			for i, expectedID := range test.expected {
				if items[i].ID != expectedID {
					t.Fatalf("expected result %d at index %d, got %d", expectedID, i, items[i].ID)
				}
			}
		})
	}

	all, total, _, _, err := svc.ListPublishedGalgames(ctx, &dto.GalgameQuery{})
	if err != nil || total != 3 || len(all) != 3 {
		t.Fatalf("expected only 3 published games, total=%d items=%d err=%v", total, len(all), err)
	}
	if _, err := svc.GetPublishedGalgame(ctx, pending.ID); !errors.Is(err, ErrGalgameNotFound) {
		t.Fatalf("expected pending detail hidden, got %v", err)
	}
}

func TestGalgameSortAndPagination(t *testing.T) {
	svc, db, userID := newCatalogTestService(t)
	ctx := context.Background()
	oldest := createTestGalgame(t, svc, userID, "oldest", nil, nil, "2010-01-01", 0, 1)
	middle := createTestGalgame(t, svc, userID, "middle", nil, nil, "2015-01-01", 0, 1)
	latest := createTestGalgame(t, svc, userID, "latest", nil, nil, "2020-01-01", 0, 1)

	if err := db.Model(&model.Galgame{}).Where("id = ?", oldest.ID).Updates(map[string]any{
		"rating_average": 9.5,
		"favorite_count": 3,
		"resource_count": 1,
		"post_count":     1,
	}).Error; err != nil {
		t.Fatalf("update oldest statistics: %v", err)
	}
	if err := db.Model(&model.Galgame{}).Where("id = ?", middle.ID).Updates(map[string]any{
		"rating_average": 7.5,
		"favorite_count": 20,
		"resource_count": 2,
		"post_count":     2,
	}).Error; err != nil {
		t.Fatalf("update middle statistics: %v", err)
	}
	if err := db.Model(&model.Galgame{}).Where("id = ?", latest.ID).Updates(map[string]any{
		"rating_average": 8.5,
		"favorite_count": 10,
		"resource_count": 20,
		"post_count":     20,
	}).Error; err != nil {
		t.Fatalf("update latest statistics: %v", err)
	}

	assertFirstGalgame(t, svc, ctx, "latest", latest.ID)
	assertFirstGalgame(t, svc, ctx, "oldest", oldest.ID)
	assertFirstGalgame(t, svc, ctx, "rating", oldest.ID)
	assertFirstGalgame(t, svc, ctx, "favorite", middle.ID)
	assertFirstGalgame(t, svc, ctx, "popular", latest.ID)

	items, total, page, limit, err := svc.ListPublishedGalgames(ctx, &dto.GalgameQuery{
		Sort:  "latest",
		Page:  2,
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("paginate galgames: %v", err)
	}
	if total != 3 || page != 2 || limit != 1 || len(items) != 1 || items[0].ID != middle.ID {
		t.Fatalf("unexpected pagination: total=%d page=%d limit=%d items=%v", total, page, limit, galgameIDs(items))
	}
}

func assertFirstGalgame(t *testing.T, svc *CatalogService, ctx context.Context, sort string, expectedID uint) {
	t.Helper()
	items, _, _, _, err := svc.ListPublishedGalgames(ctx, &dto.GalgameQuery{Sort: sort})
	if err != nil {
		t.Fatalf("sort %s: %v", sort, err)
	}
	if len(items) == 0 || items[0].ID != expectedID {
		t.Fatalf("sort %s expected first %d, got %v", sort, expectedID, galgameIDs(items))
	}
}

func galgameIDs(galgames []model.Galgame) []uint {
	ids := make([]uint, 0, len(galgames))
	for _, galgame := range galgames {
		ids = append(ids, galgame.ID)
	}
	return ids
}

func intPtr(value int) *int {
	return &value
}

func int16Ptr(value int16) *int16 {
	return &value
}

func TestCatalogValidation(t *testing.T) {
	svc, _, _ := newCatalogTestService(t)
	ctx := context.Background()

	_, _, _, _, err := svc.ListPublishedGalgames(ctx, &dto.GalgameQuery{Sort: "id desc"})
	if !errors.Is(err, ErrInvalidSort) {
		t.Fatalf("expected invalid sort error, got %v", err)
	}
	_, _, _, _, err = svc.ListPublishedGalgames(ctx, &dto.GalgameQuery{
		ReleaseFrom: intPtr(2020),
		ReleaseTo:   intPtr(2010),
	})
	if !errors.Is(err, ErrInvalidReleaseRange) {
		t.Fatalf("expected invalid release range, got %v", err)
	}

	if _, err := parseReleaseDate("2016/07/29"); !errors.Is(err, ErrInvalidReleaseDate) {
		t.Fatalf("expected invalid release date, got %v", err)
	}
	if got := fmt.Sprint(uniqueUint([]uint{1, 1, 2})); got != "[1 2]" {
		t.Fatalf("unexpected unique IDs: %s", got)
	}
	if _, err := svc.CreateDeveloper(ctx, &dto.CreateDeveloperRequest{Name: " ", Slug: " "}); !errors.Is(err, ErrInvalidCatalogInput) {
		t.Fatalf("expected whitespace input rejected, got %v", err)
	}
}
