// Package domain defines safe, immutable values for corpus opening suggestions.
package domain

import (
	"errors"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"
)

var (
	// ErrInvalidInput identifies malformed opening-suggestion data.
	ErrInvalidInput = errors.New("opening suggestion input is invalid")
	// ErrCorpusMismatch identifies values that are not owned by one corpus.
	ErrCorpusMismatch = errors.New("opening suggestion corpus ownership mismatch")
	// ErrSnapshotMismatch identifies a projection that cannot serve the active release.
	ErrSnapshotMismatch = errors.New("opening suggestion snapshot compatibility mismatch")
	// ErrUnavailableDataset identifies a dataset that is not reviewed and available.
	ErrUnavailableDataset = errors.New("opening suggestion dataset is unavailable")
	// ErrInvalidSelection identifies incomplete, ambiguous, or unsafe ranked selections.
	ErrInvalidSelection = errors.New("opening suggestion selection is invalid")
	// ErrUnresolvedExpectedEvidence identifies a selected case without exactly one resolution per requirement.
	ErrUnresolvedExpectedEvidence = errors.New("opening suggestion expected evidence is unresolved")
)

var (
	identifierPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,127}$`)
	sha256Pattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// QueryLanguage identifies the researcher interface language for a suggestion.
type QueryLanguage string

const (
	// QueryLanguageEnglish identifies English questions.
	QueryLanguageEnglish QueryLanguage = "en"
	// QueryLanguagePortuguese identifies Portuguese questions.
	QueryLanguagePortuguese QueryLanguage = "pt"
)

// SHA256 is the canonical lowercase hexadecimal checksum representation.
type SHA256 string

// Validate confirms the checksum has a canonical SHA-256 representation.
func (checksum SHA256) Validate() error {
	if !sha256Pattern.MatchString(string(checksum)) {
		return ErrInvalidInput
	}
	return nil
}

// Snapshot identifies an immutable corpus snapshot and its manifest.
type Snapshot struct {
	ID             uuid.UUID
	CorpusID       uuid.UUID
	ManifestSHA256 SHA256
}

// Validate confirms the snapshot has a stable corpus-bound identity.
func (snapshot Snapshot) Validate() error {
	if snapshot.ID == uuid.Nil || snapshot.CorpusID == uuid.Nil {
		return ErrInvalidInput
	}
	return snapshot.ManifestSHA256.Validate()
}

// DatasetPublication is the reviewed availability state required to publish suggestions.
// It intentionally contains no reviewer note, answer, evidence, scoring, provider, or prompt data.
type DatasetPublication struct {
	ID                   uuid.UUID
	DatasetRevisionID    uuid.UUID
	CorpusID             uuid.UUID
	DatasetContentSHA256 SHA256
	ReviewDecision       string
	PublicationState     string
}

// ValidateAvailable confirms this is the exact reviewed available dataset revision.
func (publication DatasetPublication) ValidateAvailable(corpusID, datasetRevisionID uuid.UUID) error {
	if publication.ID == uuid.Nil || publication.CorpusID == uuid.Nil || publication.DatasetRevisionID == uuid.Nil ||
		publication.CorpusID != corpusID || publication.DatasetRevisionID != datasetRevisionID {
		if publication.CorpusID != corpusID {
			return ErrCorpusMismatch
		}
		return ErrInvalidInput
	}
	if err := publication.DatasetContentSHA256.Validate(); err != nil {
		return err
	}
	if publication.ReviewDecision != "approved" || publication.PublicationState != "available" {
		return ErrUnavailableDataset
	}
	return nil
}

// StarterCase is the safe selected-case data needed to render one opening suggestion.
// It deliberately excludes the evaluation reference answer and expected-evidence content.
type StarterCase struct {
	ID                       uuid.UUID
	CorpusID                 uuid.UUID
	DatasetRevisionID        uuid.UUID
	ExternalCaseID           string
	ReciprocalExternalCaseID string
	Checksum                 SHA256
	QueryLanguage            QueryLanguage
	Rank                     int
	Question                 string
}

// ExpectedEvidenceRequirement identifies one reviewed requirement without exposing its locator or propositions.
type ExpectedEvidenceRequirement struct {
	ID                uuid.UUID
	CorpusID          uuid.UUID
	DatasetRevisionID uuid.UUID
	CaseID            uuid.UUID
}

// ExpectedEvidenceResolution proves one reviewed requirement resolved in a fixed snapshot.
// It deliberately stores no legal text, locator, proposition, or retrieved provider data.
type ExpectedEvidenceResolution struct {
	ExpectedEvidenceID uuid.UUID
	CorpusID           uuid.UUID
	DatasetRevisionID  uuid.UUID
	CaseID             uuid.UUID
	SnapshotID         uuid.UUID
}

// OpeningSuggestionSet is an append-only snapshot-bound projection identity.
type OpeningSuggestionSet struct {
	ID                     uuid.UUID
	CorpusID               uuid.UUID
	SnapshotID             uuid.UUID
	SnapshotManifestSHA256 SHA256
	DatasetRevisionID      uuid.UUID
	DatasetContentSHA256   SHA256
	SelectionPolicyVersion string
	PublishedBy            string
}

// OpeningSuggestionItem is safe to persist in the projection and contains only rendered-question data.
type OpeningSuggestionItem struct {
	SuggestionSetID   uuid.UUID
	CorpusID          uuid.UUID
	DatasetRevisionID uuid.UUID
	Rank              int
	CaseID            uuid.UUID
	CaseChecksum      SHA256
	QueryLanguage     QueryLanguage
	ExternalCaseID    string
	Question          string
}

// PublicOpeningSuggestion is the only question payload intended for researcher-facing reads.
type PublicOpeningSuggestion struct {
	CaseID   string
	Rank     int
	Question string
}

// Public returns the contract-safe question projection.
func (item OpeningSuggestionItem) Public() PublicOpeningSuggestion {
	return PublicOpeningSuggestion{CaseID: item.ExternalCaseID, Rank: item.Rank, Question: item.Question}
}

// Projection is the complete append-only publication written by a persistence adapter.
type Projection struct {
	Set   OpeningSuggestionSet
	Items []OpeningSuggestionItem
}

// MatchesActiveSnapshot reports whether this projection is visible for the current active release.
func (projection Projection) MatchesActiveSnapshot(active Snapshot) bool {
	return active.Validate() == nil && projection.Set.CorpusID == active.CorpusID &&
		projection.Set.SnapshotID == active.ID && projection.Set.SnapshotManifestSHA256 == active.ManifestSHA256
}

// PublicationInput contains only identifiers and safe rendered-question data needed to publish a projection.
type PublicationInput struct {
	SuggestionSetID        uuid.UUID
	CorpusID               uuid.UUID
	DatasetRevisionID      uuid.UUID
	Snapshot               Snapshot
	ActiveSnapshot         Snapshot
	DatasetPublication     DatasetPublication
	SelectionPolicyVersion string
	PublishedBy            string
	StarterCases           []StarterCase
	ExpectedEvidence       []ExpectedEvidenceRequirement
	ResolvedEvidence       []ExpectedEvidenceResolution
}

// NewProjection validates all publication gates before constructing an append-only safe projection.
func NewProjection(input PublicationInput) (Projection, error) {
	if input.SuggestionSetID == uuid.Nil || input.CorpusID == uuid.Nil || input.DatasetRevisionID == uuid.Nil ||
		strings.TrimSpace(input.SelectionPolicyVersion) == "" || strings.TrimSpace(input.PublishedBy) == "" {
		return Projection{}, ErrInvalidInput
	}
	if err := input.Snapshot.Validate(); err != nil {
		return Projection{}, err
	}
	if err := input.ActiveSnapshot.Validate(); err != nil {
		return Projection{}, err
	}
	if input.Snapshot.CorpusID != input.CorpusID || input.ActiveSnapshot.CorpusID != input.CorpusID {
		return Projection{}, ErrCorpusMismatch
	}
	if input.Snapshot.ID != input.ActiveSnapshot.ID || input.Snapshot.ManifestSHA256 != input.ActiveSnapshot.ManifestSHA256 {
		return Projection{}, ErrSnapshotMismatch
	}
	if err := input.DatasetPublication.ValidateAvailable(input.CorpusID, input.DatasetRevisionID); err != nil {
		return Projection{}, err
	}
	items, err := validateStarterCases(input.CorpusID, input.DatasetRevisionID, input.StarterCases)
	if err != nil {
		return Projection{}, err
	}
	if err := validateEvidence(input.CorpusID, input.DatasetRevisionID, input.Snapshot.ID, input.StarterCases, input.ExpectedEvidence, input.ResolvedEvidence); err != nil {
		return Projection{}, err
	}

	projection := Projection{
		Set: OpeningSuggestionSet{
			ID: input.SuggestionSetID, CorpusID: input.CorpusID, SnapshotID: input.Snapshot.ID,
			SnapshotManifestSHA256: input.Snapshot.ManifestSHA256,
			DatasetRevisionID:      input.DatasetPublication.DatasetRevisionID,
			DatasetContentSHA256:   input.DatasetPublication.DatasetContentSHA256,
			SelectionPolicyVersion: input.SelectionPolicyVersion, PublishedBy: input.PublishedBy,
		},
		Items: items,
	}
	for index := range projection.Items {
		projection.Items[index].SuggestionSetID = projection.Set.ID
	}
	return projection, nil
}

func validateStarterCases(corpusID, datasetRevisionID uuid.UUID, starterCases []StarterCase) ([]OpeningSuggestionItem, error) {
	if len(starterCases) == 0 || len(starterCases) > 10 {
		return nil, ErrInvalidSelection
	}
	items, byExternalID, ranks, err := collectStarterCaseItems(corpusID, datasetRevisionID, starterCases)
	if err != nil {
		return nil, err
	}
	if !validStarterRanks(starterCases, ranks) || !validStarterReciprocals(starterCases, byExternalID) {
		return nil, ErrInvalidSelection
	}
	sortStarterItems(items)
	return items, nil
}

func collectStarterCaseItems(corpusID, datasetRevisionID uuid.UUID, starterCases []StarterCase) ([]OpeningSuggestionItem, map[string]StarterCase, map[int]struct{}, error) {
	byExternalID := make(map[string]StarterCase, len(starterCases))
	seenCaseIDs := make(map[uuid.UUID]struct{}, len(starterCases))
	seenRankLanguages := make(map[rankLanguage]struct{}, len(starterCases))
	seenRanks := make(map[int]struct{}, 5)
	items := make([]OpeningSuggestionItem, 0, len(starterCases))
	for _, starterCase := range starterCases {
		if err := validateStarterCase(corpusID, datasetRevisionID, starterCase); err != nil {
			return nil, nil, nil, err
		}
		if _, duplicate := byExternalID[starterCase.ExternalCaseID]; duplicate {
			return nil, nil, nil, ErrInvalidSelection
		}
		if _, duplicate := seenCaseIDs[starterCase.ID]; duplicate {
			return nil, nil, nil, ErrInvalidSelection
		}
		key := rankLanguage{rank: starterCase.Rank, language: starterCase.QueryLanguage}
		if _, duplicate := seenRankLanguages[key]; duplicate {
			return nil, nil, nil, ErrInvalidSelection
		}
		byExternalID[starterCase.ExternalCaseID] = starterCase
		seenCaseIDs[starterCase.ID] = struct{}{}
		seenRankLanguages[key] = struct{}{}
		seenRanks[starterCase.Rank] = struct{}{}
		items = append(items, OpeningSuggestionItem{
			CorpusID: starterCase.CorpusID, DatasetRevisionID: starterCase.DatasetRevisionID,
			Rank: starterCase.Rank, CaseID: starterCase.ID, CaseChecksum: starterCase.Checksum,
			QueryLanguage: starterCase.QueryLanguage, ExternalCaseID: starterCase.ExternalCaseID,
			Question: starterCase.Question,
		})
	}
	return items, byExternalID, seenRanks, nil
}

func validStarterRanks(starterCases []StarterCase, seenRanks map[int]struct{}) bool {
	if len(starterCases) != len(seenRanks)*2 || len(seenRanks) > 5 {
		return false
	}
	for rank := 1; rank <= len(seenRanks); rank++ {
		if _, found := seenRanks[rank]; !found {
			return false
		}
	}
	return true
}

func validStarterReciprocals(starterCases []StarterCase, byExternalID map[string]StarterCase) bool {
	for _, starterCase := range starterCases {
		reciprocal, found := byExternalID[starterCase.ReciprocalExternalCaseID]
		if !found || reciprocal.ReciprocalExternalCaseID != starterCase.ExternalCaseID ||
			reciprocal.QueryLanguage == starterCase.QueryLanguage || reciprocal.Rank != starterCase.Rank {
			return false
		}
	}
	return true
}

func sortStarterItems(items []OpeningSuggestionItem) {
	sort.Slice(items, func(left, right int) bool {
		if items[left].Rank != items[right].Rank {
			return items[left].Rank < items[right].Rank
		}
		return items[left].QueryLanguage < items[right].QueryLanguage
	})
}

func validateStarterCase(corpusID, datasetRevisionID uuid.UUID, starterCase StarterCase) error {
	if starterCase.ID == uuid.Nil || starterCase.CorpusID != corpusID || starterCase.DatasetRevisionID != datasetRevisionID {
		if starterCase.CorpusID != corpusID {
			return ErrCorpusMismatch
		}
		return ErrInvalidSelection
	}
	if !identifierPattern.MatchString(starterCase.ExternalCaseID) ||
		!identifierPattern.MatchString(starterCase.ReciprocalExternalCaseID) ||
		starterCase.ExternalCaseID == starterCase.ReciprocalExternalCaseID || starterCase.Rank < 1 || starterCase.Rank > 5 ||
		!validQueryLanguage(starterCase.QueryLanguage) || strings.TrimSpace(starterCase.Question) == "" {
		return ErrInvalidSelection
	}
	if err := starterCase.Checksum.Validate(); err != nil {
		return err
	}
	return nil
}

func validateEvidence(
	corpusID, datasetRevisionID, snapshotID uuid.UUID,
	starterCases []StarterCase,
	requirements []ExpectedEvidenceRequirement,
	resolutions []ExpectedEvidenceResolution,
) error {
	caseIDs := make(map[uuid.UUID]struct{}, len(starterCases))
	for _, starterCase := range starterCases {
		caseIDs[starterCase.ID] = struct{}{}
	}
	requirementByID, err := validateEvidenceRequirements(corpusID, datasetRevisionID, caseIDs, requirements)
	if err != nil {
		return err
	}
	return validateEvidenceResolutions(corpusID, datasetRevisionID, snapshotID, requirementByID, resolutions)
}

func validateEvidenceRequirements(corpusID, datasetRevisionID uuid.UUID, caseIDs map[uuid.UUID]struct{}, requirements []ExpectedEvidenceRequirement) (map[uuid.UUID]ExpectedEvidenceRequirement, error) {
	requirementByID := make(map[uuid.UUID]ExpectedEvidenceRequirement, len(requirements))
	requirementCountByCase := make(map[uuid.UUID]int, len(caseIDs))
	for _, requirement := range requirements {
		if requirement.ID == uuid.Nil || requirement.CorpusID != corpusID || requirement.DatasetRevisionID != datasetRevisionID {
			if requirement.CorpusID != corpusID {
				return nil, ErrCorpusMismatch
			}
			return nil, ErrUnresolvedExpectedEvidence
		}
		if _, selected := caseIDs[requirement.CaseID]; !selected {
			return nil, ErrUnresolvedExpectedEvidence
		}
		if _, duplicate := requirementByID[requirement.ID]; duplicate {
			return nil, ErrUnresolvedExpectedEvidence
		}
		requirementByID[requirement.ID] = requirement
		requirementCountByCase[requirement.CaseID]++
	}
	for caseID := range caseIDs {
		if requirementCountByCase[caseID] == 0 {
			return nil, ErrUnresolvedExpectedEvidence
		}
	}
	return requirementByID, nil
}

func validateEvidenceResolutions(corpusID, datasetRevisionID, snapshotID uuid.UUID, requirements map[uuid.UUID]ExpectedEvidenceRequirement, resolutions []ExpectedEvidenceResolution) error {
	resolvedRequirementIDs := make(map[uuid.UUID]struct{}, len(resolutions))
	for _, resolution := range resolutions {
		requirement, found := requirements[resolution.ExpectedEvidenceID]
		if !found || resolution.CorpusID != corpusID || resolution.DatasetRevisionID != datasetRevisionID || resolution.CaseID != requirement.CaseID || resolution.SnapshotID != snapshotID {
			if resolution.CorpusID != corpusID {
				return ErrCorpusMismatch
			}
			return ErrUnresolvedExpectedEvidence
		}
		if _, duplicate := resolvedRequirementIDs[resolution.ExpectedEvidenceID]; duplicate {
			return ErrUnresolvedExpectedEvidence
		}
		resolvedRequirementIDs[resolution.ExpectedEvidenceID] = struct{}{}
	}
	if len(resolvedRequirementIDs) != len(requirements) {
		return ErrUnresolvedExpectedEvidence
	}
	return nil
}

type rankLanguage struct {
	rank     int
	language QueryLanguage
}

func validQueryLanguage(language QueryLanguage) bool {
	return language == QueryLanguageEnglish || language == QueryLanguagePortuguese
}

// CloneItems returns a detached copy of a projection's safe rendered-question items.
func (projection Projection) CloneItems() []OpeningSuggestionItem {
	return slices.Clone(projection.Items)
}
