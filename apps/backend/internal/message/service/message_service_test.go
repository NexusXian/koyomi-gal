package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"backend/internal/message/dto"
)

func TestValidateMessage(t *testing.T) {
	tests := []struct {
		name string
		req  *dto.SendMessageRequest
		want string
		err  error
	}{
		{name: "defaults to text and trims", req: &dto.SendMessageRequest{Content: "  hello  "}, want: "hello"},
		{name: "accepts 2000 code points", req: &dto.SendMessageRequest{Content: strings.Repeat("界", 2000)}},
		{name: "rejects blank", req: &dto.SendMessageRequest{Content: " \n "}, err: ErrMessageEmpty},
		{name: "rejects over 2000 code points", req: &dto.SendMessageRequest{Content: strings.Repeat("界", 2001)}, err: ErrMessageTooLong},
		{name: "rejects unsupported type", req: &dto.SendMessageRequest{Type: "image", Content: "asset"}, err: ErrInvalidMessageType},
		{name: "rejects invalid UTF-8", req: &dto.SendMessageRequest{Content: string([]byte{0xff})}, err: ErrInvalidMessage},
		{name: "rejects nil", req: nil, err: ErrMessageEmpty},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			messageType, content, err := validateMessage(test.req)
			if test.err != nil {
				if !errors.Is(err, test.err) {
					t.Fatalf("expected %v, got %v", test.err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("validate message: %v", err)
			}
			if messageType != "text" {
				t.Fatalf("expected text type, got %q", messageType)
			}
			if test.want != "" && content != test.want {
				t.Fatalf("expected content %q, got %q", test.want, content)
			}
		})
	}
}

func TestConversationCursorRoundTrip(t *testing.T) {
	wantTime := time.Date(2026, 9, 8, 12, 34, 56, 123456000, time.UTC)
	encoded, err := encodeCursor(wantTime, 42)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	cursor, err := decodeCursor(encoded)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if cursor.ID != 42 || !cursor.SortAt.Equal(wantTime) {
		t.Fatalf("unexpected cursor: %+v", cursor)
	}

	for _, value := range []string{"not-base64!", "e30", ""} {
		cursor, err := decodeCursor(value)
		if value == "" {
			if err != nil || cursor != nil {
				t.Fatalf("empty cursor should be absent, got cursor=%+v err=%v", cursor, err)
			}
			continue
		}
		if !errors.Is(err, ErrInvalidCursor) {
			t.Fatalf("expected invalid cursor for %q, got %v", value, err)
		}
	}
}

func TestCanonicalPair(t *testing.T) {
	low, high := canonicalPair(1002, 1001)
	if low != 1001 || high != 1002 {
		t.Fatalf("unexpected canonical pair: %d, %d", low, high)
	}
}
