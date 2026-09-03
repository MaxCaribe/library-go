package server

import (
	"log/slog"

	"github.com/MaxCaribe/library-go/internal/config"
)

// appInfra holds the outbound adapters: repositories, caches, external API clients.
// Construct them in newInfra and expose them as fields; nothing else in the tree
// names a concrete adapter, so swapping an implementation is a change here alone.
type appInfra struct {
	repos appRepos
}

type appRepos struct{}

func newInfra(_ config.Config, _ *slog.Logger) (*appInfra, error) {
	return &appInfra{repos: appRepos{}}, nil
}
