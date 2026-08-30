package dto

type UpdateUserRolesRequest struct {
	RoleIDs []int64 `json:"role_ids" binding:"required" example:"1,2"`
}

type UpdateRolePermissionsRequest struct {
	PermissionIDs []int64 `json:"permission_ids" binding:"required" example:"1,2,3"`
}

type MePermissionsData struct {
	Roles       []string `json:"roles" example:"admin"`
	Permissions []string `json:"permissions" example:"user:list,user:delete"`
}

type MePermissionsResponse struct {
	Code int               `json:"code" example:"0"`
	Data MePermissionsData `json:"data"`
	Msg  string            `json:"msg" example:"success"`
}
