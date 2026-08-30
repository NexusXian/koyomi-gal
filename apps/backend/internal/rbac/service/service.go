package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"backend/internal/rbac/dto"
	"backend/internal/rbac/model"
	"backend/internal/rbac/repository"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

var (
	ErrRoleNotFound          = errors.New("role not found")
	ErrPermissionNotFound    = errors.New("permission not found")
	ErrUserNotFound          = errors.New("user not found")
	ErrRoleCodeExists        = errors.New("role code already exists")
	ErrPermissionCodeExists  = errors.New("permission code already exists")
	ErrInvalidRoleCode       = errors.New("invalid role code")
	ErrInvalidPermissionCode = errors.New("invalid permission code")
	ErrUnknownRoleIDs        = errors.New("role ids contain unknown role")
	ErrUnknownPermissionIDs  = errors.New("permission ids contain unknown permission")
	ErrRoleProtected         = errors.New("role is protected")
)

var (
	roleCodePattern       = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	permissionCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}:[a-z][a-z0-9_]{0,63}$`)
)

type RBACService struct {
	repo *repository.RBACRepository
}

func NewRBACService(repo *repository.RBACRepository) *RBACService {
	return &RBACService{repo: repo}
}

func (s *RBACService) HasPermission(
	ctx context.Context,
	userID uint,
	permission string,
) (bool, error) {
	return s.repo.HasPermission(ctx, userID, permission)
}

func (s *RBACService) HasRole(
	ctx context.Context,
	userID uint,
	roleCode string,
) (bool, error) {
	return s.repo.HasRole(ctx, userID, roleCode)
}

func (s *RBACService) GetUserPermissions(ctx context.Context, userID uint) ([]string, error) {
	return s.repo.GetUserPermissions(ctx, userID)
}

func (s *RBACService) GetUserRoleCodes(ctx context.Context, userID uint) ([]string, error) {
	return s.repo.GetUserRoleCodes(ctx, userID)
}

func (s *RBACService) GetUserRoles(ctx context.Context, userID uint) ([]model.Role, error) {
	return s.repo.GetUserRoles(ctx, userID)
}

func (s *RBACService) CreateRole(ctx context.Context, req *dto.CreateRoleRequest) (*model.Role, error) {
	code := strings.ToLower(strings.TrimSpace(req.Code))
	if !roleCodePattern.MatchString(code) {
		return nil, ErrInvalidRoleCode
	}

	existing, err := s.repo.FindRoleByCode(ctx, code)
	if err != nil {
		logger.Error("find role by code", zap.String("code", code), zap.Error(err))
		return nil, err
	}
	if existing != nil {
		return nil, ErrRoleCodeExists
	}

	role := &model.Role{
		Name:        strings.TrimSpace(req.Name),
		Code:        code,
		Description: strings.TrimSpace(req.Description),
	}
	if err := s.repo.CreateRole(ctx, role); err != nil {
		logger.Error("create role", zap.String("code", code), zap.Error(err))
		return nil, err
	}
	return role, nil
}

func (s *RBACService) UpdateRole(
	ctx context.Context,
	roleID int64,
	req *dto.UpdateRoleRequest,
) (*model.Role, error) {
	role, err := s.repo.FindRoleByID(ctx, roleID)
	if err != nil {
		logger.Error("find role by id", zap.Int64("role_id", roleID), zap.Error(err))
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}

	role.Name = strings.TrimSpace(req.Name)
	role.Description = strings.TrimSpace(req.Description)
	if err := s.repo.UpdateRole(ctx, role); err != nil {
		logger.Error("update role", zap.Int64("role_id", roleID), zap.Error(err))
		return nil, err
	}
	return role, nil
}

// DeleteRole removes the role and cleans user_roles / role_permissions in one transaction.
func (s *RBACService) DeleteRole(ctx context.Context, roleID int64) error {
	role, err := s.repo.FindRoleByID(ctx, roleID)
	if err != nil {
		logger.Error("find role by id", zap.Int64("role_id", roleID), zap.Error(err))
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}
	if role.Code == RoleCodeSuperAdmin {
		return ErrRoleProtected
	}

	err = s.repo.Transaction(ctx, func(tx *repository.RBACRepository) error {
		if err := tx.DeleteUserRolesByRoleID(ctx, roleID); err != nil {
			return err
		}
		if err := tx.DeleteRolePermissionsByRoleID(ctx, roleID); err != nil {
			return err
		}
		return tx.DeleteRole(ctx, roleID)
	})
	if err != nil {
		logger.Error("delete role", zap.Int64("role_id", roleID), zap.Error(err))
		return err
	}
	return nil
}

func (s *RBACService) GetRole(ctx context.Context, roleID int64) (*model.Role, error) {
	role, err := s.repo.FindRoleByID(ctx, roleID)
	if err != nil {
		logger.Error("find role by id", zap.Int64("role_id", roleID), zap.Error(err))
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}
	return role, nil
}

func (s *RBACService) ListRoles(ctx context.Context) ([]model.Role, error) {
	return s.repo.ListRoles(ctx)
}

// CreatePermission derives resource/action from the "resource:action" code.
func (s *RBACService) CreatePermission(ctx context.Context, req *dto.CreatePermissionRequest) (*model.Permission, error) {
	code := strings.ToLower(strings.TrimSpace(req.Code))
	if !permissionCodePattern.MatchString(code) {
		return nil, ErrInvalidPermissionCode
	}

	existing, err := s.repo.FindPermissionByCode(ctx, code)
	if err != nil {
		logger.Error("find permission by code", zap.String("code", code), zap.Error(err))
		return nil, err
	}
	if existing != nil {
		return nil, ErrPermissionCodeExists
	}

	resource, action, _ := strings.Cut(code, ":")
	permission := &model.Permission{
		Name:        strings.TrimSpace(req.Name),
		Code:        code,
		Resource:    resource,
		Action:      action,
		Description: strings.TrimSpace(req.Description),
	}
	if err := s.repo.CreatePermission(ctx, permission); err != nil {
		logger.Error("create permission", zap.String("code", code), zap.Error(err))
		return nil, err
	}
	return permission, nil
}

func (s *RBACService) UpdatePermission(
	ctx context.Context,
	permissionID int64,
	req *dto.UpdatePermissionRequest,
) (*model.Permission, error) {
	permission, err := s.repo.FindPermissionByID(ctx, permissionID)
	if err != nil {
		logger.Error("find permission by id", zap.Int64("permission_id", permissionID), zap.Error(err))
		return nil, err
	}
	if permission == nil {
		return nil, ErrPermissionNotFound
	}

	permission.Name = strings.TrimSpace(req.Name)
	permission.Description = strings.TrimSpace(req.Description)
	if err := s.repo.UpdatePermission(ctx, permission); err != nil {
		logger.Error("update permission", zap.Int64("permission_id", permissionID), zap.Error(err))
		return nil, err
	}
	return permission, nil
}

// DeletePermission removes the permission and cleans role_permissions in one transaction.
func (s *RBACService) DeletePermission(ctx context.Context, permissionID int64) error {
	permission, err := s.repo.FindPermissionByID(ctx, permissionID)
	if err != nil {
		logger.Error("find permission by id", zap.Int64("permission_id", permissionID), zap.Error(err))
		return err
	}
	if permission == nil {
		return ErrPermissionNotFound
	}

	err = s.repo.Transaction(ctx, func(tx *repository.RBACRepository) error {
		if err := tx.DeleteRolePermissionsByPermissionID(ctx, permissionID); err != nil {
			return err
		}
		return tx.DeletePermission(ctx, permissionID)
	})
	if err != nil {
		logger.Error("delete permission", zap.Int64("permission_id", permissionID), zap.Error(err))
		return err
	}
	return nil
}

func (s *RBACService) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	return s.repo.ListPermissions(ctx)
}

func (s *RBACService) AssignRoleToUser(ctx context.Context, userID uint, roleID int64) error {
	if err := s.ensureUserExists(ctx, userID); err != nil {
		return err
	}
	role, err := s.repo.FindRoleByID(ctx, roleID)
	if err != nil {
		logger.Error("find role by id", zap.Int64("role_id", roleID), zap.Error(err))
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}
	return s.repo.InsertUserRoles(ctx, userID, []int64{roleID})
}

func (s *RBACService) RemoveRoleFromUser(ctx context.Context, userID uint, roleID int64) error {
	if err := s.ensureUserExists(ctx, userID); err != nil {
		return err
	}
	role, err := s.repo.FindRoleByID(ctx, roleID)
	if err != nil {
		logger.Error("find role by id", zap.Int64("role_id", roleID), zap.Error(err))
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}
	return s.repo.DeleteUserRole(ctx, userID, roleID)
}

// AssignRoleByCode binds a role looked up by code; missing roles are ignored so it can
// safely assign seeded default roles (register default role, super admin bootstrap).
func (s *RBACService) AssignRoleByCode(ctx context.Context, userID uint, roleCode string) error {
	role, err := s.repo.FindRoleByCode(ctx, roleCode)
	if err != nil {
		logger.Error("find role by code", zap.String("code", roleCode), zap.Error(err))
		return err
	}
	if role == nil {
		return nil
	}
	return s.repo.InsertUserRoles(ctx, userID, []int64{role.ID})
}

// SetUserRoles replaces all roles of the user inside one transaction.
func (s *RBACService) SetUserRoles(ctx context.Context, userID uint, roleIDs []int64) error {
	if err := s.ensureUserExists(ctx, userID); err != nil {
		return err
	}

	roleIDs = uniqueInt64(roleIDs)
	if len(roleIDs) > 0 {
		count, err := s.repo.CountRolesByIDs(ctx, roleIDs)
		if err != nil {
			logger.Error("count roles by ids", zap.Error(err))
			return err
		}
		if count != int64(len(roleIDs)) {
			return ErrUnknownRoleIDs
		}
	}

	return s.repo.Transaction(ctx, func(tx *repository.RBACRepository) error {
		if err := tx.DeleteUserRolesByUserID(ctx, userID); err != nil {
			return err
		}
		return tx.InsertUserRoles(ctx, userID, roleIDs)
	})
}

func (s *RBACService) GetRolePermissions(ctx context.Context, roleID int64) ([]model.Permission, error) {
	role, err := s.repo.FindRoleByID(ctx, roleID)
	if err != nil {
		logger.Error("find role by id", zap.Int64("role_id", roleID), zap.Error(err))
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}
	return s.repo.GetRolePermissions(ctx, roleID)
}

// SetRolePermissions replaces all permissions of the role inside one transaction.
func (s *RBACService) SetRolePermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	role, err := s.repo.FindRoleByID(ctx, roleID)
	if err != nil {
		logger.Error("find role by id", zap.Int64("role_id", roleID), zap.Error(err))
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}

	permissionIDs = uniqueInt64(permissionIDs)
	if len(permissionIDs) > 0 {
		count, err := s.repo.CountPermissionsByIDs(ctx, permissionIDs)
		if err != nil {
			logger.Error("count permissions by ids", zap.Error(err))
			return err
		}
		if count != int64(len(permissionIDs)) {
			return ErrUnknownPermissionIDs
		}
	}

	return s.repo.Transaction(ctx, func(tx *repository.RBACRepository) error {
		if err := tx.DeleteRolePermissionsByRoleID(ctx, roleID); err != nil {
			return err
		}
		return tx.InsertRolePermissions(ctx, roleID, permissionIDs)
	})
}

func (s *RBACService) ensureUserExists(ctx context.Context, userID uint) error {
	exists, err := s.repo.UserExists(ctx, userID)
	if err != nil {
		logger.Error("check user exists", zap.Uint("user_id", userID), zap.Error(err))
		return err
	}
	if !exists {
		return ErrUserNotFound
	}
	return nil
}

func uniqueInt64(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	unique := make([]int64, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}
