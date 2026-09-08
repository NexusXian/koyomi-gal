package service

import "testing"

func TestVerificationDigestIsBoundToPurposeAndIdentifier(t *testing.T) {
	service := &VerificationService{secret: []byte("verification-test-secret")}
	code := "123456"
	register := service.codeDigest("user@example.com", VerificationPurposeRegister, code)
	reset := service.codeDigest("user@example.com", VerificationPurposePasswordReset, code)
	otherIdentifier := service.codeDigest("other@example.com", VerificationPurposeRegister, code)

	if register == reset {
		t.Fatal("expected verification purpose to change digest")
	}
	if register == otherIdentifier {
		t.Fatal("expected verification identifier to change digest")
	}
}

func TestNormalizeVerificationIdentifier(t *testing.T) {
	email, err := normalizeVerificationIdentifier(" User@Example.COM ", VerificationPurposePasswordReset)
	if err != nil || email != "user@example.com" {
		t.Fatalf("normalize email identifier = %q, %v", email, err)
	}
	userID, err := normalizeVerificationIdentifier("00123", VerificationPurposeChangePassword)
	if err != nil || userID != "123" {
		t.Fatalf("normalize user identifier = %q, %v", userID, err)
	}
}
