//go:build integration

package integration_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/persistence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	evaluationLGPDCorpusID = "10000000-0000-4000-8000-000000000001"
	evaluationGDPRCorpusID = "10000000-0000-4000-8000-000000000002"
	evaluationGDPRSourceID = "20000000-0000-4000-8000-000000000002"
)

func TestEvaluationSchemaSeedsAreIsolatedFromLegacyCorpora(t *testing.T) {
	ctx, connection := openEvaluationSchemaConnection(t)

	type sourceBinding struct {
		corpusSeedKey string
		sourceSeedKey string
	}
	want := []sourceBinding{
		{"brazil-anti-corruption-white-collar-crime", "brazil-anti-corruption-law"},
		{"brazil-anti-corruption-white-collar-crime", "brazil-anti-money-laundering-law"},
		{"brazil-anti-corruption-white-collar-crime", "brazil-penal-code"},
		{"brazil-anti-corruption-white-collar-crime", "cgu-leniency-guidance"},
		{"brazil-anti-corruption-white-collar-crime", "coaf-resolution-36"},
		{"brazil-personal-data-protection", "brazil-personal-data-protection-lgpd"},
		{"us-fair-housing-disability-accommodations", "ecfr-24-100-204"},
		{"us-fair-housing-disability-accommodations", "hud-assistance-animals"},
		{"us-fair-housing-disability-accommodations", "hud-doj-reasonable-accommodations"},
		{"us-fair-housing-disability-accommodations", "hud-report-housing-discrimination"},
		{"us-fair-housing-disability-accommodations", "us-fair-housing-act-3604"},
	}

	rows, err := connection.Query(ctx, `
		SELECT corpus.seed_key, source.seed_key
		FROM sources AS source
		JOIN corpora AS corpus ON corpus.id = source.corpus_id
		WHERE source.seed_key IN (
			'brazil-anti-corruption-law',
			'brazil-anti-money-laundering-law',
			'brazil-penal-code',
			'cgu-leniency-guidance',
			'coaf-resolution-36',
			'brazil-personal-data-protection-lgpd',
			'ecfr-24-100-204',
			'hud-assistance-animals',
			'hud-doj-reasonable-accommodations',
			'hud-report-housing-discrimination',
			'us-fair-housing-act-3604'
		)
		ORDER BY corpus.seed_key, source.seed_key`)
	if err != nil {
		t.Fatalf("query evaluation source seed bindings: %v", err)
	}
	actual, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (sourceBinding, error) {
		var binding sourceBinding
		err := row.Scan(&binding.corpusSeedKey, &binding.sourceSeedKey)
		return binding, err
	})
	if err != nil {
		t.Fatalf("collect evaluation source seed bindings: %v", err)
	}
	if len(actual) != len(want) {
		t.Fatalf("evaluation source bindings = %d, want %d", len(actual), len(want))
	}
	for index := range want {
		if actual[index] != want[index] {
			t.Fatalf("evaluation source binding %d = %+v, want %+v", index, actual[index], want[index])
		}
	}

	var legacyBindings int
	if err := connection.QueryRow(ctx, `
		SELECT count(*)
		FROM sources AS source
		JOIN corpora AS corpus ON corpus.id = source.corpus_id
		WHERE source.seed_key IN (
			'brazil-anti-corruption-law',
			'brazil-anti-money-laundering-law',
			'brazil-penal-code',
			'cgu-leniency-guidance',
			'coaf-resolution-36',
			'brazil-personal-data-protection-lgpd',
			'ecfr-24-100-204',
			'hud-assistance-animals',
			'hud-doj-reasonable-accommodations',
			'hud-report-housing-discrimination',
			'us-fair-housing-act-3604'
		)
		  AND corpus.seed_key IN ('initial-gdpr-en', 'information-security')`).Scan(&legacyBindings); err != nil {
		t.Fatalf("query legacy evaluation source bindings: %v", err)
	}
	if legacyBindings != 0 {
		t.Fatalf("legacy corpus evaluation source bindings = %d, want 0", legacyBindings)
	}
}

func TestEvaluationSchemaRejectsCrossCorpusBindings(t *testing.T) {
	ctx, connection := openEvaluationSchemaConnection(t)

	t.Run("dataset source cannot use another revision corpus", func(t *testing.T) {
		transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
		defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

		revisionID := uuid.MustParse("a0000000-0000-4000-8000-000000000001")
		insertEvaluationRevision(t, ctx, transaction, revisionID, evaluationLGPDCorpusID)
		expectPostgresErrorCode(t, evaluationSchemaExecError(transaction.Exec(ctx, `
			INSERT INTO evaluation_dataset_source (
				id, dataset_revision_id, corpus_id, source_alias, title, official_url,
				issuing_authority, document_type, authority_role
			) VALUES ($1, $2, $3, 'source-a', 'Synthetic source',
				'https://example.test/source-a', 'Test authority', 'statute', 'statute')`,
			uuid.MustParse("a0000000-0000-4000-8000-000000000002"), revisionID, evaluationGDPRCorpusID,
		)), "23503")
	})

	t.Run("dataset source cannot reuse a GDPR source", func(t *testing.T) {
		transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
		defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

		revisionID := uuid.MustParse("a0000000-0000-4000-8000-000000000003")
		insertEvaluationRevision(t, ctx, transaction, revisionID, evaluationLGPDCorpusID)
		expectPostgresErrorCode(t, evaluationSchemaExecError(transaction.Exec(ctx, `
			INSERT INTO evaluation_dataset_source (
				id, dataset_revision_id, corpus_id, source_alias, title, official_url,
				issuing_authority, document_type, authority_role, corpus_source_id
			) VALUES ($1, $2, $3, 'source-b', 'Synthetic source',
				'https://example.test/source-b', 'Test authority', 'statute', 'statute', $4)`,
			uuid.MustParse("a0000000-0000-4000-8000-000000000004"), revisionID, evaluationLGPDCorpusID,
			uuid.MustParse(evaluationGDPRSourceID),
		)), "23503")
	})

	t.Run("dataset source cannot reuse an information security corpus", func(t *testing.T) {
		transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
		defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

		revisionID := uuid.MustParse("a0000000-0000-4000-8000-000000000005")
		insertEvaluationRevision(t, ctx, transaction, revisionID, evaluationLGPDCorpusID)
		foreignCorpusID := uuid.MustParse("a0000000-0000-4000-8000-000000000006")
		foreignSourceID := uuid.MustParse("a0000000-0000-4000-8000-000000000007")
		if _, err := transaction.Exec(ctx, `
			INSERT INTO corpora (id, name, description, language, jurisdiction)
			VALUES ($1, 'Synthetic information security corpus', 'Synthetic fixture.', 'en', 'Test')`, foreignCorpusID); err != nil {
			t.Fatalf("insert information security corpus: %v", err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO sources (id, corpus_id, title, kind)
			VALUES ($1, $2, 'Synthetic source', 'url')`, foreignSourceID, foreignCorpusID); err != nil {
			t.Fatalf("insert information security source: %v", err)
		}
		expectPostgresErrorCode(t, evaluationSchemaExecError(transaction.Exec(ctx, `
			INSERT INTO evaluation_dataset_source (
				id, dataset_revision_id, corpus_id, source_alias, title, official_url,
				issuing_authority, document_type, authority_role, corpus_source_id
			) VALUES ($1, $2, $3, 'source-c', 'Synthetic source',
				'https://example.test/source-c', 'Test authority', 'statute', 'statute', $4)`,
			uuid.MustParse("a0000000-0000-4000-8000-000000000008"), revisionID, evaluationLGPDCorpusID,
			foreignSourceID,
		)), "23503")
	})

	t.Run("suggestion projection cannot use another revision corpus", func(t *testing.T) {
		transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
		defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

		revisionID := uuid.MustParse("a0000000-0000-4000-8000-000000000009")
		insertEvaluationRevision(t, ctx, transaction, revisionID, evaluationLGPDCorpusID)
		snapshotID := uuid.MustParse("a0000000-0000-4000-8000-000000000010")
		if _, err := transaction.Exec(ctx, `
			INSERT INTO corpus_snapshots (id, corpus_id, manifest_sha256, created_by)
			VALUES ($1, $2, $3, 'integration-test')`,
			snapshotID, evaluationGDPRCorpusID, hashOf("e")); err != nil {
			t.Fatalf("insert foreign corpus snapshot: %v", err)
		}
		expectPostgresErrorCode(t, evaluationSchemaExecError(transaction.Exec(ctx, `
			INSERT INTO corpus_opening_suggestion_set (
				id, corpus_id, snapshot_id, snapshot_manifest_sha256, dataset_revision_id,
				source_dataset_content_sha256, selection_policy_version, published_by
			) VALUES ($1, $2, $3, $4, $5, $6, 'v1', 'integration-test')`,
			uuid.MustParse("a0000000-0000-4000-8000-000000000011"), evaluationGDPRCorpusID, snapshotID,
			hashOf("e"), revisionID, hashOf("c"),
		)), "23503")
	})
}

func TestEvaluationSchemaEnforcesAppendOnlyStorageAndStarterPairs(t *testing.T) {
	ctx, connection := openEvaluationSchemaConnection(t)
	assertEvaluationAppendOnlyTriggerTables(t, ctx, connection)
	assertEvaluationAppendOnlyMutations(t, ctx, connection)

	t.Run("each starter rank has reciprocal English and Brazilian Portuguese cases", func(t *testing.T) {
		transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
		defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

		revisionID := uuid.MustParse("b0000000-0000-4000-8000-000000000010")
		insertEvaluationRevision(t, ctx, transaction, revisionID, evaluationLGPDCorpusID)
		englishCaseID, _ := insertEvaluationCasePair(t, ctx, transaction, revisionID, "b0000000-0000-4000-8000-000000000011", "b0000000-0000-4000-8000-000000000012", 1)

		if _, err := transaction.Exec(ctx, `
			INSERT INTO evaluation_dataset_starter_case (
				id, dataset_revision_id, corpus_id, evaluation_case_id, rank, query_language,
				case_checksum, is_review_eligible
			) VALUES ($1, $2, $3, $4, 1, 'en', $5, true)`,
			uuid.MustParse("b0000000-0000-4000-8000-000000000013"), revisionID, evaluationLGPDCorpusID,
			englishCaseID, hashOf("a"),
		); err != nil {
			t.Fatalf("insert English starter selection: %v", err)
		}
		expectDeferredConstraintFailure(t, ctx, transaction)
	})

	t.Run("complete reciprocal English and Brazilian Portuguese starter pair is valid", func(t *testing.T) {
		transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
		defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

		revisionID := uuid.MustParse("b0000000-0000-4000-8000-000000000014")
		insertEvaluationRevision(t, ctx, transaction, revisionID, evaluationLGPDCorpusID)
		englishCaseID, portugueseCaseID := insertEvaluationCasePair(
			t,
			ctx,
			transaction,
			revisionID,
			"b0000000-0000-4000-8000-000000000015",
			"b0000000-0000-4000-8000-000000000016",
			1,
		)
		insertEvaluationStarterSelection(
			t,
			ctx,
			transaction,
			"b0000000-0000-4000-8000-000000000017",
			revisionID,
			englishCaseID,
			1,
			"en",
			hashOf("a"),
		)
		insertEvaluationStarterSelection(
			t,
			ctx,
			transaction,
			"b0000000-0000-4000-8000-000000000018",
			revisionID,
			portugueseCaseID,
			1,
			"pt",
			hashOf("b"),
		)

		if _, err := transaction.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE"); err != nil {
			t.Fatalf("validate complete English and Brazilian Portuguese starter pair: %v", err)
		}
	})

	t.Run("rank and language are unique within a dataset revision", func(t *testing.T) {
		transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
		defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

		revisionID := uuid.MustParse("b0000000-0000-4000-8000-000000000020")
		insertEvaluationRevision(t, ctx, transaction, revisionID, evaluationLGPDCorpusID)
		firstEnglishCaseID, _ := insertEvaluationCasePair(t, ctx, transaction, revisionID, "b0000000-0000-4000-8000-000000000021", "b0000000-0000-4000-8000-000000000022", 1)
		secondEnglishCaseID, _ := insertEvaluationCasePair(t, ctx, transaction, revisionID, "b0000000-0000-4000-8000-000000000023", "b0000000-0000-4000-8000-000000000024", 3)

		if _, err := transaction.Exec(ctx, `
			INSERT INTO evaluation_dataset_starter_case (
				id, dataset_revision_id, corpus_id, evaluation_case_id, rank, query_language,
				case_checksum, is_review_eligible
			) VALUES ($1, $2, $3, $4, 1, 'en', $5, true)`,
			uuid.MustParse("b0000000-0000-4000-8000-000000000025"), revisionID, evaluationLGPDCorpusID,
			firstEnglishCaseID, hashOf("a"),
		); err != nil {
			t.Fatalf("insert first English starter selection: %v", err)
		}
		expectPostgresErrorCode(t, evaluationSchemaExecError(transaction.Exec(ctx, `
			INSERT INTO evaluation_dataset_starter_case (
				id, dataset_revision_id, corpus_id, evaluation_case_id, rank, query_language,
				case_checksum, is_review_eligible
			) VALUES ($1, $2, $3, $4, 1, 'en', $5, true)`,
			uuid.MustParse("b0000000-0000-4000-8000-000000000026"), revisionID, evaluationLGPDCorpusID,
			secondEnglishCaseID, hashOf("c"),
		)), "23505")
	})
}

func openEvaluationSchemaConnection(t *testing.T) (context.Context, *pgx.Conn) {
	t.Helper()
	configuration, err := persistence.LoadConfig(os.LookupEnv)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), configuration.Timeout)
	t.Cleanup(cancel)
	return ctx, openCatalogTestConnection(t, ctx, configuration.Postgres)
}

func assertEvaluationAppendOnlyTriggerTables(t *testing.T, ctx context.Context, connection *pgx.Conn) {
	t.Helper()
	want := []string{
		"corpus_opening_suggestion_item",
		"corpus_opening_suggestion_set",
		"evaluation_case_expected_evidence",
		"evaluation_dataset_case",
		"evaluation_dataset_publication",
		"evaluation_dataset_revision",
		"evaluation_dataset_source",
		"evaluation_dataset_starter_case",
	}
	rows, err := connection.Query(ctx, `
		SELECT DISTINCT event_object_table
		FROM information_schema.triggers
		WHERE event_object_schema = 'public'
		  AND trigger_name IN (
			'evaluation_dataset_revision_immutable_trigger',
			'evaluation_dataset_source_binding_lock_trigger',
			'evaluation_dataset_case_immutable_trigger',
			'evaluation_case_expected_evidence_immutable_trigger',
			'evaluation_dataset_starter_case_immutable_trigger',
			'evaluation_dataset_publication_immutable_trigger',
			'corpus_opening_suggestion_set_immutable_trigger',
			'corpus_opening_suggestion_item_immutable_trigger'
		  )
		ORDER BY event_object_table`)
	if err != nil {
		t.Fatalf("query evaluation append-only triggers: %v", err)
	}
	actual, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("collect evaluation append-only triggers: %v", err)
	}
	if len(actual) != len(want) {
		t.Fatalf("append-only trigger tables = %v, want %v", actual, want)
	}
	for index := range want {
		if actual[index] != want[index] {
			t.Fatalf("append-only trigger table %d = %q, want %q", index, actual[index], want[index])
		}
	}
}

func assertEvaluationAppendOnlyMutations(t *testing.T, ctx context.Context, connection *pgx.Conn) {
	t.Helper()

	mutations := []struct {
		table     string
		updateSQL string
		deleteSQL string
		id        func(evaluationAppendOnlyFixture) uuid.UUID
	}{
		{
			table:     "evaluation_dataset_revision",
			updateSQL: "UPDATE evaluation_dataset_revision SET importer_version = 'changed' WHERE id = $1",
			deleteSQL: "DELETE FROM evaluation_dataset_revision WHERE id = $1",
			id:        func(fixture evaluationAppendOnlyFixture) uuid.UUID { return fixture.revisionID },
		},
		{
			table:     "evaluation_dataset_source",
			updateSQL: "UPDATE evaluation_dataset_source SET title = 'Changed synthetic source' WHERE id = $1",
			deleteSQL: "DELETE FROM evaluation_dataset_source WHERE id = $1",
			id:        func(fixture evaluationAppendOnlyFixture) uuid.UUID { return fixture.sourceID },
		},
		{
			table:     "evaluation_dataset_case",
			updateSQL: "UPDATE evaluation_dataset_case SET reference_answer = 'Changed synthetic answer' WHERE id = $1",
			deleteSQL: "DELETE FROM evaluation_dataset_case WHERE id = $1",
			id:        func(fixture evaluationAppendOnlyFixture) uuid.UUID { return fixture.englishCaseID },
		},
		{
			table:     "evaluation_case_expected_evidence",
			updateSQL: "UPDATE evaluation_case_expected_evidence SET display_locator = 'changed-locator' WHERE id = $1",
			deleteSQL: "DELETE FROM evaluation_case_expected_evidence WHERE id = $1",
			id:        func(fixture evaluationAppendOnlyFixture) uuid.UUID { return fixture.expectedEvidenceID },
		},
		{
			table:     "evaluation_dataset_starter_case",
			updateSQL: "UPDATE evaluation_dataset_starter_case SET is_review_eligible = false WHERE id = $1",
			deleteSQL: "DELETE FROM evaluation_dataset_starter_case WHERE id = $1",
			id:        func(fixture evaluationAppendOnlyFixture) uuid.UUID { return fixture.englishStarterID },
		},
		{
			table:     "evaluation_dataset_publication",
			updateSQL: "UPDATE evaluation_dataset_publication SET review_note = 'changed' WHERE id = $1",
			deleteSQL: "DELETE FROM evaluation_dataset_publication WHERE id = $1",
			id:        func(fixture evaluationAppendOnlyFixture) uuid.UUID { return fixture.publicationID },
		},
		{
			table:     "corpus_opening_suggestion_set",
			updateSQL: "UPDATE corpus_opening_suggestion_set SET published_by = 'changed' WHERE id = $1",
			deleteSQL: "DELETE FROM corpus_opening_suggestion_set WHERE id = $1",
			id:        func(fixture evaluationAppendOnlyFixture) uuid.UUID { return fixture.suggestionSetID },
		},
		{
			table:     "corpus_opening_suggestion_item",
			updateSQL: "UPDATE corpus_opening_suggestion_item SET question = 'Changed synthetic question' WHERE id = $1",
			deleteSQL: "DELETE FROM corpus_opening_suggestion_item WHERE id = $1",
			id:        func(fixture evaluationAppendOnlyFixture) uuid.UUID { return fixture.suggestionItemID },
		},
	}

	for _, mutation := range mutations {
		mutation := mutation
		t.Run(mutation.table+" rejects updates", func(t *testing.T) {
			transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
			defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

			fixture := insertEvaluationAppendOnlyFixture(t, ctx, transaction)
			expectPostgresErrorCode(t, evaluationSchemaExecError(transaction.Exec(ctx, mutation.updateSQL, mutation.id(fixture))), "55000")
		})
		t.Run(mutation.table+" rejects deletes", func(t *testing.T) {
			transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
			defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

			fixture := insertEvaluationAppendOnlyFixture(t, ctx, transaction)
			expectPostgresErrorCode(t, evaluationSchemaExecError(transaction.Exec(ctx, mutation.deleteSQL, mutation.id(fixture))), "55000")
		})
	}
}

type evaluationAppendOnlyFixture struct {
	revisionID         uuid.UUID
	sourceID           uuid.UUID
	englishCaseID      uuid.UUID
	expectedEvidenceID uuid.UUID
	englishStarterID   uuid.UUID
	publicationID      uuid.UUID
	suggestionSetID    uuid.UUID
	suggestionItemID   uuid.UUID
}

func insertEvaluationAppendOnlyFixture(
	t *testing.T,
	ctx context.Context,
	transaction pgx.Tx,
) evaluationAppendOnlyFixture {
	t.Helper()

	fixture := evaluationAppendOnlyFixture{
		revisionID:         uuid.MustParse("c0000000-0000-4000-8000-000000000001"),
		sourceID:           uuid.MustParse("c0000000-0000-4000-8000-000000000002"),
		englishCaseID:      uuid.MustParse("c0000000-0000-4000-8000-000000000003"),
		expectedEvidenceID: uuid.MustParse("c0000000-0000-4000-8000-000000000005"),
		englishStarterID:   uuid.MustParse("c0000000-0000-4000-8000-000000000006"),
		publicationID:      uuid.MustParse("c0000000-0000-4000-8000-000000000008"),
		suggestionSetID:    uuid.MustParse("c0000000-0000-4000-8000-000000000010"),
		suggestionItemID:   uuid.MustParse("c0000000-0000-4000-8000-000000000011"),
	}
	portugueseCaseID := uuid.MustParse("c0000000-0000-4000-8000-000000000004")
	portugueseStarterID := uuid.MustParse("c0000000-0000-4000-8000-000000000007")
	snapshotID := uuid.MustParse("c0000000-0000-4000-8000-000000000009")

	insertEvaluationRevision(t, ctx, transaction, fixture.revisionID, evaluationLGPDCorpusID)
	if _, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_dataset_source (
			id, dataset_revision_id, corpus_id, source_alias, title, official_url,
			issuing_authority, document_type, authority_role
		) VALUES ($1, $2, $3, 'source-a', 'Synthetic source',
			'https://example.test/source-a', 'Test authority', 'statute', 'statute')`,
		fixture.sourceID, fixture.revisionID, uuid.MustParse(evaluationLGPDCorpusID),
	); err != nil {
		t.Fatalf("insert append-only fixture source: %v", err)
	}
	englishCaseID, actualPortugueseCaseID := insertEvaluationCasePair(
		t,
		ctx,
		transaction,
		fixture.revisionID,
		fixture.englishCaseID.String(),
		portugueseCaseID.String(),
		1,
	)
	if englishCaseID != fixture.englishCaseID || actualPortugueseCaseID != portugueseCaseID {
		t.Fatal("append-only fixture evaluation case identifiers differ from requested values")
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_case_expected_evidence (
			id, dataset_revision_id, corpus_id, evaluation_case_id, source_alias, ordinal,
			display_locator, canonical_locator, required_propositions
		) VALUES ($1, $2, $3, $4, 'source-a', 1, 'synthetic-locator', 'article:1', '["synthetic proposition"]'::jsonb)`,
		fixture.expectedEvidenceID, fixture.revisionID, uuid.MustParse(evaluationLGPDCorpusID), fixture.englishCaseID,
	); err != nil {
		t.Fatalf("insert append-only fixture expected evidence: %v", err)
	}
	insertEvaluationStarterSelection(t, ctx, transaction, fixture.englishStarterID.String(), fixture.revisionID, fixture.englishCaseID, 1, "en", hashOf("a"))
	insertEvaluationStarterSelection(t, ctx, transaction, portugueseStarterID.String(), fixture.revisionID, portugueseCaseID, 1, "pt", hashOf("b"))
	if _, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_dataset_publication (
			id, dataset_revision_id, corpus_id, review_decision, reviewer_identity, publication_state
		) VALUES ($1, $2, $3, 'pending', 'integration-test', 'draft')`,
		fixture.publicationID, fixture.revisionID, uuid.MustParse(evaluationLGPDCorpusID),
	); err != nil {
		t.Fatalf("insert append-only fixture publication: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO corpus_snapshots (id, corpus_id, manifest_sha256, created_by)
		VALUES ($1, $2, $3, 'integration-test')`,
		snapshotID, uuid.MustParse(evaluationLGPDCorpusID), hashOf("d"),
	); err != nil {
		t.Fatalf("insert append-only fixture snapshot: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO corpus_opening_suggestion_set (
			id, corpus_id, snapshot_id, snapshot_manifest_sha256, dataset_revision_id,
			source_dataset_content_sha256, selection_policy_version, published_by
		) VALUES ($1, $2, $3, $4, $5, $6, 'v1', 'integration-test')`,
		fixture.suggestionSetID, uuid.MustParse(evaluationLGPDCorpusID), snapshotID, hashOf("d"), fixture.revisionID, hashOf("c"),
	); err != nil {
		t.Fatalf("insert append-only fixture suggestion set: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO corpus_opening_suggestion_item (
			id, suggestion_set_id, corpus_id, dataset_revision_id, rank, evaluation_case_id,
			case_checksum, query_language, question
		) VALUES ($1, $2, $3, $4, 1, $5, $6, 'en', 'Synthetic question')`,
		fixture.suggestionItemID, fixture.suggestionSetID, uuid.MustParse(evaluationLGPDCorpusID), fixture.revisionID,
		fixture.englishCaseID, hashOf("a"),
	); err != nil {
		t.Fatalf("insert append-only fixture suggestion item: %v", err)
	}

	return fixture
}

func beginEvaluationSchemaTransaction(t *testing.T, ctx context.Context, connection *pgx.Conn) pgx.Tx {
	t.Helper()
	transaction, err := connection.Begin(ctx)
	if err != nil {
		t.Fatalf("begin evaluation schema transaction: %v", err)
	}
	return transaction
}

func rollbackEvaluationSchemaTransaction(t *testing.T, ctx context.Context, transaction pgx.Tx) {
	t.Helper()
	if err := transaction.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		t.Errorf("rollback evaluation schema transaction: %v", err)
	}
}

func insertEvaluationRevision(t *testing.T, ctx context.Context, transaction pgx.Tx, revisionID uuid.UUID, corpusID string) {
	t.Helper()
	if _, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_dataset_revision (
			id, corpus_id, dataset_key, semantic_revision, jurisdiction, manifest_sha256,
			jsonl_sha256, content_sha256, manifest_path, jsonl_path, declared_snapshot_date,
			query_languages, authoritative_evidence_language, importer_version
		) VALUES ($1, $2, $3, 'v1', 'Test', $4, $5, $6, 'fixtures/manifest.json',
			'fixtures/cases.jsonl', $7, ARRAY['en', 'pt'], 'pt-BR', 'integration-test')`,
		revisionID, uuid.MustParse(corpusID), "fixture-"+revisionID.String(), hashOf("a"), hashOf("b"), hashOf("c"),
		time.Date(2026, time.August, 26, 0, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("insert evaluation revision: %v", err)
	}
}

func insertEvaluationCasePair(
	t *testing.T,
	ctx context.Context,
	transaction pgx.Tx,
	revisionID uuid.UUID,
	englishCaseIDText, portugueseCaseIDText string,
	firstPosition int,
) (uuid.UUID, uuid.UUID) {
	return insertEvaluationCasePairWithOutcome(
		t, ctx, transaction, revisionID, englishCaseIDText, portugueseCaseIDText, firstPosition, "answer", "",
	)
}

func insertEvaluationCasePairWithOutcome(
	t *testing.T,
	ctx context.Context,
	transaction pgx.Tx,
	revisionID uuid.UUID,
	englishCaseIDText, portugueseCaseIDText string,
	firstPosition int,
	expectedOutcome, expectedReasonCode string,
) (uuid.UUID, uuid.UUID) {
	t.Helper()
	var persistedReasonCode *string
	if expectedReasonCode != "" {
		persistedReasonCode = &expectedReasonCode
	}
	englishCaseID := uuid.MustParse(englishCaseIDText)
	portugueseCaseID := uuid.MustParse(portugueseCaseIDText)
	englishExternalID := "case-" + englishCaseID.String()
	portugueseExternalID := "case-" + portugueseCaseID.String()
	for _, fixture := range []struct {
		id            uuid.UUID
		externalID    string
		reciprocalID  string
		queryLanguage string
		assetLanguage string
		position      int
		checksum      string
	}{
		{englishCaseID, englishExternalID, portugueseExternalID, "en", "en", firstPosition, hashOf("a")},
		{portugueseCaseID, portugueseExternalID, englishExternalID, "pt", "pt-BR", firstPosition + 1, hashOf("b")},
	} {
		if _, err := transaction.Exec(ctx, `
			INSERT INTO evaluation_dataset_case (
				id, dataset_revision_id, corpus_id, position, external_case_id, query_language,
				asset_language, question, reference_answer, category, authoritative_evidence_language,
				expected_outcome, expected_reason_code, reciprocal_case_external_id, case_checksum
			) VALUES ($1, $2, $3, $4, $5, $6, $7, 'Synthetic question', 'Synthetic answer',
				'fixture', 'pt-BR', $8, $9, $10, $11)`,
			fixture.id, revisionID, uuid.MustParse(evaluationLGPDCorpusID), fixture.position, fixture.externalID,
			fixture.queryLanguage, fixture.assetLanguage, expectedOutcome, persistedReasonCode, fixture.reciprocalID, fixture.checksum,
		); err != nil {
			t.Fatalf("insert evaluation case %s: %v", fixture.externalID, err)
		}
	}
	return englishCaseID, portugueseCaseID
}

func insertEvaluationStarterSelection(
	t *testing.T,
	ctx context.Context,
	transaction pgx.Tx,
	selectionIDText string,
	revisionID, evaluationCaseID uuid.UUID,
	rank int,
	queryLanguage, caseChecksum string,
) {
	t.Helper()
	if _, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_dataset_starter_case (
			id, dataset_revision_id, corpus_id, evaluation_case_id, rank, query_language,
			case_checksum, is_review_eligible
		) VALUES ($1, $2, $3, $4, $5, $6, $7, true)`,
		uuid.MustParse(selectionIDText), revisionID, uuid.MustParse(evaluationLGPDCorpusID), evaluationCaseID,
		rank, queryLanguage, caseChecksum,
	); err != nil {
		t.Fatalf("insert evaluation starter selection: %v", err)
	}
}

func expectDeferredConstraintFailure(t *testing.T, ctx context.Context, transaction pgx.Tx) {
	t.Helper()
	expectPostgresErrorCode(t, evaluationSchemaExecError(transaction.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE")), "23514")
}

func evaluationSchemaExecError(_ pgconn.CommandTag, err error) error {
	return err
}

func expectPostgresErrorCode(t *testing.T, err error, wantCode string) {
	t.Helper()
	if err == nil {
		t.Fatalf("operation succeeded, want PostgreSQL error %s", wantCode)
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		t.Fatalf("operation error = %T %v, want PostgreSQL error %s", err, err, wantCode)
	}
	if postgresError.Code != wantCode {
		t.Fatalf("PostgreSQL error code = %s, want %s: %v", postgresError.Code, wantCode, err)
	}
}

func hashOf(character string) string {
	return strings.Repeat(character, 64)
}
