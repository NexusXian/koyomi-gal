package service

import (
	"context"
	"strings"

	"backend/internal/rbac/model"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

const (
	RoleCodeSuperAdmin = "super_admin"
	RoleCodeAdmin      = "admin"
	RoleCodeUser       = "user"
)

type seedRole struct {
	Code        string
	Name        string
	Description string
}

type seedPermission struct {
	Code        string
	Name        string
	Description string
}

var seedRoles = []seedRole{
	{Code: RoleCodeSuperAdmin, Name: "超级管理员", Description: "拥有系统全部权限"},
	{Code: RoleCodeAdmin, Name: "管理员", Description: "系统管理角色"},
	{Code: RoleCodeUser, Name: "普通用户", Description: "注册用户默认角色"},
}

var seedPermissions = []seedPermission{
	{Code: "user:list", Name: "查看用户列表", Description: "允许查看用户列表"},
	{Code: "user:read", Name: "查看用户详情", Description: "允许查看用户详情"},
	{Code: "user:create", Name: "创建用户", Description: "允许创建用户"},
	{Code: "user:update", Name: "更新用户", Description: "允许更新用户"},
	{Code: "user:delete", Name: "删除用户", Description: "允许删除用户"},
	{Code: "role:list", Name: "查看角色", Description: "允许查看角色列表"},
	{Code: "role:create", Name: "创建角色", Description: "允许创建角色"},
	{Code: "role:update", Name: "更新角色", Description: "允许更新角色"},
	{Code: "role:delete", Name: "删除角色", Description: "允许删除角色"},
	{Code: "role:assign", Name: "分配角色", Description: "允许分配用户角色和角色权限"},
	{Code: "permission:list", Name: "查看权限", Description: "允许查看权限列表"},
	{Code: "permission:create", Name: "创建权限", Description: "允许创建权限"},
	{Code: "permission:update", Name: "更新权限", Description: "允许更新权限"},
	{Code: "permission:delete", Name: "删除权限", Description: "允许删除权限"},
	{Code: "permission:assign", Name: "分配权限", Description: "允许调整角色权限"},
}

// seedRolePermissions maps seed role codes to granted seed permission codes.
// super_admin additionally receives every permission in the table.
var seedRolePermissions = map[string][]string{
	RoleCodeAdmin: {
		"user:list",
		"user:read",
		"user:create",
		"user:update",
		"role:list",
		"permission:list",
	},
}

// SeedDefaults creates missing default roles and permissions and re-syncs
// the seeded role-permission grants (additive only; it never revokes grants).
func (s *RBACService) SeedDefaults(ctx context.Context) error {
	roleIDs, err := s.seedRoles(ctx)
	if err != nil {
		return err
	}
	permissionIDs, err := s.seedPermissions(ctx)
	if err != nil {
		return err
	}

	allPermissionIDs := make([]int64, 0, len(permissionIDs))
	for _, permission := range seedPermissions {
		if id, ok := permissionIDs[permission.Code]; ok {
			allPermissionIDs = append(allPermissionIDs, id)
		}
	}

	roleGrants := make(map[string][]int64, len(seedRolePermissions)+1)
	roleGrants[RoleCodeSuperAdmin] = allPermissionIDs
	for roleCode, permissionCodes := range seedRolePermissions {
		ids := make([]int64, 0, len(permissionCodes))
		for _, code := range permissionCodes {
			if id, ok := permissionIDs[code]; ok {
				ids = append(ids, id)
			}
		}
		roleGrants[roleCode] = ids
	}

	for roleCode, permissionIDs := range roleGrants {
		roleID, ok := roleIDs[roleCode]
		if !ok {
			continue
		}
		if err := s.repo.InsertRolePermissions(ctx, roleID, permissionIDs); err != nil {
			logger.Error("seed role permissions", zap.String("role_code", roleCode), zap.Error(err))
			return err
		}
	}
	return nil
}

func (s *RBACService) seedRoles(ctx context.Context) (map[string]int64, error) {
	roleIDs := make(map[string]int64, len(seedRoles))
	for _, role := range seedRoles {
		existing, err := s.repo.FindRoleByCode(ctx, role.Code)
		if err != nil {
			logger.Error("find seed role", zap.String("code", role.Code), zap.Error(err))
			return nil, err
		}
		if existing != nil {
			roleIDs[role.Code] = existing.ID
			continue
		}
		created := &model.Role{
			Name:        role.Name,
			Code:        role.Code,
			Description: role.Description,
		}
		if err := s.repo.CreateRole(ctx, created); err != nil {
			logger.Error("create seed role", zap.String("code", role.Code), zap.Error(err))
			return nil, err
		}
		roleIDs[role.Code] = created.ID
	}
	return roleIDs, nil
}

func (s *RBACService) seedPermissions(ctx context.Context) (map[string]int64, error) {
	permissionIDs := make(map[string]int64, len(seedPermissions))
	for _, permission := range seedPermissions {
		existing, err := s.repo.FindPermissionByCode(ctx, permission.Code)
		if err != nil {
			logger.Error("find seed permission", zap.String("code", permission.Code), zap.Error(err))
			return nil, err
		}
		if existing != nil {
			permissionIDs[permission.Code] = existing.ID
			continue
		}
		resource, action, _ := strings.Cut(permission.Code, ":")
		created := &model.Permission{
			Name:        permission.Name,
			Code:        permission.Code,
			Resource:    resource,
			Action:      action,
			Description: permission.Description,
		}
		if err := s.repo.CreatePermission(ctx, created); err != nil {
			logger.Error("create seed permission", zap.String("code", permission.Code), zap.Error(err))
			return nil, err
		}
		permissionIDs[permission.Code] = created.ID
	}
	return permissionIDs, nil
}
