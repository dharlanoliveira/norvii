package domain

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNewProjection(t *testing.T) {
	t.Parallel()

	for _, testCase := range []projectionCase{
		{name: "reviewed available selection", mutate: func(_ *PublicationInput) {}},
		{name: "unavailable publication", mutate: func(input *PublicationInput) { input.DatasetPublication.PublicationState = "draft" }, want: ErrUnavailableDataset},
		{name: "unapproved publication", mutate: func(input *PublicationInput) { input.DatasetPublication.ReviewDecision = "pending" }, want: ErrUnavailableDataset},
		{name: "foreign publication", mutate: func(input *PublicationInput) { input.DatasetPublication.CorpusID = testID(99) }, want: ErrCorpusMismatch},
		{name: "publication for another dataset revision", mutate: func(input *PublicationInput) { input.DatasetPublication.DatasetRevisionID = testID(98) }, want: ErrInvalidInput},
		{name: "stale active snapshot", mutate: func(input *PublicationInput) { input.ActiveSnapshot.ID = testID(77) }, want: ErrSnapshotMismatch},
		{name: "foreign active snapshot", mutate: func(input *PublicationInput) { input.ActiveSnapshot.CorpusID = testID(99) }, want: ErrCorpusMismatch},
		{name: "rank above maximum", mutate: func(input *PublicationInput) { input.StarterCases[0].Rank = 6 }, want: ErrInvalidSelection},
		{name: "rank gap", mutate: func(input *PublicationInput) { input.StarterCases[0].Rank, input.StarterCases[1].Rank = 2, 2 }, want: ErrInvalidSelection},
		{name: "same language pair", mutate: func(input *PublicationInput) { input.StarterCases[1].QueryLanguage = QueryLanguageEnglish }, want: ErrInvalidSelection},
		{name: "non reciprocal pair", mutate: func(input *PublicationInput) { input.StarterCases[1].ReciprocalExternalCaseID = "other-case" }, want: ErrInvalidSelection},
		{name: "duplicate selected case", mutate: func(input *PublicationInput) { input.StarterCases = append(input.StarterCases, input.StarterCases[0]) }, want: ErrInvalidSelection},
		{name: "missing expected evidence", mutate: func(input *PublicationInput) { input.ResolvedEvidence = input.ResolvedEvidence[:1] }, want: ErrUnresolvedExpectedEvidence},
		{name: "duplicate expected evidence resolution", mutate: func(input *PublicationInput) {
			input.ResolvedEvidence = append(input.ResolvedEvidence, input.ResolvedEvidence[0])
		}, want: ErrUnresolvedExpectedEvidence},
		{name: "foreign expected evidence", mutate: func(input *PublicationInput) { input.ExpectedEvidence[0].CorpusID = testID(99) }, want: ErrCorpusMismatch},
		{name: "resolution for another snapshot", mutate: func(input *PublicationInput) { input.ResolvedEvidence[0].SnapshotID = testID(77) }, want: ErrUnresolvedExpectedEvidence},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assertProjectionCase(t, testCase)
		})
	}
}

type projectionCase struct {
	name   string
	mutate func(*PublicationInput)
	want   error
}

func assertProjectionCase(t *testing.T, testCase projectionCase) {
	t.Helper()
	input := validInput()
	testCase.mutate(&input)
	projection, err := NewProjection(input)
	if !errors.Is(err, testCase.want) {
		t.Fatalf("NewProjection() error = %v, want %v", err, testCase.want)
	}
	if testCase.want == nil {
		assertSafeProjection(t, projection)
	}
}

func assertSafeProjection(t *testing.T, projection Projection) {
	t.Helper()
	if len(projection.Items) != 2 || projection.Items[0].QueryLanguage != QueryLanguageEnglish || projection.Items[1].QueryLanguage != QueryLanguagePortuguese {
		t.Fatalf("projection items = %#v, want rank-ordered reciprocal safe items", projection.Items)
	}
	for _, item := range projection.Items {
		if item.SuggestionSetID != projection.Set.ID || item.SuggestionSetID == uuid.Nil {
			t.Fatalf("projection item suggestion set ID = %s, want parent set ID %s", item.SuggestionSetID, projection.Set.ID)
		}
	}
	if projection.Items[0].Public() != (PublicOpeningSuggestion{CaseID: "case-001-en", Rank: 1, Question: "What is required?"}) {
		t.Fatalf("Public() returned unsafe or incorrect item: %#v", projection.Items[0].Public())
	}
}

func TestProjectionMatchesActiveSnapshot(t *testing.T) {
	t.Parallel()

	input := validInput()
	projection, err := NewProjection(input)
	if err != nil {
		t.Fatalf("NewProjection() error = %v", err)
	}
	if !projection.MatchesActiveSnapshot(input.ActiveSnapshot) {
		t.Fatal("MatchesActiveSnapshot() = false, want true for publication snapshot")
	}
	later := input.ActiveSnapshot
	later.ID = testID(77)
	if projection.MatchesActiveSnapshot(later) {
		t.Fatal("MatchesActiveSnapshot() = true, want false for a later active release")
	}
}

func TestCloneItemsDetachesProjectionSlice(t *testing.T) {
	t.Parallel()

	projection, err := NewProjection(validInput())
	if err != nil {
		t.Fatalf("NewProjection() error = %v", err)
	}
	cloned := projection.CloneItems()
	cloned[0].Question = "changed"
	if projection.Items[0].Question == "changed" {
		t.Fatal("CloneItems() retained caller-owned slice backing storage")
	}
}

func validInput() PublicationInput {
	corpusID, revisionID, snapshotID := testID(1), testID(2), testID(3)
	snapshot := Snapshot{ID: snapshotID, CorpusID: corpusID, ManifestSHA256: checksum("a")}
	english, portuguese := StarterCase{
		ID: testID(10), CorpusID: corpusID, DatasetRevisionID: revisionID, ExternalCaseID: "case-001-en",
		ReciprocalExternalCaseID: "case-001-pt", Checksum: checksum("b"), QueryLanguage: QueryLanguageEnglish,
		Rank: 1, Question: "What is required?",
	}, StarterCase{
		ID: testID(11), CorpusID: corpusID, DatasetRevisionID: revisionID, ExternalCaseID: "case-001-pt",
		ReciprocalExternalCaseID: "case-001-en", Checksum: checksum("c"), QueryLanguage: QueryLanguagePortuguese,
		Rank: 1, Question: "Qual e a exigencia?",
	}
	requirements := []ExpectedEvidenceRequirement{
		{ID: testID(20), CorpusID: corpusID, DatasetRevisionID: revisionID, CaseID: english.ID},
		{ID: testID(21), CorpusID: corpusID, DatasetRevisionID: revisionID, CaseID: portuguese.ID},
	}
	return PublicationInput{
		SuggestionSetID: testID(30), CorpusID: corpusID, DatasetRevisionID: revisionID, Snapshot: snapshot, ActiveSnapshot: snapshot,
		DatasetPublication: DatasetPublication{ID: testID(40), CorpusID: corpusID, DatasetRevisionID: revisionID,
			DatasetContentSHA256: checksum("d"), ReviewDecision: "approved", PublicationState: "available"},
		SelectionPolicyVersion: "v1", PublishedBy: "test-maintainer", StarterCases: []StarterCase{english, portuguese},
		ExpectedEvidence: requirements,
		ResolvedEvidence: []ExpectedEvidenceResolution{
			{ExpectedEvidenceID: requirements[0].ID, CorpusID: corpusID, DatasetRevisionID: revisionID, CaseID: english.ID, SnapshotID: snapshotID},
			{ExpectedEvidenceID: requirements[1].ID, CorpusID: corpusID, DatasetRevisionID: revisionID, CaseID: portuguese.ID, SnapshotID: snapshotID},
		},
	}
}

func testID(value int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012d", value))
}

func checksum(character string) SHA256 { return SHA256(strings.Repeat(character, 64)) }
