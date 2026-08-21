//go:build integration

package integration_test

import (
	"context"
	"os"
	"slices"
	"testing"

	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/persistence"
	"github.com/jackc/pgx/v5"
)

func TestCorpusIngestionSchemaIsCanonicalSeededAndRepeatable(t *testing.T) {
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
		t.Fatalf(
			"migration versions = %d and %d, want 6 and 6",
			firstStatus.CurrentVersion,
			secondStatus.CurrentVersion,
		)
	}

	connection := openCatalogTestConnection(t, ctx, config.Postgres)
	expectedTables := []string{
		"corpora",
		"document_units",
		"document_versions",
		"ingestion_work",
		"pdf_origins",
		"processing_attempts",
		"source_revisions",
		"sources",
		"url_origins",
	}
	rows, err := connection.Query(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = ANY($1)
		ORDER BY table_name`, expectedTables)
	if err != nil {
		t.Fatalf("schema table query error = %v", err)
	}
	actualTables, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("collect schema tables error = %v", err)
	}
	if !slices.Equal(actualTables, expectedTables) {
		t.Fatalf("schema tables = %v, want %v", actualTables, expectedTables)
	}

	expectedIndexes := []string{
		"corpora_enabled_order_idx",
		"document_units_locator_uidx",
		"document_units_parent_order_idx",
		"document_versions_revision_pipeline_unique",
		"ingestion_work_active_source_uidx",
		"ingestion_work_pending_order_idx",
		"processing_attempts_source_order_idx",
		"sources_corpus_order_idx",
	}
	indexRows, err := connection.Query(ctx, `
		SELECT indexname
		FROM pg_indexes
		WHERE schemaname = 'public' AND indexname = ANY($1)
		ORDER BY indexname`, expectedIndexes)
	if err != nil {
		t.Fatalf("schema index query error = %v", err)
	}
	actualIndexes, err := pgx.CollectRows(indexRows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("collect schema indexes error = %v", err)
	}
	if !slices.Equal(actualIndexes, expectedIndexes) {
		t.Fatalf("schema indexes = %v, want %v", actualIndexes, expectedIndexes)
	}

	var documentUnitPrimaryKeyColumns []string
	err = connection.QueryRow(ctx, `
		SELECT array_agg(attribute.attname ORDER BY key_column.ordinality)
		FROM pg_constraint AS constraint_definition
		CROSS JOIN LATERAL unnest(constraint_definition.conkey)
			WITH ORDINALITY AS key_column(attribute_number, ordinality)
		JOIN pg_attribute AS attribute
			ON attribute.attrelid = constraint_definition.conrelid
			AND attribute.attnum = key_column.attribute_number
		WHERE constraint_definition.conrelid = 'document_units'::regclass
			AND constraint_definition.contype = 'p'`).Scan(&documentUnitPrimaryKeyColumns)
	if err != nil {
		t.Fatalf("document unit primary key query error = %v", err)
	}
	if !slices.Equal(documentUnitPrimaryKeyColumns, []string{"document_id", "id"}) {
		t.Fatalf(
			"document unit primary key = %v, want [document_id id]",
			documentUnitPrimaryKeyColumns,
		)
	}

	var corpusCount, sourceCount, originCount, workCount int
	err = connection.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM corpora WHERE seed_key IS NOT NULL),
			(SELECT count(*) FROM sources WHERE seed_key IS NOT NULL),
			(SELECT count(*) FROM url_origins),
			(SELECT count(*) FROM ingestion_work WHERE reason = 'initial')`).Scan(
		&corpusCount,
		&sourceCount,
		&originCount,
		&workCount,
	)
	if err != nil {
		t.Fatalf("seed count query error = %v", err)
	}
	if corpusCount != 2 || sourceCount != 2 || originCount != 2 || workCount != 2 {
		t.Fatalf(
			"seed counts = corpora:%d sources:%d origins:%d work:%d, want all 2",
			corpusCount,
			sourceCount,
			originCount,
			workCount,
		)
	}

	var foreignKeys int
	err = connection.QueryRow(ctx, `
		SELECT count(*)
		FROM pg_constraint
		WHERE contype = 'f'
		  AND connamespace = 'public'::regnamespace
		  AND conname LIKE '%_ownership_fk'`).Scan(&foreignKeys)
	if err != nil {
		t.Fatalf("ownership constraint query error = %v", err)
	}
	if foreignKeys < 7 {
		t.Fatalf("ownership foreign keys = %d, want at least 7", foreignKeys)
	}
}

func openCatalogTestConnection(
	t *testing.T,
	ctx context.Context,
	config persistence.PostgresConfig,
) *pgx.Conn {
	t.Helper()
	connectionConfig, err := pgx.ParseConfig("sslmode=disable")
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	connectionConfig.Host = config.Host
	connectionConfig.Port = config.Port
	connectionConfig.Database = config.Database
	connectionConfig.User = config.User
	connectionConfig.Password = config.Password
	connection, err := pgx.ConnectConfig(ctx, connectionConfig)
	if err != nil {
		t.Fatalf("ConnectConfig() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := connection.Close(context.Background()); closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
	})
	return connection
}
