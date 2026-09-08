package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"backend/internal/message/dto"
	"backend/internal/message/repository"
	"backend/internal/testutil"

	"gorm.io/gorm"
)

// TestMessageServiceAcceptance covers the private-message acceptance workflow
// against PostgreSQL and a temporary redis-server. It skips when either
// dependency is unavailable locally.
func TestMessageServiceAcceptance(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	client := testutil.NewRedis(t)

	ctx := context.Background()
	repo := repository.NewMessageRepository(db, "https://cdn.example.com")
	svc := NewMessageService(repo, repository.NewRateLimiter(client), nil)

	alice := testutil.CreateUser(t, db, "msg-alice")
	bob := testutil.CreateUser(t, db, "msg-bob")
	carol := testutil.CreateUser(t, db, "msg-carol")
	dave := testutil.CreateUser(t, db, "msg-dave")

	var conversationID uint
	t.Run("create conversation is stable in both directions", func(t *testing.T) {
		created, err := svc.CreateConversation(ctx, alice, &dto.CreateConversationRequest{UserID: bob})
		if err != nil {
			t.Fatalf("create conversation: %v", err)
		}
		if created.ID == 0 || created.Type != "direct" || created.User.ID != bob {
			t.Fatalf("unexpected conversation: %+v", created)
		}
		repeat, err := svc.CreateConversation(ctx, alice, &dto.CreateConversationRequest{UserID: bob})
		if err != nil {
			t.Fatalf("repeat conversation: %v", err)
		}
		if repeat.ID != created.ID {
			t.Fatalf("repeat returned %d, want %d", repeat.ID, created.ID)
		}
		reverse, err := svc.CreateConversation(ctx, bob, &dto.CreateConversationRequest{UserID: alice})
		if err != nil {
			t.Fatalf("reverse conversation: %v", err)
		}
		if reverse.ID != created.ID {
			t.Fatalf("reverse returned %d, want %d", reverse.ID, created.ID)
		}
		conversationID = created.ID
	})

	t.Run("cannot message self", func(t *testing.T) {
		if _, err := svc.CreateConversation(ctx, alice, &dto.CreateConversationRequest{UserID: alice}); !errors.Is(err, ErrCannotMessageSelf) {
			t.Fatalf("expected ErrCannotMessageSelf, got %v", err)
		}
	})

	var firstMessageID uint
	t.Run("send message increments receiver unread", func(t *testing.T) {
		sent, err := svc.SendMessage(ctx, alice, conversationID, &dto.SendMessageRequest{Content: " 你好 "})
		if err != nil {
			t.Fatalf("send message: %v", err)
		}
		if sent.ID == 0 || sent.SenderID != alice || sent.Content != "你好" || sent.ConversationID != conversationID {
			t.Fatalf("unexpected message: %+v", sent)
		}
		firstMessageID = sent.ID
		count, err := svc.UnreadCount(ctx, bob)
		if err != nil {
			t.Fatalf("unread count: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected receiver unread 1, got %d", count)
		}
	})

	t.Run("non member cannot access conversation", func(t *testing.T) {
		if _, err := svc.SendMessage(ctx, carol, conversationID, &dto.SendMessageRequest{Content: "hi"}); err == nil ||
			(!errors.Is(err, ErrConversationNotFound) && !errors.Is(err, ErrConversationAccessDenied)) {
			t.Fatalf("expected concealed access denial, got %v", err)
		}
	})

	var latestMessageID uint
	t.Run("mark read zeroes unread and never regresses", func(t *testing.T) {
		sent, err := svc.SendMessage(ctx, alice, conversationID, &dto.SendMessageRequest{Content: "second"})
		if err != nil {
			t.Fatalf("send second message: %v", err)
		}
		latestMessageID = sent.ID
		count, err := svc.UnreadCount(ctx, bob)
		if err != nil || count != 2 {
			t.Fatalf("expected receiver unread 2, got %d err %v", count, err)
		}
		if err := svc.MarkRead(ctx, bob, conversationID, latestMessageID); err != nil {
			t.Fatalf("mark read: %v", err)
		}
		count, err = svc.UnreadCount(ctx, bob)
		if err != nil || count != 0 {
			t.Fatalf("expected unread 0 after read, got %d err %v", count, err)
		}
		if err := svc.MarkRead(ctx, bob, conversationID, firstMessageID); err != nil {
			t.Fatalf("stale mark read: %v", err)
		}
		count, err = svc.UnreadCount(ctx, bob)
		if err != nil || count != 0 {
			t.Fatalf("stale mark read regressed unread to %d err %v", count, err)
		}
	})

	t.Run("permission none rejects new conversation", func(t *testing.T) {
		if err := repo.SetMessagePermission(ctx, dave, "none"); err != nil {
			t.Fatalf("set permission: %v", err)
		}
		if _, err := svc.CreateConversation(ctx, carol, &dto.CreateConversationRequest{UserID: dave}); !errors.Is(err, ErrMessagePermissionDenied) {
			t.Fatalf("expected ErrMessagePermissionDenied, got %v", err)
		}
	})

	t.Run("blocking denies sending in both directions", func(t *testing.T) {
		if err := svc.BlockUser(ctx, bob, alice); err != nil {
			t.Fatalf("block user: %v", err)
		}
		if _, err := svc.SendMessage(ctx, alice, conversationID, &dto.SendMessageRequest{Content: "blocked?"}); !errors.Is(err, ErrUserBlocked) {
			t.Fatalf("expected ErrUserBlocked, got %v", err)
		}
		if err := svc.UnblockUser(ctx, bob, alice); err != nil {
			t.Fatalf("unblock user: %v", err)
		}

		if err := svc.BlockUser(ctx, alice, bob); err != nil {
			t.Fatalf("block user reverse: %v", err)
		}
		if _, err := svc.SendMessage(ctx, bob, conversationID, &dto.SendMessageRequest{Content: "blocked?"}); !errors.Is(err, ErrUserBlocked) {
			t.Fatalf("expected reverse ErrUserBlocked, got %v", err)
		}
		if err := svc.UnblockUser(ctx, alice, bob); err != nil {
			t.Fatalf("unblock reverse: %v", err)
		}
	})

	t.Run("delete only own message", func(t *testing.T) {
		if err := svc.DeleteMessage(ctx, bob, latestMessageID); !errors.Is(err, ErrMessageAccessDenied) {
			t.Fatalf("expected ErrMessageAccessDenied, got %v", err)
		}
		if err := svc.DeleteMessage(ctx, alice, latestMessageID); err != nil {
			t.Fatalf("delete own message: %v", err)
		}
		messages, err := svc.ListMessages(ctx, alice, conversationID, 0, 30)
		if err != nil {
			t.Fatalf("list messages: %v", err)
		}
		for _, message := range messages.List {
			if message.ID == latestMessageID && (!message.IsDeleted || message.Content != "") {
				t.Fatalf("deleted message not sanitized: %+v", message)
			}
		}
	})

	t.Run("delete conversation only hides actor and revives on new message", func(t *testing.T) {
		if err := svc.DeleteConversation(ctx, bob, conversationID); err != nil {
			t.Fatalf("delete conversation: %v", err)
		}
		if err := listMissing(db, bob, conversationID); err != nil {
			t.Fatalf("hidden conversation still visible: %v", err)
		}
		if err := listContains(db, alice, conversationID); err != nil {
			t.Fatalf("peer lost conversation: %v", err)
		}
		count, err := svc.UnreadCount(ctx, bob)
		if err != nil || count != 0 {
			t.Fatalf("hidden conversation should drop unread, got %d err %v", count, err)
		}
		if _, err := svc.SendMessage(ctx, alice, conversationID, &dto.SendMessageRequest{Content: "revive"}); err != nil {
			t.Fatalf("send revive message: %v", err)
		}
		if err := listContains(db, bob, conversationID); err != nil {
			t.Fatalf("conversation not revived: %v", err)
		}
		count, err = svc.UnreadCount(ctx, bob)
		if err != nil || count != 1 {
			t.Fatalf("expected revived unread 1, got %d err %v", count, err)
		}
	})
}

func listContains(db *gorm.DB, userID, conversationID uint) error {
	visible, err := visibleConversation(db, userID, conversationID)
	if err != nil {
		return err
	}
	if !visible {
		return fmt.Errorf("conversation %d not visible to user %d", conversationID, userID)
	}
	return nil
}

func listMissing(db *gorm.DB, userID, conversationID uint) error {
	visible, err := visibleConversation(db, userID, conversationID)
	if err != nil {
		return err
	}
	if visible {
		return fmt.Errorf("conversation %d unexpectedly visible to user %d", conversationID, userID)
	}
	return nil
}

func visibleConversation(db *gorm.DB, userID, conversationID uint) (bool, error) {
	var count int64
	err := db.Table("conversation_members").
		Where("conversation_id = ? AND user_id = ? AND deleted_at IS NULL", conversationID, userID).
		Count(&count).Error
	return count > 0, err
}
