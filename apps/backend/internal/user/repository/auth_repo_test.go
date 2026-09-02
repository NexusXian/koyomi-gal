package repository

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapUserWriteError(t *testing.T) {
	tests := []struct {
		constraint string
		want       error
	}{
		{constraint: usernameLowerUniqueIndex, want: ErrUsernameUniqueViolation},
		{constraint: emailLowerUniqueIndex, want: ErrEmailUniqueViolation},
		{constraint: legacyEmailUniqueIndex, want: ErrEmailUniqueViolation},
	}
	for _, tt := range tests {
		err := mapUserWriteError(&pgconn.PgError{Code: "23505", ConstraintName: tt.constraint})
		if !errors.Is(err, tt.want) {
			t.Fatalf("constraint %s: expected %v, got %v", tt.constraint, tt.want, err)
		}
	}
}
