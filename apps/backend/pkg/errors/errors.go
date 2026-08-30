package errors

import "fmt"

type AppError struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func New(code int, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

const (
	CodeOK             = 0
	CodeAuth           = 205
	CodeBiz            = 233
	CodeBanned         = 234
	CodeReauthRequired = 235
)

func ErrUnauthorized(msg string) *AppError {
	return New(CodeAuth, msg, 401)
}

func ErrAuthExpired() *AppError {
	return New(CodeAuth, "用户登录失效", 401)
}

func ErrAccountBanned() *AppError {
	return New(CodeBanned, "账号已封禁", 403)
}

func ErrReauthRequired(msg string) *AppError {
	return New(CodeReauthRequired, msg, 403)
}

func ErrForbidden(msg string) *AppError {
	return New(CodeBiz, msg, 403)
}

func ErrBadRequest(msg string) *AppError {
	return New(CodeBiz, msg, 400)
}

func ErrNotFound(msg string) *AppError {
	return New(CodeBiz, msg, 404)
}

func ErrInternal(msg string) *AppError {
	return New(CodeBiz, msg, 500)
}

func ErrTooManyRequests(msg string) *AppError {
	return New(CodeBiz, msg, 429)
}

func ErrValidation(msg string) *AppError {
	return New(CodeBiz, msg, 400)
}
