package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"
	"backend/internal/testutil"

	"gorm.io/gorm"
)

type relationTestServices struct {
	catalog   *CatalogService
	rating    *RatingService
	favorite  *FavoriteService
	userState *UserStateService
	relation  *UserRelationService
}

func newRelationTestServices(t *testing.T) (*relationTestServices, *gorm.DB) {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	galgameRepo := repository.NewGalgameRepository(db)
	relationRepo := repository.NewUserRelationRepository(db)
	services := &relationTestServices{
		catalog: NewCatalogService(
			galgameRepo,
			repository.NewDeveloperRepository(db),
			repository.NewTagRepository(db),
		),
		rating:    NewRatingService(galgameRepo, relationRepo),
		favorite:  NewFavoriteService(galgameRepo, relationRepo),
		userState: NewUserStateService(galgameRepo, relationRepo),
		relation:  NewUserRelationService(galgameRepo, relationRepo),
	}
	return services, db
}

func createPublishedGalgame(t *testing.T, svc *relationTestServices, userID uint, title string) *model.Galgame {
	t.Helper()
	return createTestGalgame(
		t, svc.catalog, userID, title, nil, nil, "2020-01-01",
		model.AgeRatingAll, model.GalgameStatusPublished,
	)
}

func galgameStats(t *testing.T, db *gorm.DB, galgameID uint) (average string, ratingCount, favoriteCount int64) {
	t.Helper()
	var row struct {
		RatingAverage string
		RatingCount   int64
		FavoriteCount int64
	}
	if err := db.Raw(`
SELECT rating_average::text AS rating_average, rating_count, favorite_count
FROM galgames WHERE id = ?
`, galgameID).Scan(&row).Error; err != nil {
		t.Fatalf("read galgame stats: %v", err)
	}
	return row.RatingAverage, row.RatingCount, row.FavoriteCount
}

func TestRatingRecalculatesStatistics(t *testing.T) {
	services, db := newRelationTestServices(t)
	ctx := context.Background()
	userA := testutil.CreateUser(t, db, "rating-a")
	userB := testutil.CreateUser(t, db, "rating-b")
	galgame := createPublishedGalgame(t, services, userA, "rating-game")

	rating, err := services.rating.UpsertRating(ctx, galgame.ID, userA, 8)
	if err != nil {
		t.Fatalf("upsert first rating: %v", err)
	}
	if rating.Score != 8 {
		t.Fatalf("expected score 8, got %d", rating.Score)
	}
	average, count, _ := galgameStats(t, db, galgame.ID)
	if average != "8.00" || count != 1 {
		t.Fatalf("expected 8.00/1, got %s/%d", average, count)
	}

	if _, err := services.rating.UpsertRating(ctx, galgame.ID, userB, 10); err != nil {
		t.Fatalf("upsert second rating: %v", err)
	}
	average, count, _ = galgameStats(t, db, galgame.ID)
	if average != "9.00" || count != 2 {
		t.Fatalf("expected 9.00/2, got %s/%d", average, count)
	}

	if _, err := services.rating.UpsertRating(ctx, galgame.ID, userA, 6); err != nil {
		t.Fatalf("update first rating: %v", err)
	}
	average, count, _ = galgameStats(t, db, galgame.ID)
	if average != "8.00" || count != 2 {
		t.Fatalf("expected 8.00/2 after update, got %s/%d", average, count)
	}

	if err := services.rating.DeleteRating(ctx, galgame.ID, userB); err != nil {
		t.Fatalf("delete second rating: %v", err)
	}
	average, count, _ = galgameStats(t, db, galgame.ID)
	if average != "6.00" || count != 1 {
		t.Fatalf("expected 6.00/1 after delete, got %s/%d", average, count)
	}

	if err := services.rating.DeleteRating(ctx, galgame.ID, userA); err != nil {
		t.Fatalf("delete last rating: %v", err)
	}
	average, count, _ = galgameStats(t, db, galgame.ID)
	if average != "0.00" || count != 0 {
		t.Fatalf("expected reset 0.00/0, got %s/%d", average, count)
	}
	if err := services.rating.DeleteRating(ctx, galgame.ID, userA); !errors.Is(err, ErrRatingNotFound) {
		t.Fatalf("expected ErrRatingNotFound, got %v", err)
	}

	for _, invalid := range []int16{0, 11, -1} {
		if _, err := services.rating.UpsertRating(ctx, galgame.ID, userA, invalid); !errors.Is(err, ErrInvalidScore) {
			t.Fatalf("expected ErrInvalidScore for %d, got %v", invalid, err)
		}
	}

	if _, err := services.rating.UpsertRating(ctx, galgame.ID, userA, 10); err != nil {
		t.Fatalf("upsert full score: %v", err)
	}
	average, count, _ = galgameStats(t, db, galgame.ID)
	if average != "10.00" || count != 1 {
		t.Fatalf("expected full-mark 10.00/1, got %s/%d", average, count)
	}
}

func TestFavoriteLifecycle(t *testing.T) {
	services, db := newRelationTestServices(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, db, "favorite-user")
	galgame := createPublishedGalgame(t, services, user, "favorite-game")

	favorite, err := services.favorite.AddFavorite(ctx, galgame.ID, user)
	if err != nil {
		t.Fatalf("add favorite: %v", err)
	}
	if favorite.CreatedAt.IsZero() {
		t.Fatal("expected favorite created_at to be set")
	}
	_, _, count := galgameStats(t, db, galgame.ID)
	if count != 1 {
		t.Fatalf("expected favorite count 1, got %d", count)
	}

	if _, err := services.favorite.AddFavorite(ctx, galgame.ID, user); !errors.Is(err, ErrAlreadyFavorited) {
		t.Fatalf("expected ErrAlreadyFavorited, got %v", err)
	}
	_, _, count = galgameStats(t, db, galgame.ID)
	if count != 1 {
		t.Fatalf("expected favorite count to stay 1, got %d", count)
	}

	if err := services.favorite.RemoveFavorite(ctx, galgame.ID, user); err != nil {
		t.Fatalf("remove favorite: %v", err)
	}
	_, _, count = galgameStats(t, db, galgame.ID)
	if count != 0 {
		t.Fatalf("expected favorite count 0, got %d", count)
	}

	if err := services.favorite.RemoveFavorite(ctx, galgame.ID, user); !errors.Is(err, ErrFavoriteNotFound) {
		t.Fatalf("expected ErrFavoriteNotFound, got %v", err)
	}
	_, _, count = galgameStats(t, db, galgame.ID)
	if count != 0 {
		t.Fatalf("expected favorite count to stay 0, got %d", count)
	}
}

func TestFavoriteConcurrentCountNotLost(t *testing.T) {
	services, db := newRelationTestServices(t)
	ctx := context.Background()
	owner := testutil.CreateUser(t, db, "favorite-owner")
	galgame := createPublishedGalgame(t, services, owner, "favorite-concurrent-game")

	const users = 16
	userIDs := make([]uint, 0, users)
	for i := 0; i < users; i++ {
		userIDs = append(userIDs, testutil.CreateUser(t, db, fmt.Sprintf("favorite-user-%d", i)))
	}

	errs := make(chan error, users)
	var wg sync.WaitGroup
	for _, userID := range userIDs {
		wg.Add(1)
		go func(userID uint) {
			defer wg.Done()
			if _, err := services.favorite.AddFavorite(ctx, galgame.ID, userID); err != nil {
				errs <- err
			}
		}(userID)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent add favorite: %v", err)
	}

	_, _, count := galgameStats(t, db, galgame.ID)
	if count != users {
		t.Fatalf("expected favorite count %d, got %d", users, count)
	}

	var removed sync.WaitGroup
	for _, userID := range userIDs {
		removed.Add(1)
		go func(userID uint) {
			defer removed.Done()
			if err := services.favorite.RemoveFavorite(ctx, galgame.ID, userID); err != nil {
				t.Errorf("concurrent remove favorite: %v", err)
			}
		}(userID)
	}
	removed.Wait()

	_, _, count = galgameStats(t, db, galgame.ID)
	if count != 0 {
		t.Fatalf("expected favorite count 0 after concurrent removals, got %d", count)
	}
}

func TestUserStateCrud(t *testing.T) {
	services, db := newRelationTestServices(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, db, "state-user")
	galgame := createPublishedGalgame(t, services, user, "state-game")

	state, err := services.userState.UpsertState(ctx, galgame.ID, user, model.UserStateWish, 0)
	if err != nil {
		t.Fatalf("create state: %v", err)
	}
	if state.State != model.UserStateWish || state.PlayTimeMinutes != 0 {
		t.Fatalf("unexpected initial state: %+v", state)
	}

	state, err = services.userState.UpsertState(ctx, galgame.ID, user, model.UserStatePlaying, 120)
	if err != nil {
		t.Fatalf("update state: %v", err)
	}
	if state.State != model.UserStatePlaying || state.PlayTimeMinutes != 120 {
		t.Fatalf("unexpected updated state: %+v", state)
	}

	for _, invalid := range []int16{0, 6} {
		if _, err := services.userState.UpsertState(ctx, galgame.ID, user, invalid, 0); !errors.Is(err, ErrInvalidUserState) {
			t.Fatalf("expected ErrInvalidUserState for %d, got %v", invalid, err)
		}
	}
	if _, err := services.userState.UpsertState(ctx, galgame.ID, user, model.UserStatePlaying, -1); !errors.Is(err, ErrInvalidPlayTime) {
		t.Fatalf("expected ErrInvalidPlayTime, got %v", err)
	}

	if err := services.userState.DeleteState(ctx, galgame.ID, user); err != nil {
		t.Fatalf("delete state: %v", err)
	}
	if err := services.userState.DeleteState(ctx, galgame.ID, user); !errors.Is(err, ErrUserStateNotFound) {
		t.Fatalf("expected ErrUserStateNotFound, got %v", err)
	}
}

func TestGalgameRelationIsolation(t *testing.T) {
	services, db := newRelationTestServices(t)
	ctx := context.Background()
	userA := testutil.CreateUser(t, db, "relation-a")
	userB := testutil.CreateUser(t, db, "relation-b")
	userC := testutil.CreateUser(t, db, "relation-c")
	galgame := createPublishedGalgame(t, services, userA, "relation-game")

	if _, err := services.rating.UpsertRating(ctx, galgame.ID, userA, 8); err != nil {
		t.Fatalf("rate as userA: %v", err)
	}
	if _, err := services.rating.UpsertRating(ctx, galgame.ID, userB, 3); err != nil {
		t.Fatalf("rate as userB: %v", err)
	}
	if _, err := services.favorite.AddFavorite(ctx, galgame.ID, userA); err != nil {
		t.Fatalf("favorite as userA: %v", err)
	}
	if _, err := services.userState.UpsertState(ctx, galgame.ID, userA, model.UserStateCompleted, 600); err != nil {
		t.Fatalf("set state as userA: %v", err)
	}

	relationA, err := services.relation.GetGalgameRelation(ctx, galgame.ID, userA)
	if err != nil {
		t.Fatalf("get relation for userA: %v", err)
	}
	if relationA.Rating == nil || relationA.Rating.Score != 8 {
		t.Fatalf("expected userA score 8, got %+v", relationA.Rating)
	}
	if relationA.Favorite == nil {
		t.Fatal("expected userA favorite to exist")
	}
	if relationA.State == nil || relationA.State.State != model.UserStateCompleted {
		t.Fatalf("expected userA completed state, got %+v", relationA.State)
	}

	relationB, err := services.relation.GetGalgameRelation(ctx, galgame.ID, userB)
	if err != nil {
		t.Fatalf("get relation for userB: %v", err)
	}
	if relationB.Rating == nil || relationB.Rating.Score != 3 {
		t.Fatalf("expected userB score 3, got %+v", relationB.Rating)
	}
	if relationB.Favorite != nil || relationB.State != nil {
		t.Fatalf("expected no favorite/state for userB, got %+v", relationB)
	}

	relationC, err := services.relation.GetGalgameRelation(ctx, galgame.ID, userC)
	if err != nil {
		t.Fatalf("get relation for userC: %v", err)
	}
	if relationC.Rating != nil || relationC.Favorite != nil || relationC.State != nil {
		t.Fatalf("expected no relations for userC, got %+v", relationC)
	}

	pending := createTestGalgame(t, services.catalog, userA, "relation-pending-game", nil, nil, "",
		model.AgeRatingAll, model.GalgameStatusPending)
	ops := map[string]func() error{
		"rating": func() error {
			_, err := services.rating.UpsertRating(ctx, pending.ID, userA, 5)
			return err
		},
		"delete rating": func() error {
			return services.rating.DeleteRating(ctx, pending.ID, userA)
		},
		"favorite": func() error {
			_, err := services.favorite.AddFavorite(ctx, pending.ID, userA)
			return err
		},
		"remove favorite": func() error {
			return services.favorite.RemoveFavorite(ctx, pending.ID, userA)
		},
		"state": func() error {
			_, err := services.userState.UpsertState(ctx, pending.ID, userA, model.UserStatePlaying, 0)
			return err
		},
		"delete state": func() error {
			return services.userState.DeleteState(ctx, pending.ID, userA)
		},
		"relation": func() error {
			_, err := services.relation.GetGalgameRelation(ctx, pending.ID, userA)
			return err
		},
	}
	for name, op := range ops {
		if err := op(); !errors.Is(err, ErrGalgameNotFound) {
			t.Fatalf("%s on pending galgame: expected ErrGalgameNotFound, got %v", name, err)
		}
	}
}

func TestRelationWritesRollbackOnError(t *testing.T) {
	services, db := newRelationTestServices(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, db, "rollback-user")
	galgame := createPublishedGalgame(t, services, user, "rollback-game")

	injected := errors.New("injected failure")
	err := db.Transaction(func(tx *gorm.DB) error {
		relations := repository.NewUserRelationRepository(tx)
		inserted, err := relations.AddFavorite(ctx, galgame.ID, user)
		if err != nil {
			return err
		}
		if !inserted {
			t.Fatal("expected favorite insert")
		}
		if err := relations.IncrementFavoriteCount(ctx, galgame.ID); err != nil {
			return err
		}
		return injected
	})
	if !errors.Is(err, injected) {
		t.Fatalf("expected injected failure, got %v", err)
	}

	relations := repository.NewUserRelationRepository(db)
	if favorite, err := relations.FindFavorite(ctx, galgame.ID, user); err != nil || favorite != nil {
		t.Fatalf("expected favorite rolled back, got %+v err=%v", favorite, err)
	}
	_, _, favoriteCount := galgameStats(t, db, galgame.ID)
	if favoriteCount != 0 {
		t.Fatalf("expected favorite count rolled back to 0, got %d", favoriteCount)
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		relations := repository.NewUserRelationRepository(tx)
		if err := relations.UpsertRating(ctx, galgame.ID, user, 7); err != nil {
			return err
		}
		if err := relations.RecalculateGalgameRating(ctx, galgame.ID); err != nil {
			return err
		}
		return injected
	})
	if !errors.Is(err, injected) {
		t.Fatalf("expected injected failure, got %v", err)
	}

	if rating, err := relations.FindRating(ctx, galgame.ID, user); err != nil || rating != nil {
		t.Fatalf("expected rating rolled back, got %+v err=%v", rating, err)
	}
	average, ratingCount, _ := galgameStats(t, db, galgame.ID)
	if average != "0.00" || ratingCount != 0 {
		t.Fatalf("expected rating stats rolled back, got %s/%d", average, ratingCount)
	}
}
