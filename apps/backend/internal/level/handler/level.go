package handler

import (
	"errors"
	"strconv"

	leveldto "backend/internal/level/dto"
	levelModel "backend/internal/level/model"
	levelService "backend/internal/level/service"
	"backend/internal/middleware"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type LevelHandler struct {
	experiences *levelService.ExperienceService
	checkins    *levelService.CheckinService
}

func NewLevelHandler(
	experiences *levelService.ExperienceService,
	checkins *levelService.CheckinService,
) *LevelHandler {
	return &LevelHandler{experiences: experiences, checkins: checkins}
}

// GetUserExperience godoc
// @Summary      查看我的等级与经验
// @Description  返回当前登录用户的等级、总经验、等级进度与签到状态
// @ID           getMyExperience
// @Tags         me
// @Produce      json
// @Success      200 {object} leveldto.UserLevelResponse "等级概况"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      500 {object} response.ErrorResponse "查询失败"
// @Security     BearerAuth
// @Router       /api/v1/users/me/experience [get]
func (h *LevelHandler) GetUserExperience(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	data, err := h.experiences.GetProfile(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err, "get my experience")
		return
	}
	if checkedIn, consecutive, statusErr := h.checkins.Status(c.Request.Context(), userID); statusErr == nil {
		data.CheckedInToday = checkedIn
		data.ConsecutiveDays = consecutive
	}
	response.Ok(c, data)
}

// ListMyExperienceLogs godoc
// @Summary      查看我的经验流水
// @Description  分页返回当前登录用户的经验变动记录
// @ID           listMyExperienceLogs
// @Tags         me
// @Produce      json
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} leveldto.ExperienceLogListResponse "经验记录列表"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      500 {object} response.ErrorResponse "查询失败"
// @Security     BearerAuth
// @Router       /api/v1/users/me/experience/logs [get]
func (h *LevelHandler) ListMyExperienceLogs(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	var query leveldto.ExperienceQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	logs, total, page, limit, err := h.experiences.GetLogs(c.Request.Context(), userID, query.Page, query.Limit)
	if err != nil {
		h.respondError(c, err, "list my experience logs")
		return
	}
	items := make([]leveldto.ExperienceLogData, 0, len(logs))
	for i := range logs {
		items = append(items, newExperienceLogData(&logs[i]))
	}
	response.Ok(c, leveldto.ExperienceLogListData{Items: items, Total: total, Page: page, Limit: limit})
}

// GetUserLevel godoc
// @Summary      查看用户等级
// @Description  按用户名返回公开等级信息与进度
// @ID           getUserLevel
// @Tags         users
// @Produce      json
// @Param        username path string true "用户名"
// @Success      200 {object} leveldto.UserLevelResponse "等级概况"
// @Failure      400 {object} response.ErrorResponse "用户名格式不正确"
// @Failure      404 {object} response.ErrorResponse "用户不存在"
// @Failure      500 {object} response.ErrorResponse "查询失败"
// @Router       /api/v1/users/{username}/level [get]
func (h *LevelHandler) GetUserLevel(c *gin.Context) {
	username := c.Param("username")
	if username == "" || len(username) > 50 {
		response.Error(c, appErrors.ErrValidation("用户名格式不正确"))
		return
	}
	userID, found, err := h.experiences.FindUserIDByUsername(c.Request.Context(), username)
	if err != nil {
		h.respondError(c, err, "find user by username")
		return
	}
	if !found {
		response.Error(c, appErrors.ErrNotFound("用户不存在"))
		return
	}
	data, err := h.experiences.GetProfile(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err, "get user level")
		return
	}
	response.Ok(c, data)
}

// Checkin godoc
// @Summary      每日签到
// @Description  记录今日签到并发放签到经验；同一天只能签到一次
// @ID           checkin
// @Tags         me
// @Produce      json
// @Success      200 {object} leveldto.CheckinResultResponse "签到结果"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      409 {object} response.ErrorResponse "今天已经签到过了"
// @Failure      500 {object} response.ErrorResponse "签到失败"
// @Security     BearerAuth
// @Router       /api/v1/users/me/checkin [post]
func (h *LevelHandler) Checkin(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	result, err := h.checkins.Checkin(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err, "checkin")
		return
	}
	response.Ok(c, leveldto.CheckinResultData{
		ExpGained:       int(result.ExpGained),
		ConsecutiveDays: result.ConsecutiveDays,
		TotalExp:        result.TotalExp,
		Level:           result.Level,
		LevelName:       result.LevelName,
	})
}

// GetCheckinStatus godoc
// @Summary      查看签到状态
// @Description  返回今天是否已签到与连续签到天数
// @ID           getCheckinStatus
// @Tags         me
// @Produce      json
// @Success      200 {object} leveldto.CheckinStatusResponse "签到状态"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      500 {object} response.ErrorResponse "查询失败"
// @Security     BearerAuth
// @Router       /api/v1/users/me/checkin [get]
func (h *LevelHandler) GetCheckinStatus(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	checkedIn, consecutive, err := h.checkins.Status(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err, "get checkin status")
		return
	}
	response.Ok(c, leveldto.CheckinStatusData{
		CheckedInToday:  checkedIn,
		ConsecutiveDays: consecutive,
	})
}

func newExperienceLogData(log *levelModel.ExperienceLog) leveldto.ExperienceLogData {
	return leveldto.ExperienceLogData{
		ID:          log.ID,
		EventType:   string(log.EventType),
		ExpDelta:    log.ExpDelta,
		SourceType:  log.SourceType,
		SourceID:    log.SourceID,
		Description: log.Description,
		CreatedAt:   log.CreatedAt,
	}
}

func (h *LevelHandler) respondError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, levelService.ErrAlreadyCheckedIn):
		response.Error(c, appErrors.ErrConflict("今天已经签到过了"))
	case errors.Is(err, levelService.ErrUserNotFound):
		response.Error(c, appErrors.ErrNotFound("用户不存在"))
	case errors.Is(err, levelService.ErrLevelConfigNotFound):
		response.Error(c, appErrors.ErrInternal("等级配置缺失"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("等级系统操作失败"))
	}
}

func parseLevelID(c *gin.Context, label string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation(label+" ID 格式不正确"))
		return 0, false
	}
	return uint(id), true
}
