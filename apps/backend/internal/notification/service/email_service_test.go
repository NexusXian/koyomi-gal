package service

import (
	"context"
	"testing"
	"time"

	userService "backend/internal/user/service"
)

type captureMailer struct {
	subject string
}

func (m *captureMailer) Send(_ context.Context, _ string, subject string, _ string) error {
	m.subject = subject
	return nil
}

func TestPasswordVerificationEmailSubjects(t *testing.T) {
	tests := []struct {
		purpose string
		want    string
	}{
		{purpose: userService.VerificationPurposePasswordReset, want: "Koyomi Gal - 重置密码验证码"},
		{purpose: userService.VerificationPurposeChangePassword, want: "Koyomi Gal - 修改密码验证码"},
	}
	for _, tt := range tests {
		t.Run(tt.purpose, func(t *testing.T) {
			mailer := &captureMailer{}
			service := NewEmailService(mailer, "https://example.com")
			if err := service.SendVerificationCode(
				context.Background(),
				"user@example.com",
				tt.purpose,
				"123456",
				time.Now().Add(10*time.Minute),
			); err != nil {
				t.Fatalf("send verification email: %v", err)
			}
			if mailer.subject != tt.want {
				t.Fatalf("subject = %q, want %q", mailer.subject, tt.want)
			}
		})
	}
}
