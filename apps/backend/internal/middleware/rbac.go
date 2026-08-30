package middleware

import (
	rbacService "backend/internal/rbac/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequirePermission returns a guard factory checking the permission code via RBACService.
// Routes should depend on permissions; roles are only an organization unit.
func RequirePermission(svc *rbacService.RBACService) func(string) gin.HandlerFunc {
	return func(permission string) gin.HandlerFunc {
		return func(c *gin.Context) {
			userID, ok := CurrentUserID(c)
			if !ok {
				response.Error(c, appErrors.ErrAuthExpired())
				c.Abort()
				return
			}

			allowed, err := svc.HasPermission(c.Request.Context(), userID, permission)
			if err != nil {
				logger.Error(
					"check permission",
					zap.String("permission", permission),
					zap.Uint("user_id", userID),
					zap.Error(err),
				)
				response.Error(c, appErrors.ErrInternal("权限校验失败"))
				c.Abort()
				return
			}
			if !allowed {
				response.Error(c, appErrors.ErrForbidden("没有执行该操作的权限"))
				c.Abort()
				return
			}

			c.Next()
		}
	}
}

// RequireRole returns a guard factory checking a direct role binding.
// Prefer RequirePermission for business endpoints.
func RequireRole(svc *rbacService.RBACService) func(string) gin.HandlerFunc {
	return func(role string) gin.HandlerFunc {
		return func(c *gin.Context) {
			userID, ok := CurrentUserID(c)
			if !ok {
				response.Error(c, appErrors.ErrAuthExpired())
				c.Abort()
				return
			}

			has, err := svc.HasRole(c.Request.Context(), userID, role)
			if err != nil {
				logger.Error(
					"check role",
					zap.String("role", role),
					zap.Uint("user_id", userID),
					zap.Error(err),
				)
				response.Error(c, appErrors.ErrInternal("权限校验失败"))
				c.Abort()
				return
			}
			if !has {
				response.Error(c, appErrors.ErrForbidden("没有执行该操作的权限"))
				c.Abort()
				return
			}

			c.Next()
		}
	}
}
