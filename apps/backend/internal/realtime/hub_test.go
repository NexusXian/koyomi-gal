package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// TestRealtimeDeliversRedisEventsToWebSocketClient exercises the whole chain:
// ticket issue -> one-time consume -> upgrade -> Redis Pub/Sub -> client frame.
func TestRealtimeDeliversRedisEventsToWebSocketClient(t *testing.T) {
	client := testutil.NewRedis(t)
	gin.SetMode(gin.TestMode)

	hub := NewHub(client)
	t.Cleanup(hub.Close)
	tickets := NewTicketStore(client)
	handler := NewHandler(tickets, hub, []string{"https://koyomigal.xyz"})

	engine := gin.New()
	engine.GET("/api/v1/ws", handler.Connect)
	server := httptest.NewServer(engine)
	t.Cleanup(server.Close)

	const userID uint = 1001
	ticket, err := tickets.Issue(context.Background(), userID)
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") +
		"/api/v1/ws?ticket=" + ticket.Ticket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	publisher := NewRedisPublisher(client)
	event := Event{
		Type: EventMessageCreated,
		Data: MessageCreatedData{
			ConversationID: 8,
			Message: MessageCreatedMessage{
				ID: 42, ConversationID: 8, SenderID: 1002,
				Type: "text", Content: "你好", CreatedAt: time.Now().UTC(),
			},
		},
	}

	// Pub/Sub subscription setup is asynchronous, so publish until delivered.
	done := make(chan []byte, 1)
	go func() {
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, payload, readErr := conn.ReadMessage()
		if readErr == nil {
			done <- payload
		}
		close(done)
	}()

	deadline := time.Now().Add(4 * time.Second)
	var payload []byte
	for payload == nil {
		if err := publisher.Publish(context.Background(), []uint{userID}, event); err != nil {
			t.Fatalf("publish event: %v", err)
		}
		select {
		case received, ok := <-done:
			if !ok && received == nil {
				t.Fatal("websocket closed before delivering the event")
			}
			payload = received
		case <-time.After(100 * time.Millisecond):
			if time.Now().After(deadline) {
				t.Fatal("timed out waiting for realtime event")
			}
		}
	}

	var decoded struct {
		Type string `json:"type"`
		Data struct {
			ConversationID uint `json:"conversation_id"`
			Message        struct {
				ID       uint   `json:"id"`
				SenderID uint   `json:"sender_id"`
				Content  string `json:"content"`
			} `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode event payload: %v", err)
	}
	if decoded.Type != EventMessageCreated ||
		decoded.Data.ConversationID != 8 ||
		decoded.Data.Message.ID != 42 ||
		decoded.Data.Message.SenderID != 1002 ||
		decoded.Data.Message.Content != "你好" {
		t.Fatalf("unexpected event payload: %s", payload)
	}

	// The ticket is single-use: a second handshake must be rejected.
	_, response, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected reused ticket to be rejected")
	}
	if response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for reused ticket, got %v", response)
	}
}

// TestRealtimeRejectsDisallowedBrowserOrigin keeps browser clients restricted to
// the configured origins while native clients without Origin stay allowed.
func TestRealtimeRejectsDisallowedBrowserOrigin(t *testing.T) {
	client := testutil.NewRedis(t)
	gin.SetMode(gin.TestMode)

	hub := NewHub(client)
	t.Cleanup(hub.Close)
	tickets := NewTicketStore(client)
	handler := NewHandler(tickets, hub, []string{"https://koyomigal.xyz"})

	engine := gin.New()
	engine.GET("/api/v1/ws", handler.Connect)
	server := httptest.NewServer(engine)
	t.Cleanup(server.Close)

	ticket, err := tickets.Issue(context.Background(), 1001)
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") +
		"/api/v1/ws?ticket=" + ticket.Ticket

	header := http.Header{}
	header.Set("Origin", "https://evil.example.com")
	_, response, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err == nil {
		t.Fatal("expected disallowed origin to be rejected")
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed origin, got %v", response)
	}

	// The rejected handshake must not have consumed the ticket.
	if _, err := tickets.Consume(context.Background(), ticket.Ticket); err != nil {
		t.Fatalf("ticket should survive a rejected origin: %v", err)
	}
}
