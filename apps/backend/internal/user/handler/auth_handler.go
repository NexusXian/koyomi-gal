package handler

import (
	"errors"
	"net/http"
	"time"

	"backend/internal/middleware"
	dto "backend/internal/user/dto"
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
// @Param        request body userdto.UserRegisterRequest true "注册请求"
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
		switch {
		case errors.Is(err, service.ErrInvalidUsername):
			response.Error(c, appErrors.ErrValidation("用户名不能为空"))
		case errors.Is(err, service.ErrUsernameExists):
			response.Error(c, appErrors.ErrValidation("用户名已存在"))
		case errors.Is(err, service.ErrEmailExists):
			response.Error(c, appErrors.ErrValidation("邮箱已存在"))
		case errors.Is(err, service.ErrPasswordMismatch):
			response.Error(c, appErrors.ErrValidation("两次输入的密码不一致"))
		case errors.Is(err, service.ErrInvalidPassword):
			response.Error(c, appErrors.ErrValidation("密码长度必须为 8 到 72 字节"))
		case errors.Is(err, service.ErrInvalidVerificationCode):
			response.Error(c, appErrors.ErrValidation("验证码错误或已过期"))
		default:
			logger.Error("register user", zap.Error(err))
			response.Error(c, appErrors.ErrInternal("用户注册失败"))
		}
		return
	}
	response.OkWithMsg(c, "用户注册成功")
}

// Login godoc
// @Summary      用户登录
// @Description  校验邮箱或用户名和密码，返回 Access Token 与用户信息，并通过 Set-Cookie 写入 HttpOnly 的 Refresh Token
// @ID           login
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body userdto.UserLoginRequest true "登录请求"
// @Success      200 {object} userdto.AuthSessionResponse "登录成功"
// @Failure      400 {object} response.ErrorResponse "账号或密码格式不正确"
// @Failure      401 {object} response.ErrorResponse "账号或密码错误"
// @Failure      403 {object} response.ErrorResponse "账号已封禁"
// @Failure      500 {object} response.ErrorResponse "认证服务异常"
// @Header       200 {string} Set-Cookie "HttpOnly; Secure; SameSite=Lax; Path=/api/v1/auth 的 refresh_token Cookie"
// @Router       /api/v1/auth/login [post]
func (h *UserAuthHandler) Login(c *gin.Context) {
	var req dto.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("账号或密码格式不正确"))
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
// @Success      200 {object} userdto.AuthSessionResponse "刷新成功"
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

// ForgotPasswordCode godoc
// @Summary      发送忘记密码验证码
// @Description  无论邮箱是否存在均返回相同结果，202 表示请求已处理
// @ID           forgotPasswordCode
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body userdto.PasswordForgotCodeRequest true "忘记密码验证码请求"
// @Success      202 {object} response.MessageResponse "如果该邮箱已注册，验证码将发送到邮箱"
// @Failure      400 {object} response.ErrorResponse "邮箱格式不正确"
// @Failure      429 {object} response.ErrorResponse "请求过于频繁"
// @Failure      500 {object} response.ErrorResponse "验证码发送任务创建失败"
// @Router       /api/v1/auth/password/forgot/code [post]
func (h *UserAuthHandler) ForgotPasswordCode(c *gin.Context) {
	var req dto.PasswordForgotCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("邮箱格式不正确"))
		return
	}
	if err := h.userAuthService.RequestPasswordResetCode(c.Request.Context(), req.Email, c.ClientIP()); err != nil {
		h.respondVerificationSendError(c, err, "request password reset code")
		return
	}
	response.AcceptedWithMsg(c, "如果该邮箱已注册，验证码将发送到邮箱")
}

// ForgotPasswordVerify godoc
// @Summary      验证忘记密码验证码
// @Description  验证成功后签发十分钟有效的一次性重置凭证
// @ID           forgotPasswordVerify
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body userdto.PasswordForgotVerifyRequest true "验证码验证请求"
// @Success      200 {object} userdto.PasswordResetTokenResponse "验证成功"
// @Failure      400 {object} response.ErrorResponse "验证码错误或已过期"
// @Failure      500 {object} response.ErrorResponse "验证码验证失败"
// @Router       /api/v1/auth/password/forgot/verify [post]
func (h *UserAuthHandler) ForgotPasswordVerify(c *gin.Context) {
	var req dto.PasswordForgotVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	resetToken, err := h.userAuthService.VerifyPasswordResetCode(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidVerificationCode) {
			response.Error(c, appErrors.ErrValidation("验证码错误或已过期"))
			return
		}
		logger.Error("verify password reset code", zap.Error(err))
		response.Error(c, appErrors.ErrInternal("验证码验证失败"))
		return
	}
	response.OkWithDataAndMsg(c, dto.PasswordResetTokenData{ResetToken: resetToken}, "邮箱验证成功")
}

// ForgotPasswordReset godoc
// @Summary      重置忘记的密码
// @Description  使用一次性重置凭证设置新密码并使旧会话失效
// @ID           forgotPasswordReset
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body userdto.PasswordForgotResetRequest true "重置密码请求"
// @Success      200 {object} response.MessageResponse "密码重置成功"
// @Failure      400 {object} response.ErrorResponse "重置凭证或密码无效"
// @Failure      403 {object} response.ErrorResponse "账号已封禁"
// @Failure      500 {object} response.ErrorResponse "密码重置失败"
// @Router       /api/v1/auth/password/forgot/reset [post]
func (h *UserAuthHandler) ForgotPasswordReset(c *gin.Context) {
	var req dto.PasswordForgotResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	if err := h.userAuthService.ResetPassword(c.Request.Context(), &req); err != nil {
		h.respondPasswordError(c, err, "reset password", "密码重置失败")
		return
	}
	h.clearRefreshTokenCookie(c)
	response.OkWithMsg(c, "密码重置成功")
}

// PasswordChangeCode godoc
// @Summary      发送修改密码验证码
// @Description  向当前用户绑定邮箱发送修改密码验证码
// @ID           passwordChangeCode
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} userdto.PasswordCodeResponse "验证码已发送"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "账号已封禁"
// @Failure      429 {object} response.ErrorResponse "请求过于频繁"
// @Failure      500 {object} response.ErrorResponse "验证码发送任务创建失败"
// @Router       /api/v1/users/me/password/code [post]
func (h *UserAuthHandler) PasswordChangeCode(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	maskedEmail, err := h.userAuthService.RequestPasswordChangeCode(c.Request.Context(), userID, c.ClientIP())
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.Error(c, appErrors.ErrAuthExpired())
			return
		}
		if errors.Is(err, service.ErrAccountBanned) {
			response.Error(c, appErrors.ErrAccountBanned())
			return
		}
		h.respondVerificationSendError(c, err, "request password change code")
		return
	}
	response.OkWithDataAndMsg(c, dto.PasswordCodeData{Email: maskedEmail}, "验证码已发送")
}

// ChangePassword godoc
// @Summary      修改当前用户密码
// @Description  使用绑定当前用户的验证码设置新密码并使旧会话失效
// @ID           changePassword
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body userdto.ChangePasswordRequest true "修改密码请求"
// @Success      200 {object} response.MessageResponse "密码修改成功"
// @Failure      400 {object} response.ErrorResponse "验证码或密码无效"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "账号已封禁"
// @Failure      500 {object} response.ErrorResponse "密码修改失败"
// @Router       /api/v1/users/me/password [put]
func (h *UserAuthHandler) ChangePassword(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	if err := h.userAuthService.ChangePassword(c.Request.Context(), userID, &req); err != nil {
		h.respondPasswordError(c, err, "change password", "密码修改失败")
		return
	}
	h.clearRefreshTokenCookie(c)
	response.OkWithMsg(c, "密码修改成功")
}

func (h *UserAuthHandler) respondVerificationSendError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrInvalidVerification):
		response.Error(c, appErrors.ErrValidation("邮箱格式不正确"))
	case errors.Is(err, service.ErrVerificationCooldown):
		response.Error(c, appErrors.ErrTooManyRequests("验证码发送过于频繁，请稍后重试"))
	case errors.Is(err, service.ErrVerificationRateLimit):
		response.Error(c, appErrors.ErrTooManyRequests("请求过于频繁，请稍后重试"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("验证码发送任务创建失败"))
	}
}

func (h *UserAuthHandler) respondPasswordError(c *gin.Context, err error, operation string, internalMessage string) {
	switch {
	case errors.Is(err, service.ErrInvalidResetToken):
		response.Error(c, appErrors.ErrValidation("重置凭证无效或已过期"))
	case errors.Is(err, service.ErrInvalidVerificationCode):
		response.Error(c, appErrors.ErrValidation("验证码错误或已过期"))
	case errors.Is(err, service.ErrPasswordMismatch):
		response.Error(c, appErrors.ErrValidation("两次输入的密码不一致"))
	case errors.Is(err, service.ErrInvalidPassword):
		response.Error(c, appErrors.ErrValidation("密码长度必须为 8 到 72 字节"))
	case errors.Is(err, service.ErrSamePassword):
		response.Error(c, appErrors.ErrValidation("新密码不能与原密码相同"))
	case errors.Is(err, service.ErrUserNotFound):
		response.Error(c, appErrors.ErrAuthExpired())
	case errors.Is(err, service.ErrAccountBanned):
		response.Error(c, appErrors.ErrAccountBanned())
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal(internalMessage))
	}
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
