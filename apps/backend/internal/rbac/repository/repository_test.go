package repository

import (
	"context"
	"testing"

	"backend/internal/rbac/model"
	"backend/internal/testutil"
)

func newTestRepository(t *testing.T) *RBACRepository {
	testutil.SkipWithoutPostgres(t)
	return NewRBACRepository(testutil.NewPostgres(t))
}

func mustCreateRole(t *testing.T, repo *RBACRepository, code string) *model.Role {
	t.Helper()
	role := &model.Role{Name: code, Code: code}
	if err := repo.CreateRole(context.Background(), role); err != nil {
		t.Fatalf("create role %s: %v", code, err)
	}
	return role
}

func mustCreatePermission(t *testing.T, repo *RBACRepository, code string) *model.Permission {
	t.Helper()
	permission := &model.Permission{Name: code, Code: code, Resource: "res", Action: "act"}
	if err := repo.CreatePermission(context.Background(), permission); err != nil {
		t.Fatalf("create permission %s: %v", code, err)
	}
	return permission
}

// TestUserRolesUniqueConstraint verifies duplicate user-role bindings are
// rejected by the unique index and that OnConflict inserts stay idempotent.
func TestUserRolesUniqueConstraint(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	userID := testutil.CreateUser(t, repo.db, "dupl-user")
	role := mustCreateRole(t, repo, "member")

	first := &model.UserRole{UserID: userID, RoleID: role.ID}
	if err := repo.db.WithContext(ctx).Create(first).Error; err != nil {
		t.Fatalf("first user_roles insert should succeed: %v", err)
	}
	duplicate := &model.UserRole{UserID: userID, RoleID: role.ID}
	if err := repo.db.WithContext(ctx).Create(duplicate).Error; err == nil {
		t.Fatal("expected duplicate user_roles insert to fail on unique index")
	}

	if err := repo.InsertUserRoles(ctx, userID, []int64{role.ID}); err != nil {
		t.Fatalf("insert user role: %v", err)
	}
	if err := repo.InsertUserRoles(ctx, userID, []int64{role.ID}); err != nil {
		t.Fatalf("duplicate on-conflict insert should be ignored: %v", err)
	}

	count, err := repo.CountUserRole(ctx, userID, role.ID)
	if err != nil {
		t.Fatalf("count user role: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 user_roles row, got %d", count)
	}
}

// TestRolePermissionsUniqueConstraint verifies the role_permissions unique index.
func TestRolePermissionsUniqueConstraint(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	role := mustCreateRole(t, repo, "editor")
	permission := mustCreatePermission(t, repo, "post:review")

	first := &model.RolePermission{RoleID: role.ID, PermissionID: permission.ID}
	if err := repo.db.WithContext(ctx).Create(first).Error; err != nil {
		t.Fatalf("first role_permissions insert should succeed: %v", err)
	}
	duplicate := &model.RolePermission{RoleID: role.ID, PermissionID: permission.ID}
	if err := repo.db.WithContext(ctx).Create(duplicate).Error; err == nil {
		t.Fatal("expected duplicate role_permissions insert to fail on unique index")
	}

	if err := repo.InsertRolePermissions(ctx, role.ID, []int64{permission.ID}); err != nil {
		t.Fatalf("insert role permission: %v", err)
	}
	if err := repo.InsertRolePermissions(ctx, role.ID, []int64{permission.ID}); err != nil {
		t.Fatalf("duplicate on-conflict insert should be ignored: %v", err)
	}
}

// TestGetUserPermissionsMultiRole verifies permissions of multiple roles are merged.
func TestGetUserPermissionsMultiRole(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	userID := testutil.CreateUser(t, repo.db, "multi-user")
	roleA := mustCreateRole(t, repo, "role_a")
	roleB := mustCreateRole(t, repo, "role_b")
	permA := mustCreatePermission(t, repo, "user:list")
	permB := mustCreatePermission(t, repo, "post:delete")
	permUnbound := mustCreatePermission(t, repo, "role:create")

	if err := repo.InsertRolePermissions(ctx, roleA.ID, []int64{permA.ID}); err != nil {
		t.Fatalf("grant permission to role_a: %v", err)
	}
	if err := repo.InsertRolePermissions(ctx, roleB.ID, []int64{permB.ID}); err != nil {
		t.Fatalf("grant permission to role_b: %v", err)
	}
	if err := repo.InsertUserRoles(ctx, userID, []int64{roleA.ID, roleB.ID}); err != nil {
		t.Fatalf("assign roles: %v", err)
	}
	_ = permUnbound

	codes, err := repo.GetUserPermissions(ctx, userID)
	if err != nil {
		t.Fatalf("get user permissions: %v", err)
	}
	want := []string{"post:delete", "user:list"}
	if len(codes) != len(want) || codes[0] != want[0] || codes[1] != want[1] {
		t.Fatalf("expected permissions %v, got %v", want, codes)
	}
}

func TestHasPermissionAndHasRole(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	allowedUser := testutil.CreateUser(t, repo.db, "allowed")
	deniedUser := testutil.CreateUser(t, repo.db, "denied")
	role := mustCreateRole(t, repo, "user_manager")
	permission := mustCreatePermission(t, repo, "user:delete")

	if err := repo.InsertRolePermissions(ctx, role.ID, []int64{permission.ID}); err != nil {
		t.Fatalf("grant permission: %v", err)
	}
	if err := repo.InsertUserRoles(ctx, allowedUser, []int64{role.ID}); err != nil {
		t.Fatalf("assign role: %v", err)
	}

	allowed, err := repo.HasPermission(ctx, allowedUser, "user:delete")
	if err != nil || !allowed {
		t.Fatalf("expected allowed user to have user:delete, allowed=%v err=%v", allowed, err)
	}
	allowed, err = repo.HasPermission(ctx, deniedUser, "user:delete")
	if err != nil || allowed {
		t.Fatalf("expected denied user to lack user:delete, allowed=%v err=%v", allowed, err)
	}

	hasRole, err := repo.HasRole(ctx, allowedUser, "user_manager")
	if err != nil || !hasRole {
		t.Fatalf("expected allowed user to have role, has=%v err=%v", hasRole, err)
	}
	hasRole, err = repo.HasRole(ctx, deniedUser, "user_manager")
	if err != nil || hasRole {
		t.Fatalf("expected denied user to lack role, has=%v err=%v", hasRole, err)
	}
}

func TestGetUserRolesAndRoleCodes(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	userID := testutil.CreateUser(t, repo.db, "roles-reader")
	roleA := mustCreateRole(t, repo, "alpha")
	roleB := mustCreateRole(t, repo, "beta")
	if err := repo.InsertUserRoles(ctx, userID, []int64{roleA.ID, roleB.ID}); err != nil {
		t.Fatalf("assign roles: %v", err)
	}

	roles, err := repo.GetUserRoles(ctx, userID)
	if err != nil {
		t.Fatalf("get user roles: %v", err)
	}
	if len(roles) != 2 || roles[0].Code != "alpha" || roles[1].Code != "beta" {
		t.Fatalf("unexpected roles: %+v", roles)
	}

	codes, err := repo.GetUserRoleCodes(ctx, userID)
	if err != nil {
		t.Fatalf("get user role codes: %v", err)
	}
	if len(codes) != 2 || codes[0] != "alpha" || codes[1] != "beta" {
		t.Fatalf("unexpected role codes: %v", codes)
	}
}

func TestGetRolePermissions(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	role := mustCreateRole(t, repo, "perm-holder")
	other := mustCreateRole(t, repo, "perm-other")
	permission := mustCreatePermission(t, repo, "user:read")
	unrelated := mustCreatePermission(t, repo, "user:update")

	if err := repo.InsertRolePermissions(ctx, role.ID, []int64{permission.ID}); err != nil {
		t.Fatalf("grant permission: %v", err)
	}
	if err := repo.InsertRolePermissions(ctx, other.ID, []int64{unrelated.ID}); err != nil {
		t.Fatalf("grant permission: %v", err)
	}

	permissions, err := repo.GetRolePermissions(ctx, role.ID)
	if err != nil {
		t.Fatalf("get role permissions: %v", err)
	}
	if len(permissions) != 1 || permissions[0].Code != "user:read" {
		t.Fatalf("unexpected permissions: %+v", permissions)
	}
}

func TestUserExistsAndCountByIDs(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	userID := testutil.CreateUser(t, repo.db, "exists-user")
	exists, err := repo.UserExists(ctx, userID)
	if err != nil || !exists {
		t.Fatalf("expected user to exist, exists=%v err=%v", exists, err)
	}
	exists, err = repo.UserExists(ctx, userID+1000)
	if err != nil || exists {
		t.Fatalf("expected user to be missing, exists=%v err=%v", exists, err)
	}

	roleA := mustCreateRole(t, repo, "count_a")
	roleB := mustCreateRole(t, repo, "count_b")
	count, err := repo.CountRolesByIDs(ctx, []int64{roleA.ID, roleB.ID, roleB.ID + 1000})
	if err != nil || count != 2 {
		t.Fatalf("expected 2 existing roles, got %d err=%v", count, err)
	}

	permissionA := mustCreatePermission(t, repo, "count:list")
	count, err = repo.CountPermissionsByIDs(ctx, []int64{permissionA.ID, permissionA.ID + 1000})
	if err != nil || count != 1 {
		t.Fatalf("expected 1 existing permission, got %d err=%v", count, err)
	}
}
