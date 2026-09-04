// Package testutil provides PostgreSQL-backed helpers for integration tests.
// Tests that need a database are skipped unless RBAC_TEST_DSN is configured,
// for example:
//
//	RBAC_TEST_DSN="postgres://user:pass@localhost:5432/koyomi_rbac_test?sslmode=disable" go test ./...
package testutil

import (
	"context"
	"os"
	"strings"
	"testing"

	"backend/internal/migrations"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const dsnEnv = "RBAC_TEST_DSN"

// testDatabaseLockKey is the pg_advisory_lock key shared by all test binaries.
const testDatabaseLockKey = 727126

// PostgresDSN returns the configured integration test DSN, empty when unset.
func PostgresDSN() string {
	return strings.TrimSpace(os.Getenv(dsnEnv))
}

// SkipWithoutPostgres skips the test when RBAC_TEST_DSN is not configured.
func SkipWithoutPostgres(t *testing.T) {
	t.Helper()
	if PostgresDSN() == "" {
		t.Skipf("set %s to run this integration test", dsnEnv)
	}
}

// NewPostgres opens a gorm connection, applies all migrations, truncates the
// RBAC-related tables, and closes the connection when the test finishes.
// A session-pinned advisory lock serializes test binaries that share the same
// database, so `go test ./...` cannot interleave truncates between packages.
func NewPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := PostgresDSN()
	if dsn == "" {
		t.Fatalf("%s must be set", dsnEnv)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test postgres: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get test postgres pool: %v", err)
	}

	lockConn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("get advisory lock connection: %v", err)
	}
	if _, err := lockConn.ExecContext(context.Background(), "SELECT pg_advisory_lock($1)", testDatabaseLockKey); err != nil {
		t.Fatalf("acquire advisory lock: %v", err)
	}

	t.Cleanup(func() {
		_, _ = lockConn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", testDatabaseLockKey)
		_ = lockConn.Close()
		_ = sqlDB.Close()
	})

	if err := migrations.NewService(sqlDB).Up(context.Background()); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	TruncateTables(t, db)
	return db
}

// TruncateTables resets application tables between test cases.
func TruncateTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	err := db.Exec(
		"TRUNCATE TABLE external_match_candidates, import_jobs, galgame_external_sources, notifications, user_activities, user_privacy_settings, user_profiles, user_preferences, image_assets, articles, banners, background_presets, feedback, resource_reports, post_favorites, comment_likes, post_likes, comments, posts, resource_links, resources, user_galgames, galgame_favorites, galgame_ratings, galgame_gallery_images, galgame_contributions, galgame_tags, galgame_aliases, galgames, tags, developers, user_roles, role_permissions, permissions, roles, users RESTART IDENTITY",
	).Error
	if err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

// CreateUser inserts a users row and returns its id.
func CreateUser(t *testing.T, db *gorm.DB, username string) uint {
	t.Helper()
	var id uint
	err := db.Raw(`
INSERT INTO users (username, email, password_hash, is_banned, created_at, updated_at)
VALUES (?, ?, 'test-hash', false, NOW(), NOW())
RETURNING id
`, username, username+"@example.com").Scan(&id).Error
	if err != nil {
		t.Fatalf("create test user %s: %v", username, err)
	}
	return id
}
