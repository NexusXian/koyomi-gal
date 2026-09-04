package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"
	"backend/internal/testutil"

	"gorm.io/gorm"
)

type contributionTestEnv struct {
	db            *gorm.DB
	catalog       *CatalogService
	contributions *ContributionService
	repository    *repository.ContributionRepository
}

func newContributionTestEnv(t *testing.T) *contributionTestEnv {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	galgames := repository.NewGalgameRepository(db)
	contributionRepository := repository.NewContributionRepository(db, "https://img.example.com")
	contributions := NewContributionService(contributionRepository, galgames, "https://img.example.com")
	catalog := NewCatalogService(
		galgames,
		repository.NewDeveloperRepository(db),
		repository.NewTagRepository(db),
	)
	catalog.SetContributionService(contributions)
	return &contributionTestEnv{
		db:            db,
		catalog:       catalog,
		contributions: contributions,
		repository:    contributionRepository,
	}
}

func TestContributionReviewLifecycleAndIdempotency(t *testing.T) {
	env := newContributionTestEnv(t)
	ctx := context.Background()
	creator := testutil.CreateUser(t, env.db, "contribution-creator")
	reviewer := testutil.CreateUser(t, env.db, "contribution-reviewer")

	rejected := createTestGalgame(t, env.catalog, creator, "contribution-rejected", nil, nil, "", 0, model.GalgameStatusPending)
	assertContributionRows(t, env.db, rejected.ID, 0)
	if _, err := env.catalog.ReviewGalgame(ctx, reviewer, rejected.ID, &dto.ReviewGalgameRequest{Status: model.GalgameStatusRejected}); err != nil {
		t.Fatalf("reject galgame: %v", err)
	}
	assertContributionRows(t, env.db, rejected.ID, 0)

	approved := createTestGalgame(t, env.catalog, creator, "contribution-approved", nil, nil, "", 0, model.GalgameStatusPending)
	assertContributionRows(t, env.db, approved.ID, 0)
	if _, err := env.catalog.ReviewGalgame(ctx, reviewer, approved.ID, &dto.ReviewGalgameRequest{Status: model.GalgameStatusPublished}); err != nil {
		t.Fatalf("approve galgame: %v", err)
	}
	if _, err := env.catalog.ReviewGalgame(ctx, reviewer, approved.ID, &dto.ReviewGalgameRequest{Status: model.GalgameStatusPublished}); err != nil {
		t.Fatalf("repeat approval: %v", err)
	}
	assertContributionRows(t, env.db, approved.ID, 1)

	sourceType := model.ContributionSourceGalgameCreate
	sourceID := approved.ID
	if err := env.contributions.RecordContribution(ctx, RecordContributionInput{
		GalgameID:  approved.ID,
		UserID:     creator,
		Action:     model.ContributionActionCreate,
		SourceType: &sourceType,
		SourceID:   &sourceID,
	}); err != nil {
		t.Fatalf("repeat source record: %v", err)
	}
	assertContributionRows(t, env.db, approved.ID, 1)

	for range 2 {
		if err := env.contributions.RecordContribution(ctx, RecordContributionInput{
			GalgameID: approved.ID,
			UserID:    creator,
			Action:    model.ContributionActionEdit,
		}); err != nil {
			t.Fatalf("record repeated user contribution: %v", err)
		}
	}
	assertContributionRows(t, env.db, approved.ID, 3)

	direct := createTestGalgame(t, env.catalog, creator, "contribution-direct-publish", nil, nil, "", 0, model.GalgameStatusPending)
	ageRating := direct.AgeRating
	coverSensitive := direct.CoverSensitive
	publishedStatus := model.GalgameStatusPublished
	if _, err := env.catalog.UpdateGalgame(ctx, direct.ID, &dto.UpdateGalgameRequest{
		Title:          direct.Title,
		OriginalTitle:  direct.OriginalTitle,
		RomajiTitle:    direct.RomajiTitle,
		Slug:           direct.Slug,
		Description:    direct.Description,
		CoverURL:       direct.CoverURL,
		BannerURL:      direct.BannerURL,
		AgeRating:      &ageRating,
		CoverSensitive: &coverSensitive,
		Status:         &publishedStatus,
	}, reviewer); err != nil {
		t.Fatalf("publish galgame through update: %v", err)
	}
	var directContribution model.GalgameContribution
	if err := env.db.Where("galgame_id = ?", direct.ID).Take(&directContribution).Error; err != nil {
		t.Fatalf("find direct publication contribution: %v", err)
	}
	if directContribution.UserID != creator || directContribution.Action != model.ContributionActionCreate {
		t.Fatalf("direct publication attribution: %+v", directContribution)
	}
}

func TestContributorAggregationOrderingAndPrivacy(t *testing.T) {
	env := newContributionTestEnv(t)
	ctx := context.Background()
	owner := testutil.CreateUser(t, env.db, "contribution-owner")
	first := testutil.CreateUser(t, env.db, "contribution-first")
	second := testutil.CreateUser(t, env.db, "contribution-second")
	game := createTestGalgame(t, env.catalog, owner, "contribution-aggregate", nil, nil, "", 0, model.GalgameStatusPublished)

	for range 3 {
		if err := env.contributions.RecordContribution(ctx, RecordContributionInput{
			GalgameID: game.ID, UserID: first, Action: model.ContributionActionEdit,
		}); err != nil {
			t.Fatalf("record first contributor: %v", err)
		}
	}
	for range 2 {
		if err := env.contributions.RecordContribution(ctx, RecordContributionInput{
			GalgameID: game.ID, UserID: second, Action: model.ContributionActionGallery,
		}); err != nil {
			t.Fatalf("record second contributor: %v", err)
		}
	}

	data, err := env.contributions.ListContributorsByGalgameID(ctx, game.ID, 1, 20)
	if err != nil {
		t.Fatalf("list contributors: %v", err)
	}
	if data.Total != 3 || len(data.Items) != 3 {
		t.Fatalf("contributors: total=%d items=%d", data.Total, len(data.Items))
	}
	if data.Items[0].UserID != first || data.Items[0].ContributionCount != 3 {
		t.Fatalf("first contributor ordering/count: %+v", data.Items[0])
	}
	if data.Items[1].UserID != second || data.Items[1].ContributionCount != 2 {
		t.Fatalf("second contributor ordering/count: %+v", data.Items[1])
	}
	if data.Items[0].FirstContributedAt.After(data.Items[0].LastContributedAt) {
		t.Fatalf("invalid contribution timestamps: %+v", data.Items[0])
	}

	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal contributor DTO: %v", err)
	}
	serialized := string(payload)
	for _, privateField := range []string{"email", "password", "role", "is_banned", "privacy"} {
		if strings.Contains(serialized, privateField) {
			t.Fatalf("private field %q leaked in %s", privateField, serialized)
		}
	}
}

func TestContributorMissingUserAndTransactionRollback(t *testing.T) {
	env := newContributionTestEnv(t)
	ctx := context.Background()
	owner := testutil.CreateUser(t, env.db, "contribution-rollback-owner")
	game := createTestGalgame(t, env.catalog, owner, "contribution-rollback", nil, nil, "", 0, model.GalgameStatusPublished)

	if err := env.contributions.RecordContribution(ctx, RecordContributionInput{
		GalgameID: game.ID,
		UserID:    99999999,
		Action:    model.ContributionActionEdit,
	}); err != nil {
		t.Fatalf("record missing user contribution: %v", err)
	}
	data, err := env.contributions.ListContributorsByGalgameID(ctx, game.ID, 1, 20)
	if err != nil {
		t.Fatalf("list missing user contributor: %v", err)
	}
	var missing *dto.ContributorData
	for index := range data.Items {
		if data.Items[index].UserID == 99999999 {
			missing = &data.Items[index]
		}
	}
	if len(data.Items) != 2 || missing == nil || missing.Username != "" {
		t.Fatalf("missing user contributor response: %+v", data.Items)
	}

	pending := createTestGalgame(t, env.catalog, owner, "contribution-transaction", nil, nil, "", 0, model.GalgameStatusPending)
	rollbackErr := errors.New("force rollback")
	err = env.contributions.Transaction(ctx, func(tx *gorm.DB) error {
		if err := repository.NewGalgameRepository(tx).UpdateStatus(ctx, pending.ID, model.GalgameStatusPublished); err != nil {
			return err
		}
		if err := env.contributions.RecordContribution(ctx, RecordContributionInput{
			GalgameID: pending.ID,
			UserID:    owner,
			Action:    model.ContributionActionCreate,
		}, tx); err != nil {
			return err
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("transaction error: got %v", err)
	}
	stored, err := repository.NewGalgameRepository(env.db).FindByID(ctx, pending.ID)
	if err != nil {
		t.Fatalf("reload rolled back galgame: %v", err)
	}
	if stored.Status != model.GalgameStatusPending {
		t.Fatalf("galgame status persisted after rollback: %d", stored.Status)
	}
	assertContributionRows(t, env.db, pending.ID, 0)
}

func TestContributorTieSortsByLatestContribution(t *testing.T) {
	env := newContributionTestEnv(t)
	owner := testutil.CreateUser(t, env.db, "contribution-sort-owner")
	older := testutil.CreateUser(t, env.db, "contribution-sort-older")
	newer := testutil.CreateUser(t, env.db, "contribution-sort-newer")
	game := createTestGalgame(t, env.catalog, owner, "contribution-sort", nil, nil, "", 0, model.GalgameStatusPublished)

	if err := env.db.Exec(`
INSERT INTO galgame_contributions (galgame_id, user_id, action, created_at)
VALUES (?, ?, 'edit', ?), (?, ?, 'edit', ?)
`, game.ID, older, time.Now().Add(-time.Hour), game.ID, newer, time.Now()).Error; err != nil {
		t.Fatalf("insert sorted contributions: %v", err)
	}
	contributors, _, err := env.repository.ListContributorsByGalgameID(context.Background(), game.ID, 1, 20)
	if err != nil {
		t.Fatalf("list sorted contributors: %v", err)
	}
	positions := make(map[uint]int, len(contributors))
	for index, contributor := range contributors {
		positions[contributor.UserID] = index
	}
	if positions[newer] >= positions[older] {
		t.Fatalf("latest contribution ordering: %+v", contributors)
	}
}

func assertContributionRows(t *testing.T, db *gorm.DB, galgameID uint, want int64) {
	t.Helper()
	var count int64
	if err := db.Table("galgame_contributions").Where("galgame_id = ?", galgameID).Count(&count).Error; err != nil {
		t.Fatalf("count contributions: %v", err)
	}
	if count != want {
		t.Fatalf("contribution rows: got %d want %d", count, want)
	}
}
