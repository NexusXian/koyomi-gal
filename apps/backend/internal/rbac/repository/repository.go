package repository

import (
	"context"
	"errors"
	"time"

	"backend/internal/rbac/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RBACRepository struct {
	db *gorm.DB
}

func NewRBACRepository(db *gorm.DB) *RBACRepository {
	return &RBACRepository{db: db}
}

// Transaction runs fn with a transaction-bound RBACRepository.
func (r *RBACRepository) Transaction(ctx context.Context, fn func(tx *RBACRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&RBACRepository{db: tx})
	})
}

// CreateRole inserts a new role row.
func (r *RBACRepository) CreateRole(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

// UpdateRole persists mutable role fields; code is immutable.
func (r *RBACRepository) UpdateRole(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).
		Model(&model.Role{}).
		Where("id = ?", role.ID).
		Updates(map[string]any{
			"name":        role.Name,
			"description": role.Description,
			"updated_at":  time.Now(),
		}).Error
}

func (r *RBACRepository) DeleteRole(ctx context.Context, roleID int64) error {
	return r.db.WithContext(ctx).Delete(&model.Role{}, roleID).Error
}

// FindRoleByID returns nil, nil when the role does not exist.
func (r *RBACRepository) FindRoleByID(ctx context.Context, roleID int64) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).First(&role, roleID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// FindRoleByCode returns nil, nil when the role does not exist.
func (r *RBACRepository) FindRoleByCode(ctx context.Context, code string) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *RBACRepository) ListRoles(ctx context.Context) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.WithContext(ctx).Order("id").Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// CountRolesByIDs returns how many of the given role ids exist.
func (r *RBACRepository) CountRolesByIDs(ctx context.Context, roleIDs []int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Role{}).
		Where("id IN ?", roleIDs).
		Count(&count).Error
	return count, err
}

func (r *RBACRepository) CreatePermission(ctx context.Context, permission *model.Permission) error {
	return r.db.WithContext(ctx).Create(permission).Error
}

// UpdatePermission persists mutable permission fields; code/resource/action are immutable.
func (r *RBACRepository) UpdatePermission(ctx context.Context, permission *model.Permission) error {
	return r.db.WithContext(ctx).
		Model(&model.Permission{}).
		Where("id = ?", permission.ID).
		Updates(map[string]any{
			"name":        permission.Name,
			"description": permission.Description,
			"updated_at":  time.Now(),
		}).Error
}

func (r *RBACRepository) DeletePermission(ctx context.Context, permissionID int64) error {
	return r.db.WithContext(ctx).Delete(&model.Permission{}, permissionID).Error
}

// FindPermissionByID returns nil, nil when the permission does not exist.
func (r *RBACRepository) FindPermissionByID(ctx context.Context, permissionID int64) (*model.Permission, error) {
	var permission model.Permission
	err := r.db.WithContext(ctx).First(&permission, permissionID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &permission, nil
}

// FindPermissionByCode returns nil, nil when the permission does not exist.
func (r *RBACRepository) FindPermissionByCode(ctx context.Context, code string) (*model.Permission, error) {
	var permission model.Permission
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&permission).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &permission, nil
}

func (r *RBACRepository) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	var permissions []model.Permission
	err := r.db.WithContext(ctx).Order("id").Find(&permissions).Error
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

// CountPermissionsByIDs returns how many of the given permission ids exist.
func (r *RBACRepository) CountPermissionsByIDs(ctx context.Context, permissionIDs []int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Permission{}).
		Where("id IN ?", permissionIDs).
		Count(&count).Error
	return count, err
}

func (r *RBACRepository) GetUserRoles(ctx context.Context, userID uint) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.WithContext(ctx).
		Table("roles AS r").
		Select("r.*").
		Joins("JOIN user_roles ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID).
		Order("r.id").
		Scan(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *RBACRepository) GetUserRoleCodes(ctx context.Context, userID uint) ([]string, error) {
	var codes []string
	err := r.db.WithContext(ctx).
		Table("roles AS r").
		Select("r.code").
		Joins("JOIN user_roles ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID).
		Order("r.code").
		Scan(&codes).Error
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// InsertUserRoles binds roles to a user, ignoring pairs that already exist.
func (r *RBACRepository) InsertUserRoles(ctx context.Context, userID uint, roleIDs []int64) error {
	if len(roleIDs) == 0 {
		return nil
	}
	rows := make([]model.UserRole, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		rows = append(rows, model.UserRole{UserID: userID, RoleID: roleID})
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&rows).Error
}

func (r *RBACRepository) DeleteUserRole(ctx context.Context, userID uint, roleID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&model.UserRole{}).Error
}

func (r *RBACRepository) DeleteUserRolesByUserID(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&model.UserRole{}).Error
}

func (r *RBACRepository) DeleteUserRolesByRoleID(ctx context.Context, roleID int64) error {
	return r.db.WithContext(ctx).
		Where("role_id = ?", roleID).
		Delete(&model.UserRole{}).Error
}

func (r *RBACRepository) CountUserRole(ctx context.Context, userID uint, roleID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.UserRole{}).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Count(&count).Error
	return count, err
}

func (r *RBACRepository) GetRolePermissions(ctx context.Context, roleID int64) ([]model.Permission, error) {
	var permissions []model.Permission
	err := r.db.WithContext(ctx).
		Table("permissions AS p").
		Select("p.*").
		Joins("JOIN role_permissions rp ON rp.permission_id = p.id").
		Where("rp.role_id = ?", roleID).
		Order("p.id").
		Scan(&permissions).Error
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

// InsertRolePermissions grants permissions to a role, ignoring pairs that already exist.
func (r *RBACRepository) InsertRolePermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	if len(permissionIDs) == 0 {
		return nil
	}
	rows := make([]model.RolePermission, 0, len(permissionIDs))
	for _, permissionID := range permissionIDs {
		rows = append(rows, model.RolePermission{RoleID: roleID, PermissionID: permissionID})
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&rows).Error
}

func (r *RBACRepository) DeleteRolePermissionsByRoleID(ctx context.Context, roleID int64) error {
	return r.db.WithContext(ctx).
		Where("role_id = ?", roleID).
		Delete(&model.RolePermission{}).Error
}

func (r *RBACRepository) DeleteRolePermissionsByPermissionID(ctx context.Context, permissionID int64) error {
	return r.db.WithContext(ctx).
		Where("permission_id = ?", permissionID).
		Delete(&model.RolePermission{}).Error
}

// GetUserPermissions returns the distinct permission codes granted to the user through its roles.
func (r *RBACRepository) GetUserPermissions(ctx context.Context, userID uint) ([]string, error) {
	var codes []string
	err := r.db.WithContext(ctx).Raw(`
SELECT DISTINCT p.code
FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = ?
ORDER BY p.code
`, userID).Scan(&codes).Error
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// HasPermission reports whether any role of the user grants the permission code.
func (r *RBACRepository) HasPermission(ctx context.Context, userID uint, permission string) (bool, error) {
	var allowed bool
	err := r.db.WithContext(ctx).Raw(`
SELECT EXISTS (
    SELECT 1
    FROM user_roles ur
    JOIN role_permissions rp ON rp.role_id = ur.role_id
    JOIN permissions p ON p.id = rp.permission_id
    WHERE ur.user_id = ? AND p.code = ?
)
`, userID, permission).Scan(&allowed).Error
	if err != nil {
		return false, err
	}
	return allowed, nil
}

// HasRole reports whether the user is bound to the role code.
func (r *RBACRepository) HasRole(ctx context.Context, userID uint, roleCode string) (bool, error) {
	var has bool
	err := r.db.WithContext(ctx).Raw(`
SELECT EXISTS (
    SELECT 1
    FROM user_roles ur
    JOIN roles r ON r.id = ur.role_id
    WHERE ur.user_id = ? AND r.code = ?
)
`, userID, roleCode).Scan(&has).Error
	if err != nil {
		return false, err
	}
	return has, nil
}

// UserExists reports whether the users table contains the id.
func (r *RBACRepository) UserExists(ctx context.Context, userID uint) (bool, error) {
	var exists bool
	err := r.db.WithContext(ctx).
		Table("users").
		Select("EXISTS (SELECT 1 FROM users WHERE id = ?)", userID).
		Scan(&exists).Error
	if err != nil {
		return false, err
	}
	return exists, nil
}
