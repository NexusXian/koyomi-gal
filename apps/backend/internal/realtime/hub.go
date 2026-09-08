package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"backend/pkg/logger"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	privateMessagePattern = privateMessageChannelPrefix + "*"
	writeTimeout          = 10 * time.Second
	pongTimeout           = 60 * time.Second
	pingInterval          = 25 * time.Second
	maxInboundMessageSize = 1024
	outboundBufferSize    = 64
)

type Hub struct {
	mu      sync.Mutex
	clients map[uint]map[*Client]struct{}
	pubsub  *redis.PubSub
	cancel  context.CancelFunc
	done    chan struct{}
	once    sync.Once
	closed  bool
}

type Client struct {
	hub    *Hub
	userID uint
	conn   *websocket.Conn
	send   chan []byte
}

func NewHub(client *redis.Client) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	hub := &Hub{
		clients: make(map[uint]map[*Client]struct{}),
		pubsub:  client.PSubscribe(ctx, privateMessagePattern),
		cancel:  cancel,
		done:    make(chan struct{}),
	}
	go hub.run(ctx)
	return hub
}

func (h *Hub) Serve(userID uint, conn *websocket.Conn) {
	client := &Client{hub: h, userID: userID, conn: conn, send: make(chan []byte, outboundBufferSize)}
	if !h.register(client) {
		_ = conn.Close()
		return
	}
	go client.writePump()
	client.readPump()
}

func (h *Hub) Close() {
	h.once.Do(func() {
		h.cancel()
		_ = h.pubsub.Close()
		<-h.done
	})
}

func (h *Hub) run(ctx context.Context) {
	defer close(h.done)
	for {
		message, err := h.pubsub.ReceiveMessage(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) && ctx.Err() == nil {
				logger.Error("realtime Redis subscription stopped", zap.Error(err))
			}
			h.disconnectAll()
			return
		}
		userID, ok := channelUserID(message.Channel)
		if !ok || !json.Valid([]byte(message.Payload)) {
			logger.Warn("discard invalid realtime Redis message", zap.String("channel", message.Channel))
			continue
		}
		h.deliver(userID, []byte(message.Payload))
	}
}

func (h *Hub) register(client *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return false
	}
	if h.clients[client.userID] == nil {
		h.clients[client.userID] = make(map[*Client]struct{})
	}
	h.clients[client.userID][client] = struct{}{}
	return true
}

func (h *Hub) unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	connections := h.clients[client.userID]
	if _, exists := connections[client]; !exists {
		return
	}
	delete(connections, client)
	close(client.send)
	if len(connections) == 0 {
		delete(h.clients, client.userID)
	}
}

func (h *Hub) deliver(userID uint, payload []byte) {
	h.mu.Lock()
	connections := h.clients[userID]
	var dropped []*Client
	for client := range connections {
		select {
		case client.send <- payload:
		default:
			delete(connections, client)
			close(client.send)
			dropped = append(dropped, client)
		}
	}
	if len(connections) == 0 {
		delete(h.clients, userID)
	}
	h.mu.Unlock()
	for _, client := range dropped {
		_ = client.conn.Close()
	}
}

func (h *Hub) disconnectAll() {
	h.mu.Lock()
	h.closed = true
	var clients []*Client
	for userID, connections := range h.clients {
		for client := range connections {
			close(client.send)
			clients = append(clients, client)
		}
		delete(h.clients, userID)
	}
	h.mu.Unlock()
	for _, client := range clients {
		_ = client.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutting down"),
			time.Now().Add(writeTimeout),
		)
		_ = client.conn.Close()
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister(c)
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(maxInboundMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case payload, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func OriginAllowed(allowedOrigins []string) func(*http.Request) bool {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}
	return func(request *http.Request) bool {
		origin := request.Header.Get("Origin")
		if origin == "" {
			return true
		}
		_, ok := allowed[origin]
		return ok
	}
}

func channelUserID(channel string) (uint, bool) {
	value, ok := strings.CutPrefix(channel, privateMessageChannelPrefix)
	if !ok {
		return 0, false
	}
	userID, err := strconv.ParseUint(value, 10, 0)
	return uint(userID), err == nil && userID > 0
}
