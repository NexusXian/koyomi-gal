package realtime

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	ticketPrefix   = "ws:ticket:"
	ticketTTL      = 30 * time.Second
	ticketByteSize = 32
	issueAttempts  = 3
)

var ErrInvalidTicket = errors.New("invalid or expired websocket ticket")

var consumeTicketScript = redis.NewScript(`
local value = redis.call('GET', KEYS[1])
if value then
    redis.call('DEL', KEYS[1])
end
return value
`)

type TicketStore struct {
	redis redis.Cmdable
}

type TicketData struct {
	Ticket    string    `json:"ticket"`
	ExpiresAt time.Time `json:"expires_at"`
}

type TicketResponse struct {
	Code int        `json:"code" example:"0"`
	Data TicketData `json:"data"`
	Msg  string     `json:"msg" example:"success"`
}

func NewTicketStore(client redis.Cmdable) *TicketStore {
	return &TicketStore{redis: client}
}

func (s *TicketStore) Issue(ctx context.Context, userID uint) (*TicketData, error) {
	if userID == 0 {
		return nil, errors.New("websocket ticket user ID is required")
	}
	for range issueAttempts {
		ticket, err := generateTicket()
		if err != nil {
			return nil, err
		}
		expiresAt := time.Now().Add(ticketTTL)
		created, err := s.redis.SetNX(ctx, ticketKey(ticket), strconv.FormatUint(uint64(userID), 10), ticketTTL).Result()
		if err != nil {
			return nil, fmt.Errorf("store websocket ticket: %w", err)
		}
		if created {
			return &TicketData{Ticket: ticket, ExpiresAt: expiresAt}, nil
		}
	}
	return nil, errors.New("generate unique websocket ticket")
}

func (s *TicketStore) Consume(ctx context.Context, ticket string) (uint, error) {
	if !validTicket(ticket) {
		return 0, ErrInvalidTicket
	}
	value, err := consumeTicketScript.Run(ctx, s.redis, []string{ticketKey(ticket)}).Text()
	if errors.Is(err, redis.Nil) {
		return 0, ErrInvalidTicket
	}
	if err != nil {
		return 0, fmt.Errorf("consume websocket ticket: %w", err)
	}
	userID, err := strconv.ParseUint(value, 10, 0)
	if err != nil || userID == 0 {
		return 0, ErrInvalidTicket
	}
	return uint(userID), nil
}

func generateTicket() (string, error) {
	random := make([]byte, ticketByteSize)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate websocket ticket: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(random), nil
}

func validTicket(ticket string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(ticket)
	return err == nil && len(decoded) == ticketByteSize &&
		base64.RawURLEncoding.EncodeToString(decoded) == ticket
}

func ticketKey(ticket string) string {
	return ticketPrefix + ticket
}
