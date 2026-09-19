package service

import (
	"context"
	"errors"
	"math"
	"testing"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/repository"
	notificationRepo "backend/internal/notification/repository"
	notificationService "backend/internal/notification/service"
	"backend/internal/testutil"
	userRepository "backend/internal/user/repository"
)

func ratingValue(value int16) *int16 { return &value }

func TestMultidimensionalRatingLifecycle(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	cache := testutil.NewRedis(t)
	galgames := repository.NewGalgameRepository(db)
	relations := repository.NewUserRelationRepository(db, "https://img.example.com")
	services := &relationTestServices{
		catalog: NewCatalogService(galgames, repository.NewDeveloperRepository(db), repository.NewTagRepository(db)),
		rating:  NewRatingService(galgames, relations, cache),
	}
	ctx := context.Background()
	userA := testutil.CreateUser(t, db, "rating-detail-a")
	userB := testutil.CreateUser(t, db, "rating-detail-b")
	galgame := createPublishedGalgame(t, services, userA, "rating-detail-game")
	missing, err := services.rating.GetMyRating(ctx, galgame.ID, userA)
	if err != nil || missing != nil {
		t.Fatalf("expected no current-user rating without error, got rating=%+v err=%v", missing, err)
	}

	empty, err := services.rating.Summary(ctx, galgame.ID)
	if err != nil {
		t.Fatalf("get empty summary: %v", err)
	}
	if empty.Count != 0 || empty.Overall != nil || empty.Visual.Average != nil || empty.Visual.Count != 0 {
		t.Fatalf("expected null empty averages, got %+v", empty)
	}
	if ttl := cache.TTL(ctx, ratingSummaryCacheKey(galgame.ID)).Val(); ttl <= 0 || ttl > ratingSummaryTTL {
		t.Fatalf("expected summary cache TTL up to %v, got %v", ratingSummaryTTL, ttl)
	}

	recommend := int16(2)
	review := "Detailed review"
	ratingA, err := services.rating.PutRating(ctx, galgame.ID, userA, &dto.PutRatingRequest{
		Overall:        8,
		Visual:         ratingValue(8),
		Music:          ratingValue(10),
		Recommendation: &recommend,
		ReviewText:     &review,
		SpoilerLevel:   2,
	})
	if err != nil {
		t.Fatalf("put first detailed rating: %v", err)
	}
	if ratingA.Score != 8 || ratingA.Visual == nil || *ratingA.Visual != 8 || ratingA.Story != nil || ratingA.ReviewText == nil {
		t.Fatalf("unexpected first detailed rating: %+v", ratingA)
	}
	if cache.Exists(ctx, ratingSummaryCacheKey(galgame.ID)).Val() != 0 {
		t.Fatal("expected detailed upsert to invalidate summary cache")
	}

	if _, err := services.rating.PutRating(ctx, galgame.ID, userB, &dto.PutRatingRequest{
		Overall: 6,
		Visual:  ratingValue(10),
		Story:   ratingValue(4),
	}); err != nil {
		t.Fatalf("put second detailed rating: %v", err)
	}
	summary, err := services.rating.Summary(ctx, galgame.ID)
	if err != nil {
		t.Fatalf("get populated summary: %v", err)
	}
	if summary.Count != 2 || summary.Overall == nil || math.Abs(*summary.Overall-7) > 0.0001 {
		t.Fatalf("unexpected overall summary: %+v", summary)
	}
	if summary.Visual.Count != 2 || summary.Visual.Average == nil || math.Abs(*summary.Visual.Average-9) > 0.0001 {
		t.Fatalf("unexpected visual summary: %+v", summary.Visual)
	}
	if summary.Story.Count != 1 || summary.Story.Average == nil || *summary.Story.Average != 4 || summary.Replay.Average != nil {
		t.Fatalf("unexpected nullable dimension summaries: %+v", summary)
	}

	if _, err := services.rating.UpsertRating(ctx, galgame.ID, userA, 10); err != nil {
		t.Fatalf("legacy score update: %v", err)
	}
	preserved, err := services.rating.GetMyRating(ctx, galgame.ID, userA)
	if err != nil {
		t.Fatalf("get rating after legacy update: %v", err)
	}
	if preserved.Score != 10 || preserved.Visual == nil || *preserved.Visual != 8 || preserved.ReviewText == nil || *preserved.ReviewText != review {
		t.Fatalf("legacy update did not preserve extended fields: %+v", preserved)
	}

	if err := relations.UpsertUserState(ctx, galgame.ID, userA, 3, 0); err != nil {
		t.Fatalf("set play status: %v", err)
	}
	liked, err := services.rating.LikeRating(ctx, ratingA.ID, userB)
	if err != nil {
		t.Fatalf("like rating: %v", err)
	}
	liked, err = services.rating.LikeRating(ctx, ratingA.ID, userB)
	if err != nil {
		t.Fatalf("repeat like rating: %v", err)
	}
	if liked.LikeCount != 1 || !liked.Liked {
		t.Fatalf("expected idempotent like count 1, got %+v", liked)
	}

	items, total, err := services.rating.ListRatings(ctx, galgame.ID, &userB, 1, 20, "popular")
	if err != nil {
		t.Fatalf("list popular ratings: %v", err)
	}
	if total != 2 || len(items) != 2 || items[0].ID != ratingA.ID || !items[0].Liked || items[0].PlayStatus == nil || *items[0].PlayStatus != 3 {
		t.Fatalf("unexpected authenticated rating list: total=%d items=%+v", total, items)
	}
	items, _, err = services.rating.ListRatings(ctx, galgame.ID, nil, 1, 20, "highest")
	if err != nil {
		t.Fatalf("list anonymous ratings: %v", err)
	}
	if len(items) != 2 || items[0].Score != 10 || items[0].Liked {
		t.Fatalf("unexpected anonymous highest list: %+v", items)
	}

	unliked, err := services.rating.UnlikeRating(ctx, ratingA.ID, userB)
	if err != nil {
		t.Fatalf("unlike rating: %v", err)
	}
	unliked, err = services.rating.UnlikeRating(ctx, ratingA.ID, userB)
	if err != nil {
		t.Fatalf("repeat unlike rating: %v", err)
	}
	if unliked.LikeCount != 0 || unliked.Liked {
		t.Fatalf("expected idempotent unlike count 0, got %+v", unliked)
	}

	if _, err := services.rating.Summary(ctx, galgame.ID); err != nil {
		t.Fatalf("repopulate summary cache: %v", err)
	}
	if err := services.rating.DeleteRating(ctx, galgame.ID, userA); err != nil {
		t.Fatalf("delete detailed rating: %v", err)
	}
	if cache.Exists(ctx, ratingSummaryCacheKey(galgame.ID)).Val() != 0 {
		t.Fatal("expected delete to invalidate summary cache")
	}
}

func TestPutRatingValidation(t *testing.T) {
	service := NewRatingService(nil, nil)
	tests := []dto.PutRatingRequest{
		{Overall: 0},
		{Overall: 11},
		{Overall: 8, SpoilerLevel: -1},
		{Overall: 8, SpoilerLevel: 3},
		{Overall: 8, Visual: ratingValue(0)},
		{Overall: 8, Replay: ratingValue(11)},
		{Overall: 8, Recommendation: ratingValue(-2)},
		{Overall: 8, Recommendation: ratingValue(3)},
	}
	for _, request := range tests {
		if _, err := service.PutRating(context.Background(), 1, 1, &request); !errors.Is(err, ErrInvalidRating) {
			t.Fatalf("expected ErrInvalidRating for %+v, got %v", request, err)
		}
	}
}

func TestRatingLikeNotification(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	galgames := repository.NewGalgameRepository(db)
	relations := repository.NewUserRelationRepository(db, "https://img.example.com")
	notifications := notificationService.NewNotificationService(
		notificationRepo.NewNotificationRepository(db, "https://img.example.com"),
	)
	rating := NewRatingService(galgames, relations)
	rating.SetNotificationService(notifications)
	catalog := NewCatalogService(galgames, repository.NewDeveloperRepository(db), repository.NewTagRepository(db))
	ctx := context.Background()

	author := testutil.CreateUser(t, db, "rating-notify-author")
	liker := testutil.CreateUser(t, db, "rating-notify-liker")
	galgame := createPublishedGalgame(t, &relationTestServices{catalog: catalog, rating: rating}, author, "rating-notify-game")
	mine, err := rating.PutRating(ctx, galgame.ID, author, &dto.PutRatingRequest{Overall: 9})
	if err != nil {
		t.Fatalf("put rating: %v", err)
	}

	countNotifications := func() int64 {
		var count int64
		db.Table("notifications").
			Where("recipient_id = ? AND type = ?", author, "rating_liked").
			Count(&count)
		return count
	}

	if _, err := rating.LikeRating(ctx, mine.ID, liker); err != nil {
		t.Fatalf("like rating: %v", err)
	}
	if countNotifications() != 1 {
		t.Fatal("expected one rating_liked notification for the author")
	}
	if _, err := rating.LikeRating(ctx, mine.ID, liker); err != nil {
		t.Fatalf("repeat like rating: %v", err)
	}
	if countNotifications() != 1 {
		t.Fatal("expected idempotent like not to duplicate notification")
	}

	other := testutil.CreateUser(t, db, "rating-notify-self")
	if _, err := rating.PutRating(ctx, galgame.ID, other, &dto.PutRatingRequest{Overall: 7}); err != nil {
		t.Fatalf("put self rating: %v", err)
	}
	selfRating, err := rating.GetMyRating(ctx, galgame.ID, other)
	if err != nil {
		t.Fatalf("find self rating: %v", err)
	}
	if _, err := rating.LikeRating(ctx, selfRating.ID, other); err != nil {
		t.Fatalf("self like rating: %v", err)
	}
	var selfCount int64
	db.Table("notifications").Where("recipient_id = ?", other).Count(&selfCount)
	if selfCount != 0 {
		t.Fatal("expected no notification for self like")
	}
}

func TestRatingListPrivacyFilter(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	galgames := repository.NewGalgameRepository(db)
	relations := repository.NewUserRelationRepository(db, "https://img.example.com")
	rating := NewRatingService(galgames, relations)
	catalog := NewCatalogService(galgames, repository.NewDeveloperRepository(db), repository.NewTagRepository(db))
	ctx := context.Background()

	publicUser := testutil.CreateUser(t, db, "rating-privacy-public")
	privateUser := testutil.CreateUser(t, db, "rating-privacy-private")
	viewer := testutil.CreateUser(t, db, "rating-privacy-viewer")
	galgame := createPublishedGalgame(t, &relationTestServices{catalog: catalog, rating: rating}, publicUser, "rating-privacy-game")
	if _, err := rating.PutRating(ctx, galgame.ID, publicUser, &dto.PutRatingRequest{Overall: 8}); err != nil {
		t.Fatalf("put public rating: %v", err)
	}
	if _, err := rating.PutRating(ctx, galgame.ID, privateUser, &dto.PutRatingRequest{Overall: 6}); err != nil {
		t.Fatalf("put private rating: %v", err)
	}
	if err := db.Exec("UPDATE user_privacy_settings SET show_ratings = FALSE WHERE user_id = ?", privateUser).Error; err != nil {
		t.Fatalf("hide private user ratings: %v", err)
	}

	items, total, err := rating.ListRatings(ctx, galgame.ID, &viewer, 1, 20, "newest")
	if err != nil {
		t.Fatalf("list ratings for other viewer: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].UserID != publicUser {
		t.Fatalf("expected only the public author's rating, total=%d items=%+v", total, items)
	}

	items, total, err = rating.ListRatings(ctx, galgame.ID, nil, 1, 20, "newest")
	if err != nil {
		t.Fatalf("list ratings anonymously: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("expected anonymous list to hide private rating, total=%d items=%+v", total, items)
	}

	items, total, err = rating.ListRatings(ctx, galgame.ID, &privateUser, 1, 20, "newest")
	if err != nil {
		t.Fatalf("list ratings as private author: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("expected author to still see own rating, total=%d items=%+v", total, items)
	}
}

func TestProfileRatingsIncludeDimensions(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	galgames := repository.NewGalgameRepository(db)
	relations := repository.NewUserRelationRepository(db, "https://img.example.com")
	rating := NewRatingService(galgames, relations)
	catalog := NewCatalogService(galgames, repository.NewDeveloperRepository(db), repository.NewTagRepository(db))
	ctx := context.Background()

	user := testutil.CreateUser(t, db, "rating-profile-dims")
	galgame := createPublishedGalgame(t, &relationTestServices{catalog: catalog, rating: rating}, user, "rating-profile-game")
	review := "Great story"
	recommend := int16(2)
	if _, err := rating.PutRating(ctx, galgame.ID, user, &dto.PutRatingRequest{
		Overall: 9, Story: ratingValue(10), Voice: ratingValue(8),
		Recommendation: &recommend, ReviewText: &review, SpoilerLevel: 1,
	}); err != nil {
		t.Fatalf("put rating: %v", err)
	}

	profiles := userRepository.NewUserProfileRepository(db, "https://img.example.com")
	items, total, err := profiles.ListRatings(ctx, user, 1, 20)
	if err != nil {
		t.Fatalf("list profile ratings: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("expected one profile rating, total=%d items=%+v", total, items)
	}
	item := items[0]
	if item.Score == nil || *item.Score != 9 || item.Story == nil || *item.Story != 10 || item.Voice == nil || *item.Voice != 8 {
		t.Fatalf("expected overall and dimension scores, got %+v", item)
	}
	if item.Visual != nil || item.Replay != nil {
		t.Fatalf("expected unset dimensions to stay null, got %+v", item)
	}
	if item.Recommendation == nil || *item.Recommendation != 2 || item.ReviewText == nil || *item.ReviewText != review || item.SpoilerLevel == nil || *item.SpoilerLevel != 1 {
		t.Fatalf("expected review fields on profile rating, got %+v", item)
	}
}
