package service

import (
	"strings"
)

func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 72 {
		return ErrInvalidPassword
	}
	return nil
}

func MaskEmail(email string) string {
	local, domain, found := strings.Cut(email, "@")
	if !found || local == "" || domain == "" {
		return "***"
	}
	runes := []rune(local)
	visible := 2
	if len(runes) < 3 {
		visible = 1
	}
	return string(runes[:visible]) + "***@" + domain
}
