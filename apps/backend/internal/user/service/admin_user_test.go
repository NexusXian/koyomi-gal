package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	"backend/internal/testutil"
	"backend/internal/user/dto"
	userRepo "backend/internal/user/repository"
	"backend/pkg/bcrypt"

	"gorm.io/gorm"
)

type rollbackUserAdminRBAC struct {
	db  *gorm.DB
	err error
}

func (r *rollbackUserAdminRBAC) AssignRoleByCode(context.Context, uint, string) error {
	return nil
}

func (r *rollbackUserAdminRBAC) RunUserAdminMutation(
	ctx context.Context,
	_, _ uint,
	_ bool,
	mutate func(tx *gorm.DB) error,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := mutate(tx); err != nil {
			return err
		}
		return r.err
	})
}

func TestUserAdminLifecycleAndSearch(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	ctx := context.Background()
	rbac := rbacService.NewRBACService(rbacRepo.NewRBACRepository(db))
	if err := rbac.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed rbac: %v", err)
	}
	users := NewUserAdminService(userRepo.NewUserAdminRepository(db), rbac)

	banned := true
	created, err := users.Create(ctx, &dto.CreateAdminUserRequest{
		Username: "AdminCreated", Email: "ADMIN@Example.com", Password: "password123", IsBanned: &banned,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.Email != "admin@example.com" || !created.IsBanned || bcrypt.ComparePassword(created.PasswordHash, "password123") != nil {
		t.Fatalf("unexpected created user: %+v", created)
	}
	roles, err := rbac.GetUserRoleCodes(ctx, created.ID)
	if err != nil || len(roles) != 1 || roles[0] != rbacService.RoleCodeUser {
		t.Fatalf("expected default user role, roles=%v err=%v", roles, err)
	}

	for _, keyword := range []string{"admincreated", "EXAMPLE.COM"} {
		listed, total, page, limit, err := users.List(ctx, &dto.AdminUserQuery{Keyword: keyword})
		if err != nil || total != 1 || len(listed) != 1 || page != 1 || limit != 20 {
			t.Fatalf("search %q: total=%d items=%d page=%d limit=%d err=%v", keyword, total, len(listed), page, limit, err)
		}
	}
	listed, total, _, _, err := users.List(ctx, &dto.AdminUserQuery{Keyword: strconv.FormatUint(uint64(created.ID), 10)})
	if err != nil || total != 1 || len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("numeric ID search: total=%d items=%v err=%v", total, listed, err)
	}
	listed, total, _, _, err = users.List(ctx, &dto.AdminUserQuery{Keyword: strings.TrimSpace(strings.Repeat(" ", 1))})
	if err != nil || total != 1 || len(listed) != 1 {
		t.Fatalf("empty keyword list: total=%d items=%d err=%v", total, len(listed), err)
	}

	newUsername := "Renamed"
	unbanned := false
	updated, err := users.Update(ctx, created.ID, created.ID, &dto.UpdateAdminUserRequest{
		Username: &newUsername, IsBanned: &unbanned,
	})
	if err != nil {
		t.Fatalf("update user: %v", err)
	}
	if updated.Username != newUsername || updated.Email != created.Email || updated.IsBanned {
		t.Fatalf("unexpected updated user: %+v", updated)
	}
	if _, err := users.Update(ctx, created.ID, created.ID, &dto.UpdateAdminUserRequest{Email: &updated.Email}); err != nil {
		t.Fatalf("same user email must not conflict: %v", err)
	}
	shortPassword := "short"
	if _, err := users.Update(ctx, created.ID, created.ID, &dto.UpdateAdminUserRequest{Password: &shortPassword}); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("expected short password rejection, got %v", err)
	}

	other, err := users.Create(ctx, &dto.CreateAdminUserRequest{
		Username: "Other", Email: "other@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("create other user: %v", err)
	}
	duplicate := strings.ToUpper(updated.Username)
	if _, err := users.Update(ctx, created.ID, other.ID, &dto.UpdateAdminUserRequest{Username: &duplicate}); !errors.Is(err, ErrUsernameExists) {
		t.Fatalf("expected case-insensitive username conflict, got %v", err)
	}
	if err := users.Delete(ctx, other.ID, other.ID); !errors.Is(err, ErrSelfDelete) {
		t.Fatalf("expected self-delete rejection, got %v", err)
	}
	if err := users.Delete(ctx, created.ID, other.ID); err != nil {
		t.Fatalf("delete other user: %v", err)
	}
	if _, err := users.Get(ctx, other.ID); !errors.Is(err, ErrAdminUserNotFound) {
		t.Fatalf("expected deleted user not found, got %v", err)
	}
	roles, err = rbac.GetUserRoleCodes(ctx, other.ID)
	if err != nil || len(roles) != 0 {
		t.Fatalf("expected deleted user roles removed, roles=%v err=%v", roles, err)
	}
}

func TestUserAdminSafeguards(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	ctx := context.Background()
	rbac := rbacService.NewRBACService(rbacRepo.NewRBACRepository(db))
	if err := rbac.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed rbac: %v", err)
	}
	users := NewUserAdminService(userRepo.NewUserAdminRepository(db), rbac)
	actorID := testutil.CreateUser(t, db, "admin-actor")
	superAdminID := testutil.CreateUser(t, db, "protected-super")
	if err := rbac.AssignRoleByCode(ctx, superAdminID, rbacService.RoleCodeSuperAdmin); err != nil {
		t.Fatalf("assign super admin: %v", err)
	}

	banned := true
	if _, err := users.Update(ctx, actorID, actorID, &dto.UpdateAdminUserRequest{IsBanned: &banned}); !errors.Is(err, ErrSelfBan) {
		t.Fatalf("expected self-ban rejection, got %v", err)
	}
	username := "blocked-update"
	if _, err := users.Update(ctx, actorID, superAdminID, &dto.UpdateAdminUserRequest{Username: &username}); !errors.Is(err, ErrSuperAdminUserProtected) {
		t.Fatalf("expected super-admin update rejection, got %v", err)
	}
	if err := users.Delete(ctx, actorID, superAdminID); !errors.Is(err, ErrSuperAdminUserProtected) {
		t.Fatalf("expected super-admin delete rejection, got %v", err)
	}
}

func TestAdminUserPagination(t *testing.T) {
	if page, limit := adminUserPagination(0, 0); page != 1 || limit != 20 {
		t.Fatalf("unexpected defaults: page=%d limit=%d", page, limit)
	}
	if _, limit := adminUserPagination(1, 101); limit != 100 {
		t.Fatalf("expected limit cap 100, got %d", limit)
	}
}

func TestUserAdminMutationUsesGuardTransaction(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	ctx := context.Background()
	rollbackErr := errors.New("force rollback")
	repo := userRepo.NewUserAdminRepository(db)
	users := NewUserAdminService(repo, &rollbackUserAdminRBAC{db: db, err: rollbackErr})
	actorID := testutil.CreateUser(t, db, "tx-actor")

	updateID := testutil.CreateUser(t, db, "tx-update")
	updatedUsername := "tx-update-changed"
	if _, err := users.Update(ctx, actorID, updateID, &dto.UpdateAdminUserRequest{Username: &updatedUsername}); !errors.Is(err, rollbackErr) {
		t.Fatalf("expected rollback error from update, got %v", err)
	}
	updated, err := repo.FindByID(ctx, updateID)
	if err != nil || updated == nil || updated.Username != "tx-update" {
		t.Fatalf("update escaped guard transaction: user=%+v err=%v", updated, err)
	}

	deleteID := testutil.CreateUser(t, db, "tx-delete")
	if err := users.Delete(ctx, actorID, deleteID); !errors.Is(err, rollbackErr) {
		t.Fatalf("expected rollback error from delete, got %v", err)
	}
	deleted, err := repo.FindByID(ctx, deleteID)
	if err != nil || deleted == nil {
		t.Fatalf("delete escaped guard transaction: user=%+v err=%v", deleted, err)
	}
}
