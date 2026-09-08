package repository

import (
	"context"
	"errors"
	"sync"
	"testing"

	"backend/internal/message/model"
	"backend/internal/testutil"
)

func TestConcurrentDirectConversationCreationUsesOneConversation(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	firstID := testutil.CreateUser(t, db, "message-first")
	secondID := testutil.CreateUser(t, db, "message-second")
	repo := NewMessageRepository(db, "https://cdn.example.com")

	start := make(chan struct{})
	errorsChannel := make(chan error, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			err := repo.Transaction(context.Background(), func(tx *MessageRepository) error {
				if err := tx.LockUsers(context.Background(), firstID, secondID); err != nil {
					return err
				}
				conversationID, err := tx.FindDirectConversationID(context.Background(), firstID, secondID)
				if err != nil || conversationID != 0 {
					return err
				}
				return tx.CreateDirectConversation(context.Background(), &model.Conversation{
					ConversationType: model.ConversationTypeDirect,
				}, firstID, secondID)
			})
			errorsChannel <- err
		}()
	}
	close(start)
	wait.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			t.Fatalf("create direct conversation: %v", err)
		}
	}

	var conversations, directConversations, members int64
	if err := db.Table("conversations").Count(&conversations).Error; err != nil {
		t.Fatalf("count conversations: %v", err)
	}
	if err := db.Table("direct_conversations").Count(&directConversations).Error; err != nil {
		t.Fatalf("count direct conversations: %v", err)
	}
	if err := db.Table("conversation_members").Count(&members).Error; err != nil {
		t.Fatalf("count conversation members: %v", err)
	}
	if conversations != 1 || directConversations != 1 || members != 2 {
		t.Fatalf("unexpected rows: conversations=%d direct=%d members=%d", conversations, directConversations, members)
	}

	conversationID, err := repo.FindDirectConversationID(context.Background(), firstID, secondID)
	if err != nil {
		t.Fatalf("find created conversation: %v", err)
	}
	record, err := repo.Conversation(context.Background(), firstID, conversationID)
	if err != nil {
		t.Fatalf("load empty conversation: %v", err)
	}
	if record.ConversationType != model.ConversationTypeDirect || record.LastMessageAt != nil || record.SortAt.IsZero() {
		t.Fatalf("unexpected empty conversation: %+v", record)
	}
	listed, err := repo.ListConversations(context.Background(), firstID, nil, 20)
	if err != nil {
		t.Fatalf("list empty conversation: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != conversationID {
		t.Fatalf("expected empty conversation in list, got %+v", listed)
	}

	outsiderID := testutil.CreateUser(t, db, "message-outsider")
	if _, err := repo.ListMessages(context.Background(), outsiderID, conversationID, 0, 30); !errors.Is(err, ErrConversationAccessDenied) {
		t.Fatalf("expected conversation access denied, got %v", err)
	}
	if _, err := repo.ListMessages(context.Background(), firstID, conversationID+1000, 0, 30); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected conversation not found, got %v", err)
	}

	message := &model.Message{
		ConversationID: conversationID, SenderID: firstID, ReceiverID: secondID,
		MessageType: model.MessageTypeText, Content: "access test",
	}
	if err := repo.CreateMessage(context.Background(), message); err != nil {
		t.Fatalf("create access test message: %v", err)
	}
	if _, err := repo.LockMessageForDelete(context.Background(), outsiderID, message.ID); !errors.Is(err, ErrMessageAccessDenied) {
		t.Fatalf("expected message access denied, got %v", err)
	}
	if _, err := repo.LockMessageForDelete(context.Background(), firstID, message.ID+1000); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("expected message not found, got %v", err)
	}
}

func TestSetMessagePermissionCreatesMissingDefaults(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	userID := testutil.CreateUser(t, db, "message-settings")
	if err := db.Exec("DELETE FROM user_privacy_settings WHERE user_id = ?", userID).Error; err != nil {
		t.Fatalf("delete generated privacy settings: %v", err)
	}

	repo := NewMessageRepository(db, "")
	if err := repo.SetMessagePermission(context.Background(), userID, model.MessagePermissionNone); err != nil {
		t.Fatalf("create missing message settings: %v", err)
	}
	permission, err := repo.MessagePermission(context.Background(), userID)
	if err != nil {
		t.Fatalf("load message permission: %v", err)
	}
	if permission != model.MessagePermissionNone {
		t.Fatalf("expected none permission, got %q", permission)
	}
	var profileVisibility string
	if err := db.Table("user_privacy_settings").Select("profile_visibility").
		Where("user_id = ?", userID).Scan(&profileVisibility).Error; err != nil {
		t.Fatalf("load default privacy fields: %v", err)
	}
	if profileVisibility != "public" {
		t.Fatalf("expected default profile visibility, got %q", profileVisibility)
	}
}
