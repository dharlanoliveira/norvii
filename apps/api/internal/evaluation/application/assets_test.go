package application

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	"github.com/google/uuid"
)

func TestImporterValidatesEveryOwnedDatasetWithoutNetwork(t *testing.T) {
	repositoryRoot := repositoryRoot(t)
	reader := AssetReaderFunc(func(projectRelativePath string) (io.ReadCloser, error) {
		return os.Open(filepath.Join(repositoryRoot, filepath.FromSlash(projectRelativePath)))
	})
	importer := NewImporter(reader)

	for _, testCase := range []ownedDatasetExpectation{
		{
			name: "Brazil LGPD", request: assetRequest("brazil-lgpd", "brazil-lgpd"),
			wantCases: 18, wantSources: 1, wantStarters: 10,
		},
		{
			name: "Brazil anti-corruption", request: assetRequest("brazil-anti-corruption", "brazil-anti-corruption"),
			wantCases: 16, wantSources: 5, wantStarters: 10,
		},
		{
			name: "United States fair housing", request: assetRequest("us-fair-housing-disability-accommodations", "us-fair-housing"),
			wantCases: 18, wantSources: 5, wantStarters: 10,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assertOwnedDataset(t, importer, testCase)
		})
	}
}

type ownedDatasetExpectation struct {
	name         string
	request      AssetRequest
	wantCases    int
	wantSources  int
	wantStarters int
}

func assertOwnedDataset(t *testing.T, importer *Importer, want ownedDatasetExpectation) {
	t.Helper()
	asset, err := importer.Validate(want.request)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(asset.Cases) != want.wantCases || len(asset.Sources) != want.wantSources || len(asset.StarterSelections) != want.wantStarters {
		t.Fatalf("asset counts = cases:%d sources:%d starters:%d, want %d/%d/%d", len(asset.Cases), len(asset.Sources), len(asset.StarterSelections), want.wantCases, want.wantSources, want.wantStarters)
	}
	assertExpectedEvidenceLocators(t, asset)
	assertCompoundSelectorAtomicMappings(t, asset)
	if err := domain.ValidateStarterSelections(asset.Revision, asset.Cases, asset.StarterSelections); err != nil {
		t.Fatalf("validated starter selections = %v", err)
	}
}

func assertExpectedEvidenceLocators(t *testing.T, asset DatasetAsset) {
	t.Helper()
	if len(asset.ExpectedEvidence) == 0 {
		t.Fatal("asset expected evidence is empty")
	}
	for _, evidence := range asset.ExpectedEvidence {
		if evidence.CanonicalLocator == evidence.DisplayLocator || evidence.CanonicalLocator == "" {
			t.Fatalf("expected evidence locator mapping = %+v, want a distinct exact canonical locator", evidence)
		}
	}
}

func TestImporterRejectsUnsafeOrMismatchedAssetRequestsBeforeReading(t *testing.T) {
	reads := 0
	importer := NewImporter(AssetReaderFunc(func(string) (io.ReadCloser, error) {
		reads++
		return nil, nil
	}))
	valid := syntheticRequest()

	requests := []AssetRequest{
		{CorpusID: valid.CorpusID, CorpusKey: valid.CorpusKey, ExpectedDatasetKey: valid.ExpectedDatasetKey, ManifestPath: "/tmp/dataset.manifest.json", JSONLPath: valid.JSONLPath},
		{CorpusID: valid.CorpusID, CorpusKey: valid.CorpusKey, ExpectedDatasetKey: valid.ExpectedDatasetKey, ManifestPath: "data/corpora/example/evaluation/../dataset.manifest.json", JSONLPath: valid.JSONLPath},
		{CorpusID: valid.CorpusID, CorpusKey: valid.CorpusKey, ExpectedDatasetKey: valid.ExpectedDatasetKey, ManifestPath: valid.ManifestPath, JSONLPath: "data/corpora/other/evaluation/example-v1.jsonl"},
		{CorpusID: valid.CorpusID, CorpusKey: valid.CorpusKey, ManifestPath: valid.ManifestPath, JSONLPath: valid.JSONLPath},
	}
	for _, request := range requests {
		if _, err := importer.Validate(request); !errors.Is(err, ErrAssetValidation) {
			t.Fatalf("Validate(%+v) error = %v, want asset validation error", request, err)
		}
	}
	if reads != 0 {
		t.Fatalf("asset reader calls = %d, want zero for invalid paths", reads)
	}
}

func TestImporterRejectsOversizedAssets(t *testing.T) {
	request := syntheticRequest()
	reads := 0
	oversized := &countingAsset{remaining: int64(maxManifestBytes + 1024)}
	importer := NewImporter(AssetReaderFunc(func(projectRelativePath string) (io.ReadCloser, error) {
		reads++
		if projectRelativePath == request.ManifestPath {
			return oversized, nil
		}
		return io.NopCloser(strings.NewReader(syntheticCases(1, 1, "", ""))), nil
	}))

	if _, err := importer.Validate(request); !errors.Is(err, ErrAssetValidation) {
		t.Fatalf("Validate() error = %v, want asset validation error", err)
	}
	if reads != 1 {
		t.Fatalf("asset reader calls = %d, want only the oversized manifest read", reads)
	}
	if oversized.readBytes != int64(maxManifestBytes+1) {
		t.Fatalf("oversized reader bytes = %d, want %d", oversized.readBytes, maxManifestBytes+1)
	}
}

func TestImporterRejectsUnsupportedCorpusIDsBeforeReading(t *testing.T) {
	reads := 0
	importer := NewImporter(AssetReaderFunc(func(string) (io.ReadCloser, error) {
		reads++
		return nil, nil
	}))
	informationSecurityCorpusID := uuid.MustParse("10000000-0000-4000-8000-000000000099")

	for _, request := range []AssetRequest{
		assetRequest("brazil-lgpd", "brazil-lgpd"),
		assetRequest("brazil-anti-corruption", "brazil-anti-corruption"),
		assetRequest("us-fair-housing-disability-accommodations", "us-fair-housing"),
	} {
		request.CorpusID = informationSecurityCorpusID
		if _, err := importer.Validate(request); !errors.Is(err, ErrAssetValidation) {
			t.Fatalf("Validate(%s) error = %v, want asset validation error", request.CorpusKey, err)
		}
	}
	if reads != 0 {
		t.Fatalf("asset reader calls = %d, want zero for unsupported corpus IDs", reads)
	}
}

func TestImporterRejectsTypedNilAssetReader(t *testing.T) {
	var reader AssetReaderFunc
	importer := NewImporter(reader)

	if _, err := importer.Validate(assetRequest("brazil-lgpd", "brazil-lgpd")); !errors.Is(err, ErrAssetValidation) {
		t.Fatalf("Validate() error = %v, want asset validation error", err)
	}
}

func TestImporterRejectsStrictJSONViolationsAndUnsafeRecords(t *testing.T) {
	validManifest := syntheticManifest()
	validCases := syntheticCases(1, 1, "", "")
	testCases := []struct {
		name     string
		manifest string
		cases    string
	}{
		{
			name: "unknown manifest field", manifest: strings.Replace(validManifest, "\n}", ",\n  \"unexpected\": true\n}", 1), cases: validCases,
		},
		{
			name: "duplicate manifest field", manifest: strings.Replace(validManifest, "\n}", ",\n  \"status\": \"draft\"\n}", 1), cases: validCases,
		},
		{
			name: "unknown case field", manifest: validManifest, cases: strings.Replace(validCases, "}\n{", ",\"unexpected\":true}\n{", 1),
		},
		{
			name: "unrecognized evidence source", manifest: validManifest, cases: strings.Replace(validCases, "\"official-law\"", "\"other-law\"", 1),
		},
		{
			name: "missing canonical evidence locator", manifest: validManifest, cases: strings.Replace(validCases, `,"canonical_locator":"article:1"`, "", 1),
		},
		{
			name: "human canonical evidence locator", manifest: validManifest, cases: strings.Replace(validCases, "\"article:1\"", "\"Article 1\"", 1),
		},
		{
			name: "duplicate atomic canonical evidence locator", manifest: validManifest, cases: duplicateAtomicEvidence(validCases),
		},
		{
			name: "duplicate case identifier", manifest: validManifest, cases: strings.Replace(validCases, `{"id":"case-001-pt"`, `{"id":"case-001-en"`, 1),
		},
		{
			name: "broken reciprocal pair", manifest: validManifest, cases: strings.Replace(validCases, "case-001-pt\",\"starter_rank", "case-002-pt\",\"starter_rank", 1),
		},
		{
			name: "asymmetric starter rank", manifest: validManifest, cases: syntheticCases(1, 2, "", ""),
		},
		{
			name: "invalid abstention reason", manifest: validManifest, cases: syntheticCases(1, 1, "abstain", ""),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			importer := NewImporter(newMemoryReader(map[string]string{
				syntheticRequest().ManifestPath: testCase.manifest,
				syntheticRequest().JSONLPath:    testCase.cases,
			}))
			if _, err := importer.Validate(syntheticRequest()); !errors.Is(err, ErrAssetValidation) {
				t.Fatalf("Validate() error = %v, want asset validation error", err)
			}
		})
	}
}

func TestImporterAcceptsExpandedAtomicSelectorsWithOneDisplayLocator(t *testing.T) {
	request := syntheticRequest()
	asset, err := NewImporter(newMemoryReader(map[string]string{
		request.ManifestPath: syntheticManifest(),
		request.JSONLPath:    expandedAtomicEvidence(syntheticCases(1, 1, "", "")),
	})).Validate(request)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(asset.ExpectedEvidence) != 4 {
		t.Fatalf("expected evidence count = %d, want four atomic records", len(asset.ExpectedEvidence))
	}
	for _, evidence := range asset.ExpectedEvidence {
		if evidence.DisplayLocator != "Article 1" || !strings.HasPrefix(evidence.CanonicalLocator, "article:1/item:") {
			t.Fatalf("expanded evidence = %+v, want preserved display locator and an atomic canonical locator", evidence)
		}
	}
}

func TestImporterUsesCanonicalHashesAndValidatesAbstentionPairs(t *testing.T) {
	request := syntheticRequest()
	manifest := syntheticManifest()
	cases := syntheticCases(1, 1, "abstain", "not-enough-evidence")
	importer := NewImporter(newMemoryReader(map[string]string{
		request.ManifestPath: manifest,
		request.JSONLPath:    cases,
	}))
	first, err := importer.Validate(request)
	if err != nil {
		t.Fatalf("first Validate() error = %v", err)
	}

	formattedManifest := strings.ReplaceAll(manifest, "  ", "    ")
	formattedCases := strings.ReplaceAll(cases, ",", ", ")
	second, err := NewImporter(newMemoryReader(map[string]string{
		request.ManifestPath: formattedManifest,
		request.JSONLPath:    formattedCases,
	})).Validate(request)
	if err != nil {
		t.Fatalf("second Validate() error = %v", err)
	}
	if first.Revision.ManifestSHA256 != second.Revision.ManifestSHA256 || first.Revision.JSONLSHA256 != second.Revision.JSONLSHA256 || first.Revision.ContentSHA256 != second.Revision.ContentSHA256 {
		t.Fatalf("canonical hashes changed: first=%+v second=%+v", first.Revision, second.Revision)
	}
	if first.Cases[0].ExpectedOutcome != domain.ExpectedOutcomeAbstain || first.Cases[0].ExpectedReasonCode != "not-enough-evidence" {
		t.Fatalf("abstention case = %+v, want preserved abstention outcome", first.Cases[0])
	}
}

func assetRequest(corpusKey, datasetKey string) AssetRequest {
	datasetID := datasetKey + "-v1"
	supported, found := supportedAssetCorpusFor(corpusKey)
	if !found {
		panic("test asset request has an unsupported corpus key")
	}
	return AssetRequest{
		CorpusID: supported.corpusID, CorpusKey: corpusKey, ExpectedDatasetKey: datasetKey,
		ManifestPath: "data/corpora/" + corpusKey + "/evaluation/" + datasetID + ".manifest.json",
		JSONLPath:    "data/corpora/" + corpusKey + "/evaluation/" + datasetID + ".jsonl",
	}
}

func syntheticRequest() AssetRequest { return assetRequest("brazil-lgpd", "brazil-lgpd") }

func newMemoryReader(assets map[string]string) AssetReader {
	return AssetReaderFunc(func(projectRelativePath string) (io.ReadCloser, error) {
		contents, found := assets[projectRelativePath]
		if !found {
			return nil, os.ErrNotExist
		}
		return io.NopCloser(strings.NewReader(contents)), nil
	})
}

func syntheticManifest() string {
	return `{
  "dataset_id": "brazil-lgpd-v1",
  "status": "draft",
  "snapshot_date": "2026-08-25",
  "jurisdiction": "Test",
  "authoritative_source_language": "en",
  "supported_query_languages": ["en", "pt-BR"],
  "source_manifest": [{
    "id": "official-law",
    "title": "Official law",
    "authority": "Test authority",
    "document_type": "law",
    "official_url": "https://example.test/law"
  }]
}`
}

func syntheticCases(englishRank, portugueseRank int, outcome, reason string) string {
	expectedOutcome := ""
	if outcome != "" {
		expectedOutcome = `,"expected_outcome":"` + outcome + `","expected_reason_code":"` + reason + `"`
	}
	return `{"id":"case-001-en","language":"en","question":"Which synthetic obligation applies?","expected_answer":"A synthetic obligation applies.","expected_evidence":[{"source_id":"official-law","locator":"Article 1","canonical_locator":"article:1","required_propositions":["synthetic obligation"]}],"category":"synthetic","paired_item_id":"case-001-pt","starter_rank":` + integer(englishRank) + `,"source_language":"en"` + expectedOutcome + `}
{"id":"case-001-pt","language":"pt-BR","question":"Which translated synthetic obligation applies?","expected_answer":"A translated synthetic obligation applies.","expected_evidence":[{"source_id":"official-law","locator":"Article 1","canonical_locator":"article:1","required_propositions":["synthetic obligation"]}],"category":"synthetic","paired_item_id":"case-001-en","starter_rank":` + integer(portugueseRank) + `,"source_language":"en"` + expectedOutcome + `}`
}

func expandedAtomicEvidence(cases string) string {
	return strings.ReplaceAll(cases,
		`[{"source_id":"official-law","locator":"Article 1","canonical_locator":"article:1","required_propositions":["synthetic obligation"]}]`,
		`[{"source_id":"official-law","locator":"Article 1","canonical_locator":"article:1/item:a","required_propositions":["first synthetic obligation"]},{"source_id":"official-law","locator":"Article 1","canonical_locator":"article:1/item:b","required_propositions":["second synthetic obligation"]}]`,
	)
}

func duplicateAtomicEvidence(cases string) string {
	return strings.Replace(expandedAtomicEvidence(cases), "article:1/item:b", "article:1/item:a", 1)
}

func assertCompoundSelectorAtomicMappings(t *testing.T, asset DatasetAsset) {
	t.Helper()

	type selectorExpectation struct {
		caseID            string
		sourceAlias       string
		displayLocator    string
		canonicalLocators []string
	}
	expectations := []selectorExpectation{
		{"lgpd-002-pt", "lgpd", "Art. 5\u00ba, VI e VII", []string{"article:5/item:vi", "article:5/item:vii"}},
		{"lgpd-002-en", "lgpd", "Art. 5\u00ba, VI e VII", []string{"article:5/item:vi", "article:5/item:vii"}},
		{"lgpd-003-pt", "lgpd", "Art. 6\u00ba, I e III", []string{"article:6/item:i", "article:6/item:iii"}},
		{"lgpd-003-en", "lgpd", "Art. 6\u00ba, I e III", []string{"article:6/item:i", "article:6/item:iii"}},
		{"lgpd-004-pt", "lgpd", "Art. 18, caput, I e II", []string{"article:18", "article:18/item:i", "article:18/item:ii"}},
		{"lgpd-004-en", "lgpd", "Art. 18, caput, I e II", []string{"article:18", "article:18/item:i", "article:18/item:ii"}},
		{"brac-003-pt", "lac", "Art. 6\u00ba, I e II", []string{"article:6/item:i", "article:6/item:ii"}},
		{"brac-003-en", "lac", "Art. 6\u00ba, I e II", []string{"article:6/item:i", "article:6/item:ii"}},
		{"brac-007-pt", "aml-law", "Arts. 10 e 11", []string{"article:10", "article:11"}},
		{"brac-007-en", "aml-law", "Arts. 10 e 11", []string{"article:10", "article:11"}},
		{"fh-004-en", "fair-housing-act-3604", "42 U.S.C. \u00a7 3604(f)(3)(A)-(B)", []string{"us-code:42/section:3604/item:f/item:3/item:a", "us-code:42/section:3604/item:f/item:3/item:b"}},
		{"fh-004-pt", "fair-housing-act-3604", "42 U.S.C. \u00a7 3604(f)(3)(A)-(B)", []string{"us-code:42/section:3604/item:f/item:3/item:a", "us-code:42/section:3604/item:f/item:3/item:b"}},
		{"fh-005-en", "hud-assistance-animals", "What Is an Assistance Animal?; Obligations of Housing Providers", []string{"section:what-is-an-assistance-animal", "section:obligations-of-housing-providers"}},
		{"fh-005-pt", "hud-assistance-animals", "What Is an Assistance Animal?; Obligations of Housing Providers", []string{"section:what-is-an-assistance-animal", "section:obligations-of-housing-providers"}},
	}

	caseExternalIDs := make(map[uuid.UUID]string, len(asset.Cases))
	availableExternalIDs := make(map[string]struct{}, len(asset.Cases))
	for _, evaluationCase := range asset.Cases {
		caseExternalIDs[evaluationCase.ID] = evaluationCase.ExternalID
		availableExternalIDs[evaluationCase.ExternalID] = struct{}{}
	}
	actual := make(map[string][]string, len(expectations))
	expected := make(map[string][]string, len(expectations))
	for _, expectation := range expectations {
		if _, belongsToAsset := availableExternalIDs[expectation.caseID]; !belongsToAsset {
			continue
		}
		key := expectation.caseID + "\x00" + expectation.sourceAlias + "\x00" + expectation.displayLocator
		expected[key] = expectation.canonicalLocators
	}
	for _, evidence := range asset.ExpectedEvidence {
		key := caseExternalIDs[evidence.CaseID] + "\x00" + evidence.SourceAlias + "\x00" + evidence.DisplayLocator
		if _, required := expected[key]; required {
			actual[key] = append(actual[key], evidence.CanonicalLocator)
		}
	}
	for key, want := range expected {
		if got := actual[key]; strings.Join(got, "\x00") != strings.Join(want, "\x00") {
			t.Fatalf("compound selector atomic mapping for %q = %v, want %v", key, got, want)
		}
	}
}

func integer(value int) string { return strconv.Itoa(value) }

type countingAsset struct {
	remaining int64
	readBytes int64
}

func (asset *countingAsset) Read(destination []byte) (int, error) {
	if asset.remaining == 0 {
		return 0, io.EOF
	}
	count := int64(len(destination))
	if count > asset.remaining {
		count = asset.remaining
	}
	asset.remaining -= count
	asset.readBytes += count
	for index := range destination[:count] {
		destination[index] = 'x'
	}
	return int(count), nil
}

func (*countingAsset) Close() error { return nil }

func repositoryRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	return filepath.Clean(filepath.Join(directory, "../../../../../"))
}
