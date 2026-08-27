package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	"github.com/google/uuid"
)

func TestPreflightServiceCheckRejectsEveryGateWithoutModelWork(t *testing.T) {
	t.Parallel()

	for _, testCase := range []preflightGateCase{
		{
			name: "draft dataset", configure: func(catalog *preflightCatalogFake, _ *PreflightRequest) {
				catalog.publication.PublicationState = domain.PublicationStateDraft
				catalog.publication.ReviewDecision = domain.ReviewDecisionPending
			}, want: ErrDatasetUnavailable, catalogs: 2,
		},
		{
			name: "wrong corpus", configure: func(catalog *preflightCatalogFake, request *PreflightRequest) {
				request.CorpusID = testPreflightID(99)
				catalog.datasetCorpus = testPreflightID(1)
			}, want: ErrPreflightCorpusMismatch, catalogs: 1,
		},
		{
			name: "missing source", configure: func(catalog *preflightCatalogFake, _ *PreflightRequest) {
				catalog.sourceMember = false
			}, want: ErrSnapshotIncompatible, catalogs: 6,
		},
		{
			name: "unresolved locator", configure: func(catalog *preflightCatalogFake, _ *PreflightRequest) {
				catalog.resolveErr = errors.New("not found")
			}, want: ErrLocatorUnresolved, catalogs: 6,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assertPreflightGateRejection(t, testCase)
		})
	}
}

type preflightGateCase struct {
	name      string
	configure func(*preflightCatalogFake, *PreflightRequest)
	want      error
	catalogs  int
}

func assertPreflightGateRejection(t *testing.T, testCase preflightGateCase) {
	t.Helper()
	catalog, request := validPreflightInputs()
	testCase.configure(&catalog, &request)
	_, err := NewPreflightService(&catalog).Check(context.Background(), request)
	if !errors.Is(err, testCase.want) {
		t.Fatalf("Check() error = %v, want %v", err, testCase.want)
	}
	if catalog.calls != testCase.catalogs {
		t.Fatalf("catalog calls = %d, want %d", catalog.calls, testCase.catalogs)
	}
	if catalog.model.calls != 0 {
		t.Fatalf("model calls = %d, want zero after rejected preflight", catalog.model.calls)
	}
	compatibility, ok := err.(*CompatibilityError)
	if !ok {
		t.Fatalf("Check() error type = %T, want *CompatibilityError", err)
	}
	if len(compatibility.MissingRequirements()) > maxMissingRequirements {
		t.Fatalf("missing requirements exceeds bound: %d", len(compatibility.MissingRequirements()))
	}
}

func TestPreflightServiceCheckReturnsOnlyCompleteResolutionPlan(t *testing.T) {
	t.Parallel()

	catalog, request := validPreflightInputs()
	result, err := NewPreflightService(&catalog).Check(context.Background(), request)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.CorpusID != request.CorpusID || result.DatasetRevisionID != request.DatasetRevisionID ||
		result.SnapshotID != request.SnapshotID || len(result.ResolvedLocators) != 1 {
		t.Fatalf("Check() result = %#v, want complete fixed-snapshot resolution", result)
	}
	result.ResolvedLocators[0].CanonicalLocator = "changed"
	if catalog.resolved.CanonicalLocator != "article:1" {
		t.Fatalf("result mutation changed catalog resolution: %#v", catalog.resolved)
	}
}

func TestPreflightServiceCheckReturnsNotFoundForAnAbsentRevision(t *testing.T) {
	t.Parallel()

	catalog, request := validPreflightInputs()
	catalog.datasetCorpusErr = ErrDatasetNotFound

	_, err := NewPreflightService(&catalog).Check(context.Background(), request)
	if !errors.Is(err, ErrDatasetNotFound) {
		t.Fatalf("Check() error = %v, want %v", err, ErrDatasetNotFound)
	}
	if catalog.calls != 1 {
		t.Fatalf("catalog calls = %d, want 1", catalog.calls)
	}
	if catalog.model.calls != 0 {
		t.Fatalf("model calls = %d, want zero after absent revision", catalog.model.calls)
	}
}

func TestPreflightServiceCheckResolvesEveryAtomicTargetForOneDisplayLocator(t *testing.T) {
	t.Parallel()

	catalog, request := validPreflightInputs()
	catalog.evidence = []ExpectedEvidenceRequirement{
		{
			ID: testPreflightID(7), CorpusID: request.CorpusID, DatasetRevisionID: request.DatasetRevisionID, CaseID: testPreflightID(8),
			SourceAlias: "official-source", DisplayLocator: "Article 1(A)-(B)", CanonicalLocator: "article:1/item:a",
		},
		{
			ID: testPreflightID(12), CorpusID: request.CorpusID, DatasetRevisionID: request.DatasetRevisionID, CaseID: testPreflightID(8),
			SourceAlias: "official-source", DisplayLocator: "Article 1(A)-(B)", CanonicalLocator: "article:1/item:b",
		},
	}

	result, err := NewPreflightService(&catalog).Check(context.Background(), request)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(result.ResolvedLocators) != 2 ||
		result.ResolvedLocators[0].CanonicalLocator != "article:1/item:a" ||
		result.ResolvedLocators[1].CanonicalLocator != "article:1/item:b" ||
		result.ResolvedLocators[0].DisplayLocator != "Article 1(A)-(B)" ||
		result.ResolvedLocators[1].DisplayLocator != "Article 1(A)-(B)" {
		t.Fatalf("Check() resolved locators = %#v, want each atomic locator with the original display locator", result.ResolvedLocators)
	}
}

func TestPreflightServiceCheckReturnsSourceAndLocatorDiagnosticsTogether(t *testing.T) {
	t.Parallel()

	catalog, request := validPreflightInputs()
	catalog.sourceMember = false
	catalog.resolveErr = errors.New("not found")

	_, err := NewPreflightService(&catalog).Check(context.Background(), request)
	if !errors.Is(err, ErrSnapshotIncompatible) {
		t.Fatalf("Check() error = %v, want %v", err, ErrSnapshotIncompatible)
	}
	compatibility, ok := err.(*CompatibilityError)
	if !ok {
		t.Fatalf("Check() error type = %T, want *CompatibilityError", err)
	}
	missing := compatibility.MissingRequirements()
	if len(missing) != 2 || missing[0].Reason != "The required source is not a member of the selected snapshot." ||
		missing[1].Reason != "The locator did not resolve uniquely." {
		t.Fatalf("missing requirements = %#v, want source and locator diagnostics", missing)
	}
}

func TestAppendMissingRequirementsPreservesBothGateCategoriesAtTheBound(t *testing.T) {
	t.Parallel()

	snapshotMissing := make([]MissingRequirement, maxMissingRequirements)
	for index := range snapshotMissing {
		snapshotMissing[index] = MissingRequirement{SourceAlias: fmt.Sprintf("source-%d", index), Reason: "source"}
	}
	locatorMissing := MissingRequirement{SourceAlias: "source-locator", Locator: "Article 1", Reason: "locator"}
	combined := appendMissingRequirements(missingRequirementGroups{current: snapshotMissing, additional: []MissingRequirement{locatorMissing}})

	if len(combined) != maxMissingRequirements || combined[maxMissingRequirements-1] != locatorMissing {
		t.Fatalf("combined missing requirements = %#v, want bounded diagnostics including the locator gate", combined)
	}
}

func TestPublicationServiceAppendsValidatedLifecycle(t *testing.T) {
	t.Parallel()

	store := &publicationStoreFake{}
	service := NewPublicationService(store, func() uuid.UUID { return testPreflightID(40) }, func() time.Time {
		return time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
	})
	command := ReviewCommand{
		CorpusID: testPreflightID(1), DatasetRevisionID: testPreflightID(2), ReviewDecision: domain.ReviewDecisionApproved,
		ReviewerIdentity: "maintainer", ReviewNote: "Reviewed against the named snapshot.", PublicationState: domain.PublicationStateAvailable,
	}
	publication, err := service.Review(context.Background(), command)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if store.appended != 1 || publication.ID != testPreflightID(40) || store.publication != publication {
		t.Fatalf("Review() append = %#v, calls = %d", store.publication, store.appended)
	}

	store.latest, store.found = publication, true
	if _, err := service.Review(context.Background(), command); !errors.Is(err, domain.ErrInvalidLifecycleTransition) {
		t.Fatalf("Review() repeated available error = %v, want lifecycle transition error", err)
	}
	if store.appended != 1 {
		t.Fatalf("append calls = %d, want one", store.appended)
	}
}

type preflightCatalogFake struct {
	datasetCorpus    uuid.UUID
	datasetCorpusErr error
	publication      domain.Publication
	publicationOK    bool
	sources          []SourceRequirement
	evidence         []ExpectedEvidenceRequirement
	sourceMember     bool
	resolveErr       error
	resolved         ResolvedLocator
	model            *modelCallSentinel
	calls            int
}

func (fake *preflightCatalogFake) DatasetCorpus(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
	fake.calls++
	return fake.datasetCorpus, fake.datasetCorpusErr
}

func (fake *preflightCatalogFake) LatestPublication(_ context.Context, _ uuid.UUID, _ uuid.UUID) (domain.Publication, bool, error) {
	fake.calls++
	return fake.publication, fake.publicationOK, nil
}

func (fake *preflightCatalogFake) SourceRequirements(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]SourceRequirement, error) {
	fake.calls++
	return fake.sources, nil
}

func (fake *preflightCatalogFake) ExpectedEvidence(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]ExpectedEvidenceRequirement, error) {
	fake.calls++
	return fake.evidence, nil
}

func (fake *preflightCatalogFake) SnapshotContainsSource(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	fake.calls++
	return fake.sourceMember, nil
}

func (fake *preflightCatalogFake) ResolvePreflightLocator(_ context.Context, request LocatorRequest) (ResolvedLocator, error) {
	fake.calls++
	if fake.resolveErr != nil {
		return ResolvedLocator{}, fake.resolveErr
	}
	resolved := fake.resolved
	resolved.DisplayLocator = request.DisplayLocator
	resolved.CanonicalLocator = request.CanonicalLocator
	return resolved, nil
}

type modelCallSentinel struct{ calls int }

func (sentinel *modelCallSentinel) Call() { sentinel.calls++ }

type publicationStoreFake struct {
	latest      domain.Publication
	found       bool
	publication domain.Publication
	appended    int
}

func (fake *publicationStoreFake) LatestPublication(_ context.Context, _ uuid.UUID, _ uuid.UUID) (domain.Publication, bool, error) {
	return fake.latest, fake.found, nil
}

func (fake *publicationStoreFake) AppendPublication(_ context.Context, publication domain.Publication) error {
	fake.appended++
	fake.publication = publication
	return nil
}

func validPreflightInputs() (preflightCatalogFake, PreflightRequest) {
	corpusID, revisionID, snapshotID, sourceID := testPreflightID(1), testPreflightID(2), testPreflightID(3), testPreflightID(4)
	return preflightCatalogFake{
		datasetCorpus: corpusID,
		publication: domain.Publication{
			ID: testPreflightID(5), DatasetRevisionID: revisionID, CorpusID: corpusID,
			ReviewDecision: domain.ReviewDecisionApproved, ReviewerIdentity: "maintainer",
			PublicationState: domain.PublicationStateAvailable, ReviewedAt: time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC),
		},
		publicationOK: true,
		sources: []SourceRequirement{{
			ID: testPreflightID(6), CorpusID: corpusID, DatasetRevisionID: revisionID, SourceAlias: "official-source", CorpusSourceID: &sourceID,
		}},
		evidence: []ExpectedEvidenceRequirement{{
			ID: testPreflightID(7), CorpusID: corpusID, DatasetRevisionID: revisionID, CaseID: testPreflightID(8),
			SourceAlias: "official-source", DisplayLocator: "Article 1", CanonicalLocator: "article:1",
		}},
		sourceMember: true,
		model:        &modelCallSentinel{},
		resolved: ResolvedLocator{
			CorpusID: corpusID, SnapshotID: snapshotID, SourceID: sourceID, SourceRevisionID: testPreflightID(9),
			DocumentID: testPreflightID(10), UnitID: testPreflightID(11), CanonicalLocator: "article:1", DisplayLocator: "Article 1",
		},
	}, PreflightRequest{CorpusID: corpusID, DatasetRevisionID: revisionID, SnapshotID: snapshotID}
}

func testPreflightID(value int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012d", value))
}
