package server

import (
	"context"
	"log/slog"

	"github.com/MaxCaribe/library-go/internal/config"
	"github.com/MaxCaribe/library-go/internal/infrastructure/postgres"
)

// appInfra holds the outbound adapters: the database pool, and the
// repositories built on it. Construct them in newInfra; nothing else in the
// tree names a concrete adapter, so swapping an implementation stays local.
type appInfra struct {
	db    *postgres.Client
	repos appRepos
}

type appRepos struct{}

func newInfra(ctx context.Context, cfg config.Config, logger *slog.Logger) (*appInfra, error) {
	db, err := postgres.NewClient(ctx, postgres.Config{
		URL:     cfg.Database.URL,
		MinConn: cfg.Database.MinConn,
		MaxConn: cfg.Database.MaxConn,
	}, logger)
	if err != nil {
		return nil, err
	}

	return &appInfra{db: db, repos: appRepos{}}, nil
}

func (a *appInfra) Close() error {
	return a.db.Close()
}
