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
	{Code: "role:assign", Name: "分配角色", Description: "允许分配用户角色"},
	{Code: "permission:list", Name: "查看权限", Description: "允许查看权限列表"},
	{Code: "permission:create", Name: "创建权限", Description: "允许创建权限"},
	{Code: "permission:update", Name: "更新权限", Description: "允许更新权限"},
	{Code: "permission:delete", Name: "删除权限", Description: "允许删除权限"},
	{Code: "permission:assign", Name: "分配权限", Description: "允许调整角色权限"},
	{Code: "galgame:create", Name: "创建 Galgame", Description: "允许创建 Galgame、开发商和 Tag"},
	{Code: "galgame:update", Name: "更新 Galgame", Description: "允许更新 Galgame、开发商和 Tag"},
	{Code: "galgame:delete", Name: "删除 Galgame", Description: "允许删除 Galgame"},
	{Code: "galgame:review", Name: "审核 Galgame", Description: "允许查看全部状态的 Galgame"},
	{Code: "resource:update", Name: "更新资源", Description: "允许更新任何用户上传的资源"},
	{Code: "resource:delete", Name: "删除资源", Description: "允许删除任何用户上传的资源"},
	{Code: "resource:review", Name: "审核资源", Description: "允许审核资源发布状态"},
	{Code: "resource_report:list", Name: "查看资源举报", Description: "允许查看资源举报列表"},
	{Code: "resource_report:handle", Name: "处理资源举报", Description: "允许处理资源举报"},
	{Code: "feedback:read", Name: "查看反馈", Description: "允许查看意见反馈与版权投诉"},
	{Code: "feedback:handle", Name: "处理反馈", Description: "允许处理意见反馈与版权投诉"},
	{Code: "post:moderate", Name: "管理帖子", Description: "允许管理任何用户的帖子"},
	{Code: "comment:moderate", Name: "管理评论", Description: "允许管理任何用户的评论"},
	{Code: "banner:read", Name: "查看 Banner", Description: "允许查看管理端 Banner"},
	{Code: "banner:create", Name: "创建 Banner", Description: "允许创建 Banner"},
	{Code: "banner:update", Name: "更新 Banner", Description: "允许更新 Banner"},
	{Code: "banner:delete", Name: "删除 Banner", Description: "允许删除 Banner"},
	{Code: "background_preset:read", Name: "查看背景预设", Description: "允许查看管理端背景预设"},
	{Code: "background_preset:create", Name: "创建背景预设", Description: "允许创建背景预设"},
	{Code: "background_preset:update", Name: "更新背景预设", Description: "允许更新背景预设"},
	{Code: "background_preset:delete", Name: "删除背景预设", Description: "允许删除背景预设"},
	{Code: "article:read", Name: "查看文章", Description: "允许查看管理端文章"},
	{Code: "article:create", Name: "创建文章", Description: "允许创建文章"},
	{Code: "article:update", Name: "更新文章", Description: "允许更新文章内容和置顶状态"},
	{Code: "article:delete", Name: "删除文章", Description: "允许删除文章"},
	{Code: "article:publish", Name: "发布文章", Description: "允许发布、定时发布和取消发布文章"},
	{Code: "image:manage", Name: "上传管理图片", Description: "允许上传 galgames、banners、admin 分类的图片"},
	{Code: "image:read", Name: "查看图片资源", Description: "允许查看管理端图片资源列表"},
	{Code: "image:delete", Name: "删除图片资源", Description: "允许删除任意用户上传的图片"},
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
		"galgame:create",
		"galgame:update",
		"galgame:delete",
		"galgame:review",
		"resource:update",
		"resource:delete",
		"resource:review",
		"resource_report:list",
		"resource_report:handle",
		"feedback:read",
		"feedback:handle",
		"post:moderate",
		"comment:moderate",
		"banner:read",
		"banner:create",
		"banner:update",
		"banner:delete",
		"background_preset:read",
		"background_preset:create",
		"background_preset:update",
		"background_preset:delete",
		"article:read",
		"article:create",
		"article:update",
		"article:delete",
		"article:publish",
		"image:manage",
		"image:read",
		"image:delete",
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
