package middleware

import (
	"strconv"
	"strings"

	appErrors "backend/pkg/errors"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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

// Auth validates the Bearer access token and stores the userID in the context.
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), bearerPrefix)
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

		c.Set(contextUserIDKey, userID)
		c.Next()
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
