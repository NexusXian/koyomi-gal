package migrations

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var migrationFiles embed.FS

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Up(ctx context.Context) (err error) {
	sourceDriver, err := iofs.New(migrationFiles, ".")
	if err != nil {
		return fmt.Errorf("open migration files: %w", err)
	}

	connection, err := s.db.Conn(ctx)
	if err != nil {
		_ = sourceDriver.Close()
		return fmt.Errorf("get migration connection: %w", err)
	}
	databaseDriver, err := postgres.WithConnection(ctx, connection, &postgres.Config{})
	if err != nil {
		_ = connection.Close()
		_ = sourceDriver.Close()
		return fmt.Errorf("initialize migration database driver: %w", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", databaseDriver)
	if err != nil {
		_ = databaseDriver.Close()
		_ = sourceDriver.Close()
		return fmt.Errorf("initialize migration service: %w", err)
	}
	defer func() {
		sourceErr, databaseErr := migrator.Close()
		if sourceErr != nil {
			err = errors.Join(err, fmt.Errorf("close migration source: %w", sourceErr))
		}
		if databaseErr != nil {
			err = errors.Join(err, fmt.Errorf("close migration database driver: %w", databaseErr))
		}
	}()

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
