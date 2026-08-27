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

	connection := openCatalogTestConnection(t, ctx, config.Postgres)
	expectedTables := []string{
		"corpora",
		"corpus_opening_suggestion_item",
		"corpus_opening_suggestion_set",
		"corpus_snapshot_documents",
		"corpus_snapshot_releases",
		"corpus_snapshots",
		"document_units",
		"document_versions",
		"evaluation_case_expected_evidence",
		"evaluation_dataset_case",
		"evaluation_dataset_publication",
		"evaluation_dataset_revision",
		"evaluation_dataset_source",
		"evaluation_dataset_starter_case",
		"evaluation_run",
		"evaluation_run_actual_evidence",
		"evaluation_run_case",
		"evaluation_run_expected_evidence",
		"evaluation_run_metric",
		"graph_release_assertions",
		"graph_release_entities",
		"graph_release_legal_units",
		"graph_releases",
		"ingestion_work",
		"normative_assertions",
		"pdf_origins",
		"processing_attempts",
		"semantic_entities",
		"semantic_extraction_runs",
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
		"corpus_opening_suggestion_item_rank_idx",
		"corpus_opening_suggestion_set_snapshot_idx",
		"corpus_snapshot_documents_document_idx",
		"document_units_canonical_locator_uidx",
		"document_units_locator_uidx",
		"document_units_parent_order_idx",
		"document_versions_revision_pipeline_unique",
		"evaluation_case_expected_evidence_case_idx",
		"evaluation_dataset_case_revision_position_idx",
		"evaluation_dataset_publication_revision_latest_idx",
		"evaluation_dataset_revision_corpus_imported_idx",
		"evaluation_dataset_source_binding_idx",
		"evaluation_run_case_claim_idx",
		"evaluation_run_case_run_state_idx",
		"graph_releases_snapshot_status_idx",
		"ingestion_work_active_source_uidx",
		"ingestion_work_pending_order_idx",
		"normative_assertions_extraction_supported_idx",
		"processing_attempts_source_order_idx",
		"semantic_entities_extraction_supported_idx",
		"semantic_extraction_runs_document_ready_idx",
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

	assertPrimaryKeyColumns(t, ctx, connection, "graph_release_assertions", []string{"graph_release_id", "normative_assertion_id"})
	assertPrimaryKeyColumns(t, ctx, connection, "graph_release_legal_units", []string{"graph_release_id", "document_id", "legal_unit_id"})

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
	if corpusCount != 4 || sourceCount != 12 || originCount != 12 || workCount != 12 {
		t.Fatalf(
			"seed counts = corpora:%d sources:%d origins:%d work:%d, want 4/12/12/12",
			corpusCount,
			sourceCount,
			originCount,
			workCount,
		)
	}

	var queuedEvaluationSourceCount int
	err = connection.QueryRow(ctx, `
		SELECT count(*)
		FROM sources AS source
		JOIN ingestion_work AS work
		  ON work.source_id = source.id
		 AND work.corpus_id = source.corpus_id
		WHERE source.seed_key = ANY($1)
		  AND work.reason = 'initial'
		  AND work.status = 'pending'`, []string{
		"brazil-anti-corruption-law",
		"brazil-penal-code",
		"brazil-anti-money-laundering-law",
		"coaf-resolution-36",
		"cgu-leniency-guidance",
		"us-fair-housing-act-3604",
		"hud-assistance-animals",
		"hud-doj-reasonable-accommodations",
		"hud-report-housing-discrimination",
		"ecfr-24-100-204",
	}).Scan(&queuedEvaluationSourceCount)
	if err != nil {
		t.Fatalf("evaluation source queue query error = %v", err)
	}
	if queuedEvaluationSourceCount != 10 {
		t.Fatalf("queued evaluation sources = %d, want 10", queuedEvaluationSourceCount)
	}

	expectedEvaluationCorpusSeedKeys := []string{
		"brazil-anti-corruption-white-collar-crime",
		"brazil-personal-data-protection",
		"us-fair-housing-disability-accommodations",
	}
	rows, err = connection.Query(ctx, `
		SELECT seed_key
		FROM corpora
		WHERE seed_key = ANY($1)
		ORDER BY seed_key`, expectedEvaluationCorpusSeedKeys)
	if err != nil {
		t.Fatalf("evaluation corpus seed query error = %v", err)
	}
	actualEvaluationCorpusSeedKeys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("collect evaluation corpus seeds error = %v", err)
	}
	if !slices.Equal(actualEvaluationCorpusSeedKeys, expectedEvaluationCorpusSeedKeys) {
		t.Fatalf(
			"evaluation corpus seed keys = %v, want %v",
			actualEvaluationCorpusSeedKeys,
			expectedEvaluationCorpusSeedKeys,
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

func assertPrimaryKeyColumns(
	t *testing.T,
	ctx context.Context,
	connection *pgx.Conn,
	table string,
	want []string,
) {
	t.Helper()
	var actual []string
	err := connection.QueryRow(ctx, `
		SELECT array_agg(attribute.attname ORDER BY key_column.ordinality)
		FROM pg_constraint AS constraint_definition
		CROSS JOIN LATERAL unnest(constraint_definition.conkey)
			WITH ORDINALITY AS key_column(attribute_number, ordinality)
		JOIN pg_attribute AS attribute
			ON attribute.attrelid = constraint_definition.conrelid
			AND attribute.attnum = key_column.attribute_number
		WHERE constraint_definition.conrelid = $1::regclass
			AND constraint_definition.contype = 'p'`, table).Scan(&actual)
	if err != nil {
		t.Fatalf("%s primary key query error = %v", table, err)
	}
	if !slices.Equal(actual, want) {
		t.Fatalf("%s primary key = %v, want %v", table, actual, want)
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
