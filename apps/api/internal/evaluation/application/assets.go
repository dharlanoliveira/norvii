// Package application validates project-owned evaluation dataset assets.
package application

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	"github.com/google/uuid"
)

const (
	maxManifestBytes              = 256 * 1024
	maxJSONLBytes                 = 2 * 1024 * 1024
	maxJSONLLine                  = 64 * 1024
	maxCaseCount                  = 256
	brazilLGPDCorpusKey           = "brazil-lgpd"
	brazilAntiCorruptionCorpusKey = "brazil-anti-corruption"
	manifestFileSuffix            = ".manifest.json"
	jsonlFileSuffix               = ".jsonl"
)

var (
	// ErrAssetValidation identifies an invalid, unsafe, or unsupported local asset.
	ErrAssetValidation           = errors.New("evaluation asset validation failed")
	assetPathPattern             = regexp.MustCompile(`^data/corpora/[a-z0-9][a-z0-9-]{0,127}/evaluation/[a-z0-9][a-z0-9._-]{0,127}$`)
	datasetIDPattern             = regexp.MustCompile(`^([a-z0-9][a-z0-9-]{0,127})-(v[0-9]+)$`)
	identifierPattern            = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,127}$`)
	assetNamespace               = uuid.MustParse("c1dddb7d-ff8d-4bae-ae3b-b476aa7c13a8")
	brazilLGPDCorpusID           = uuid.MustParse("10000000-0000-4000-8000-000000000001")
	brazilAntiCorruptionCorpusID = uuid.MustParse("10000000-0000-4000-8000-000000000003")
	usFairHousingCorpusID        = uuid.MustParse("10000000-0000-4000-8000-000000000004")
)

type supportedAssetCorpus struct {
	corpusID   uuid.UUID
	datasetKey string
}

// AssetReader is the local-boundary port used to stream project-owned assets.
// Implementations must not fetch remote resources or materialize assets before
// the importer applies its byte limit.
type AssetReader interface {
	OpenAsset(projectRelativePath string) (io.ReadCloser, error)
}

// AssetReaderFunc adapts a function into an AssetReader.
type AssetReaderFunc func(projectRelativePath string) (io.ReadCloser, error)

// OpenAsset implements AssetReader.
func (reader AssetReaderFunc) OpenAsset(projectRelativePath string) (io.ReadCloser, error) {
	return reader(projectRelativePath)
}

// AssetRequest identifies one manifest and JSONL pair beneath a corpus evaluation directory.
type AssetRequest struct {
	CorpusID           uuid.UUID
	CorpusKey          string
	ExpectedDatasetKey string
	ManifestPath       string
	JSONLPath          string
}

// OwnedAssetRequests returns fresh requests for every repository-owned evaluation dataset.
// The fixed corpus bindings prevent legal assets from being redirected into another corpus.
func OwnedAssetRequests() []AssetRequest {
	return []AssetRequest{
		assetRequestFor(brazilLGPDCorpusKey, brazilLGPDCorpusID, brazilLGPDCorpusKey),
		assetRequestFor(brazilAntiCorruptionCorpusKey, brazilAntiCorruptionCorpusID, brazilAntiCorruptionCorpusKey),
		assetRequestFor("us-fair-housing-disability-accommodations", usFairHousingCorpusID, "us-fair-housing"),
	}
}

func assetRequestFor(corpusKey string, corpusID uuid.UUID, datasetKey string) AssetRequest {
	datasetID := datasetKey + "-v1"
	directory := "data/corpora/" + corpusKey + "/evaluation/"
	return AssetRequest{
		CorpusID: corpusID, CorpusKey: corpusKey, ExpectedDatasetKey: datasetKey,
		ManifestPath: directory + datasetID + manifestFileSuffix,
		JSONLPath:    directory + datasetID + jsonlFileSuffix,
	}
}

// DatasetAsset is a fully validated immutable input for a later persistence operation.
// It deliberately contains no publication, snapshot, network, or model behavior.
type DatasetAsset struct {
	Revision          domain.DatasetRevision
	ManifestPath      string
	JSONLPath         string
	Sources           []domain.SourceRequirement
	Cases             []domain.Case
	ExpectedEvidence  []domain.ExpectedEvidence
	StarterSelections []domain.StarterSelection
}

// Importer validates local manifest and JSONL assets into domain values.
type Importer struct{ reader AssetReader }

// NewImporter constructs an asset-only importer around a caller-owned local reader.
func NewImporter(reader AssetReader) *Importer { return &Importer{reader: reader} }

// Validate verifies one project-owned manifest/JSONL pair without network, database, model, or HTTP work.
func (importer *Importer) Validate(request AssetRequest) (DatasetAsset, error) {
	if importer == nil || importer.reader == nil || isNilAssetReader(importer.reader) || request.CorpusID == uuid.Nil || !safeCorpusKey(request.CorpusKey) ||
		!matchesSupportedCorpus(request) ||
		!safeAssetPaths(request.CorpusKey, request.ManifestPath, request.JSONLPath) {
		return DatasetAsset{}, assetValidationError("unsafe asset request")
	}

	manifestBytes, err := importer.readBounded(request.ManifestPath, maxManifestBytes)
	if err != nil {
		return DatasetAsset{}, err
	}
	jsonlBytes, err := importer.readBounded(request.JSONLPath, maxJSONLBytes)
	if err != nil {
		return DatasetAsset{}, err
	}

	manifest, manifestHash, err := decodeManifest(manifestBytes)
	if err != nil {
		return DatasetAsset{}, err
	}
	records, jsonlHash, err := decodeCases(jsonlBytes)
	if err != nil {
		return DatasetAsset{}, err
	}
	return buildDatasetAsset(request, manifest, manifestHash, records, jsonlHash)
}

func (importer *Importer) readBounded(projectRelativePath string, limit int) ([]byte, error) {
	asset, err := importer.reader.OpenAsset(projectRelativePath)
	if err != nil {
		return nil, assetValidationError("asset read failed")
	}
	if asset == nil {
		return nil, assetValidationError("asset reader returned no content")
	}
	defer asset.Close()
	contents, err := io.ReadAll(io.LimitReader(asset, int64(limit)+1))
	if err != nil {
		return nil, assetValidationError("asset read failed")
	}
	if len(contents) == 0 || len(contents) > limit {
		return nil, assetValidationError("asset size is outside the supported limit")
	}
	return contents, nil
}

type assetManifest struct {
	DatasetID                   string        `json:"dataset_id"`
	Status                      string        `json:"status"`
	SnapshotDate                string        `json:"snapshot_date"`
	Jurisdiction                string        `json:"jurisdiction"`
	AuthoritativeSourceLanguage string        `json:"authoritative_source_language"`
	SupportedQueryLanguages     []string      `json:"supported_query_languages"`
	SourceManifest              []assetSource `json:"source_manifest"`
}

type assetSource struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Authority    string `json:"authority"`
	DocumentType string `json:"document_type"`
	OfficialURL  string `json:"official_url"`
}

type assetCase struct {
	ID                 string                  `json:"id"`
	Language           string                  `json:"language"`
	Question           string                  `json:"question"`
	ExpectedAnswer     string                  `json:"expected_answer"`
	ExpectedEvidence   []assetExpectedEvidence `json:"expected_evidence"`
	Category           string                  `json:"category"`
	PairedItemID       string                  `json:"paired_item_id"`
	StarterRank        *int                    `json:"starter_rank"`
	SourceLanguage     string                  `json:"source_language"`
	ExpectedOutcome    string                  `json:"expected_outcome"`
	ExpectedReasonCode string                  `json:"expected_reason_code"`
}

type assetExpectedEvidence struct {
	SourceID             string   `json:"source_id"`
	Locator              string   `json:"locator"`
	CanonicalLocator     string   `json:"canonical_locator"`
	RequiredPropositions []string `json:"required_propositions"`
}

func decodeManifest(contents []byte) (assetManifest, domain.SHA256, error) {
	if err := validateNoDuplicateFields(contents); err != nil {
		return assetManifest{}, "", err
	}
	var manifest assetManifest
	if err := strictDecode(contents, &manifest); err != nil {
		return assetManifest{}, "", err
	}
	canonical, err := canonicalJSON(contents)
	if err != nil {
		return assetManifest{}, "", err
	}
	return manifest, checksum(canonical), nil
}

func decodeCases(contents []byte) ([]assetCase, domain.SHA256, error) {
	lines := bytes.Split(contents, []byte{'\n'})
	if len(lines) > 1 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 || len(lines) > maxCaseCount {
		return nil, "", assetValidationError("case count is outside the supported limit")
	}

	records := make([]assetCase, 0, len(lines))
	canonicalLines := make([][]byte, 0, len(lines))
	for _, line := range lines {
		line = bytes.TrimSuffix(line, []byte{'\r'})
		if len(line) == 0 || len(line) > maxJSONLLine {
			return nil, "", assetValidationError("case line is outside the supported limit")
		}
		if err := validateNoDuplicateFields(line); err != nil {
			return nil, "", err
		}
		var record assetCase
		if err := strictDecode(line, &record); err != nil {
			return nil, "", err
		}
		canonical, err := canonicalJSON(line)
		if err != nil {
			return nil, "", err
		}
		records = append(records, record)
		canonicalLines = append(canonicalLines, canonical)
	}
	return records, checksum(bytes.Join(canonicalLines, []byte{'\n'})), nil
}

func buildDatasetAsset(
	request AssetRequest,
	manifest assetManifest,
	manifestHash domain.SHA256,
	records []assetCase,
	jsonlHash domain.SHA256,
) (DatasetAsset, error) {
	datasetKey, semanticRevision, err := validateManifest(manifest)
	if err != nil {
		return DatasetAsset{}, err
	}
	if request.ExpectedDatasetKey != datasetKey || !matchesAssetFilenames(request, manifest.DatasetID) {
		return DatasetAsset{}, assetValidationError("asset identity does not match the requested corpus dataset")
	}
	contentHash := checksum([]byte(string(manifestHash) + "\n" + string(jsonlHash)))
	revisionID := deterministicID("revision", request.CorpusID.String(), string(contentHash))
	snapshotDate, _ := time.Parse(time.DateOnly, manifest.SnapshotDate)
	revision := domain.DatasetRevision{
		ID: revisionID, CorpusID: request.CorpusID, DatasetKey: datasetKey, SemanticRevision: semanticRevision,
		Jurisdiction: manifest.Jurisdiction, ManifestSHA256: manifestHash, JSONLSHA256: jsonlHash, ContentSHA256: contentHash,
		DeclaredSnapshotDate: snapshotDate, QueryLanguages: []domain.QueryLanguage{domain.QueryLanguageEnglish, domain.QueryLanguagePortuguese},
		AuthoritativeEvidenceLanguage: domain.AssetLanguage(manifest.AuthoritativeSourceLanguage),
	}
	if err := revision.Validate(); err != nil {
		return DatasetAsset{}, domainValidationError(err)
	}

	sources, err := buildSources(revision, manifest.SourceManifest)
	if err != nil {
		return DatasetAsset{}, err
	}
	cases, evidence, selections, err := buildCases(revision, records, sources)
	if err != nil {
		return DatasetAsset{}, err
	}
	if err := domain.ValidatePairedCases(revision, cases); err != nil {
		return DatasetAsset{}, domainValidationError(err)
	}
	if err := validatePairedOutcomes(cases); err != nil {
		return DatasetAsset{}, err
	}
	if err := domain.ValidateStarterSelections(revision, cases, selections); err != nil {
		return DatasetAsset{}, domainValidationError(err)
	}
	if err := validateStarterRanks(selections); err != nil {
		return DatasetAsset{}, err
	}
	return DatasetAsset{
		Revision: revision, ManifestPath: request.ManifestPath, JSONLPath: request.JSONLPath,
		Sources: sources, Cases: cases, ExpectedEvidence: evidence, StarterSelections: selections,
	}, nil
}

func validateManifest(manifest assetManifest) (string, string, error) {
	matches := datasetIDPattern.FindStringSubmatch(manifest.DatasetID)
	if len(matches) != 3 || manifest.Status != "draft" || strings.TrimSpace(manifest.Jurisdiction) == "" ||
		manifest.AuthoritativeSourceLanguage != string(domain.AssetLanguageEnglish) &&
			manifest.AuthoritativeSourceLanguage != string(domain.AssetLanguagePortugueseBrazil) {
		return "", "", assetValidationError("manifest has invalid required fields")
	}
	if _, err := time.Parse(time.DateOnly, manifest.SnapshotDate); err != nil {
		return "", "", assetValidationError("manifest snapshot date is invalid")
	}
	if len(manifest.SupportedQueryLanguages) != 2 || !containsExactly(manifest.SupportedQueryLanguages, "en", "pt-BR") || len(manifest.SourceManifest) == 0 {
		return "", "", assetValidationError("manifest language or source requirements are invalid")
	}
	return matches[1], matches[2], nil
}

func buildSources(revision domain.DatasetRevision, records []assetSource) ([]domain.SourceRequirement, error) {
	sources := make([]domain.SourceRequirement, 0, len(records))
	aliases := make(map[string]struct{}, len(records))
	for _, record := range records {
		if !safeIdentifier(record.ID) || strings.TrimSpace(record.Title) == "" || strings.TrimSpace(record.Authority) == "" ||
			strings.TrimSpace(record.DocumentType) == "" || !strings.HasPrefix(record.OfficialURL, "https://") {
			return nil, assetValidationError("manifest source is invalid")
		}
		if _, exists := aliases[record.ID]; exists {
			return nil, assetValidationError("manifest source aliases must be unique")
		}
		source := domain.SourceRequirement{
			ID: deterministicID("source", revision.ID.String(), record.ID), DatasetRevisionID: revision.ID, CorpusID: revision.CorpusID,
			SourceAlias: record.ID, Title: record.Title, OfficialURL: record.OfficialURL, IssuingAuthority: record.Authority,
			DocumentType: record.DocumentType, AuthorityRole: authorityRole(record.DocumentType),
		}
		if source.AuthorityRole == "" {
			return nil, assetValidationError("manifest source document type is unsupported")
		}
		if err := source.ValidateAgainst(revision); err != nil {
			return nil, domainValidationError(err)
		}
		aliases[record.ID] = struct{}{}
		sources = append(sources, source)
	}
	return sources, nil
}

func buildCases(
	revision domain.DatasetRevision,
	records []assetCase,
	sources []domain.SourceRequirement,
) ([]domain.Case, []domain.ExpectedEvidence, []domain.StarterSelection, error) {
	cases := make([]domain.Case, 0, len(records))
	evidence := make([]domain.ExpectedEvidence, 0, len(records))
	selections := make([]domain.StarterSelection, 0, 10)
	externalIDs := make(map[string]struct{}, len(records))
	aliases := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		aliases[source.SourceAlias] = struct{}{}
	}
	for position, record := range records {
		if _, exists := externalIDs[record.ID]; exists {
			return nil, nil, nil, assetValidationError("case identifiers must be unique")
		}
		evaluationCase, err := buildCase(revision, position+1, record)
		if err != nil {
			return nil, nil, nil, err
		}
		caseEvidence, err := buildEvidence(revision, evaluationCase, record.ExpectedEvidence, aliases, sources)
		if err != nil {
			return nil, nil, nil, err
		}
		if record.StarterRank != nil {
			selections = append(selections, domain.StarterSelection{
				ID: deterministicID("starter", evaluationCase.ID.String()), DatasetRevisionID: revision.ID, CorpusID: revision.CorpusID,
				CaseID: evaluationCase.ID, Rank: *record.StarterRank, QueryLanguage: evaluationCase.QueryLanguage,
				CaseChecksum: evaluationCase.Checksum, ReviewEligible: true,
			})
		}
		externalIDs[record.ID] = struct{}{}
		cases = append(cases, evaluationCase)
		evidence = append(evidence, caseEvidence...)
	}
	return cases, evidence, selections, nil
}

func buildCase(revision domain.DatasetRevision, position int, record assetCase) (domain.Case, error) {
	queryLanguage, assetLanguage, err := normalizeLanguage(record.Language)
	if err != nil {
		return domain.Case{}, err
	}
	expectedOutcome := domain.ExpectedOutcome(record.ExpectedOutcome)
	if expectedOutcome == "" {
		expectedOutcome = domain.ExpectedOutcomeAnswer
	}
	if !safeIdentifier(record.ID) || !safeIdentifier(record.PairedItemID) || strings.TrimSpace(record.Question) == "" ||
		strings.TrimSpace(record.ExpectedAnswer) == "" || strings.TrimSpace(record.Category) == "" ||
		(record.SourceLanguage != string(domain.AssetLanguageEnglish) && record.SourceLanguage != string(domain.AssetLanguagePortugueseBrazil)) ||
		(expectedOutcome == domain.ExpectedOutcomeAnswer && record.ExpectedReasonCode != "") ||
		(expectedOutcome == domain.ExpectedOutcomeAbstain && !safeIdentifier(record.ExpectedReasonCode)) ||
		(expectedOutcome != domain.ExpectedOutcomeAnswer && expectedOutcome != domain.ExpectedOutcomeAbstain) {
		return domain.Case{}, assetValidationError("case fields or expected outcome are invalid")
	}
	caseChecksum, err := canonicalCaseChecksum(record)
	if err != nil {
		return domain.Case{}, err
	}
	evaluationCase := domain.Case{
		ID: deterministicID("case", revision.ID.String(), record.ID), DatasetRevisionID: revision.ID, CorpusID: revision.CorpusID,
		Position: position, ExternalID: record.ID, QueryLanguage: queryLanguage, AssetLanguage: assetLanguage,
		Question: record.Question, ReferenceAnswer: record.ExpectedAnswer, Category: record.Category,
		AuthoritativeEvidenceLanguage: domain.AssetLanguage(record.SourceLanguage), ExpectedOutcome: expectedOutcome,
		ExpectedReasonCode: record.ExpectedReasonCode, ReciprocalExternalID: record.PairedItemID, Checksum: caseChecksum,
	}
	if err := evaluationCase.ValidateAgainst(revision); err != nil {
		return domain.Case{}, domainValidationError(err)
	}
	return evaluationCase, nil
}

func buildEvidence(
	revision domain.DatasetRevision,
	evaluationCase domain.Case,
	records []assetExpectedEvidence,
	aliases map[string]struct{},
	sources []domain.SourceRequirement,
) ([]domain.ExpectedEvidence, error) {
	if len(records) == 0 {
		return nil, assetValidationError("case expected evidence is required")
	}
	evidence := make([]domain.ExpectedEvidence, 0, len(records))
	seenSelectors := make(map[string]struct{}, len(records))
	for index, record := range records {
		if _, found := aliases[record.SourceID]; !found || strings.TrimSpace(record.Locator) == "" ||
			strings.TrimSpace(record.CanonicalLocator) == "" || len(record.RequiredPropositions) == 0 {
			return nil, assetValidationError("case expected evidence is invalid")
		}
		selector := record.SourceID + "\x00" + record.CanonicalLocator
		if _, duplicate := seenSelectors[selector]; duplicate {
			return nil, assetValidationError("case expected evidence selectors must be unique")
		}
		seenSelectors[selector] = struct{}{}
		propositions := make(map[string]struct{}, len(record.RequiredPropositions))
		for _, proposition := range record.RequiredPropositions {
			if strings.TrimSpace(proposition) == "" {
				return nil, assetValidationError("case expected evidence propositions are invalid")
			}
			if _, duplicate := propositions[proposition]; duplicate {
				return nil, assetValidationError("case expected evidence propositions must be unique")
			}
			propositions[proposition] = struct{}{}
		}
		item := domain.ExpectedEvidence{
			ID:                deterministicID("evidence", evaluationCase.ID.String(), fmt.Sprintf("%d", index+1)),
			DatasetRevisionID: revision.ID, CorpusID: revision.CorpusID, CaseID: evaluationCase.ID,
			SourceAlias: record.SourceID, Ordinal: index + 1, DisplayLocator: record.Locator, CanonicalLocator: record.CanonicalLocator,
			RequiredPropositions: append([]string(nil), record.RequiredPropositions...),
		}
		if err := item.ValidateAgainst(revision, evaluationCase, sources); err != nil {
			return nil, domainValidationError(err)
		}
		evidence = append(evidence, item)
	}
	return evidence, nil
}

func canonicalCaseChecksum(record assetCase) (domain.SHA256, error) {
	contents, err := json.Marshal(record)
	if err != nil {
		return "", assetValidationError("case checksum serialization failed")
	}
	canonical, err := canonicalJSON(contents)
	if err != nil {
		return "", err
	}
	return checksum(canonical), nil
}

func normalizeLanguage(value string) (domain.QueryLanguage, domain.AssetLanguage, error) {
	switch value {
	case "en":
		return domain.QueryLanguageEnglish, domain.AssetLanguageEnglish, nil
	case "pt-BR":
		return domain.QueryLanguagePortuguese, domain.AssetLanguagePortugueseBrazil, nil
	default:
		return "", "", assetValidationError("case language is unsupported")
	}
}

func authorityRole(documentType string) string {
	switch documentType {
	case "law", "decree-law", "statute":
		return "statute"
	case "resolution", "current-regulation-web-edition":
		return "regulation"
	case "official-guidance-page", "guidance-pdf":
		return "guidance"
	case "official-procedure-page":
		return "procedure"
	default:
		return ""
	}
}

func safeAssetPaths(corpusKey, manifestPath, jsonlPath string) bool {
	if !assetPathPattern.MatchString(manifestPath) || !assetPathPattern.MatchString(jsonlPath) ||
		path.IsAbs(manifestPath) || path.IsAbs(jsonlPath) || strings.Contains(manifestPath, "..") || strings.Contains(jsonlPath, "..") {
		return false
	}
	prefix := "data/corpora/" + corpusKey + "/evaluation/"
	return strings.HasPrefix(manifestPath, prefix) && strings.HasPrefix(jsonlPath, prefix) &&
		strings.HasSuffix(manifestPath, manifestFileSuffix) && strings.HasSuffix(jsonlPath, jsonlFileSuffix)
}

func matchesSupportedCorpus(request AssetRequest) bool {
	supported, found := supportedAssetCorpusFor(request.CorpusKey)
	return found && request.CorpusID == supported.corpusID && request.ExpectedDatasetKey == supported.datasetKey
}

// supportedAssetCorpusFor is an immutable manifest binding. It prevents a
// caller from importing a legal dataset into another corpus, including an
// information-security corpus.
func supportedAssetCorpusFor(corpusKey string) (supportedAssetCorpus, bool) {
	switch corpusKey {
	case brazilLGPDCorpusKey:
		return supportedAssetCorpus{corpusID: brazilLGPDCorpusID, datasetKey: brazilLGPDCorpusKey}, true
	case brazilAntiCorruptionCorpusKey:
		return supportedAssetCorpus{corpusID: brazilAntiCorruptionCorpusID, datasetKey: brazilAntiCorruptionCorpusKey}, true
	case "us-fair-housing-disability-accommodations":
		return supportedAssetCorpus{corpusID: usFairHousingCorpusID, datasetKey: "us-fair-housing"}, true
	default:
		return supportedAssetCorpus{}, false
	}
}

func isNilAssetReader(reader AssetReader) bool {
	function, isFunction := reader.(AssetReaderFunc)
	return isFunction && function == nil
}

func matchesAssetFilenames(request AssetRequest, datasetID string) bool {
	return path.Base(request.ManifestPath) == datasetID+manifestFileSuffix && path.Base(request.JSONLPath) == datasetID+jsonlFileSuffix
}

func validatePairedOutcomes(cases []domain.Case) error {
	byExternalID := make(map[string]domain.Case, len(cases))
	for _, evaluationCase := range cases {
		byExternalID[evaluationCase.ExternalID] = evaluationCase
	}
	for _, evaluationCase := range cases {
		pairedCase := byExternalID[evaluationCase.ReciprocalExternalID]
		if pairedCase.ExpectedOutcome != evaluationCase.ExpectedOutcome || pairedCase.ExpectedReasonCode != evaluationCase.ExpectedReasonCode {
			return assetValidationError("reciprocal case outcomes must match")
		}
	}
	return nil
}

func validateStarterRanks(selections []domain.StarterSelection) error {
	if len(selections) == 0 {
		return nil
	}
	rankCounts := make(map[int]int, 5)
	maxRank := 0
	for _, selection := range selections {
		rankCounts[selection.Rank]++
		if selection.Rank > maxRank {
			maxRank = selection.Rank
		}
	}
	for rank := 1; rank <= maxRank; rank++ {
		if rankCounts[rank] != 2 {
			return assetValidationError("starter ranks must contain one reciprocal language pair")
		}
	}
	return nil
}

func safeCorpusKey(value string) bool { return safeIdentifier(value) }

func safeIdentifier(value string) bool {
	return identifierPattern.MatchString(value)
}

func containsExactly(values []string, expected ...string) bool {
	if len(values) != len(expected) {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	for _, value := range expected {
		if _, exists := seen[value]; !exists {
			return false
		}
	}
	return true
}

func deterministicID(parts ...string) uuid.UUID {
	return uuid.NewSHA1(assetNamespace, []byte(strings.Join(parts, "\x00")))
}

func checksum(contents []byte) domain.SHA256 {
	sum := sha256.Sum256(contents)
	return domain.SHA256(hex.EncodeToString(sum[:]))
}

func strictDecode(contents []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return assetValidationError("asset JSON structure is invalid")
	}
	if err := expectEOF(decoder); err != nil {
		return assetValidationError("asset JSON contains trailing data")
	}
	return nil
}

func canonicalJSON(contents []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil || expectEOF(decoder) != nil {
		return nil, assetValidationError("asset JSON canonicalization failed")
	}
	var canonical bytes.Buffer
	if err := writeCanonicalJSON(&canonical, value); err != nil {
		return nil, assetValidationError("asset JSON canonicalization failed")
	}
	return canonical.Bytes(), nil
}

func writeCanonicalJSON(destination *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		destination.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				destination.WriteByte(',')
			}
			encodedKey, err := json.Marshal(key)
			if err != nil {
				return err
			}
			destination.Write(encodedKey)
			destination.WriteByte(':')
			if err := writeCanonicalJSON(destination, typed[key]); err != nil {
				return err
			}
		}
		destination.WriteByte('}')
	case []any:
		destination.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				destination.WriteByte(',')
			}
			if err := writeCanonicalJSON(destination, item); err != nil {
				return err
			}
		}
		destination.WriteByte(']')
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return err
		}
		destination.Write(encoded)
	}
	return nil
}

func validateNoDuplicateFields(contents []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(contents))
	if err := consumeJSONValue(decoder); err != nil || expectEOF(decoder) != nil {
		return assetValidationError("asset JSON contains duplicate or malformed fields")
	}
	return nil
}

func consumeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		fields := make(map[string]struct{})
		for decoder.More() {
			field, err := decoder.Token()
			if err != nil {
				return err
			}
			name, isName := field.(string)
			if !isName {
				return errors.New("JSON object key is invalid")
			}
			if _, exists := fields[name]; exists {
				return errors.New("duplicate JSON field")
			}
			fields[name] = struct{}{}
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	default:
		return errors.New("JSON delimiter is invalid")
	}
}

func expectEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("unexpected JSON value")
		}
		return err
	}
	return nil
}

func assetValidationError(message string) error {
	return fmt.Errorf("%w: %s", ErrAssetValidation, message)
}

func domainValidationError(_ error) error {
	return fmt.Errorf("%w: domain value is invalid", ErrAssetValidation)
}
