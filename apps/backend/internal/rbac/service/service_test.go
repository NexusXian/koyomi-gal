package service

import (
	"context"
	"errors"
	"testing"

	"backend/internal/rbac/dto"
	"backend/internal/rbac/model"
	"backend/internal/rbac/repository"
	"backend/internal/testutil"

	"gorm.io/gorm"
)

func newTestService(t *testing.T) (*RBACService, *gorm.DB) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	return NewRBACService(repository.NewRBACRepository(db)), db
}

func createRole(t *testing.T, svc *RBACService, code string) *model.Role {
	t.Helper()
	role, err := svc.CreateRole(context.Background(), &dto.CreateRoleRequest{
		Name: code,
		Code: code,
	})
	if err != nil {
		t.Fatalf("create role %s: %v", code, err)
	}
	return role
}

func createPermission(t *testing.T, svc *RBACService, code string) *model.Permission {
	t.Helper()
	permission, err := svc.CreatePermission(context.Background(), &dto.CreatePermissionRequest{
		Name: code,
		Code: code,
	})
	if err != nil {
		t.Fatalf("create permission %s: %v", code, err)
	}
	return permission
}

// TestHasPermissionThroughRole covers the core User -> Role -> Permission flow:
// a user bound to a role with user:delete is allowed, a role-less user is denied.
func TestHasPermissionThroughRole(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	adminUser := testutil.CreateUser(t, db, "svc-admin")
	plainUser := testutil.CreateUser(t, db, "svc-plain")
	role := createRole(t, svc, "deleter")
	permission := createPermission(t, svc, "user:delete")
	if err := svc.SetRolePermissions(ctx, role.ID, []int64{permission.ID}); err != nil {
		t.Fatalf("set role permissions: %v", err)
	}
	if err := svc.AssignRoleToUser(ctx, adminUser, role.ID); err != nil {
		t.Fatalf("assign role: %v", err)
	}

	allowed, err := svc.HasPermission(ctx, adminUser, "user:delete")
	if err != nil || !allowed {
		t.Fatalf("expected admin user allowed, allowed=%v err=%v", allowed, err)
	}
	allowed, err = svc.HasPermission(ctx, plainUser, "user:delete")
	if err != nil || allowed {
		t.Fatalf("expected plain user denied, allowed=%v err=%v", allowed, err)
	}
}

func TestCreateRoleValidation(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.CreateRole(ctx, &dto.CreateRoleRequest{Name: "x", Code: "Bad Code"}); !errors.Is(err, ErrInvalidRoleCode) {
		t.Fatalf("expected ErrInvalidRoleCode, got %v", err)
	}

	createRole(t, svc, "existing")
	_, err := svc.CreateRole(ctx, &dto.CreateRoleRequest{Name: "x", Code: "existing"})
	if !errors.Is(err, ErrRoleCodeExists) {
		t.Fatalf("expected ErrRoleCodeExists, got %v", err)
	}

	if _, err := svc.CreatePermission(ctx, &dto.CreatePermissionRequest{Name: "x", Code: "user-delete"}); !errors.Is(err, ErrInvalidPermissionCode) {
		t.Fatalf("expected ErrInvalidPermissionCode, got %v", err)
	}
	createPermission(t, svc, "user:kick")
	if _, err := svc.CreatePermission(ctx, &dto.CreatePermissionRequest{Name: "x", Code: "user:kick"}); !errors.Is(err, ErrPermissionCodeExists) {
		t.Fatalf("expected ErrPermissionCodeExists, got %v", err)
	}
}

// TestAssignRoleTwiceSingleRow ensures repeated assignment cannot create duplicates.
func TestAssignRoleTwiceSingleRow(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	userID := testutil.CreateUser(t, db, "dupl-assign")
	role := createRole(t, svc, "solo")

	for range 2 {
		if err := svc.AssignRoleToUser(ctx, userID, role.ID); err != nil {
			t.Fatalf("assign role: %v", err)
		}
	}

	roles, err := svc.GetUserRoles(ctx, userID)
	if err != nil {
		t.Fatalf("get user roles: %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("expected exactly 1 role after duplicate assignment, got %d", len(roles))
	}
}

func TestSetUserRoles(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	userID := testutil.CreateUser(t, db, "replace-me")
	roleA := createRole(t, svc, "replace_a")
	roleB := createRole(t, svc, "replace_b")
	roleC := createRole(t, svc, "replace_c")

	if err := svc.SetUserRoles(ctx, userID, []int64{roleA.ID, roleA.ID}); err != nil {
		t.Fatalf("set user roles: %v", err)
	}
	roles, err := svc.GetUserRoles(ctx, userID)
	if err != nil || len(roles) != 1 {
		t.Fatalf("expected 1 unique role, got %d err=%v", len(roles), err)
	}

	if err := svc.SetUserRoles(ctx, userID, []int64{roleB.ID, roleC.ID}); err != nil {
		t.Fatalf("set user roles: %v", err)
	}
	roles, err = svc.GetUserRoles(ctx, userID)
	if err != nil || len(roles) != 2 || roles[0].ID != roleB.ID || roles[1].ID != roleC.ID {
		t.Fatalf("expected roles [replace_b replace_c], got %+v err=%v", roles, err)
	}

	if err := svc.SetUserRoles(ctx, userID, []int64{}); err != nil {
		t.Fatalf("clear user roles: %v", err)
	}
	roles, err = svc.GetUserRoles(ctx, userID)
	if err != nil || len(roles) != 0 {
		t.Fatalf("expected roles cleared, got %+v err=%v", roles, err)
	}

	err = svc.SetUserRoles(ctx, userID, []int64{roleB.ID, roleB.ID + 1000})
	if !errors.Is(err, ErrUnknownRoleIDs) {
		t.Fatalf("expected ErrUnknownRoleIDs, got %v", err)
	}

	err = svc.SetUserRoles(ctx, userID+1000, []int64{roleB.ID})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestRemoveRoleFromUser(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	userID := testutil.CreateUser(t, db, "remove-me")
	role := createRole(t, svc, "removable")
	if err := svc.AssignRoleToUser(ctx, userID, role.ID); err != nil {
		t.Fatalf("assign role: %v", err)
	}
	if err := svc.RemoveRoleFromUser(ctx, userID, role.ID); err != nil {
		t.Fatalf("remove role: %v", err)
	}
	roles, err := svc.GetUserRoles(ctx, userID)
	if err != nil || len(roles) != 0 {
		t.Fatalf("expected no roles after removal, got %+v err=%v", roles, err)
	}
}

// TestDeleteRoleCleansAssociations ensures deleting a role leaves no orphaned
// user_roles or role_permissions rows.
func TestDeleteRoleCleansAssociations(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	userID := testutil.CreateUser(t, db, "orphan-check")
	role := createRole(t, svc, "doomed")
	permission := createPermission(t, svc, "doomed:act")
	if err := svc.SetRolePermissions(ctx, role.ID, []int64{permission.ID}); err != nil {
		t.Fatalf("set role permissions: %v", err)
	}
	if err := svc.AssignRoleToUser(ctx, userID, role.ID); err != nil {
		t.Fatalf("assign role: %v", err)
	}

	if err := svc.DeleteRole(ctx, role.ID); err != nil {
		t.Fatalf("delete role: %v", err)
	}

	var userRoleCount, rolePermissionCount int64
	db.Model(&model.UserRole{}).Where("role_id = ?", role.ID).Count(&userRoleCount)
	db.Model(&model.RolePermission{}).Where("role_id = ?", role.ID).Count(&rolePermissionCount)
	if userRoleCount != 0 || rolePermissionCount != 0 {
		t.Fatalf("expected no associations after delete, got user_roles=%d role_permissions=%d",
			userRoleCount, rolePermissionCount)
	}

	if err := svc.DeleteRole(ctx, role.ID); !errors.Is(err, ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound on second delete, got %v", err)
	}
}

// TestDeletePermissionCleansAssociations ensures deleting a permission leaves no
// orphaned role_permissions rows.
func TestDeletePermissionCleansAssociations(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	role := createRole(t, svc, "perm_loser")
	permission := createPermission(t, svc, "gone:act")
	if err := svc.SetRolePermissions(ctx, role.ID, []int64{permission.ID}); err != nil {
		t.Fatalf("set role permissions: %v", err)
	}

	if err := svc.DeletePermission(ctx, permission.ID); err != nil {
		t.Fatalf("delete permission: %v", err)
	}

	var rolePermissionCount int64
	db.Model(&model.RolePermission{}).Where("permission_id = ?", permission.ID).Count(&rolePermissionCount)
	if rolePermissionCount != 0 {
		t.Fatalf("expected no role_permissions after delete, got %d", rolePermissionCount)
	}
}

func TestSetRolePermissions(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	role := createRole(t, svc, "grants")
	permissionA := createPermission(t, svc, "grant:a")
	permissionB := createPermission(t, svc, "grant:b")

	if err := svc.SetRolePermissions(ctx, role.ID, []int64{permissionA.ID, permissionA.ID}); err != nil {
		t.Fatalf("set role permissions: %v", err)
	}
	permissions, err := svc.GetRolePermissions(ctx, role.ID)
	if err != nil || len(permissions) != 1 {
		t.Fatalf("expected 1 unique permission, got %d err=%v", len(permissions), err)
	}

	if err := svc.SetRolePermissions(ctx, role.ID, []int64{permissionB.ID}); err != nil {
		t.Fatalf("set role permissions: %v", err)
	}
	permissions, err = svc.GetRolePermissions(ctx, role.ID)
	if err != nil || len(permissions) != 1 || permissions[0].ID != permissionB.ID {
		t.Fatalf("expected permissions replaced with grant:b, got %+v err=%v", permissions, err)
	}

	err = svc.SetRolePermissions(ctx, role.ID, []int64{permissionB.ID + 1000})
	if !errors.Is(err, ErrUnknownPermissionIDs) {
		t.Fatalf("expected ErrUnknownPermissionIDs, got %v", err)
	}

	err = svc.SetRolePermissions(ctx, role.ID+1000, []int64{permissionB.ID})
	if !errors.Is(err, ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestSeedDefaults(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if err := svc.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}
	if err := svc.SeedDefaults(ctx); err != nil {
		t.Fatalf("re-seed defaults should be idempotent: %v", err)
	}

	repo := svc.repo
	superAdmin, err := repo.FindRoleByCode(ctx, RoleCodeSuperAdmin)
	if err != nil || superAdmin == nil {
		t.Fatalf("super_admin role missing: %v", err)
	}
	admin, err := repo.FindRoleByCode(ctx, RoleCodeAdmin)
	if err != nil || admin == nil {
		t.Fatalf("admin role missing: %v", err)
	}
	user, err := repo.FindRoleByCode(ctx, RoleCodeUser)
	if err != nil || user == nil {
		t.Fatalf("user role missing: %v", err)
	}

	allPermissions, err := repo.ListPermissions(ctx)
	if err != nil {
		t.Fatalf("list permissions: %v", err)
	}
	if len(allPermissions) != len(seedPermissions) {
		t.Fatalf("expected %d seeded permissions, got %d", len(seedPermissions), len(allPermissions))
	}

	superAdminPermissions, err := repo.GetRolePermissions(ctx, superAdmin.ID)
	if err != nil {
		t.Fatalf("get super_admin permissions: %v", err)
	}
	if len(superAdminPermissions) != len(seedPermissions) {
		t.Fatalf("expected super_admin to hold all %d permissions, got %d",
			len(seedPermissions), len(superAdminPermissions))
	}

	adminPermissions, err := repo.GetRolePermissions(ctx, admin.ID)
	if err != nil {
		t.Fatalf("get admin permissions: %v", err)
	}
	if len(adminPermissions) != len(seedRolePermissions[RoleCodeAdmin]) {
		t.Fatalf("expected admin to hold %d permissions, got %d",
			len(seedRolePermissions[RoleCodeAdmin]), len(adminPermissions))
	}

	if err := svc.DeleteRole(ctx, superAdmin.ID); !errors.Is(err, ErrRoleProtected) {
		t.Fatalf("expected ErrRoleProtected when deleting super_admin, got %v", err)
	}
}

// TestAssignRoleByCode covers default-role assignment used on registration.
func TestAssignRoleByCode(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	if err := svc.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	userID := testutil.CreateUser(t, db, "by-code")
	if err := svc.AssignRoleByCode(ctx, userID, RoleCodeUser); err != nil {
		t.Fatalf("assign role by code: %v", err)
	}
	codes, err := svc.GetUserRoleCodes(ctx, userID)
	if err != nil || len(codes) != 1 || codes[0] != RoleCodeUser {
		t.Fatalf("expected [%s], got %v err=%v", RoleCodeUser, codes, err)
	}

	if err := svc.AssignRoleByCode(ctx, userID, "missing_role"); err != nil {
		t.Fatalf("missing role code should be ignored, got %v", err)
	}
}
