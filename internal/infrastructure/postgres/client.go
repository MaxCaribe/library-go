package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

const (
	pingRetries     = 10
	pingInitialWait = 200 * time.Millisecond
	pingMaxWait     = 2 * time.Second
	connMaxIdleTime = 3 * time.Minute
)

type Config struct {
	URL     string
	MinConn int
	MaxConn int
}

type Client struct {
	Pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewClient(ctx context.Context, cfg Config, logger *slog.Logger) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("postgres: DATABASE_URL is required")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse url: %w", err)
	}

	poolConfig.MinConns = int32(cfg.MinConn)
	poolConfig.MaxConns = int32(cfg.MaxConn)
	poolConfig.MaxConnIdleTime = connMaxIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	client := &Client{Pool: pool, logger: logger.With("package", "postgres")}

	if err := client.pingWithRetry(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	client.logger.InfoContext(ctx, "connected to postgres", "max_conn", cfg.MaxConn, "min_conn", cfg.MinConn)
	return client, nil
}

func (c *Client) Close() error {
	c.Pool.Close()
	return nil
}

// SQLDB exposes the pool as a *sql.DB. goose speaks database/sql, so migrations
// borrow the same pool rather than opening a second one. The returned handle
// keeps zero idle connections of its own, so it cannot starve pgx users.
func (c *Client) SQLDB() *sql.DB {
	return stdlib.OpenDBFromPool(c.Pool)
}

func (c *Client) pingWithRetry(ctx context.Context) error {
	wait := pingInitialWait

	var err error
	for attempt := 1; attempt <= pingRetries; attempt++ {
		if err = c.Pool.Ping(ctx); err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		c.logger.WarnContext(ctx, "postgres not ready, retrying", "attempt", attempt, "wait", wait.String())
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
		wait = min(wait*2, pingMaxWait)
	}
	return fmt.Errorf("postgres: unreachable after %d attempts: %w", pingRetries, err)
}
