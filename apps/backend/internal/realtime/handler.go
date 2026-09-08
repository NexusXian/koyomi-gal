package realtime

import (
	"errors"
	"net/http"
	"time"

	"backend/internal/middleware"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Handler struct {
	tickets       *TicketStore
	hub           *Hub
	originAllowed func(*http.Request) bool
	upgrader      websocket.Upgrader
}

func NewHandler(tickets *TicketStore, hub *Hub, allowedOrigins []string) *Handler {
	originAllowed := OriginAllowed(allowedOrigins)
	return &Handler{
		tickets:       tickets,
		hub:           hub,
		originAllowed: originAllowed,
		upgrader: websocket.Upgrader{
			HandshakeTimeout: 10 * time.Second,
			CheckOrigin:      originAllowed,
		},
	}
}

// CreateTicket godoc
// @Summary      创建 WebSocket 连接票据
// @Description  创建一个 30 秒内有效且仅可使用一次的 WebSocket 连接票据
// @ID           createWebSocketTicket
// @Tags         websocket
// @Produce      json
// @Success      200 {object} TicketResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/ws/ticket [post]
func (h *Handler) CreateTicket(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	data, err := h.tickets.Issue(c.Request.Context(), userID)
	if err != nil {
		logger.Error("issue websocket ticket", zap.Uint("user_id", userID), zap.Error(err))
		response.Error(c, appErrors.ErrInternal("WebSocket 票据创建失败"))
		return
	}
	response.Ok(c, data)
}

func (h *Handler) Connect(c *gin.Context) {
	if !h.originAllowed(c.Request) {
		response.Error(c, appErrors.ErrForbidden("WebSocket Origin 不允许"))
		return
	}
	values := c.Request.URL.Query()["ticket"]
	if len(values) != 1 {
		response.Error(c, appErrors.ErrUnauthorized("WebSocket 票据无效或已过期"))
		return
	}
	userID, err := h.tickets.Consume(c.Request.Context(), values[0])
	if errors.Is(err, ErrInvalidTicket) {
		response.Error(c, appErrors.ErrUnauthorized("WebSocket 票据无效或已过期"))
		return
	}
	if err != nil {
		logger.Error("consume websocket ticket", zap.Error(err))
		response.Error(c, appErrors.ErrInternal("WebSocket 连接失败"))
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Warn("upgrade websocket connection", zap.Uint("user_id", userID), zap.Error(err))
		return
	}
	h.hub.Serve(userID, conn)
}
