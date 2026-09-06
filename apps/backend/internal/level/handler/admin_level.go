package handler

import (
	"errors"

	leveldto "backend/internal/level/dto"
	levelService "backend/internal/level/service"
	"backend/internal/middleware"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AdminLevelHandler struct {
	configs      *levelService.LevelConfigService
	experiences  *levelService.ExperienceService
}

func NewAdminLevelHandler(
	configs *levelService.LevelConfigService,
	experiences *levelService.ExperienceService,
) *AdminLevelHandler {
	return &AdminLevelHandler{configs: configs, experiences: experiences}
}

// ListLevels godoc
// @Summary      管理端查询等级配置
// @Description  返回全部等级定义；需要 level_config:read 权限
// @ID           listAdminLevels
// @Tags         admin
// @Produce      json
// @Success      200 {object} leveldto.LevelConfigListResponse "等级配置列表"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/levels [get]
func (h *AdminLevelHandler) ListLevels(c *gin.Context) {
	configs, err := h.configs.ListLevels(c.Request.Context())
	if err != nil {
		h.respondError(c, err, "list admin levels")
		return
	}
	items := leveldto.NewLevelConfigList(configs)
	response.Ok(c, leveldto.LevelConfigListData{Items: items, Total: int64(len(items))})
}

// CreateLevel godoc
// @Summary      创建等级配置
// @Description  新增一个等级定义；需要 level_config:create 权限
// @ID           createAdminLevel
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body leveldto.CreateLevelConfigRequest true "创建等级请求"
// @Success      200 {object} leveldto.LevelConfigDataResponse "创建成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      409 {object} response.ErrorResponse "等级已存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/levels [post]
func (h *AdminLevelHandler) CreateLevel(c *gin.Context) {
	var req leveldto.CreateLevelConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	config, err := h.configs.CreateLevel(c.Request.Context(), &req)
	if err != nil {
		h.respondError(c, err, "create level config")
		return
	}
	response.Ok(c, leveldto.NewLevelConfigData(config))
}

// UpdateLevel godoc
// @Summary      更新等级配置
// @Description  更新等级名称、所需经验、图标、颜色、描述与启用状态；需要 level_config:update 权限
// @ID           updateAdminLevel
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "等级配置 ID"
// @Param        request body leveldto.UpdateLevelConfigRequest true "更新等级请求"
// @Success      200 {object} leveldto.LevelConfigDataResponse "更新成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "等级配置不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/levels/{id} [put]
func (h *AdminLevelHandler) UpdateLevel(c *gin.Context) {
	id, ok := parseLevelID(c, "等级配置")
	if !ok {
		return
	}
	var req leveldto.UpdateLevelConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	config, err := h.configs.UpdateLevel(c.Request.Context(), id, &req)
	if err != nil {
		h.respondError(c, err, "update level config")
		return
	}
	response.Ok(c, leveldto.NewLevelConfigData(config))
}

// DeleteLevel godoc
// @Summary      删除等级配置
// @Description  删除未被用户持有的等级；仍有用户处于该等级时仅禁用；需要 level_config:update 权限
// @ID           deleteAdminLevel
// @Tags         admin
// @Produce      json
// @Param        id path int true "等级配置 ID"
// @Success      200 {object} response.MessageResponse "等级已删除或已禁用"
// @Failure      400 {object} response.ErrorResponse "等级配置 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "等级配置不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/levels/{id} [delete]
func (h *AdminLevelHandler) DeleteLevel(c *gin.Context) {
	id, ok := parseLevelID(c, "等级配置")
	if !ok {
		return
	}
	disabled, err := h.configs.DeleteLevel(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err, "delete level config")
		return
	}
	if disabled {
		response.OkWithMsg(c, "仍有用户处于该等级，已改为禁用")
		return
	}
	response.OkWithMsg(c, "等级配置已删除")
}

// ListExperienceRules godoc
// @Summary      管理端查询经验规则
// @Description  返回全部经验事件规则；需要 experience_rule:read 权限
// @ID           listAdminExperienceRules
// @Tags         admin
// @Produce      json
// @Success      200 {object} leveldto.ExperienceRuleListResponse "经验规则列表"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/experience-rules [get]
func (h *AdminLevelHandler) ListExperienceRules(c *gin.Context) {
	rules, err := h.configs.ListRules(c.Request.Context())
	if err != nil {
		h.respondError(c, err, "list experience rules")
		return
	}
	items := leveldto.NewExperienceRuleList(rules)
	response.Ok(c, leveldto.ExperienceRuleListData{Items: items, Total: int64(len(items))})
}

// UpdateExperienceRule godoc
// @Summary      更新经验规则
// @Description  更新经验事件的名称、经验值、每日次数、每日经验上限、冷却与启用状态；需要 experience_rule:update 权限
// @ID           updateAdminExperienceRule
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "经验规则 ID"
// @Param        request body leveldto.UpdateExperienceRuleRequest true "更新经验规则请求"
// @Success      200 {object} leveldto.ExperienceRuleDataResponse "更新成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "经验规则不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/experience-rules/{id} [put]
func (h *AdminLevelHandler) UpdateExperienceRule(c *gin.Context) {
	id, ok := parseLevelID(c, "经验规则")
	if !ok {
		return
	}
	var req leveldto.UpdateExperienceRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	rule, err := h.configs.UpdateRule(c.Request.Context(), id, &req)
	if err != nil {
		h.respondError(c, err, "update experience rule")
		return
	}
	response.Ok(c, leveldto.NewExperienceRuleData(rule))
}

// AdjustUserExperience godoc
// @Summary      管理员调整用户经验
// @Description  手动增加或扣减用户经验（扣减后不低于 0）并写入经验流水；需要 experience:adjust 权限
// @ID           adjustUserExperience
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "用户 ID"
// @Param        request body leveldto.AdjustExperienceRequest true "调整经验请求"
// @Success      200 {object} leveldto.UserLevelResponse "调整后的等级概况"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "用户不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/users/{id}/experience/adjust [post]
func (h *AdminLevelHandler) AdjustUserExperience(c *gin.Context) {
	userID, ok := parseLevelID(c, "用户")
	if !ok {
		return
	}
	var req leveldto.AdjustExperienceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	operatorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	if _, err := h.experiences.Adjust(c.Request.Context(), userID, req.Exp, req.Reason, operatorID); err != nil {
		h.respondError(c, err, "adjust user experience")
		return
	}
	data, err := h.experiences.GetProfile(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err, "reload user experience")
		return
	}
	response.Ok(c, data)
}

func (h *AdminLevelHandler) respondError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, levelService.ErrLevelConfigNotFound):
		response.Error(c, appErrors.ErrNotFound("等级配置不存在"))
	case errors.Is(err, levelService.ErrLevelConfigExists):
		response.Error(c, appErrors.ErrConflict("该等级已存在"))
	case errors.Is(err, levelService.ErrLevelConfigInvalid),
		errors.Is(err, levelService.ErrExperienceRuleInvalid),
		errors.Is(err, levelService.ErrInvalidAdjustment):
		response.Error(c, appErrors.ErrValidation("配置参数不正确"))
	case errors.Is(err, levelService.ErrExperienceRuleNotFound):
		response.Error(c, appErrors.ErrNotFound("经验规则不存在"))
	case errors.Is(err, levelService.ErrUserNotFound):
		response.Error(c, appErrors.ErrNotFound("用户不存在"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("等级配置操作失败"))
	}
}
