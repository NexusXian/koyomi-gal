package handler

import (
	"errors"
	"net/http"
	"time"

	"backend/internal/user/dto"
	"backend/internal/user/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	refreshTokenCookieName = "refresh_token"
	refreshTokenCookiePath = "/api/v1/auth"
)

type UserAuthHandler struct {
	userAuthService *service.UserAuthService
	refreshTokenTTL time.Duration
}

func NewUserAuthHandler(
	userAuthService *service.UserAuthService,
	refreshTokenTTL time.Duration,
) *UserAuthHandler {
	return &UserAuthHandler{
		userAuthService: userAuthService,
		refreshTokenTTL: refreshTokenTTL,
	}
}

// Register godoc
// @Summary      用户注册
// @Description  使用邮箱验证码创建新账号
// @ID           register
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.UserRegisterRequest true "注册请求"
// @Success      200 {object} response.MessageResponse "用户注册成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      500 {object} response.ErrorResponse "用户注册失败"
// @Router       /api/v1/auth/register [post]
func (h *UserAuthHandler) Register(c *gin.Context) {
	var req dto.UserRegisterRequest
	ctx := c.Request.Context()
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}

	if err := h.userAuthService.UserRegister(ctx, &req); err != nil {
		response.Error(c, appErrors.ErrInternal("用户注册失败"))
		return
	}
	response.OkWithMsg(c, "用户注册成功")
}

// Login godoc
// @Summary      用户登录
// @Description  校验邮箱和密码，返回 Access Token 与用户信息，并通过 Set-Cookie 写入 HttpOnly 的 Refresh Token
// @ID           login
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.UserLoginRequest true "登录请求"
// @Success      200 {object} dto.AuthSessionResponse "登录成功"
// @Failure      400 {object} response.ErrorResponse "邮箱或密码格式不正确"
// @Failure      401 {object} response.ErrorResponse "邮箱或密码错误"
// @Failure      403 {object} response.ErrorResponse "账号已封禁"
// @Failure      500 {object} response.ErrorResponse "认证服务异常"
// @Header       200 {string} Set-Cookie "HttpOnly; Secure; SameSite=Lax; Path=/api/v1/auth 的 refresh_token Cookie"
// @Router       /api/v1/auth/login [post]
func (h *UserAuthHandler) Login(c *gin.Context) {
	var req dto.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("邮箱或密码格式不正确"))
		return
	}

	session, refreshToken, err := h.userAuthService.UserLogin(c.Request.Context(), &req)
	if err != nil {
		h.respondAuthError(c, err, "login")
		return
	}

	h.setRefreshTokenCookie(c, refreshToken)
	response.Ok(c, session)
}

// Refresh godoc
// @Summary      刷新登录会话
// @Description  使用 refresh_token Cookie 轮换 Refresh Token，返回新的 Access Token 与用户信息
// @ID           refreshSession
// @Tags         auth
// @Produce      json
// @Success      200 {object} dto.AuthSessionResponse "刷新成功"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "账号已封禁"
// @Failure      500 {object} response.ErrorResponse "认证服务异常"
// @Header       200 {string} Set-Cookie "轮换后的 refresh_token Cookie"
// @Router       /api/v1/auth/refresh [post]
func (h *UserAuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(refreshTokenCookieName)
	if err != nil {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}

	session, replacementToken, err := h.userAuthService.RefreshSession(
		c.Request.Context(),
		refreshToken,
	)
	if err != nil {
		h.respondAuthError(c, err, "refresh session")
		return
	}

	h.setRefreshTokenCookie(c, replacementToken)
	response.Ok(c, session)
}

// Logout godoc
// @Summary      退出登录
// @Description  撤销刷新会话并清除 refresh_token Cookie
// @ID           logout
// @Tags         auth
// @Produce      json
// @Success      200 {object} response.MessageResponse "退出成功"
// @Failure      400 {object} response.ErrorResponse "Refresh Token 格式不正确"
// @Failure      500 {object} response.ErrorResponse "退出登录失败"
// @Header       200 {string} Set-Cookie "已过期的 refresh_token Cookie"
// @Router       /api/v1/auth/logout [post]
func (h *UserAuthHandler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie(refreshTokenCookieName)
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		response.Error(c, appErrors.ErrBadRequest("Refresh Token 格式不正确"))
		return
	}

	if err := h.userAuthService.Logout(c.Request.Context(), refreshToken); err != nil {
		logger.Error("logout", zap.Error(err))
		response.Error(c, appErrors.ErrInternal("退出登录失败"))
		return
	}

	h.clearRefreshTokenCookie(c)
	response.OkWithMsg(c, "success")
}

func (h *UserAuthHandler) respondAuthError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		response.Error(c, appErrors.ErrUnauthorized("邮箱或密码错误"))
	case errors.Is(err, service.ErrInvalidRefreshToken):
		response.Error(c, appErrors.ErrAuthExpired())
	case errors.Is(err, service.ErrAccountBanned):
		response.Error(c, appErrors.ErrAccountBanned())
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("认证服务异常"))
	}
}

func (h *UserAuthHandler) setRefreshTokenCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    token,
		Path:     refreshTokenCookiePath,
		MaxAge:   int(h.refreshTokenTTL / time.Second),
		Expires:  time.Now().Add(h.refreshTokenTTL),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *UserAuthHandler) clearRefreshTokenCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshTokenCookieName,
		Path:     refreshTokenCookiePath,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
