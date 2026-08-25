//go:build integration

package integration_test

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/persistence"
	"github.com/dharlanoliveira/norvii/apps/api/migrations"
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
	expectedVersion := latestEmbeddedMigrationVersion(t)
	if firstStatus.CurrentVersion != expectedVersion || secondStatus.CurrentVersion != expectedVersion {
		t.Fatalf(
			"migration versions = %d and %d, want %d and %d",
			firstStatus.CurrentVersion,
			secondStatus.CurrentVersion,
			expectedVersion,
			expectedVersion,
		)
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

func latestEmbeddedMigrationVersion(t *testing.T) int32 {
	t.Helper()
	entries, err := migrations.Files.ReadDir(".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	var latest int32
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		versionText, _, found := strings.Cut(entry.Name(), "_")
		if !found {
			t.Fatalf("embedded migration %q has no numeric version prefix", entry.Name())
		}
		version, parseErr := strconv.ParseInt(versionText, 10, 32)
		if parseErr != nil || version <= 0 {
			t.Fatalf("embedded migration %q has an invalid version prefix", entry.Name())
		}
		if int32(version) > latest {
			latest = int32(version)
		}
	}
	if latest == 0 {
		t.Fatal("no embedded SQL migrations found")
	}
	return latest
}
