package middleware

import (
	"context"
	"strconv"
	"strings"

	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

const (
	contextUserIDKey  = "userID"
	accessTokenIssuer = "koyomi-gal"
	accessTokenType   = "access"
	bearerPrefix      = "Bearer "
)

type accessTokenClaims struct {
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

type AccessUserChecker interface {
	AccessUserStatus(ctx context.Context, userID uint) (exists, banned bool, err error)
}

// Auth validates the Bearer access token and stores the userID in the context.
func Auth(secret string) gin.HandlerFunc {
	return AuthWithUserChecker(secret, nil)
}

// AuthWithUserChecker additionally rejects tokens for deleted or banned users.
func AuthWithUserChecker(secret string, checker AccessUserChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token, ok := strings.CutPrefix(header, bearerPrefix)
		if !ok || strings.TrimSpace(token) == "" {
			response.Error(c, appErrors.ErrAuthExpired())
			c.Abort()
			return
		}

		userID, err := parseAccessToken(strings.TrimSpace(token), secret)
		if err != nil {
			response.Error(c, appErrors.ErrAuthExpired())
			c.Abort()
			return
		}

		if checker != nil {
			exists, banned, err := checker.AccessUserStatus(c.Request.Context(), userID)
			if err != nil {
				logger.Error("check access token user", zap.Uint("user_id", userID), zap.Error(err))
				response.Error(c, appErrors.ErrInternal("用户状态校验失败"))
				c.Abort()
				return
			}
			if !exists {
				response.Error(c, appErrors.ErrAuthExpired())
				c.Abort()
				return
			}
			if banned {
				response.Error(c, appErrors.ErrAccountBanned())
				c.Abort()
				return
			}
		}

		c.Set(contextUserIDKey, userID)
		c.Next()
	}
}

// OptionalAuthWithUserChecker permits anonymous requests only when the
// Authorization header is absent. A supplied token is validated exactly like
// a token on a protected route.
func OptionalAuthWithUserChecker(secret string, checker AccessUserChecker) gin.HandlerFunc {
	protected := AuthWithUserChecker(secret, checker)
	return func(c *gin.Context) {
		if len(c.Request.Header.Values("Authorization")) == 0 {
			c.Next()
			return
		}
		protected(c)
	}
}

// CurrentUserID returns the authenticated userID stored by Auth.
func CurrentUserID(c *gin.Context) (uint, bool) {
	value, ok := c.Get(contextUserIDKey)
	if !ok {
		return 0, false
	}
	userID, ok := value.(uint)
	return userID, ok
}

func parseAccessToken(token string, secret string) (uint, error) {
	claims := &accessTokenClaims{}
	parsed, err := jwt.ParseWithClaims(
		token,
		claims,
		func(*jwt.Token) (any, error) { return []byte(secret), nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(accessTokenIssuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return 0, err
	}
	if !parsed.Valid || claims.TokenType != accessTokenType {
		return 0, jwt.ErrTokenInvalidClaims
	}

	userID, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil || userID == 0 {
		return 0, jwt.ErrTokenInvalidClaims
	}
	return uint(userID), nil
}
