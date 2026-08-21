//go:build integration

package integration_test

import (
	"context"
	"os"
	"testing"

	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/persistence"
	"github.com/jackc/pgx/v5"
)

func TestCanonicalInitializationIsVectorCapableAndRepeatable(t *testing.T) {
	config, err := persistence.LoadConfig(os.LookupEnv)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	migrator, err := persistence.OpenPostgresMigrator(ctx, config.Postgres)
	if err != nil {
		t.Fatalf("OpenPostgresMigrator() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := migrator.Close(context.Background()); closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
	})
	service := persistence.NewMigrationService(migrator)

	firstStatus, err := service.Apply(ctx)
	if err != nil {
		t.Fatalf("first Apply() error = %v", err)
	}
	secondStatus, err := service.Apply(ctx)
	if err != nil {
		t.Fatalf("second Apply() error = %v", err)
	}
	if firstStatus.CurrentVersion != 6 || secondStatus.CurrentVersion != 6 {
		t.Fatalf("migration versions = %d and %d, want 6 and 6", firstStatus.CurrentVersion, secondStatus.CurrentVersion)
	}

	connectionConfig, err := pgx.ParseConfig("sslmode=disable")
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	connectionConfig.Host = config.Postgres.Host
	connectionConfig.Port = config.Postgres.Port
	connectionConfig.Database = config.Postgres.Database
	connectionConfig.User = config.Postgres.User
	connectionConfig.Password = config.Postgres.Password
	connection, err := pgx.ConnectConfig(ctx, connectionConfig)
	if err != nil {
		t.Fatalf("ConnectConfig() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := connection.Close(context.Background()); closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
	})

	var extensionVersion string
	if err := connection.QueryRow(
		ctx,
		"SELECT extversion FROM pg_extension WHERE extname = 'vector'",
	).Scan(&extensionVersion); err != nil {
		t.Fatalf("vector extension query error = %v", err)
	}
	if extensionVersion != "0.8.6" {
		t.Fatalf("vector extension version = %q, want 0.8.6", extensionVersion)
	}
}
