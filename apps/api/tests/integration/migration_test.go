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

const evaluationDatasetsMigrationVersion int32 = 13

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
	if firstStatus.CurrentVersion < evaluationDatasetsMigrationVersion {
		t.Fatalf(
			"migration version = %d, want Feature 012 evaluation dataset migration %d or later",
			firstStatus.CurrentVersion,
			evaluationDatasetsMigrationVersion,
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

func TestEvaluationDatasetsMigrationIsEmbedded(t *testing.T) {
	migration, err := migrations.Files.ReadFile("013_evaluation_datasets.sql")
	if err != nil {
		t.Fatalf("read Feature 012 evaluation dataset migration: %v", err)
	}

	for _, requiredStatement := range []string{
		"CREATE TABLE evaluation_dataset_revision",
		"CREATE TABLE evaluation_dataset_starter_case",
		"CREATE TABLE corpus_opening_suggestion_set",
		"CREATE TABLE corpus_opening_suggestion_item",
		"CREATE FUNCTION reject_evaluation_immutable_mutation()",
	} {
		if !strings.Contains(string(migration), requiredStatement) {
			t.Errorf("Feature 012 migration does not contain %q", requiredStatement)
		}
	}
}

func TestEvaluationExpectedEvidenceCanonicalLocatorMigrationIsEmbedded(t *testing.T) {
	migration, err := migrations.Files.ReadFile("015_evaluation_expected_evidence_canonical_locators.sql")
	if err != nil {
		t.Fatalf("read evaluation canonical-locator migration: %v", err)
	}

	for _, requiredStatement := range []string{
		"ADD COLUMN canonical_locator text",
		"canonical_locator ~ '^[a-z][a-z-]*:[a-z0-9.-]+",
		"DROP COLUMN IF EXISTS canonical_locator",
	} {
		if !strings.Contains(string(migration), requiredStatement) {
			t.Errorf("canonical-locator migration does not contain %q", requiredStatement)
		}
	}
}

func TestEvaluationRunLedgerMigrationIsEmbedded(t *testing.T) {
	migration, err := migrations.Files.ReadFile("016_evaluation_runs.sql")
	if err != nil {
		t.Fatalf("read evaluation run ledger migration: %v", err)
	}

	for _, requiredStatement := range []string{
		"CREATE TABLE evaluation_run",
		"CREATE TABLE evaluation_run_case",
		"CREATE TABLE evaluation_run_expected_evidence",
		"CREATE TABLE evaluation_run_actual_evidence",
		"CREATE TABLE evaluation_run_metric",
		"CREATE FUNCTION protect_evaluation_run_case_terminal_result()",
	} {
		if !strings.Contains(string(migration), requiredStatement) {
			t.Errorf("evaluation run ledger migration does not contain %q", requiredStatement)
		}
	}
}

func TestEvaluationTerminalResultLedgerMigrationIsEmbedded(t *testing.T) {
	migration, err := migrations.Files.ReadFile("017_evaluation_terminal_result_ledger.sql")
	if err != nil {
		t.Fatalf("read evaluation terminal result ledger migration: %v", err)
	}

	for _, requiredStatement := range []string{
		"CREATE FUNCTION reject_evaluation_run_case_child_after_terminal()",
		"CREATE FUNCTION require_terminal_cases_for_evaluation_run_aggregate()",
		"CREATE TRIGGER evaluation_run_actual_evidence_terminal_child_trigger",
		"CREATE TRIGGER evaluation_run_metric_terminal_aggregate_trigger",
	} {
		if !strings.Contains(string(migration), requiredStatement) {
			t.Errorf("evaluation terminal result ledger migration does not contain %q", requiredStatement)
		}
	}
}

func TestEvaluationComparisonMetricLedgerMigrationIsEmbedded(t *testing.T) {
	migration, err := migrations.Files.ReadFile("018_evaluation_comparison_metric_ledger.sql")
	if err != nil {
		t.Fatalf("read evaluation comparison metric ledger migration: %v", err)
	}

	for _, requiredStatement := range []string{
		"CREATE FUNCTION require_evaluation_run_metric_scorer_version()",
		"CREATE FUNCTION require_evaluation_terminal_case_metric_ledger()",
		"CREATE FUNCTION require_evaluation_terminal_run_metric_ledgers()",
		"CREATE TRIGGER evaluation_run_metric_scorer_version_trigger",
		"CREATE TRIGGER evaluation_run_case_metric_ledger_trigger",
		"CREATE TRIGGER evaluation_run_terminal_metric_ledger_trigger",
	} {
		if !strings.Contains(string(migration), requiredStatement) {
			t.Errorf("evaluation comparison metric ledger migration does not contain %q", requiredStatement)
		}
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
