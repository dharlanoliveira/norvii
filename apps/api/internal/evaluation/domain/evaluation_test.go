package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

const validChecksum = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestDatasetRevisionValidate(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name     string
		revision DatasetRevision
		wantErr  error
	}{
		{name: "valid", revision: validRevision(), wantErr: nil},
		{
			name: "empty dataset key",
			revision: func() DatasetRevision {
				revision := validRevision()
				revision.DatasetKey = ""
				return revision
			}(),
			wantErr: ErrInvalidInput,
		},
		{
			name: "duplicate query language",
			revision: func() DatasetRevision {
				revision := validRevision()
				revision.QueryLanguages = []QueryLanguage{QueryLanguageEnglish, QueryLanguageEnglish}
				return revision
			}(),
			wantErr: ErrInvalidInput,
		},
		{
			name: "uppercase checksum",
			revision: func() DatasetRevision {
				revision := validRevision()
				revision.ContentSHA256 = SHA256(strings.ToUpper(validChecksum))
				return revision
			}(),
			wantErr: ErrInvalidInput,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := testCase.revision.Validate(); !errors.Is(err, testCase.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestValidatedValuesDetachCallerOwnedSlices(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name     string
		validate func(t *testing.T)
	}{
		{
			name: "dataset revision query languages",
			validate: func(t *testing.T) {
				languages := []QueryLanguage{QueryLanguageEnglish, QueryLanguagePortuguese}
				revision := validRevision()
				revision.QueryLanguages = languages
				if err := revision.Validate(); err != nil {
					t.Fatalf("Validate() error = %v", err)
				}

				languages[0] = QueryLanguagePortuguese
				if revision.QueryLanguages[0] != QueryLanguageEnglish {
					t.Fatalf("QueryLanguages = %v, want validation to copy caller-owned slice", revision.QueryLanguages)
				}
			},
		},
		{
			name: "expected evidence required propositions",
			validate: func(t *testing.T) {
				revision := validRevision()
				evaluationCase := validCases(revision)[0]
				requiredPropositions := []string{"the obligation applies"}
				evidence := validEvidence(revision, evaluationCase)
				evidence.RequiredPropositions = requiredPropositions
				if err := evidence.ValidateAgainst(revision, evaluationCase, []SourceRequirement{validSource(revision)}); err != nil {
					t.Fatalf("ValidateAgainst() error = %v", err)
				}

				requiredPropositions[0] = "changed after validation"
				if evidence.RequiredPropositions[0] != "the obligation applies" {
					t.Fatalf("RequiredPropositions = %v, want validation to copy caller-owned slice", evidence.RequiredPropositions)
				}
			},
		},
	} {
		t.Run(testCase.name, testCase.validate)
	}
}

func TestSHA256Validate(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name     string
		checksum SHA256
		wantErr  error
	}{
		{name: "lowercase hexadecimal", checksum: SHA256(validChecksum)},
		{name: "wrong length", checksum: SHA256("abc"), wantErr: ErrInvalidInput},
		{name: "uppercase", checksum: SHA256(strings.ToUpper(validChecksum)), wantErr: ErrInvalidInput},
		{name: "non hexadecimal", checksum: SHA256(strings.Repeat("g", 64)), wantErr: ErrInvalidInput},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := testCase.checksum.Validate(); !errors.Is(err, testCase.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestSourceRequirementAndEvidenceOwnership(t *testing.T) {
	t.Parallel()

	revision := validRevision()
	source := validSource(revision)
	evaluationCase := validCases(revision)[0]
	evidence := validEvidence(revision, evaluationCase)

	for _, testCase := range []struct {
		name     string
		validate func() error
		wantErr  error
	}{
		{
			name:     "valid source and evidence",
			validate: func() error { return evidence.ValidateAgainst(revision, evaluationCase, []SourceRequirement{source}) },
		},
		{
			name: "source from another corpus",
			validate: func() error {
				foreignSource := source
				foreignSource.CorpusID = fixtureID("00000000-0000-4000-8000-000000000099")
				return foreignSource.ValidateAgainst(revision)
			},
			wantErr: ErrCorpusMismatch,
		},
		{
			name: "evidence for another case",
			validate: func() error {
				foreignEvidence := evidence
				foreignEvidence.CaseID = fixtureID("00000000-0000-4000-8000-000000000098")
				return foreignEvidence.ValidateAgainst(revision, evaluationCase, []SourceRequirement{source})
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "unknown evidence source",
			validate: func() error {
				unknownSource := evidence
				unknownSource.SourceAlias = "other-source"
				return unknownSource.ValidateAgainst(revision, evaluationCase, []SourceRequirement{source})
			},
			wantErr: ErrInvalidInput,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := testCase.validate(); !errors.Is(err, testCase.wantErr) {
				t.Fatalf("validation error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestCaseValidationDefaultsExpectedOutcome(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name     string
		validate func(revision DatasetRevision, cases []Case) error
		outcome  func(cases []Case) ExpectedOutcome
	}{
		{
			name: "direct case validation",
			validate: func(revision DatasetRevision, cases []Case) error {
				return cases[0].ValidateAgainst(revision)
			},
			outcome: func(cases []Case) ExpectedOutcome { return cases[0].ExpectedOutcome },
		},
		{
			name: "paired case validation",
			validate: func(revision DatasetRevision, cases []Case) error {
				return ValidatePairedCases(revision, cases)
			},
			outcome: func(cases []Case) ExpectedOutcome { return cases[0].ExpectedOutcome },
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			revision := validRevision()
			cases := validCases(revision)
			cases[0].ExpectedOutcome = ""
			if err := testCase.validate(revision, cases); err != nil {
				t.Fatalf("validation error = %v", err)
			}
			if outcome := testCase.outcome(cases); outcome != ExpectedOutcomeAnswer {
				t.Fatalf("ExpectedOutcome = %q, want %q", outcome, ExpectedOutcomeAnswer)
			}
		})
	}
}

func TestValidatePairedCases(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name    string
		cases   func(DatasetRevision) []Case
		wantErr error
	}{
		{name: "valid reciprocal languages", cases: validCases},
		{
			name: "missing reciprocal case",
			cases: func(revision DatasetRevision) []Case {
				return validCases(revision)[:1]
			},
			wantErr: ErrInvalidPair,
		},
		{
			name: "same query language",
			cases: func(revision DatasetRevision) []Case {
				cases := validCases(revision)
				cases[1].QueryLanguage = QueryLanguageEnglish
				cases[1].AssetLanguage = AssetLanguageEnglish
				return cases
			},
			wantErr: ErrInvalidPair,
		},
		{
			name: "non reciprocal link",
			cases: func(revision DatasetRevision) []Case {
				cases := validCases(revision)
				cases[1].ReciprocalExternalID = "case-003-en"
				return cases
			},
			wantErr: ErrInvalidPair,
		},
		{
			name: "case owned by another corpus",
			cases: func(revision DatasetRevision) []Case {
				cases := validCases(revision)
				cases[1].CorpusID = fixtureID("00000000-0000-4000-8000-000000000099")
				return cases
			},
			wantErr: ErrCorpusMismatch,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			revision := validRevision()
			if err := ValidatePairedCases(revision, testCase.cases(revision)); !errors.Is(err, testCase.wantErr) {
				t.Fatalf("ValidatePairedCases() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestValidateStarterSelections(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name       string
		selections func(DatasetRevision, []Case) []StarterSelection
		wantErr    error
	}{
		{name: "empty draft selection", selections: func(_ DatasetRevision, _ []Case) []StarterSelection { return nil }},
		{name: "reviewed reciprocal rank", selections: validSelections},
		{
			name: "missing reciprocal selection",
			selections: func(revision DatasetRevision, cases []Case) []StarterSelection {
				return validSelections(revision, cases)[:1]
			},
			wantErr: ErrInvalidStarterSelection,
		},
		{
			name: "different reciprocal rank",
			selections: func(revision DatasetRevision, cases []Case) []StarterSelection {
				selections := validSelections(revision, cases)
				selections[1].Rank = 2
				return selections
			},
			wantErr: ErrInvalidStarterSelection,
		},
		{
			name: "unreviewed selection",
			selections: func(revision DatasetRevision, cases []Case) []StarterSelection {
				selections := validSelections(revision, cases)
				selections[0].ReviewEligible = false
				return selections
			},
			wantErr: ErrInvalidStarterSelection,
		},
		{
			name: "checksum differs from case",
			selections: func(revision DatasetRevision, cases []Case) []StarterSelection {
				selections := validSelections(revision, cases)
				selections[0].CaseChecksum = SHA256(strings.Repeat("a", 64))
				return selections
			},
			wantErr: ErrInvalidStarterSelection,
		},
		{
			name: "rank outside published range",
			selections: func(revision DatasetRevision, cases []Case) []StarterSelection {
				selections := validSelections(revision, cases)
				selections[0].Rank = 6
				return selections
			},
			wantErr: ErrInvalidStarterSelection,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			revision := validRevision()
			cases := validCases(revision)
			if err := ValidateStarterSelections(revision, cases, testCase.selections(revision, cases)); !errors.Is(err, testCase.wantErr) {
				t.Fatalf("ValidateStarterSelections() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestValidatePublicationTransition(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name     string
		previous PublicationState
		next     PublicationState
		wantErr  error
	}{
		{name: "draft becomes available", previous: PublicationStateDraft, next: PublicationStateAvailable},
		{name: "draft becomes withdrawn", previous: PublicationStateDraft, next: PublicationStateWithdrawn},
		{name: "available becomes superseded", previous: PublicationStateAvailable, next: PublicationStateSuperseded},
		{name: "available becomes withdrawn", previous: PublicationStateAvailable, next: PublicationStateWithdrawn},
		{name: "draft remains draft", previous: PublicationStateDraft, next: PublicationStateDraft, wantErr: ErrInvalidLifecycleTransition},
		{name: "available cannot become draft", previous: PublicationStateAvailable, next: PublicationStateDraft, wantErr: ErrInvalidLifecycleTransition},
		{name: "withdrawn is terminal", previous: PublicationStateWithdrawn, next: PublicationStateAvailable, wantErr: ErrInvalidLifecycleTransition},
		{name: "superseded is terminal", previous: PublicationStateSuperseded, next: PublicationStateAvailable, wantErr: ErrInvalidLifecycleTransition},
		{name: "unknown state", previous: PublicationState("unknown"), next: PublicationStateAvailable, wantErr: ErrInvalidLifecycleTransition},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := ValidatePublicationTransition(testCase.previous, testCase.next); !errors.Is(err, testCase.wantErr) {
				t.Fatalf("ValidatePublicationTransition() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestPublicationValidateAgainst(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name        string
		publication func(DatasetRevision) Publication
		wantErr     error
	}{
		{name: "available approved publication", publication: validPublication},
		{
			name: "available rejected publication",
			publication: func(revision DatasetRevision) Publication {
				publication := validPublication(revision)
				publication.ReviewDecision = ReviewDecisionRejected
				return publication
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "publication from another corpus",
			publication: func(revision DatasetRevision) Publication {
				publication := validPublication(revision)
				publication.CorpusID = fixtureID("00000000-0000-4000-8000-000000000099")
				return publication
			},
			wantErr: ErrCorpusMismatch,
		},
		{
			name: "publication review note exceeds bound",
			publication: func(revision DatasetRevision) Publication {
				publication := validPublication(revision)
				publication.ReviewNote = strings.Repeat("a", 2001)
				return publication
			},
			wantErr: ErrInvalidInput,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			revision := validRevision()
			if err := testCase.publication(revision).ValidateAgainst(revision); !errors.Is(err, testCase.wantErr) {
				t.Fatalf("ValidateAgainst() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestOpeningSuggestionSetSnapshotCompatibility(t *testing.T) {
	t.Parallel()

	revision := validRevision()
	snapshot := validSnapshot(revision)
	set := validSuggestionSet(revision, snapshot)

	for _, testCase := range []struct {
		name    string
		set     OpeningSuggestionSet
		active  Snapshot
		wantErr error
	}{
		{name: "matching active snapshot", set: set, active: snapshot},
		{
			name:    "different snapshot identifier",
			set:     set,
			active:  Snapshot{ID: fixtureID("00000000-0000-4000-8000-000000000077"), CorpusID: snapshot.CorpusID, ManifestSHA256: snapshot.ManifestSHA256},
			wantErr: ErrSnapshotMismatch,
		},
		{
			name:    "different snapshot manifest",
			set:     set,
			active:  Snapshot{ID: snapshot.ID, CorpusID: snapshot.CorpusID, ManifestSHA256: SHA256(strings.Repeat("b", 64))},
			wantErr: ErrSnapshotMismatch,
		},
		{
			name:    "different corpus",
			set:     set,
			active:  Snapshot{ID: snapshot.ID, CorpusID: fixtureID("00000000-0000-4000-8000-000000000099"), ManifestSHA256: snapshot.ManifestSHA256},
			wantErr: ErrSnapshotMismatch,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := testCase.set.ValidateActiveSnapshot(testCase.active); !errors.Is(err, testCase.wantErr) {
				t.Fatalf("ValidateActiveSnapshot() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}

	publication := validPublication(revision)
	if err := set.ValidateAgainst(revision, publication, snapshot); err != nil {
		t.Fatalf("ValidateAgainst() error = %v", err)
	}
	foreignSet := set
	foreignSet.CorpusID = fixtureID("00000000-0000-4000-8000-000000000099")
	if err := foreignSet.ValidateAgainst(revision, publication, snapshot); !errors.Is(err, ErrCorpusMismatch) {
		t.Fatalf("ValidateAgainst() error = %v, want %v", err, ErrCorpusMismatch)
	}
}

func TestOpeningSuggestionSetRequiresAvailablePublication(t *testing.T) {
	t.Parallel()

	revision := validRevision()
	snapshot := validSnapshot(revision)
	set := validSuggestionSet(revision, snapshot)

	for _, testCase := range []struct {
		name        string
		publication func(Publication) Publication
		wantErr     error
	}{
		{name: "available approved", publication: func(publication Publication) Publication { return publication }},
		{
			name: "draft publication",
			publication: func(publication Publication) Publication {
				publication.PublicationState = PublicationStateDraft
				publication.ReviewDecision = ReviewDecisionPending
				return publication
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "rejected publication",
			publication: func(publication Publication) Publication {
				publication.PublicationState = PublicationStateDraft
				publication.ReviewDecision = ReviewDecisionRejected
				return publication
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "superseded publication",
			publication: func(publication Publication) Publication {
				publication.PublicationState = PublicationStateSuperseded
				return publication
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "withdrawn publication",
			publication: func(publication Publication) Publication {
				publication.PublicationState = PublicationStateWithdrawn
				return publication
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "publication for another revision",
			publication: func(publication Publication) Publication {
				publication.DatasetRevisionID = fixtureID("00000000-0000-4000-8000-000000000099")
				return publication
			},
			wantErr: ErrInvalidInput,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			publication := testCase.publication(validPublication(revision))
			err := set.ValidateAgainst(revision, publication, snapshot)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("ValidateAgainst() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestOpeningSuggestionItemPublic(t *testing.T) {
	t.Parallel()

	revision := validRevision()
	snapshot := validSnapshot(revision)
	evaluationCase := validCases(revision)[0]
	set := validSuggestionSet(revision, snapshot)
	item := OpeningSuggestionItem{
		ID: fixtureID("00000000-0000-4000-8000-000000000050"), SuggestionSetID: set.ID, CorpusID: revision.CorpusID,
		DatasetRevisionID: revision.ID, CaseID: evaluationCase.ID, CaseChecksum: evaluationCase.Checksum,
		Rank: 1, QueryLanguage: QueryLanguageEnglish, Question: evaluationCase.Question,
	}
	if err := item.ValidateAgainst(set, evaluationCase); err != nil {
		t.Fatalf("ValidateAgainst() error = %v", err)
	}
	suggestion, err := item.Public("case-001-en")
	if err != nil {
		t.Fatalf("Public() error = %v", err)
	}
	if suggestion != (PublicOpeningSuggestion{CaseID: "case-001-en", Rank: 1, Question: item.Question}) {
		t.Fatalf("Public() = %#v, want public-safe opening suggestion", suggestion)
	}

	foreignItem := item
	foreignItem.CorpusID = fixtureID("00000000-0000-4000-8000-000000000099")
	if err := foreignItem.ValidateAgainst(set, evaluationCase); !errors.Is(err, ErrCorpusMismatch) {
		t.Fatalf("ValidateAgainst() error = %v, want %v", err, ErrCorpusMismatch)
	}
}

func validRevision() DatasetRevision {
	return DatasetRevision{
		ID: fixtureID("00000000-0000-4000-8000-000000000001"), CorpusID: fixtureID("00000000-0000-4000-8000-000000000002"),
		DatasetKey: "test-dataset", SemanticRevision: "v1", Jurisdiction: "Test", ManifestSHA256: SHA256(validChecksum),
		JSONLSHA256: SHA256(strings.Repeat("a", 64)), ContentSHA256: SHA256(strings.Repeat("b", 64)),
		DeclaredSnapshotDate: time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC),
		QueryLanguages:       []QueryLanguage{QueryLanguageEnglish, QueryLanguagePortuguese}, AuthoritativeEvidenceLanguage: AssetLanguageEnglish,
	}
}

func validSource(revision DatasetRevision) SourceRequirement {
	return SourceRequirement{
		ID: fixtureID("00000000-0000-4000-8000-000000000010"), DatasetRevisionID: revision.ID, CorpusID: revision.CorpusID,
		SourceAlias: "official-law", Title: "Official law", OfficialURL: "https://example.test/law", IssuingAuthority: "Test authority",
		DocumentType: "law", AuthorityRole: "statute",
	}
}

func validCases(revision DatasetRevision) []Case {
	return []Case{
		{
			ID: fixtureID("00000000-0000-4000-8000-000000000020"), DatasetRevisionID: revision.ID, CorpusID: revision.CorpusID,
			Position: 1, ExternalID: "case-001-en", QueryLanguage: QueryLanguageEnglish, AssetLanguage: AssetLanguageEnglish,
			Question: "What is the relevant obligation?", ReferenceAnswer: "The obligation applies.", Category: "direct-provision",
			AuthoritativeEvidenceLanguage: AssetLanguageEnglish, ExpectedOutcome: ExpectedOutcomeAnswer, ReciprocalExternalID: "case-001-pt", Checksum: SHA256(validChecksum),
		},
		{
			ID: fixtureID("00000000-0000-4000-8000-000000000021"), DatasetRevisionID: revision.ID, CorpusID: revision.CorpusID,
			Position: 2, ExternalID: "case-001-pt", QueryLanguage: QueryLanguagePortuguese, AssetLanguage: AssetLanguagePortugueseBrazil,
			Question: "Qual e a obrigacao aplicavel?", ReferenceAnswer: "A obrigacao se aplica.", Category: "direct-provision",
			AuthoritativeEvidenceLanguage: AssetLanguageEnglish, ExpectedOutcome: ExpectedOutcomeAnswer, ReciprocalExternalID: "case-001-en", Checksum: SHA256(strings.Repeat("c", 64)),
		},
	}
}

func validEvidence(revision DatasetRevision, evaluationCase Case) ExpectedEvidence {
	return ExpectedEvidence{
		ID: fixtureID("00000000-0000-4000-8000-000000000030"), DatasetRevisionID: revision.ID, CorpusID: revision.CorpusID,
		CaseID: evaluationCase.ID, SourceAlias: "official-law", Ordinal: 1, DisplayLocator: "Article 1", CanonicalLocator: "article:1", RequiredPropositions: []string{"the obligation applies"},
	}
}

func validSelections(revision DatasetRevision, cases []Case) []StarterSelection {
	return []StarterSelection{
		{
			ID: fixtureID("00000000-0000-4000-8000-000000000040"), DatasetRevisionID: revision.ID, CorpusID: revision.CorpusID,
			CaseID: cases[0].ID, Rank: 1, QueryLanguage: cases[0].QueryLanguage, CaseChecksum: cases[0].Checksum, ReviewEligible: true,
		},
		{
			ID: fixtureID("00000000-0000-4000-8000-000000000041"), DatasetRevisionID: revision.ID, CorpusID: revision.CorpusID,
			CaseID: cases[1].ID, Rank: 1, QueryLanguage: cases[1].QueryLanguage, CaseChecksum: cases[1].Checksum, ReviewEligible: true,
		},
	}
}

func validPublication(revision DatasetRevision) Publication {
	return Publication{
		ID: fixtureID("00000000-0000-4000-8000-000000000060"), DatasetRevisionID: revision.ID, CorpusID: revision.CorpusID,
		ReviewDecision: ReviewDecisionApproved, ReviewerIdentity: "test-reviewer", PublicationState: PublicationStateAvailable,
		ReviewedAt: time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC),
	}
}

func validSnapshot(revision DatasetRevision) Snapshot {
	return Snapshot{
		ID: fixtureID("00000000-0000-4000-8000-000000000070"), CorpusID: revision.CorpusID, ManifestSHA256: revision.ManifestSHA256,
	}
}

func validSuggestionSet(revision DatasetRevision, snapshot Snapshot) OpeningSuggestionSet {
	return OpeningSuggestionSet{
		ID: fixtureID("00000000-0000-4000-8000-000000000080"), CorpusID: revision.CorpusID, SnapshotID: snapshot.ID,
		SnapshotManifestSHA256: snapshot.ManifestSHA256, DatasetRevisionID: revision.ID, DatasetContentSHA256: revision.ContentSHA256,
		SelectionPolicyVersion: "v1", PublishedBy: "test-reviewer",
	}
}

func fixtureID(value string) uuid.UUID {
	return uuid.MustParse(value)
}
