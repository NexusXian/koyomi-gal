package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"backend/internal/rbac/dto"
	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	"backend/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const testTokenSecret = "integration-test-access-token-secret"

func issueTestAccessToken(t *testing.T, userID uint) string {
	t.Helper()
	claims := accessTokenClaims{
		TokenType: accessTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    accessTokenIssuer,
			Subject:   strconv.FormatUint(uint64(userID), 10),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testTokenSecret))
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	return token
}

type rbacTestServer struct {
	db     *gorm.DB
	engine *gin.Engine
	svc    *rbacService.RBACService
}

func newRBACTestServer(t *testing.T) *rbacTestServer {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	gin.SetMode(gin.TestMode)

	svc := rbacService.NewRBACService(rbacRepo.NewRBACRepository(db))
	engine := gin.New()
	engine.Use(Auth(testTokenSecret))
	requirePermission := RequirePermission(svc)
	requireRole := RequireRole(svc)
	okHandler := func(c *gin.Context) { c.Status(http.StatusOK) }

	engine.DELETE("/users/:id", requirePermission("user:delete"), okHandler)
	engine.GET("/check/user:list", requirePermission("user:list"), okHandler)
	engine.GET("/check/post:delete", requirePermission("post:delete"), okHandler)
	engine.GET("/check/role-admin", requireRole("admin"), okHandler)

	return &rbacTestServer{db: db, engine: engine, svc: svc}
}

func (s *rbacTestServer) do(t *testing.T, method, path, token string) int {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	s.engine.ServeHTTP(recorder, req)
	return recorder.Code
}

// bindRoles maps role code -> granted permission codes and binds every role to the user.
func (s *rbacTestServer) bindRoles(t *testing.T, userID uint, rolePermissions map[string][]string) {
	t.Helper()
	ctx := context.Background()
	for roleCode, permissionCodes := range rolePermissions {
		role, err := s.svc.CreateRole(ctx, &dto.CreateRoleRequest{Name: roleCode, Code: roleCode})
		if err != nil {
			t.Fatalf("create role %s: %v", roleCode, err)
		}
		permissionIDs := make([]int64, 0, len(permissionCodes))
		for _, permissionCode := range permissionCodes {
			permission, err := s.svc.CreatePermission(ctx, &dto.CreatePermissionRequest{
				Name: permissionCode,
				Code: permissionCode,
			})
			if err != nil {
				t.Fatalf("create permission %s: %v", permissionCode, err)
			}
			permissionIDs = append(permissionIDs, permission.ID)
		}
		if err := s.svc.SetRolePermissions(ctx, role.ID, permissionIDs); err != nil {
			t.Fatalf("set role permissions for %s: %v", roleCode, err)
		}
		if err := s.svc.AssignRoleToUser(ctx, userID, role.ID); err != nil {
			t.Fatalf("assign role %s: %v", roleCode, err)
		}
	}
}

// TestAuthMiddlewareUnauthorized covers missing and invalid tokens returning 401.
func TestAuthMiddlewareUnauthorized(t *testing.T) {
	server := newRBACTestServer(t)

	if code := server.do(t, http.MethodDelete, "/users/1", ""); code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", code)
	}
	if code := server.do(t, http.MethodDelete, "/users/1", "not-a-jwt"); code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with invalid token, got %d", code)
	}
}

// TestRequirePermissionAllowedAndForbidden covers the core flow:
// User -> admin -> user:delete may call DELETE /users/:id,
// a role-less user receives 403.
func TestRequirePermissionAllowedAndForbidden(t *testing.T) {
	server := newRBACTestServer(t)

	adminID := testutil.CreateUser(t, server.db, "mw-admin")
	plainID := testutil.CreateUser(t, server.db, "mw-plain")
	server.bindRoles(t, adminID, map[string][]string{
		"admin": {"user:delete"},
	})

	if code := server.do(t, http.MethodDelete, "/users/"+strconv.FormatUint(uint64(plainID), 10), issueTestAccessToken(t, adminID)); code != http.StatusOK {
		t.Fatalf("expected admin to be allowed, got %d", code)
	}
	if code := server.do(t, http.MethodDelete, "/users/1", issueTestAccessToken(t, plainID)); code != http.StatusForbidden {
		t.Fatalf("expected plain user to be forbidden, got %d", code)
	}
}

// TestRequirePermissionMultiRole verifies a user with multiple roles holds the
// union of their permissions.
func TestRequirePermissionMultiRole(t *testing.T) {
	server := newRBACTestServer(t)

	userID := testutil.CreateUser(t, server.db, "mw-multi")
	server.bindRoles(t, userID, map[string][]string{
		"viewer":    {"user:list"},
		"moderator": {"post:delete"},
	})

	token := issueTestAccessToken(t, userID)
	if code := server.do(t, http.MethodGet, "/check/user:list", token); code != http.StatusOK {
		t.Fatalf("expected user:list from role A to be allowed, got %d", code)
	}
	if code := server.do(t, http.MethodGet, "/check/post:delete", token); code != http.StatusOK {
		t.Fatalf("expected post:delete from role B to be allowed, got %d", code)
	}
	if code := server.do(t, http.MethodGet, "/check/role-admin", token); code != http.StatusForbidden {
		t.Fatalf("expected unrelated role check to be forbidden, got %d", code)
	}
}

func TestRequireRole(t *testing.T) {
	server := newRBACTestServer(t)

	adminID := testutil.CreateUser(t, server.db, "mw-role-admin")
	plainID := testutil.CreateUser(t, server.db, "mw-role-plain")
	server.bindRoles(t, adminID, map[string][]string{
		"admin": {"role:list"},
	})

	if code := server.do(t, http.MethodGet, "/check/role-admin", issueTestAccessToken(t, adminID)); code != http.StatusOK {
		t.Fatalf("expected admin role to be allowed, got %d", code)
	}
	if code := server.do(t, http.MethodGet, "/check/role-admin", issueTestAccessToken(t, plainID)); code != http.StatusForbidden {
		t.Fatalf("expected plain user to be forbidden, got %d", code)
	}
}

// TestRequirePermissionAfterRoleDeletion verifies permissions disappear once the
// binding role is deleted.
func TestRequirePermissionAfterRoleDeletion(t *testing.T) {
	server := newRBACTestServer(t)
	ctx := context.Background()

	userID := testutil.CreateUser(t, server.db, "mw-deleted")
	server.bindRoles(t, userID, map[string][]string{
		"temp_admin": {"user:delete"},
	})

	token := issueTestAccessToken(t, userID)
	if code := server.do(t, http.MethodDelete, "/users/1", token); code != http.StatusOK {
		t.Fatalf("expected permission before role deletion, got %d", code)
	}

	roles, err := server.svc.GetUserRoles(ctx, userID)
	if err != nil || len(roles) != 1 {
		t.Fatalf("expected 1 role, got %+v err=%v", roles, err)
	}
	if err := server.svc.DeleteRole(ctx, roles[0].ID); err != nil {
		t.Fatalf("delete role: %v", err)
	}

	if code := server.do(t, http.MethodDelete, "/users/1", token); code != http.StatusForbidden {
		t.Fatalf("expected forbidden after role deletion, got %d", code)
	}
}
