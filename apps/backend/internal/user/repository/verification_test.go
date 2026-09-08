package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestVerificationCodePurposeAttemptsAndConsume(t *testing.T) {
	repo := newTestVerificationRepository(t)
	ctx := context.Background()

	reserveTestCode(t, repo, "user@example.com", "register", "register-request", "register-digest", time.Minute)
	reserveTestCode(t, repo, "user@example.com", "password_reset", "reset-request", "reset-digest", time.Minute)

	valid, err := repo.ConsumeVerificationCode(ctx, "user@example.com", "password_reset", "register-digest", 5)
	if err != nil || valid {
		t.Fatalf("cross-purpose digest should be invalid: valid=%v err=%v", valid, err)
	}
	valid, err = repo.ConsumeVerificationCode(ctx, "user@example.com", "register", "register-digest", 5)
	if err != nil || !valid {
		t.Fatalf("register code should remain valid: valid=%v err=%v", valid, err)
	}
	valid, err = repo.ConsumeVerificationCode(ctx, "user@example.com", "password_reset", "reset-digest", 5)
	if err != nil || !valid {
		t.Fatalf("reset code should remain valid: valid=%v err=%v", valid, err)
	}
	valid, err = repo.ConsumeVerificationCode(ctx, "user@example.com", "register", "register-digest", 5)
	if err != nil || valid {
		t.Fatalf("consumed code should be invalid: valid=%v err=%v", valid, err)
	}

	reserveTestCode(t, repo, "attempts@example.com", "password_reset", "attempts-request", "attempts-digest", time.Minute)
	for attempt := 1; attempt <= 5; attempt++ {
		valid, err = repo.ConsumeVerificationCode(ctx, "attempts@example.com", "password_reset", "wrong", 5)
		if err != nil || valid {
			t.Fatalf("attempt %d should be invalid: valid=%v err=%v", attempt, valid, err)
		}
	}
	valid, err = repo.ConsumeVerificationCode(ctx, "attempts@example.com", "password_reset", "attempts-digest", 5)
	if err != nil || valid {
		t.Fatalf("code should be deleted after fifth wrong attempt: valid=%v err=%v", valid, err)
	}
}

func TestVerificationCodeOverwriteResetsAttempts(t *testing.T) {
	repo := newTestVerificationRepository(t)
	ctx := context.Background()
	identifier := "overwrite@example.com"

	reserveTestCode(t, repo, identifier, "password_reset", "first", "first-digest", time.Millisecond)
	for range 4 {
		if valid, err := repo.ConsumeVerificationCode(ctx, identifier, "password_reset", "wrong", 5); err != nil || valid {
			t.Fatalf("wrong attempt: valid=%v err=%v", valid, err)
		}
	}
	time.Sleep(3 * time.Millisecond)
	reserveTestCode(t, repo, identifier, "password_reset", "second", "second-digest", time.Minute)
	if valid, err := repo.ConsumeVerificationCode(ctx, identifier, "password_reset", "wrong", 5); err != nil || valid {
		t.Fatalf("new-code wrong attempt: valid=%v err=%v", valid, err)
	}
	if valid, err := repo.ConsumeVerificationCode(ctx, identifier, "password_reset", "second-digest", 5); err != nil || !valid {
		t.Fatalf("new code should survive reset attempt count: valid=%v err=%v", valid, err)
	}
}

func TestVerificationIdentifierLimitSurvivesCancel(t *testing.T) {
	repo := newTestVerificationRepository(t)
	ctx := context.Background()
	identifier := "limited@example.com"

	for index := range 10 {
		requestID := fmt.Sprintf("request-%d", index)
		reservation, err := repo.ReserveVerificationCode(
			ctx, identifier, "register", "127.0.0.1", requestID, "digest",
			time.Minute, time.Minute, time.Hour, 100, 24*time.Hour, 10,
		)
		if err != nil {
			t.Fatalf("reserve %d: %v", index+1, err)
		}
		if err := repo.CancelVerificationCode(ctx, reservation); err != nil {
			t.Fatalf("cancel %d: %v", index+1, err)
		}
	}
	_, err := repo.ReserveVerificationCode(
		ctx, identifier, "register", "127.0.0.1", "request-11", "digest",
		time.Minute, time.Minute, time.Hour, 100, 24*time.Hour, 10,
	)
	if !errors.Is(err, ErrVerificationRateLimit) {
		t.Fatalf("expected identifier rate limit after canceled reservations, got %v", err)
	}
}

func TestPasswordResetTicketIsHashedAndOneUse(t *testing.T) {
	repo := newTestVerificationRepository(t)
	ctx := context.Background()
	token := "opaque-reset-token"

	if key := passwordResetTicketKey(token); key == "password_reset_ticket:"+token {
		t.Fatal("ticket key must not contain the raw token")
	}
	if err := repo.CreatePasswordResetTicket(ctx, token, 42, 10*time.Minute); err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	userID, err := repo.ConsumePasswordResetTicket(ctx, token)
	if err != nil || userID != 42 {
		t.Fatalf("consume ticket: userID=%d err=%v", userID, err)
	}
	if _, err := repo.ConsumePasswordResetTicket(ctx, token); !errors.Is(err, ErrResetTicketNotFound) {
		t.Fatalf("expected consumed ticket to be unavailable, got %v", err)
	}
}

func reserveTestCode(
	t *testing.T,
	repo *VerificationRepository,
	identifier string,
	purpose string,
	requestID string,
	digest string,
	resendInterval time.Duration,
) {
	t.Helper()
	_, err := repo.ReserveVerificationCode(
		context.Background(), identifier, purpose, "127.0.0.1", requestID, digest,
		time.Minute, resendInterval, time.Hour, 100, 24*time.Hour, 100,
	)
	if err != nil {
		t.Fatalf("reserve verification code: %v", err)
	}
}

func newTestVerificationRepository(t *testing.T) *VerificationRepository {
	t.Helper()
	serverPath, err := exec.LookPath("redis-server")
	if err != nil {
		t.Skip("redis-server is required for verification repository tests")
	}
	socket := fmt.Sprintf("/tmp/kg-redis-%d-%d.sock", os.Getpid(), time.Now().UnixNano())
	t.Cleanup(func() { _ = os.Remove(socket) })
	var output bytes.Buffer
	command := exec.Command(
		serverPath,
		"--port", "0",
		"--unixsocket", socket,
		"--unixsocketperm", "700",
		"--save", "",
		"--appendonly", "no",
	)
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Start(); err != nil {
		t.Skipf("start redis-server: %v", err)
	}
	t.Cleanup(func() {
		_ = command.Process.Signal(os.Interrupt)
		_ = command.Wait()
	})

	client := redis.NewClient(&redis.Options{Network: "unix", Addr: socket})
	t.Cleanup(func() { _ = client.Close() })
	deadline := time.Now().Add(3 * time.Second)
	for {
		if err := client.Ping(context.Background()).Err(); err == nil {
			return NewVerificationRepository(client)
		}
		if time.Now().After(deadline) {
			t.Fatalf("redis-server did not become ready: %s", output.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
}
