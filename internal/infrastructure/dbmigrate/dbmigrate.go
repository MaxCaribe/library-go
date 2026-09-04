package dbmigrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3"

	"github.com/MaxCaribe/library-go/migrations"
)

const DefaultMigrationsDir = "migrations"

type Migrator struct {
	provider *goose.Provider
}

// New builds a migrator over an already-connected database. goose runs with
// out-of-order migrations allowed: it records every applied version, so a
// branch merged after a higher version was applied still runs instead of being
// silently skipped.
func New(db *sql.DB) (*Migrator, error) {
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS,
		goose.WithAllowOutofOrder(true),
	)
	if err != nil {
		return nil, fmt.Errorf("dbmigrate: create provider: %w", err)
	}
	return &Migrator{provider: provider}, nil
}

func (m *Migrator) Up(ctx context.Context) error {
	_, err := m.provider.Up(ctx)
	return err
}

func (m *Migrator) Down(ctx context.Context, steps int) error {
	if steps < 0 {
		steps = -steps
	}

	for range steps {
		if _, err := m.provider.Down(ctx); err != nil {
			if errors.Is(err, goose.ErrNoNextVersion) {
				return nil
			}
			return err
		}
	}
	return nil
}

func (m *Migrator) Status(ctx context.Context) ([]*goose.MigrationStatus, error) {
	return m.provider.Status(ctx)
}

func Create(name, dir string) error {
	return goose.Create(nil, dir, name, "sql")
}
