package dto

import (
	"encoding/json"
	"testing"
	"time"

	userdto "backend/internal/user/dto"
)

func TestMessageSettingsJSONContract(t *testing.T) {
	assertJSONKeys(t, UpdateMessageSettingsRequest{Permission: "everyone"}, []string{"permission"}, []string{"message_permission"})
	assertJSONKeys(t, MessageSettingsData{Permission: "none"}, []string{"permission"}, []string{"message_permission"})
}

func TestConversationJSONContract(t *testing.T) {
	conversation := ConversationData{
		ID: 1, Type: "direct",
		User: userdto.PublicUserSummary{ID: 2, Username: "peer", DisplayName: "Peer"},
	}
	assertJSONKeys(
		t,
		conversation,
		[]string{"id", "type", "user", "last_message", "last_message_at", "unread_count", "is_blocked", "can_send", "created_at", "updated_at"},
		[]string{"participant"},
	)
	encoded, err := json.Marshal(conversation)
	if err != nil {
		t.Fatalf("marshal conversation: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("decode conversation: %v", err)
	}
	if fields["type"] != "direct" || fields["last_message_at"] != nil {
		t.Fatalf("unexpected conversation contract: %s", encoded)
	}
	assertJSONKeys(
		t,
		ConversationListData{List: []ConversationData{}, NextCursor: "cursor"},
		[]string{"list", "next_cursor", "has_more"},
		[]string{"items"},
	)
}

func TestMessageJSONContract(t *testing.T) {
	message := MessageData{
		ID: 3, ConversationID: 1, SenderID: 2,
		Sender: userdto.PublicUserSummary{ID: 2, Username: "sender", DisplayName: "Sender"},
		Type:   "text", Content: "hello", CreatedAt: time.Now(),
	}
	assertJSONKeys(
		t,
		message,
		[]string{"id", "conversation_id", "sender_id", "sender", "receiver_id", "type", "content", "is_deleted", "created_at", "updated_at"},
		nil,
	)
	assertJSONKeys(
		t,
		MessageListData{List: []MessageData{}, NextCursor: 3},
		[]string{"list", "next_cursor", "has_more"},
		[]string{"items", "next_before_id"},
	)
}

func assertJSONKeys(t *testing.T, value any, required, forbidden []string) {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("decode JSON object: %v", err)
	}
	for _, key := range required {
		if _, ok := object[key]; !ok {
			t.Errorf("missing JSON key %q in %s", key, encoded)
		}
	}
	for _, key := range forbidden {
		if _, ok := object[key]; ok {
			t.Errorf("unexpected JSON key %q in %s", key, encoded)
		}
	}
}
