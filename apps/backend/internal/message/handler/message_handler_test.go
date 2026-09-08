package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/message/service"

	"github.com/gin-gonic/gin"
)

func TestRespondErrorContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		err        error
		statusCode int
		message    string
	}{
		{name: "empty", err: service.ErrMessageEmpty, statusCode: http.StatusBadRequest, message: "消息内容不能为空"},
		{name: "too long", err: service.ErrMessageTooLong, statusCode: http.StatusBadRequest, message: "消息内容不能超过 2000 个字符"},
		{name: "type", err: service.ErrInvalidMessageType, statusCode: http.StatusBadRequest, message: "消息类型不支持"},
		{name: "self", err: service.ErrCannotMessageSelf, statusCode: http.StatusBadRequest, message: "不能向自己发起私信操作"},
		{name: "blocked", err: service.ErrUserBlocked, statusCode: http.StatusForbidden, message: "UserBlocked"},
		{name: "permission", err: service.ErrMessagePermissionDenied, statusCode: http.StatusForbidden, message: "MessagePermissionDenied"},
		{name: "conversation not found", err: service.ErrConversationNotFound, statusCode: http.StatusNotFound, message: "会话不存在"},
		{name: "conversation access is concealed", err: service.ErrConversationAccessDenied, statusCode: http.StatusNotFound, message: "会话不存在"},
		{name: "message not found", err: service.ErrMessageNotFound, statusCode: http.StatusNotFound, message: "消息不存在"},
		{name: "message access", err: service.ErrMessageAccessDenied, statusCode: http.StatusForbidden, message: "MessageAccessDenied"},
		{name: "message rate limit", err: service.ErrMessageRateLimited, statusCode: http.StatusTooManyRequests, message: "消息发送过于频繁"},
		{name: "conversation rate limit", err: service.ErrConversationRateLimit, statusCode: http.StatusTooManyRequests, message: "新建会话过于频繁"},
		{name: "rate limiter unavailable", err: service.ErrRateLimitUnavailable, statusCode: http.StatusServiceUnavailable, message: "私信服务暂时不可用"},
	}

	handler := &MessageHandler{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			handler.respondError(context, test.err, "test")
			if recorder.Code != test.statusCode {
				t.Fatalf("expected status %d, got %d", test.statusCode, recorder.Code)
			}
			var body struct {
				Msg string `json:"msg"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Msg != test.message {
				t.Fatalf("expected message %q, got %q", test.message, body.Msg)
			}
		})
	}
}
