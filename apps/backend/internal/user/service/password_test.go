package service

import (
	"errors"
	"strings"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{name: "seven bytes", password: "1234567"},
		{name: "eight bytes", password: "12345678", valid: true},
		{name: "seventy two bytes", password: strings.Repeat("a", 72), valid: true},
		{name: "seventy three bytes", password: strings.Repeat("a", 73)},
		{name: "multibyte length uses bytes", password: "密码密码密码", valid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.valid && err != nil {
				t.Fatalf("expected valid password, got %v", err)
			}
			if !tt.valid && !errors.Is(err, ErrInvalidPassword) {
				t.Fatalf("expected ErrInvalidPassword, got %v", err)
			}
		})
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		email string
		want  string
	}{
		{email: "nexus@example.com", want: "ne***@example.com"},
		{email: "abc@example.com", want: "ab***@example.com"},
		{email: "ab@example.com", want: "a***@example.com"},
		{email: "a@example.com", want: "a***@example.com"},
		{email: "invalid", want: "***"},
	}

	for _, tt := range tests {
		if got := MaskEmail(tt.email); got != tt.want {
			t.Errorf("MaskEmail(%q) = %q, want %q", tt.email, got, tt.want)
		}
	}
}
