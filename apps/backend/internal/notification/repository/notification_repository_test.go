package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	imageModel "backend/internal/image/model"
	"backend/internal/notification/dto"
	"backend/internal/notification/model"
	"backend/internal/testutil"
)

func TestOwnListReadIsolationReadAllAndUnreadCount(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	repo := NewNotificationRepository(db, "https://img.example.com/")
	ctx := context.Background()
	recipient := testutil.CreateUser(t, db, "notification-recipient")
	other := testutil.CreateUser(t, db, "notification-other")
	actor := testutil.CreateUser(t, db, "notification-actor")

	var avatarID uint
	if err := db.Raw(`
INSERT INTO image_assets (
    user_id, object_key, original_name, mime_type, extension, size, category, status, created_at, updated_at
) VALUES (?, 'avatars/actor.webp', 'actor.webp', 'image/webp', 'webp', 10, 'avatars', ?, NOW(), NOW())
RETURNING id
`, actor, imageModel.ImageStatusActive).Scan(&avatarID).Error; err != nil {
		t.Fatalf("create actor avatar: %v", err)
	}
	if err := db.Exec("UPDATE users SET avatar_asset_id = ? WHERE id = ?", avatarID, actor).Error; err != nil {
		t.Fatalf("assign actor avatar: %v", err)
	}

	entityType := "post"
	entityID := uint(1)
	targetURL := "/posts/1"
	older := model.Notification{
		RecipientID: recipient, ActorID: &actor, Category: model.CategoryInteraction,
		Type: model.TypePostLiked, EntityType: &entityType, EntityID: &entityID,
		Title: "older", TargetURL: &targetURL,
		Metadata: model.Metadata{"post_id": float64(1)}, CreatedAt: time.Now().Add(-time.Minute),
	}
	newer := model.Notification{
		RecipientID: recipient, Category: model.CategorySystem,
		Type: model.TypeSystem, Title: "newer", Metadata: model.Metadata{}, CreatedAt: time.Now(),
	}
	foreign := model.Notification{
		RecipientID: other, ActorID: &actor, Category: model.CategoryInteraction,
		Type: model.TypePostLiked, Title: "foreign", Metadata: model.Metadata{},
	}
	for _, notification := range []*model.Notification{&older, &newer, &foreign} {
		if err := repo.Create(ctx, notification); err != nil {
			t.Fatalf("create notification %s: %v", notification.Title, err)
		}
	}

	items, total, err := repo.List(ctx, ListOptions{RecipientID: recipient, Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if total != 2 || len(items) != 2 || items[0].ID != newer.ID || items[1].ID != older.ID {
		t.Fatalf("unexpected own list: total=%d items=%+v", total, items)
	}
	if items[1].ActorName != "notification-actor" || items[1].ActorAvatar != "https://img.example.com/avatars/actor.webp" {
		t.Fatalf("unexpected resolved actor: %+v", items[1])
	}
	if _, exists := items[1].Metadata["post_id"]; !exists {
		t.Fatalf("metadata did not round trip: %+v", items[1].Metadata)
	}
	if items[1].EntityType == nil || *items[1].EntityType != "post" || items[1].EntityID == nil || *items[1].EntityID != 1 || items[1].TargetURL == nil || *items[1].TargetURL != "/posts/1" {
		t.Fatalf("entity navigation did not round trip: %+v", items[1])
	}
	if err := db.Exec("DELETE FROM users WHERE id = ?", actor).Error; err != nil {
		t.Fatalf("delete actor: %v", err)
	}
	items, _, err = repo.List(ctx, ListOptions{RecipientID: recipient, Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("list after actor deletion: %v", err)
	}
	if items[1].ActorID != nil || dto.NewNotificationData(&items[1]).Actor != nil {
		t.Fatalf("deleted actor should be nil: %+v", items[1])
	}

	count, err := repo.UnreadCount(ctx, recipient)
	if err != nil || count != 2 {
		t.Fatalf("expected two unread notifications, count=%d err=%v", count, err)
	}
	if _, err := repo.MarkRead(ctx, other, older.ID); !errors.Is(err, ErrNotificationNotFound) {
		t.Fatalf("foreign mark read should be not found, got %v", err)
	}
	marked, err := repo.MarkRead(ctx, recipient, older.ID)
	if err != nil {
		t.Fatalf("mark own notification read: %v", err)
	}
	if !marked.IsRead || marked.ReadAt == nil {
		t.Fatalf("marked notification should have is_read and read_at set: %+v", marked)
	}
	read := false
	readItems, readTotal, err := repo.List(ctx, ListOptions{
		RecipientID: recipient, Page: 1, Limit: 20,
		Category: model.CategoryInteraction, Unread: &read,
	})
	if err != nil || readTotal != 1 || len(readItems) != 1 || readItems[0].ID != older.ID {
		t.Fatalf("unexpected read interaction filter: total=%d items=%+v err=%v", readTotal, readItems, err)
	}
	count, err = repo.UnreadCount(ctx, recipient)
	if err != nil || count != 1 {
		t.Fatalf("expected one unread notification, count=%d err=%v", count, err)
	}

	updated, err := repo.MarkAllRead(ctx, recipient)
	if err != nil || updated != 1 {
		t.Fatalf("mark all read: updated=%d err=%v", updated, err)
	}
	var markedAll model.Notification
	if err := db.First(&markedAll, newer.ID).Error; err != nil || !markedAll.IsRead || markedAll.ReadAt == nil {
		t.Fatalf("mark all should set is_read and read_at: notification=%+v err=%v", markedAll, err)
	}
	count, err = repo.UnreadCount(ctx, recipient)
	if err != nil || count != 0 {
		t.Fatalf("expected no unread notifications, count=%d err=%v", count, err)
	}
	foreignCount, err := repo.UnreadCount(ctx, other)
	if err != nil || foreignCount != 1 {
		t.Fatalf("foreign notification should remain unread, count=%d err=%v", foreignCount, err)
	}
}
