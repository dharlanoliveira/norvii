package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore verifies the canonical store through the production pgx driver.
type PostgresStore struct {
	connection *pgx.Conn
}

// OpenPostgresPool opens the bounded caller-owned pool used by online repositories.
func OpenPostgresPool(ctx context.Context, config PostgresConfig) (*pgxpool.Pool, error) {
	connectionConfig, err := newPGXConfig(config)
	if err != nil {
		return nil, err
	}
	poolConfig, err := pgxpool.ParseConfig("sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool configuration: %w", err)
	}
	poolConfig.ConnConfig = connectionConfig
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("connect PostgreSQL pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("verify PostgreSQL pool: %w", err)
	}
	return pool, nil
}

// OpenPostgresStore opens an authenticated canonical-store connection.
func OpenPostgresStore(ctx context.Context, config PostgresConfig) (*PostgresStore, error) {
	connectionConfig, err := newPGXConfig(config)
	if err != nil {
		return nil, err
	}
	connection, err := pgx.ConnectConfig(ctx, connectionConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	return &PostgresStore{connection: connection}, nil
}

// Name returns the safe diagnostic identity for the canonical store.
func (store *PostgresStore) Name() string {
	return "PostgreSQL"
}

// Verify executes a constant read-only query.
func (store *PostgresStore) Verify(ctx context.Context) error {
	var ready int
	if err := store.connection.QueryRow(ctx, "SELECT 1").Scan(&ready); err != nil {
		return fmt.Errorf("execute canonical readiness query: %w", err)
	}
	if ready != 1 {
		return fmt.Errorf("canonical readiness query returned an unexpected value")
	}
	return nil
}

// Close releases the pgx connection.
func (store *PostgresStore) Close(ctx context.Context) error {
	return store.connection.Close(ctx)
}
