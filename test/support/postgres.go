// Package support wires real dependencies for integration tests.
package support

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/MaxCaribe/library-go/internal/application"
	"github.com/MaxCaribe/library-go/internal/infrastructure/dbmigrate"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/handlers"
	"github.com/MaxCaribe/library-go/internal/infrastructure/postgres"
	"github.com/MaxCaribe/library-go/internal/infrastructure/postgres/repositories"
)

const (
	databaseURLEnv  = "TEST_DATABASE_URL"
	containerImage  = "postgres:17-alpine"
	containerDBName = "library_test"
	startupTimeout  = time.Minute
)

// One container is shared by every test in a package: starting Postgres costs
// seconds, and truncating between tests is cheap. Testcontainers' reaper
// removes it when the test binary exits.
var (
	containerOnce sync.Once
	containerURL  string
	containerErr  error
)

// API is the real stack against a throwaway database. Pool is exposed so tests
// can assert on persisted state that no endpoint exposes yet.
type API struct {
	Mux  *http.ServeMux
	Pool *pgxpool.Pool
}

// NewBookAPI builds pool, migrations, repository, service and handler against a
// truncated database.
func NewBookAPI(t *testing.T) API {
	t.Helper()

	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	client, err := postgres.NewClient(ctx, postgres.Config{URL: DatabaseURL(t), MinConn: 1, MaxConn: 4}, logger)
	require.NoError(t, err)
	t.Cleanup(func() { client.Close() })

	migrate(ctx, t, client)

	_, err = client.Pool.Exec(ctx, "TRUNCATE books, changes")
	require.NoError(t, err)

	books := repositories.NewBookRepository(client.Pool)
	changes := repositories.NewChangeRepository(client.Pool)

	mux := http.NewServeMux()
	handlers.NewBookHandler(application.NewBookService(books, logger), logger).RegisterRoutes(mux)
	handlers.NewHistoryHandler(application.NewHistoryService(changes, books), logger).RegisterRoutes(mux)
	return API{Mux: mux, Pool: client.Pool}
}

// DatabaseURL prefers an already-running database so the suite can be pointed
// at one without Docker; otherwise it starts a container. When neither is
// available the test skips rather than failing, so `go test ./...` still works
// on a machine with no Docker.
func DatabaseURL(t *testing.T) string {
	t.Helper()

	if url := os.Getenv(databaseURLEnv); url != "" {
		return url
	}

	containerOnce.Do(startContainer)
	if containerErr != nil {
		t.Skipf("no test database: set %s, or start Docker so a container can be used (%v)", databaseURLEnv, containerErr)
	}
	return containerURL
}

func startContainer() {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, containerImage,
		tcpostgres.WithDatabase(containerDBName),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			// Postgres logs readiness twice: once for the bootstrap server used
			// to create the database, then once for the real one.
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(startupTimeout),
		),
	)
	if err != nil {
		containerErr = err
		return
	}

	containerURL, containerErr = container.ConnectionString(ctx, "sslmode=disable")
}

func migrate(ctx context.Context, t *testing.T, client *postgres.Client) {
	t.Helper()

	sqlDB := client.SQLDB()
	defer sqlDB.Close()

	migrator, err := dbmigrate.New(sqlDB)
	require.NoError(t, err)
	require.NoError(t, migrator.Up(ctx))
}
