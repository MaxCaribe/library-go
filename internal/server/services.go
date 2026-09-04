package server

import (
	"log/slog"

	"github.com/MaxCaribe/library-go/internal/application"
	"github.com/MaxCaribe/library-go/internal/config"
)

// appServices holds the use-case services. Each takes its dependencies from
// infra.repos as interfaces declared in the application layer.
type appServices struct {
	book *application.BookService
}

func newServices(infra *appInfra, _ config.Config, logger *slog.Logger) (*appServices, error) {
	return &appServices{
		book: application.NewBookService(infra.repos.book, logger),
	}, nil
}
