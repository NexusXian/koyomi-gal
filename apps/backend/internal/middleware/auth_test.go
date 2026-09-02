package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type stubAccessUserChecker struct {
	exists bool
	banned bool
	err    error
}

func (s stubAccessUserChecker) AccessUserStatus(context.Context, uint) (bool, bool, error) {
	return s.exists, s.banned, s.err
}

func TestAuthWithUserChecker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "middleware-auth-test-secret"
	token := testAccessToken(t, secret, 42)
	tests := []struct {
		name    string
		checker AccessUserChecker
		status  int
	}{
		{name: "active", checker: stubAccessUserChecker{exists: true}, status: http.StatusNoContent},
		{name: "deleted", checker: stubAccessUserChecker{}, status: http.StatusUnauthorized},
		{name: "banned", checker: stubAccessUserChecker{exists: true, banned: true}, status: http.StatusForbidden},
		{name: "database failure", checker: stubAccessUserChecker{err: errors.New("database unavailable")}, status: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/", AuthWithUserChecker(secret, tt.checker), func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Fatalf("expected status %d, got %d", tt.status, response.Code)
			}
		})
	}
}

func TestAuthRemainsUsableWithoutChecker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "middleware-auth-test-secret"
	router := gin.New()
	router.GET("/", Auth(secret), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+testAccessToken(t, secret, 7))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
}

func testAccessToken(t *testing.T, secret string, userID uint) string {
	t.Helper()
	now := time.Now()
	claims := accessTokenClaims{
		TokenType: accessTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    accessTokenIssuer,
			Subject:   strconv.FormatUint(uint64(userID), 10),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}
