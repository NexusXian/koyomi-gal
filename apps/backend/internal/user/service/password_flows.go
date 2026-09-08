package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	dto "backend/internal/user/dto"
	"backend/internal/user/repository"
	"backend/pkg/bcrypt"
)

func (s *UserAuthService) RequestPasswordResetCode(ctx context.Context, email string, ip string) error {
	normalizedEmail, err := normalizeEmail(email)
	if err != nil {
		return ErrInvalidVerification
	}
	prepared, err := s.verificationService.PrepareCode(
		ctx,
		normalizedEmail,
		normalizedEmail,
		VerificationPurposePasswordReset,
		ip,
	)
	if err != nil {
		return err
	}

	user, err := s.authRepo.FindUserByEmail(ctx, normalizedEmail)
	if err != nil {
		return fmt.Errorf("find password reset user: %w", err)
	}
	if user == nil || user.IsBanned {
		return nil
	}
	return s.verificationService.EnqueuePrepared(ctx, prepared)
}

func (s *UserAuthService) VerifyPasswordResetCode(
	ctx context.Context,
	req *dto.PasswordForgotVerifyRequest,
) (string, error) {
	normalizedEmail, err := normalizeEmail(req.Email)
	if err != nil {
		return "", ErrInvalidVerificationCode
	}
	user, err := s.authRepo.FindUserByEmail(ctx, normalizedEmail)
	if err != nil {
		return "", fmt.Errorf("find password reset user: %w", err)
	}
	if user == nil || user.IsBanned {
		return "", ErrInvalidVerificationCode
	}

	valid, err := s.verificationService.ConsumeCode(
		ctx,
		normalizedEmail,
		VerificationPurposePasswordReset,
		req.Code,
	)
	if err != nil {
		return "", err
	}
	if !valid {
		return "", ErrInvalidVerificationCode
	}
	return s.verificationService.IssuePasswordResetTicket(ctx, user.ID)
}

func (s *UserAuthService) ResetPassword(
	ctx context.Context,
	req *dto.PasswordForgotResetRequest,
) error {
	if req.Password != req.ConfirmPassword {
		return ErrPasswordMismatch
	}
	if err := ValidatePassword(req.Password); err != nil {
		return err
	}

	userID, err := s.verificationService.ConsumePasswordResetTicket(ctx, req.ResetToken)
	if errors.Is(err, repository.ErrResetTicketNotFound) {
		return ErrInvalidResetToken
	}
	if err != nil {
		return err
	}

	user, err := s.authRepo.FindUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find password reset user: %w", err)
	}
	if user == nil {
		return ErrInvalidResetToken
	}
	if user.IsBanned {
		return ErrAccountBanned
	}
	if bcrypt.ComparePassword(user.PasswordHash, req.Password) == nil {
		return ErrSamePassword
	}

	passwordHash, err := bcrypt.HashPassword(req.Password)
	if err != nil {
		return fmt.Errorf("hash reset password: %w", err)
	}
	if err := s.authRepo.UpdatePassword(ctx, user.ID, passwordHash); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrInvalidResetToken
		}
		return fmt.Errorf("update reset password: %w", err)
	}
	return nil
}

func (s *UserAuthService) RequestPasswordChangeCode(
	ctx context.Context,
	userID uint,
	ip string,
) (string, error) {
	user, err := s.authRepo.FindUserByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("find password change user: %w", err)
	}
	if user == nil {
		return "", ErrUserNotFound
	}
	if user.IsBanned {
		return "", ErrAccountBanned
	}
	identifier := strconv.FormatUint(uint64(user.ID), 10)
	prepared, err := s.verificationService.PrepareCode(
		ctx,
		user.Email,
		identifier,
		VerificationPurposeChangePassword,
		ip,
	)
	if err != nil {
		return "", err
	}
	if err := s.verificationService.EnqueuePrepared(ctx, prepared); err != nil {
		return "", err
	}
	return MaskEmail(user.Email), nil
}

func (s *UserAuthService) ChangePassword(
	ctx context.Context,
	userID uint,
	req *dto.ChangePasswordRequest,
) error {
	user, err := s.authRepo.FindUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find password change user: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}
	if user.IsBanned {
		return ErrAccountBanned
	}

	valid, err := s.verificationService.ConsumeCode(
		ctx,
		strconv.FormatUint(uint64(user.ID), 10),
		VerificationPurposeChangePassword,
		req.Code,
	)
	if err != nil {
		return err
	}
	if !valid {
		return ErrInvalidVerificationCode
	}
	if req.NewPassword != req.ConfirmPassword {
		return ErrPasswordMismatch
	}
	if err := ValidatePassword(req.NewPassword); err != nil {
		return err
	}
	if bcrypt.ComparePassword(user.PasswordHash, req.NewPassword) == nil {
		return ErrSamePassword
	}

	passwordHash, err := bcrypt.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash changed password: %w", err)
	}
	if err := s.authRepo.UpdatePassword(ctx, user.ID, passwordHash); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("update changed password: %w", err)
	}
	return nil
}
