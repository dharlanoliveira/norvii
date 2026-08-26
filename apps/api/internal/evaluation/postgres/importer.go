package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const datasetImporterVersion = "evaluation-dataset-importer-v1"

type importDatabase interface {
	Begin(context.Context) (pgx.Tx, error)
}

// DatasetImporter persists validated immutable evaluation assets in PostgreSQL.
type DatasetImporter struct {
	database importDatabase
}

// NewDatasetImporter constructs an immutable asset writer around caller-owned persistence.
func NewDatasetImporter(database importDatabase) *DatasetImporter {
	return &DatasetImporter{database: database}
}

// StoreDataset writes a new draft revision, or returns the already stored identical revision.
// It never binds sources, publishes a revision, mutates an active release, or performs network work.
func (importer *DatasetImporter) StoreDataset(
	ctx context.Context,
	asset application.DatasetAsset,
) (application.DatasetImportResult, error) {
	if importer == nil || importer.database == nil {
		return application.DatasetImportResult{}, application.ErrDatasetStore
	}
	if err := validateDatasetAsset(asset); err != nil {
		return application.DatasetImportResult{}, err
	}

	transaction, err := importer.database.Begin(ctx)
	if err != nil {
		return application.DatasetImportResult{}, fmt.Errorf("begin evaluation dataset import: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	created, err := insertRevision(ctx, transaction, asset.Revision, asset.ManifestPath, asset.JSONLPath)
	if err != nil {
		return application.DatasetImportResult{}, err
	}
	if !created {
		if err := verifyExistingRevision(ctx, transaction, asset.Revision); err != nil {
			return application.DatasetImportResult{}, err
		}
		if err := transaction.Commit(ctx); err != nil {
			return application.DatasetImportResult{}, fmt.Errorf("commit existing evaluation dataset import: %w", err)
		}
		return importResult(asset, false), nil
	}

	if err := insertSources(ctx, transaction, asset.Sources); err != nil {
		return application.DatasetImportResult{}, err
	}
	if err := insertCases(ctx, transaction, asset.Cases); err != nil {
		return application.DatasetImportResult{}, err
	}
	if err := insertEvidence(ctx, transaction, asset.ExpectedEvidence); err != nil {
		return application.DatasetImportResult{}, err
	}
	if err := insertStarterSelections(ctx, transaction, asset.StarterSelections); err != nil {
		return application.DatasetImportResult{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return application.DatasetImportResult{}, fmt.Errorf("commit evaluation dataset import: %w", err)
	}
	return importResult(asset, true), nil
}

func validateDatasetAsset(asset application.DatasetAsset) error {
	revision := asset.Revision
	if err := revision.Validate(); err != nil || len(asset.Sources) == 0 || len(asset.Cases) == 0 || len(asset.ExpectedEvidence) == 0 {
		return application.ErrDatasetStore
	}
	for index := range asset.Sources {
		if err := asset.Sources[index].ValidateAgainst(revision); err != nil {
			return application.ErrDatasetStore
		}
	}
	for index := range asset.Cases {
		if err := asset.Cases[index].ValidateAgainst(revision); err != nil {
			return application.ErrDatasetStore
		}
	}
	if err := domain.ValidatePairedCases(revision, asset.Cases); err != nil {
		return application.ErrDatasetStore
	}
	for index := range asset.ExpectedEvidence {
		evidence := asset.ExpectedEvidence[index]
		matchingCase, found := findCase(asset.Cases, evidence.CaseID)
		if !found || evidence.ValidateAgainst(revision, matchingCase, asset.Sources) != nil {
			return application.ErrDatasetStore
		}
	}
	if err := domain.ValidateStarterSelections(revision, asset.Cases, asset.StarterSelections); err != nil {
		return application.ErrDatasetStore
	}
	return nil
}

func findCase(cases []domain.Case, caseID uuid.UUID) (domain.Case, bool) {
	for _, evaluationCase := range cases {
		if evaluationCase.ID == caseID {
			return evaluationCase, true
		}
	}
	return domain.Case{}, false
}

func insertRevision(ctx context.Context, transaction pgx.Tx, revision domain.DatasetRevision, manifestPath, jsonlPath string) (bool, error) {
	queryLanguages := make([]string, len(revision.QueryLanguages))
	for index, language := range revision.QueryLanguages {
		queryLanguages[index] = string(language)
	}
	command, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_dataset_revision (
			id, corpus_id, dataset_key, semantic_revision, jurisdiction, manifest_sha256,
			jsonl_sha256, content_sha256, manifest_path, jsonl_path, declared_snapshot_date,
			query_languages, authoritative_evidence_language, importer_version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (corpus_id, content_sha256) DO NOTHING`,
		revision.ID, revision.CorpusID, revision.DatasetKey, revision.SemanticRevision, revision.Jurisdiction,
		string(revision.ManifestSHA256), string(revision.JSONLSHA256), string(revision.ContentSHA256), manifestPath,
		jsonlPath, revision.DeclaredSnapshotDate.UTC(), queryLanguages, string(revision.AuthoritativeEvidenceLanguage), datasetImporterVersion,
	)
	if err != nil {
		return false, fmt.Errorf("insert evaluation dataset revision: %w", err)
	}
	return command.RowsAffected() == 1, nil
}

func verifyExistingRevision(ctx context.Context, transaction pgx.Tx, revision domain.DatasetRevision) error {
	var existingID uuid.UUID
	err := transaction.QueryRow(ctx, `
		SELECT id
		FROM evaluation_dataset_revision
		WHERE corpus_id = $1 AND content_sha256 = $2`,
		revision.CorpusID, string(revision.ContentSHA256),
	).Scan(&existingID)
	if err != nil {
		return fmt.Errorf("read existing evaluation dataset revision: %w", err)
	}
	if existingID != revision.ID {
		return application.ErrDatasetStore
	}
	return nil
}

func insertSources(ctx context.Context, transaction pgx.Tx, sources []domain.SourceRequirement) error {
	for _, source := range sources {
		if _, err := transaction.Exec(ctx, `
			INSERT INTO evaluation_dataset_source (
				id, dataset_revision_id, corpus_id, source_alias, title, official_url,
				issuing_authority, document_type, authority_role, corpus_source_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			source.ID, source.DatasetRevisionID, source.CorpusID, source.SourceAlias, source.Title, source.OfficialURL,
			source.IssuingAuthority, source.DocumentType, source.AuthorityRole, source.CorpusSourceID,
		); err != nil {
			return fmt.Errorf("insert evaluation dataset source: %w", err)
		}
	}
	return nil
}

func insertCases(ctx context.Context, transaction pgx.Tx, cases []domain.Case) error {
	for _, evaluationCase := range cases {
		if _, err := transaction.Exec(ctx, `
			INSERT INTO evaluation_dataset_case (
				id, dataset_revision_id, corpus_id, position, external_case_id, query_language,
				asset_language, question, reference_answer, category, authoritative_evidence_language,
				expected_outcome, expected_reason_code, reciprocal_case_external_id, case_checksum
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
			evaluationCase.ID, evaluationCase.DatasetRevisionID, evaluationCase.CorpusID, evaluationCase.Position,
			evaluationCase.ExternalID, string(evaluationCase.QueryLanguage), string(evaluationCase.AssetLanguage),
			evaluationCase.Question, evaluationCase.ReferenceAnswer, evaluationCase.Category,
			string(evaluationCase.AuthoritativeEvidenceLanguage), string(evaluationCase.ExpectedOutcome),
			nullableString(evaluationCase.ExpectedReasonCode), evaluationCase.ReciprocalExternalID, string(evaluationCase.Checksum),
		); err != nil {
			return fmt.Errorf("insert evaluation dataset case: %w", err)
		}
	}
	return nil
}

func insertEvidence(ctx context.Context, transaction pgx.Tx, evidence []domain.ExpectedEvidence) error {
	for _, requirement := range evidence {
		propositions, err := json.Marshal(requirement.RequiredPropositions)
		if err != nil {
			return application.ErrDatasetStore
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO evaluation_case_expected_evidence (
				id, dataset_revision_id, corpus_id, evaluation_case_id, source_alias,
				ordinal, display_locator, canonical_locator, required_propositions
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)`,
			requirement.ID, requirement.DatasetRevisionID, requirement.CorpusID, requirement.CaseID,
			requirement.SourceAlias, requirement.Ordinal, requirement.DisplayLocator, requirement.CanonicalLocator, string(propositions),
		); err != nil {
			return fmt.Errorf("insert evaluation expected evidence: %w", err)
		}
	}
	return nil
}

func insertStarterSelections(ctx context.Context, transaction pgx.Tx, selections []domain.StarterSelection) error {
	for _, selection := range selections {
		if _, err := transaction.Exec(ctx, `
			INSERT INTO evaluation_dataset_starter_case (
				id, dataset_revision_id, corpus_id, evaluation_case_id, rank, query_language,
				case_checksum, is_review_eligible
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			selection.ID, selection.DatasetRevisionID, selection.CorpusID, selection.CaseID, selection.Rank,
			string(selection.QueryLanguage), string(selection.CaseChecksum), selection.ReviewEligible,
		); err != nil {
			return fmt.Errorf("insert evaluation starter selection: %w", err)
		}
	}
	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func importResult(asset application.DatasetAsset, created bool) application.DatasetImportResult {
	return application.DatasetImportResult{
		RevisionID: asset.Revision.ID,
		CorpusID:   asset.Revision.CorpusID,
		DatasetKey: asset.Revision.DatasetKey,
		Created:    created,
		Sources:    len(asset.Sources),
		Cases:      len(asset.Cases),
		Evidence:   len(asset.ExpectedEvidence),
		Starters:   len(asset.StarterSelections),
	}
}

var _ application.DatasetStore = (*DatasetImporter)(nil)
