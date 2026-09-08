package realtime

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventJSONShapes(t *testing.T) {
	createdAt := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		event Event
		want  string
	}{
		{
			name: "message created",
			event: Event{Type: EventMessageCreated, Data: MessageCreatedData{
				ConversationID: 8,
				Message: MessageCreatedMessage{
					ID: 42, ConversationID: 8, SenderID: 1001, Type: "text",
					Content: "Hello", IsDeleted: false, CreatedAt: createdAt,
				},
			}},
			want: `{"type":"message.created","data":{"conversation_id":8,"message":{"id":42,"conversation_id":8,"sender_id":1001,"type":"text","content":"Hello","is_deleted":false,"created_at":"2026-09-08T12:00:00Z"}}}`,
		},
		{
			name:  "conversation read",
			event: Event{Type: EventConversationRead, Data: ConversationReadData{ConversationID: 8, UserID: 1002, MessageID: 42}},
			want:  `{"type":"conversation.read","data":{"conversation_id":8,"user_id":1002,"message_id":42}}`,
		},
		{
			name:  "message deleted",
			event: Event{Type: EventMessageDeleted, Data: MessageDeletedData{ConversationID: 8, MessageID: 42}},
			want:  `{"type":"message.deleted","data":{"conversation_id":8,"message_id":42}}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := json.Marshal(test.event)
			if err != nil {
				t.Fatalf("marshal event: %v", err)
			}
			if string(got) != test.want {
				t.Fatalf("unexpected event JSON\nwant: %s\n got: %s", test.want, got)
			}
		})
	}
}
