package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/alecthomas/kingpin/v2"

	"github.com/MaxCaribe/library-go/internal/config"
	"github.com/MaxCaribe/library-go/internal/infrastructure/dbmigrate"
	"github.com/MaxCaribe/library-go/internal/infrastructure/postgres"
	"github.com/MaxCaribe/library-go/pkg/logging"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.InitConfig()

	upCmd := kingpin.Command("up", "Apply all pending migrations")
	downCmd := kingpin.Command("down", "Roll migrations back")
	downSteps := downCmd.Flag("steps", "Number of migrations to roll back").Default("1").Int()
	statusCmd := kingpin.Command("status", "Show which migrations have been applied")
	createCmd := kingpin.Command("create", "Scaffold a new migration file")
	createName := createCmd.Arg("name", "Migration description, e.g. add_book_isbn").Required().String()
	createDir := createCmd.Flag("dir", "Migrations directory").Default(dbmigrate.DefaultMigrationsDir).String()

	command := kingpin.MustParse(kingpin.CommandLine.Parse(os.Args[1:]))
	logger := logging.CreateRootSlogger(!cfg.Debug).With("package", "migrate")

	// Scaffolding only touches the filesystem, so it must work with no database.
	if command == createCmd.FullCommand() {
		if err := dbmigrate.Create(*createName, *createDir); err != nil {
			return err
		}
		return nil
	}

	ctx := context.Background()

	client, err := postgres.NewClient(ctx, postgres.Config{
		URL:     cfg.Database.URL,
		MinConn: cfg.Database.MinConn,
		MaxConn: cfg.Database.MaxConn,
	}, logger)
	if err != nil {
		return err
	}
	defer client.Close()

	sqlDB := client.SQLDB()
	defer sqlDB.Close()

	migrator, err := dbmigrate.New(sqlDB)
	if err != nil {
		return err
	}

	switch command {
	case upCmd.FullCommand():
		if err := migrator.Up(ctx); err != nil {
			return err
		}
		logger.InfoContext(ctx, "migrations applied")
	case downCmd.FullCommand():
		if err := migrator.Down(ctx, *downSteps); err != nil {
			return err
		}
		logger.InfoContext(ctx, "migrations rolled back", "steps", *downSteps)
	case statusCmd.FullCommand():
		return printStatus(ctx, migrator, logger)
	}
	return nil
}

func printStatus(ctx context.Context, migrator *dbmigrate.Migrator, logger *slog.Logger) error {
	statuses, err := migrator.Status(ctx)
	if err != nil {
		return err
	}

	for _, status := range statuses {
		logger.InfoContext(ctx, "migration",
			"version", status.Source.Version,
			"path", status.Source.Path,
			"state", status.State,
			"applied_at", status.AppliedAt,
		)
	}
	return nil
}
