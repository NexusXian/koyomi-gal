package realtime

import (
	"context"
	"encoding/base64"
	"testing"

	"backend/internal/testutil"
)

func TestTicketFormatEntropyTTLAndConsumeOnce(t *testing.T) {
	client := testutil.NewRedis(t)
	store := NewTicketStore(client)
	ctx := context.Background()
	seen := make(map[string]struct{}, 128)

	for range 128 {
		data, err := store.Issue(ctx, 1001)
		if err != nil {
			t.Fatalf("issue ticket: %v", err)
		}
		decoded, err := base64.RawURLEncoding.DecodeString(data.Ticket)
		if err != nil || len(decoded) != ticketByteSize {
			t.Fatalf("ticket is not %d bytes of URL-safe entropy: ticket=%q err=%v", ticketByteSize, data.Ticket, err)
		}
		if _, exists := seen[data.Ticket]; exists {
			t.Fatalf("duplicate ticket generated: %q", data.Ticket)
		}
		seen[data.Ticket] = struct{}{}
		if got, want := ticketKey(data.Ticket), "ws:ticket:"+data.Ticket; got != want {
			t.Fatalf("unexpected ticket key: want=%q got=%q", want, got)
		}
		if exists, err := client.Exists(ctx, ticketKey(data.Ticket)).Result(); err != nil || exists != 1 {
			t.Fatalf("ticket key was not stored: exists=%d err=%v", exists, err)
		}
		ttl, err := client.TTL(ctx, ticketKey(data.Ticket)).Result()
		if err != nil || ttl <= 0 || ttl > ticketTTL {
			t.Fatalf("unexpected ticket TTL: %v (err=%v)", ttl, err)
		}
	}

	data, err := store.Issue(ctx, 2002)
	if err != nil {
		t.Fatalf("issue consumable ticket: %v", err)
	}
	userID, err := store.Consume(ctx, data.Ticket)
	if err != nil || userID != 2002 {
		t.Fatalf("consume ticket: userID=%d err=%v", userID, err)
	}
	if _, err := store.Consume(ctx, data.Ticket); err != ErrInvalidTicket {
		t.Fatalf("second consume should fail with ErrInvalidTicket, got %v", err)
	}
	if _, err := store.Consume(ctx, "eyJhbGciOiJIUzI1NiJ9.payload.signature"); err != ErrInvalidTicket {
		t.Fatalf("JWT-like query value should not be accepted, got %v", err)
	}
}
