package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	userService "backend/internal/user/service"
)

type Mailer interface {
	Send(context.Context, string, string, string) error
}

type EmailService struct {
	mailer Mailer
}

func NewEmailService(mailer Mailer) *EmailService {
	return &EmailService{mailer: mailer}
}

func (s *EmailService) SendVerificationCode(
	ctx context.Context,
	email string,
	purpose string,
	code string,
	expiresAt time.Time,
) error {
	var purposeName string
	switch purpose {
	case userService.VerificationPurposeRegister:
		purposeName = "注册"
	case userService.VerificationPurposePasswordReset:
		purposeName = "重置密码"
	default:
		return errors.New("unsupported verification purpose")
	}

	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		return errors.New("verification code expired")
	}
	minutes := int((remaining + time.Minute - 1) / time.Minute)
	subject := fmt.Sprintf("Koyomi Gal %s验证码", purposeName)
	body := fmt.Sprintf(
		"您的%s验证码是：%s\n\n验证码将在 %d 分钟内过期。若非本人操作，请忽略此邮件。",
		purposeName,
		code,
		minutes,
	)
	return s.mailer.Send(ctx, email, subject, body)
}
