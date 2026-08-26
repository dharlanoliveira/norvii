//go:build integration

package integration_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	evaluationpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/postgres"
	"github.com/jackc/pgx/v5"
)

func TestEvaluationDatasetImporterImportsOwnedAssetsIdempotently(t *testing.T) {
	ctx, connection := openEvaluationSchemaConnection(t)
	reader := newRepositoryDatasetReader(t)
	service := application.NewDatasetImportService(
		application.NewImporter(reader),
		evaluationpostgres.NewDatasetImporter(connection),
	)

	first := make([]application.DatasetImportResult, 0, 3)
	for _, request := range application.OwnedAssetRequests() {
		result, err := service.Import(ctx, request)
		if err != nil {
			t.Fatalf("first Import(%s) error = %v", request.CorpusKey, err)
		}
		first = append(first, result)
	}
	for index, request := range application.OwnedAssetRequests() {
		result, err := service.Import(ctx, request)
		if err != nil {
			t.Fatalf("second Import(%s) error = %v", request.CorpusKey, err)
		}
		if result.Created {
			t.Fatalf("second Import(%s) created a duplicate revision", request.CorpusKey)
		}
		if result.RevisionID != first[index].RevisionID || result.Cases != first[index].Cases {
			t.Fatalf("repeat import result = %+v, want stable identity and count %+v", result, first[index])
		}
	}
	for assetPath := range expectedDatasetAssetPaths {
		if reader.opened[assetPath] != 2 {
			t.Fatalf("local asset reads for %s = %d, want two and no external reader", assetPath, reader.opened[assetPath])
		}
	}

	var revisions, cases, evidence, canonicalEvidence int
	if err := connection.QueryRow(ctx, `
		SELECT count(*)
		FROM evaluation_dataset_revision
		WHERE corpus_id = ANY($1::uuid[])`, evaluationDatasetCorpusIDs()).Scan(&revisions); err != nil {
		t.Fatalf("count imported revisions: %v", err)
	}
	if err := connection.QueryRow(ctx, `
		SELECT count(*)
		FROM evaluation_dataset_case
		WHERE corpus_id = ANY($1::uuid[])`, evaluationDatasetCorpusIDs()).Scan(&cases); err != nil {
		t.Fatalf("count imported cases: %v", err)
	}
	if revisions != 3 || cases != 52 {
		t.Fatalf("imported revision/case counts = %d/%d, want 3/52", revisions, cases)
	}
	assertImportedLanguagePairs(t, ctx, connection)
	if err := connection.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE canonical_locator IS NOT NULL AND canonical_locator <> display_locator)
		FROM evaluation_case_expected_evidence
		WHERE corpus_id = ANY($1::uuid[])`, evaluationDatasetCorpusIDs()).Scan(&evidence, &canonicalEvidence); err != nil {
		t.Fatalf("count imported canonical evidence locators: %v", err)
	}
	if evidence != 76 || canonicalEvidence != 76 {
		t.Fatalf("imported canonical evidence locators = %d/%d, want 76/76", evidence, canonicalEvidence)
	}
}

func assertImportedLanguagePairs(t *testing.T, ctx context.Context, connection *pgx.Conn) {
	t.Helper()

	rows, err := connection.Query(ctx, `
		SELECT
			case_record.corpus_id::text,
			count(*) AS total_cases,
			count(*) FILTER (WHERE case_record.query_language = 'en') AS english_cases,
			count(*) FILTER (WHERE case_record.query_language = 'pt') AS portuguese_cases,
			count(*) FILTER (
				WHERE reciprocal_case.id IS NOT NULL
					AND reciprocal_case.query_language <> case_record.query_language
					AND reciprocal_case.reciprocal_case_external_id = case_record.external_case_id
			) AS reciprocal_cases
		FROM evaluation_dataset_case AS case_record
		LEFT JOIN evaluation_dataset_case AS reciprocal_case
			ON reciprocal_case.dataset_revision_id = case_record.dataset_revision_id
			AND reciprocal_case.corpus_id = case_record.corpus_id
			AND reciprocal_case.external_case_id = case_record.reciprocal_case_external_id
		WHERE case_record.corpus_id = ANY($1::uuid[])
		GROUP BY case_record.corpus_id
		ORDER BY case_record.corpus_id`, evaluationDatasetCorpusIDs())
	if err != nil {
		t.Fatalf("query imported reciprocal language pairs: %v", err)
	}
	defer rows.Close()

	wantCases := map[string]int{
		"10000000-0000-4000-8000-000000000001": 18,
		"10000000-0000-4000-8000-000000000003": 16,
		"10000000-0000-4000-8000-000000000004": 18,
	}
	found := make(map[string]bool, len(wantCases))
	for rows.Next() {
		var corpusID string
		var totalCases, englishCases, portugueseCases, reciprocalCases int
		if err := rows.Scan(&corpusID, &totalCases, &englishCases, &portugueseCases, &reciprocalCases); err != nil {
			t.Fatalf("scan imported reciprocal language pairs: %v", err)
		}
		wantTotal, knownCorpus := wantCases[corpusID]
		if !knownCorpus {
			t.Fatalf("imported language pair corpus = %s, want only owned corpus IDs", corpusID)
		}
		if totalCases != wantTotal || englishCases != wantTotal/2 || portugueseCases != wantTotal/2 || reciprocalCases != wantTotal {
			t.Fatalf(
				"imported language pairs for corpus %s = total %d, en %d, pt %d, reciprocal %d; want %d paired cases",
				corpusID,
				totalCases,
				englishCases,
				portugueseCases,
				reciprocalCases,
				wantTotal,
			)
		}
		found[corpusID] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate imported reciprocal language pairs: %v", err)
	}
	if len(found) != len(wantCases) {
		t.Fatalf("imported language pair corpora = %v, want all three owned corpora", found)
	}
}

func evaluationDatasetCorpusIDs() []string {
	return []string{
		"10000000-0000-4000-8000-000000000001",
		"10000000-0000-4000-8000-000000000003",
		"10000000-0000-4000-8000-000000000004",
	}
}

type repositoryDatasetReader struct {
	repositoryRoot string
	opened         map[string]int
}

func newRepositoryDatasetReader(t *testing.T) *repositoryDatasetReader {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	return &repositoryDatasetReader{
		repositoryRoot: filepath.Clean(filepath.Join(workingDirectory, "../../../../")),
		opened:         make(map[string]int),
	}
}

func (reader *repositoryDatasetReader) OpenAsset(projectRelativePath string) (io.ReadCloser, error) {
	if _, allowed := expectedDatasetAssetPaths[projectRelativePath]; !allowed {
		return nil, os.ErrPermission
	}
	reader.opened[projectRelativePath]++
	return os.Open(filepath.Join(reader.repositoryRoot, filepath.FromSlash(projectRelativePath)))
}

var expectedDatasetAssetPaths = map[string]struct{}{
	"data/corpora/brazil-lgpd/evaluation/brazil-lgpd-v1.manifest.json":                                   {},
	"data/corpora/brazil-lgpd/evaluation/brazil-lgpd-v1.jsonl":                                           {},
	"data/corpora/brazil-anti-corruption/evaluation/brazil-anti-corruption-v1.manifest.json":             {},
	"data/corpora/brazil-anti-corruption/evaluation/brazil-anti-corruption-v1.jsonl":                     {},
	"data/corpora/us-fair-housing-disability-accommodations/evaluation/us-fair-housing-v1.manifest.json": {},
	"data/corpora/us-fair-housing-disability-accommodations/evaluation/us-fair-housing-v1.jsonl":         {},
}
