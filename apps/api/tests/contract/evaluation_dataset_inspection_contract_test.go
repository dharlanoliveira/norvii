package contract_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	evaluationcontract "github.com/dharlanoliveira/norvii/apps/api/tests/contract"
	"github.com/google/uuid"
)

func TestDatasetInspectionFixturesPreserveTheVersionedContract(t *testing.T) {
	var catalog []evaluationcontract.DatasetCatalogResponse
	decodeDatasetInspectionFixture(t, "dataset-catalog-response.json", &catalog)
	if len(catalog) != 1 {
		t.Fatalf("catalog entries = %d, want 1", len(catalog))
	}
	validateDatasetCatalogFixture(t, catalog[0])

	var detail evaluationcontract.DatasetDetailResponse
	decodeDatasetInspectionFixture(t, "dataset-detail-response.json", &detail)
	validateDatasetCatalogFixture(t, detail.DatasetCatalogResponse)
	if !reflect.DeepEqual(detail.Revision, catalog[0].Revision) || detail.Available != catalog[0].Available || detail.Review == nil || catalog[0].Review == nil || *detail.Review != *catalog[0].Review {
		t.Fatalf("detail identity or review differs from catalog: detail=%#v catalog=%#v", detail.DatasetCatalogResponse, catalog[0])
	}
	if len(detail.Sources) != 1 || len(detail.Starters) != 1 {
		t.Fatalf("detail sources/starters = %d/%d, want 1/1", len(detail.Sources), len(detail.Starters))
	}
	source := detail.Sources[0]
	if !source.Bound || source.CorpusSourceID == nil || !validDatasetFixtureUUID(source.ID) || !validDatasetFixtureUUID(*source.CorpusSourceID) ||
		strings.TrimSpace(source.SourceAlias) == "" || strings.TrimSpace(source.OfficialURL) == "" || strings.TrimSpace(source.IssuingAuthority) == "" {
		t.Fatalf("invalid source authority fixture: %#v", source)
	}
	starter := detail.Starters[0]
	if !starter.ReviewEligible || starter.Rank < 1 || starter.Rank > 5 || !validDatasetFixtureUUID(starter.ID) || !validDatasetFixtureUUID(starter.CaseID) ||
		(starter.QueryLanguage != "en" && starter.QueryLanguage != "pt") {
		t.Fatalf("invalid starter fixture: %#v", starter)
	}

	var preflight evaluationcontract.DatasetPreflightResponse
	decodeDatasetInspectionFixture(t, "dataset-preflight-success.json", &preflight)
	if !preflight.Compatible || len(preflight.MissingRequirements) != 0 || preflight.DatasetRevisionID != detail.Revision.ID || preflight.CorpusID != detail.Revision.CorpusID || !validDatasetFixtureUUID(preflight.SnapshotID) {
		t.Fatalf("invalid successful preflight identity: %#v", preflight)
	}
}

func TestDatasetPreflightIncompatibilityFixtureIsBoundedAndSafe(t *testing.T) {
	var fixture evaluationcontract.DatasetPreflightErrorResponse
	decodeDatasetInspectionFixture(t, "dataset-preflight-error-snapshot-incompatible.json", &fixture)
	if fixture.Error.Code != "snapshot_incompatible" || strings.TrimSpace(fixture.Error.Message) == "" || !validDatasetFixtureUUID(fixture.Error.RequestID) {
		t.Fatalf("invalid preflight error identity: %#v", fixture.Error)
	}
	if len(fixture.Error.MissingRequirements) == 0 || len(fixture.Error.MissingRequirements) > 32 {
		t.Fatalf("missing requirements = %d, want 1 through 32", len(fixture.Error.MissingRequirements))
	}
	encoded, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	for _, value := range []string{"question", "referenceAnswer", "requiredPropositions", "providerPayload", "prompt", "credential"} {
		if bytes.Contains(bytes.ToLower(encoded), []byte(strings.ToLower(value))) {
			t.Fatalf("preflight error fixture disclosed %q: %s", value, encoded)
		}
	}
}

func TestDatasetPreflightNotFoundFixturePreservesThePublicErrorEnvelope(t *testing.T) {
	var fixture evaluationcontract.DatasetPreflightErrorResponse
	decodeDatasetInspectionFixture(t, "dataset-preflight-error-not-found.json", &fixture)
	if fixture.Error.Code != "not_found" || strings.TrimSpace(fixture.Error.Message) == "" || !validDatasetFixtureUUID(fixture.Error.RequestID) {
		t.Fatalf("invalid absent preflight fixture: %#v", fixture.Error)
	}
	if fixture.Error.MissingRequirements != nil {
		t.Fatalf("not-found missing requirements = %#v, want omitted", fixture.Error.MissingRequirements)
	}
}

func decodeDatasetInspectionFixture(t *testing.T, name string, destination any) {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "contracts", "evaluation", "v1", "fixtures", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	if err := decoder.Decode(&struct{}{}); !errorsDatasetFixtureEOF(err) {
		t.Fatalf("decode %s trailing content: %v", name, err)
	}
}

func errorsDatasetFixtureEOF(err error) bool { return err == io.EOF }

func validateDatasetCatalogFixture(t *testing.T, fixture evaluationcontract.DatasetCatalogResponse) {
	t.Helper()
	revision := fixture.Revision
	if !validDatasetFixtureUUID(revision.ID) || !validDatasetFixtureUUID(revision.CorpusID) || strings.TrimSpace(revision.DatasetKey) == "" ||
		strings.TrimSpace(revision.SemanticRevision) == "" || strings.TrimSpace(revision.Jurisdiction) == "" ||
		!validDatasetFixtureHash(revision.ManifestSHA256) || !validDatasetFixtureHash(revision.JSONLSHA256) || !validDatasetFixtureHash(revision.ContentSHA256) ||
		len(revision.QueryLanguages) == 0 || (revision.AuthoritativeEvidenceLanguage != "en" && revision.AuthoritativeEvidenceLanguage != "pt-BR") {
		t.Fatalf("invalid catalog revision: %#v", revision)
	}
	if _, err := time.Parse("2006-01-02", revision.DeclaredSnapshotDate); err != nil {
		t.Fatalf("invalid declared snapshot date %q: %v", revision.DeclaredSnapshotDate, err)
	}
	for _, language := range revision.QueryLanguages {
		if language != "en" && language != "pt" {
			t.Fatalf("invalid query language %q", language)
		}
	}
	if fixture.Review == nil || (fixture.Available && (fixture.Review.Decision != "approved" || fixture.Review.PublicationState != "available")) {
		t.Fatalf("invalid availability/review relationship: available=%t review=%#v", fixture.Available, fixture.Review)
	}
	if _, err := time.Parse(time.RFC3339, fixture.Review.ReviewedAt); err != nil {
		t.Fatalf("invalid review timestamp %q: %v", fixture.Review.ReviewedAt, err)
	}
}

func validDatasetFixtureUUID(value string) bool {
	identifier, err := uuid.Parse(value)
	return err == nil && identifier != uuid.Nil
}

func validDatasetFixtureHash(value string) bool {
	return len(value) == 64 && strings.Trim(value, "0123456789abcdef") == ""
}
