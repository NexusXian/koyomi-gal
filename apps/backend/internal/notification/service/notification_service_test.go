package service

import (
	"context"
	"testing"

	"backend/internal/notification/model"
	"backend/internal/notification/repository"
	"backend/internal/testutil"
)

func TestNotificationFromInputNullableNavigation(t *testing.T) {
	notification, err := notificationFromInput(CreateInput{
		RecipientID: 1, Category: model.CategorySystem, Type: model.TypeSystem, Title: "system",
	})
	if err != nil {
		t.Fatalf("build notification: %v", err)
	}
	if notification.EntityType != nil || notification.EntityID != nil || notification.TargetURL != nil {
		t.Fatalf("empty navigation input should become nil: %+v", notification)
	}
}

func TestCreateAndCreateManySuppressSelfNotifications(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	svc := NewNotificationService(repository.NewNotificationRepository(db, "https://img.example.com"))
	ctx := context.Background()
	recipient := testutil.CreateUser(t, db, "notification-service-recipient")
	actor := testutil.CreateUser(t, db, "notification-service-actor")

	self, err := svc.Create(ctx, CreateInput{
		RecipientID: recipient, ActorID: &recipient, Category: model.CategoryInteraction,
		Type: model.TypePostLiked, Title: "self", Metadata: map[string]any{"post_id": recipient},
	})
	if err != nil || self != nil {
		t.Fatalf("self notification should be suppressed, notification=%+v err=%v", self, err)
	}

	created, err := svc.CreateMany(ctx, []CreateInput{
		{
			RecipientID: recipient, ActorID: &recipient, Category: model.CategoryInteraction,
			Type: model.TypePostLiked, Title: "self bulk",
		},
		{
			RecipientID: recipient, ActorID: &actor, Category: model.CategoryInteraction,
			Type: model.TypePostLiked, EntityType: "post", EntityID: 2,
			Title: "delivered", TargetURL: "/posts/2",
			Metadata: map[string]any{"nested": map[string]any{"ok": true}},
		},
		{
			RecipientID: actor, Category: model.CategorySystem,
			Type: model.TypeSystem, Title: "system",
		},
	})
	if err != nil {
		t.Fatalf("create many notifications: %v", err)
	}
	if len(created) != 2 || created[0].ID == 0 || created[1].ID == 0 {
		t.Fatalf("expected two persisted notifications, got %+v", created)
	}
	if created[0].EntityType == nil || *created[0].EntityType != "post" || created[0].EntityID == nil || *created[0].EntityID != 2 || created[0].TargetURL == nil || *created[0].TargetURL != "/posts/2" {
		t.Fatalf("expected entity navigation pointers, got %+v", created[0])
	}
	if created[1].EntityType != nil || created[1].EntityID != nil || created[1].TargetURL != nil {
		t.Fatalf("expected empty navigation input to persist as null, got %+v", created[1])
	}

	var count int64
	if err := db.Model(&model.Notification{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("expected two notification rows, count=%d err=%v", count, err)
	}
}

func TestCreateRejectsUnencodableMetadata(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	svc := NewNotificationService(repository.NewNotificationRepository(db, ""))
	recipient := testutil.CreateUser(t, db, "notification-service-metadata")

	_, err := svc.Create(context.Background(), CreateInput{
		RecipientID: recipient, Category: model.CategorySystem,
		Type: model.TypeSystem, Title: "bad",
		Metadata: map[string]any{"unsupported": make(chan int)},
	})
	if err == nil {
		t.Fatal("expected unsupported metadata to fail")
	}
}
