// Package domain defines immutable evaluation dataset values and validation rules.
package domain

import (
	"errors"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrInvalidInput identifies a malformed evaluation domain value.
	ErrInvalidInput = errors.New("evaluation input is invalid")
	// ErrCorpusMismatch identifies values that do not belong to the same corpus.
	ErrCorpusMismatch = errors.New("evaluation corpus ownership mismatch")
	// ErrSnapshotMismatch identifies a suggestion set that cannot serve an active snapshot.
	ErrSnapshotMismatch = errors.New("evaluation snapshot compatibility mismatch")
	// ErrInvalidPair identifies an incomplete or non-reciprocal language pair.
	ErrInvalidPair = errors.New("evaluation case pair is invalid")
	// ErrInvalidStarterSelection identifies an incomplete or unsafe starter selection.
	ErrInvalidStarterSelection = errors.New("evaluation starter selection is invalid")
	// ErrInvalidLifecycleTransition identifies an unsupported publication state transition.
	ErrInvalidLifecycleTransition = errors.New("evaluation publication transition is invalid")
)

var (
	safeIdentifierPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,127}$`)
	sha256Pattern           = regexp.MustCompile(`^[0-9a-f]{64}$`)
	canonicalLocatorPattern = regexp.MustCompile(`^[a-z][a-z-]*:[a-z0-9.-]+(?:/[a-z][a-z-]*:[a-z0-9.-]+)*$`)
)

// QueryLanguage identifies the language in which a researcher asks a question.
type QueryLanguage string

const (
	QueryLanguageEnglish    QueryLanguage = "en"
	QueryLanguagePortuguese QueryLanguage = "pt"
)

// AssetLanguage identifies the language preserved in a source dataset asset.
type AssetLanguage string

const (
	AssetLanguageEnglish          AssetLanguage = "en"
	AssetLanguagePortugueseBrazil AssetLanguage = "pt-BR"
)

// ExpectedOutcome defines whether a case expects an answer or an abstention.
type ExpectedOutcome string

const (
	ExpectedOutcomeAnswer  ExpectedOutcome = "answer"
	ExpectedOutcomeAbstain ExpectedOutcome = "abstain"
)

// PublicationState describes the append-only review lifecycle of a dataset revision.
type PublicationState string

const (
	PublicationStateDraft      PublicationState = "draft"
	PublicationStateAvailable  PublicationState = "available"
	PublicationStateSuperseded PublicationState = "superseded"
	PublicationStateWithdrawn  PublicationState = "withdrawn"
)

// ReviewDecision records the review outcome associated with one publication record.
type ReviewDecision string

const (
	ReviewDecisionPending  ReviewDecision = "pending"
	ReviewDecisionApproved ReviewDecision = "approved"
	ReviewDecisionRejected ReviewDecision = "rejected"
)

// SHA256 is a lowercase hexadecimal SHA-256 checksum.
type SHA256 string

// Validate reports whether checksum has the stable SHA-256 representation used by the catalog.
func (checksum SHA256) Validate() error {
	if !sha256Pattern.MatchString(string(checksum)) {
		return ErrInvalidInput
	}
	return nil
}

// DatasetRevision identifies one immutable imported evaluation dataset revision.
type DatasetRevision struct {
	ID                            uuid.UUID
	CorpusID                      uuid.UUID
	DatasetKey                    string
	SemanticRevision              string
	Jurisdiction                  string
	ManifestSHA256                SHA256
	JSONLSHA256                   SHA256
	ContentSHA256                 SHA256
	DeclaredSnapshotDate          time.Time
	QueryLanguages                []QueryLanguage
	AuthoritativeEvidenceLanguage AssetLanguage
}

// Validate confirms the revision identity and immutable manifest metadata.
//
// A successful validation detaches the revision from the caller-owned language
// slice so a later mutation of the import buffer cannot change the validated
// domain value.
func (revision *DatasetRevision) Validate() error {
	if revision.ID == uuid.Nil || revision.CorpusID == uuid.Nil ||
		!safeIdentifier(revision.DatasetKey) || strings.TrimSpace(revision.SemanticRevision) == "" ||
		strings.TrimSpace(revision.Jurisdiction) == "" || revision.DeclaredSnapshotDate.IsZero() ||
		!validAssetLanguage(revision.AuthoritativeEvidenceLanguage) {
		return ErrInvalidInput
	}
	if err := revision.ManifestSHA256.Validate(); err != nil {
		return err
	}
	if err := revision.JSONLSHA256.Validate(); err != nil {
		return err
	}
	if err := revision.ContentSHA256.Validate(); err != nil {
		return err
	}
	if len(revision.QueryLanguages) == 0 || len(revision.QueryLanguages) > 2 {
		return ErrInvalidInput
	}

	seen := make(map[QueryLanguage]struct{}, len(revision.QueryLanguages))
	for _, language := range revision.QueryLanguages {
		if !validQueryLanguage(language) {
			return ErrInvalidInput
		}
		if _, duplicate := seen[language]; duplicate {
			return ErrInvalidInput
		}
		seen[language] = struct{}{}
	}
	revision.QueryLanguages = slices.Clone(revision.QueryLanguages)
	return nil
}

// SourceRequirement is one official source declared by a dataset revision.
type SourceRequirement struct {
	ID                uuid.UUID
	DatasetRevisionID uuid.UUID
	CorpusID          uuid.UUID
	SourceAlias       string
	Title             string
	OfficialURL       string
	IssuingAuthority  string
	DocumentType      string
	AuthorityRole     string
	CorpusSourceID    *uuid.UUID
}

// ValidateAgainst confirms source ownership and safe official-source metadata.
func (source SourceRequirement) ValidateAgainst(revision DatasetRevision) error {
	if err := revision.Validate(); err != nil {
		return err
	}
	if source.ID == uuid.Nil || source.DatasetRevisionID != revision.ID || source.CorpusID != revision.CorpusID {
		return ownershipError(source.CorpusID, revision.CorpusID)
	}
	if !safeIdentifier(source.SourceAlias) || strings.TrimSpace(source.Title) == "" ||
		strings.TrimSpace(source.IssuingAuthority) == "" || strings.TrimSpace(source.DocumentType) == "" ||
		strings.TrimSpace(source.AuthorityRole) == "" || !validOfficialURL(source.OfficialURL) {
		return ErrInvalidInput
	}
	if source.CorpusSourceID != nil && *source.CorpusSourceID == uuid.Nil {
		return ErrInvalidInput
	}
	return nil
}

// Case is one immutable multilingual question and expected answer.
type Case struct {
	ID                            uuid.UUID
	DatasetRevisionID             uuid.UUID
	CorpusID                      uuid.UUID
	Position                      int
	ExternalID                    string
	QueryLanguage                 QueryLanguage
	AssetLanguage                 AssetLanguage
	Question                      string
	ReferenceAnswer               string
	Category                      string
	AuthoritativeEvidenceLanguage AssetLanguage
	ExpectedOutcome               ExpectedOutcome
	ExpectedReasonCode            string
	ReciprocalExternalID          string
	Checksum                      SHA256
}

// ValidateAgainst confirms a case belongs to the revision and has valid independent fields.
func (evaluationCase *Case) ValidateAgainst(revision DatasetRevision) error {
	if err := revision.Validate(); err != nil {
		return err
	}
	if evaluationCase.ID == uuid.Nil || evaluationCase.DatasetRevisionID != revision.ID || evaluationCase.CorpusID != revision.CorpusID {
		return ownershipError(evaluationCase.CorpusID, revision.CorpusID)
	}
	if evaluationCase.Position < 1 || !safeIdentifier(evaluationCase.ExternalID) ||
		!safeIdentifier(evaluationCase.ReciprocalExternalID) ||
		evaluationCase.ExternalID == evaluationCase.ReciprocalExternalID ||
		strings.TrimSpace(evaluationCase.Question) == "" || strings.TrimSpace(evaluationCase.ReferenceAnswer) == "" ||
		strings.TrimSpace(evaluationCase.Category) == "" || !validQueryLanguage(evaluationCase.QueryLanguage) ||
		!validAssetLanguage(evaluationCase.AssetLanguage) || !validAssetLanguage(evaluationCase.AuthoritativeEvidenceLanguage) ||
		!matchingAssetLanguage(evaluationCase.QueryLanguage, evaluationCase.AssetLanguage) {
		return ErrInvalidInput
	}
	if err := evaluationCase.Checksum.Validate(); err != nil {
		return err
	}
	if !revisionIncludesLanguage(revision, evaluationCase.QueryLanguage) {
		return ErrInvalidInput
	}
	if evaluationCase.ExpectedOutcome == "" {
		evaluationCase.ExpectedOutcome = ExpectedOutcomeAnswer
	}
	switch evaluationCase.ExpectedOutcome {
	case ExpectedOutcomeAnswer:
		if evaluationCase.ExpectedReasonCode != "" {
			return ErrInvalidInput
		}
	case ExpectedOutcomeAbstain:
		if !safeIdentifier(evaluationCase.ExpectedReasonCode) {
			return ErrInvalidInput
		}
	default:
		return ErrInvalidInput
	}
	return nil
}

// ExpectedEvidence identifies an ordered source locator required to evaluate one case.
type ExpectedEvidence struct {
	ID                   uuid.UUID
	DatasetRevisionID    uuid.UUID
	CorpusID             uuid.UUID
	CaseID               uuid.UUID
	SourceAlias          string
	Ordinal              int
	DisplayLocator       string
	CanonicalLocator     string
	RequiredPropositions []string
}

// ValidateAgainst confirms evidence belongs to its case and a declared source requirement.
//
// A successful validation detaches the evidence from the caller-owned
// propositions slice so later importer mutations cannot alter the validated
// domain value.
func (evidence *ExpectedEvidence) ValidateAgainst(revision DatasetRevision, evaluationCase Case, sources []SourceRequirement) error {
	if err := evaluationCase.ValidateAgainst(revision); err != nil {
		return err
	}
	if evidence.ID == uuid.Nil || evidence.DatasetRevisionID != revision.ID || evidence.CorpusID != revision.CorpusID || evidence.CaseID != evaluationCase.ID {
		return ownershipError(evidence.CorpusID, revision.CorpusID)
	}
	if evidence.Ordinal < 1 || !safeIdentifier(evidence.SourceAlias) || strings.TrimSpace(evidence.DisplayLocator) == "" ||
		!canonicalLocatorPattern.MatchString(evidence.CanonicalLocator) || len(evidence.RequiredPropositions) == 0 {
		return ErrInvalidInput
	}
	for _, proposition := range evidence.RequiredPropositions {
		if strings.TrimSpace(proposition) == "" {
			return ErrInvalidInput
		}
	}
	for _, source := range sources {
		if source.SourceAlias == evidence.SourceAlias {
			if err := source.ValidateAgainst(revision); err != nil {
				return err
			}
			evidence.RequiredPropositions = slices.Clone(evidence.RequiredPropositions)
			return nil
		}
	}
	return ErrInvalidInput
}

// ValidatePairedCases confirms all cases form reciprocal pairs in distinct query languages.
func ValidatePairedCases(revision DatasetRevision, cases []Case) error {
	if err := revision.Validate(); err != nil {
		return err
	}
	if len(cases) == 0 {
		return ErrInvalidPair
	}

	byExternalID := make(map[string]Case, len(cases))
	positions := make(map[int]struct{}, len(cases))
	for index := range cases {
		if err := cases[index].ValidateAgainst(revision); err != nil {
			return err
		}
		evaluationCase := cases[index]
		if _, duplicate := byExternalID[evaluationCase.ExternalID]; duplicate {
			return ErrInvalidPair
		}
		if _, duplicate := positions[evaluationCase.Position]; duplicate {
			return ErrInvalidPair
		}
		byExternalID[evaluationCase.ExternalID] = evaluationCase
		positions[evaluationCase.Position] = struct{}{}
	}
	for _, evaluationCase := range cases {
		pairedCase, found := byExternalID[evaluationCase.ReciprocalExternalID]
		if !found || pairedCase.ReciprocalExternalID != evaluationCase.ExternalID || pairedCase.QueryLanguage == evaluationCase.QueryLanguage {
			return ErrInvalidPair
		}
	}
	return nil
}

// StarterSelection chooses one reviewed case for an empty-chat opening suggestion rank.
type StarterSelection struct {
	ID                uuid.UUID
	DatasetRevisionID uuid.UUID
	CorpusID          uuid.UUID
	CaseID            uuid.UUID
	Rank              int
	QueryLanguage     QueryLanguage
	CaseChecksum      SHA256
	ReviewEligible    bool
}

// ValidateStarterSelections confirms selected cases form reviewed reciprocal language pairs.
func ValidateStarterSelections(revision DatasetRevision, cases []Case, selections []StarterSelection) error {
	if err := ValidatePairedCases(revision, cases); err != nil {
		return err
	}
	if len(selections) == 0 {
		return nil
	}

	caseByID := make(map[uuid.UUID]Case, len(cases))
	for _, evaluationCase := range cases {
		caseByID[evaluationCase.ID] = evaluationCase
	}
	selectionByCaseID := make(map[uuid.UUID]StarterSelection, len(selections))
	selectionByRankLanguage := make(map[starterRankLanguage]struct{}, len(selections))
	for _, selection := range selections {
		if selection.ID == uuid.Nil || selection.DatasetRevisionID != revision.ID || selection.CorpusID != revision.CorpusID {
			return starterOwnershipError(selection.CorpusID, revision.CorpusID)
		}
		if selection.Rank < 1 || selection.Rank > 5 || !validQueryLanguage(selection.QueryLanguage) || !selection.ReviewEligible {
			return ErrInvalidStarterSelection
		}
		if err := selection.CaseChecksum.Validate(); err != nil {
			return err
		}
		evaluationCase, found := caseByID[selection.CaseID]
		if !found || evaluationCase.QueryLanguage != selection.QueryLanguage || evaluationCase.Checksum != selection.CaseChecksum {
			return ErrInvalidStarterSelection
		}
		if _, duplicate := selectionByCaseID[selection.CaseID]; duplicate {
			return ErrInvalidStarterSelection
		}
		key := starterRankLanguage{rank: selection.Rank, language: selection.QueryLanguage}
		if _, duplicate := selectionByRankLanguage[key]; duplicate {
			return ErrInvalidStarterSelection
		}
		selectionByCaseID[selection.CaseID] = selection
		selectionByRankLanguage[key] = struct{}{}
	}
	for _, selection := range selections {
		evaluationCase := caseByID[selection.CaseID]
		pairedCase := findCaseByExternalID(cases, evaluationCase.ReciprocalExternalID)
		pairedSelection, found := selectionByCaseID[pairedCase.ID]
		if !found || pairedSelection.Rank != selection.Rank || pairedSelection.QueryLanguage == selection.QueryLanguage {
			return ErrInvalidStarterSelection
		}
	}
	return nil
}

// Publication records one immutable review decision in the dataset lifecycle.
type Publication struct {
	ID                uuid.UUID
	DatasetRevisionID uuid.UUID
	CorpusID          uuid.UUID
	ReviewDecision    ReviewDecision
	ReviewerIdentity  string
	ReviewNote        string
	PublicationState  PublicationState
	ReviewedAt        time.Time
}

// Validate confirms publication fields that do not depend on a dataset revision lookup.
func (publication Publication) Validate() error {
	if publication.ID == uuid.Nil || publication.DatasetRevisionID == uuid.Nil || publication.CorpusID == uuid.Nil {
		return ErrInvalidInput
	}
	if strings.TrimSpace(publication.ReviewerIdentity) == "" || publication.ReviewedAt.IsZero() ||
		!validReviewDecision(publication.ReviewDecision) || !validPublicationState(publication.PublicationState) {
		return ErrInvalidInput
	}
	if len([]rune(publication.ReviewNote)) > 2000 {
		return ErrInvalidInput
	}
	if publication.PublicationState == PublicationStateAvailable && publication.ReviewDecision != ReviewDecisionApproved {
		return ErrInvalidInput
	}
	return nil
}

// ValidateAgainst confirms publication ownership and state-specific review requirements.
func (publication Publication) ValidateAgainst(revision DatasetRevision) error {
	if err := revision.Validate(); err != nil {
		return err
	}
	if err := publication.Validate(); err != nil {
		return err
	}
	if publication.DatasetRevisionID != revision.ID || publication.CorpusID != revision.CorpusID {
		return ownershipError(publication.CorpusID, revision.CorpusID)
	}
	return nil
}

// ValidatePublicationTransition applies the append-only publication lifecycle rules.
func ValidatePublicationTransition(previous, next PublicationState) error {
	if !validPublicationState(previous) || !validPublicationState(next) {
		return ErrInvalidLifecycleTransition
	}
	switch previous {
	case PublicationStateDraft:
		if next == PublicationStateAvailable || next == PublicationStateWithdrawn {
			return nil
		}
	case PublicationStateAvailable:
		if next == PublicationStateSuperseded || next == PublicationStateWithdrawn {
			return nil
		}
	}
	return ErrInvalidLifecycleTransition
}

// Snapshot identifies the active immutable corpus snapshot used by a projection.
type Snapshot struct {
	ID             uuid.UUID
	CorpusID       uuid.UUID
	ManifestSHA256 SHA256
}

// Validate reports whether the snapshot has a stable corpus-bound identity.
func (snapshot Snapshot) Validate() error {
	if snapshot.ID == uuid.Nil || snapshot.CorpusID == uuid.Nil {
		return ErrInvalidInput
	}
	return snapshot.ManifestSHA256.Validate()
}

// OpeningSuggestionSet is an immutable, snapshot-bound projection of an available revision.
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

// ValidateAgainst confirms a suggestion set is owned by an available revision and its named snapshot.
func (set OpeningSuggestionSet) ValidateAgainst(revision DatasetRevision, publication Publication, snapshot Snapshot) error {
	if err := revision.Validate(); err != nil {
		return err
	}
	if err := publication.ValidateAgainst(revision); err != nil {
		return err
	}
	if err := snapshot.Validate(); err != nil {
		return err
	}
	if publication.PublicationState != PublicationStateAvailable || publication.ReviewDecision != ReviewDecisionApproved {
		return ErrInvalidInput
	}
	if set.ID == uuid.Nil || set.CorpusID != revision.CorpusID || set.CorpusID != snapshot.CorpusID ||
		set.DatasetRevisionID != revision.ID || set.SnapshotID != snapshot.ID ||
		set.SnapshotManifestSHA256 != snapshot.ManifestSHA256 || set.DatasetContentSHA256 != revision.ContentSHA256 {
		return ownershipOrSnapshotError(set, revision, snapshot)
	}
	if strings.TrimSpace(set.SelectionPolicyVersion) == "" || strings.TrimSpace(set.PublishedBy) == "" {
		return ErrInvalidInput
	}
	return nil
}

// ValidateActiveSnapshot confirms the set can safely serve the supplied active snapshot.
func (set OpeningSuggestionSet) ValidateActiveSnapshot(active Snapshot) error {
	if err := active.Validate(); err != nil {
		return err
	}
	if set.CorpusID != active.CorpusID || set.SnapshotID != active.ID || set.SnapshotManifestSHA256 != active.ManifestSHA256 {
		return ErrSnapshotMismatch
	}
	return nil
}

// OpeningSuggestionItem is the immutable selection from which a public suggestion is rendered.
type OpeningSuggestionItem struct {
	ID                uuid.UUID
	SuggestionSetID   uuid.UUID
	CorpusID          uuid.UUID
	DatasetRevisionID uuid.UUID
	Rank              int
	CaseID            uuid.UUID
	CaseChecksum      SHA256
	QueryLanguage     QueryLanguage
	Question          string
}

// ValidateAgainst confirms an opening-suggestion item remains bound to its selected case.
func (item OpeningSuggestionItem) ValidateAgainst(set OpeningSuggestionSet, evaluationCase Case) error {
	if item.ID == uuid.Nil || item.SuggestionSetID != set.ID || item.CorpusID != set.CorpusID ||
		item.DatasetRevisionID != set.DatasetRevisionID || item.CaseID != evaluationCase.ID ||
		item.CorpusID != evaluationCase.CorpusID || item.DatasetRevisionID != evaluationCase.DatasetRevisionID {
		if item.CorpusID != set.CorpusID || item.CorpusID != evaluationCase.CorpusID {
			return ErrCorpusMismatch
		}
		return ErrInvalidInput
	}
	if item.Rank < 1 || item.Rank > 5 || item.QueryLanguage != evaluationCase.QueryLanguage ||
		item.CaseChecksum != evaluationCase.Checksum || item.Question != evaluationCase.Question {
		return ErrInvalidInput
	}
	return item.CaseChecksum.Validate()
}

// PublicOpeningSuggestion is deliberately safe for the normal chat read contract.
type PublicOpeningSuggestion struct {
	CaseID   string
	Rank     int
	Question string
}

// Public returns a contract-safe suggestion without answers, evidence, or evaluation metadata.
func (item OpeningSuggestionItem) Public(externalCaseID string) (PublicOpeningSuggestion, error) {
	if item.Rank < 1 || item.Rank > 5 || !safeIdentifier(externalCaseID) ||
		!validQueryLanguage(item.QueryLanguage) || strings.TrimSpace(item.Question) == "" {
		return PublicOpeningSuggestion{}, ErrInvalidInput
	}
	if err := item.CaseChecksum.Validate(); err != nil {
		return PublicOpeningSuggestion{}, err
	}
	return PublicOpeningSuggestion{CaseID: externalCaseID, Rank: item.Rank, Question: item.Question}, nil
}

type starterRankLanguage struct {
	rank     int
	language QueryLanguage
}

func safeIdentifier(value string) bool {
	return safeIdentifierPattern.MatchString(value)
}

func validQueryLanguage(language QueryLanguage) bool {
	return language == QueryLanguageEnglish || language == QueryLanguagePortuguese
}

func validAssetLanguage(language AssetLanguage) bool {
	return language == AssetLanguageEnglish || language == AssetLanguagePortugueseBrazil
}

func matchingAssetLanguage(queryLanguage QueryLanguage, assetLanguage AssetLanguage) bool {
	return (queryLanguage == QueryLanguageEnglish && assetLanguage == AssetLanguageEnglish) ||
		(queryLanguage == QueryLanguagePortuguese && assetLanguage == AssetLanguagePortugueseBrazil)
}

func revisionIncludesLanguage(revision DatasetRevision, language QueryLanguage) bool {
	for _, supportedLanguage := range revision.QueryLanguages {
		if supportedLanguage == language {
			return true
		}
	}
	return false
}

func validOfficialURL(rawURL string) bool {
	parsed, err := url.ParseRequestURI(rawURL)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func validReviewDecision(decision ReviewDecision) bool {
	return decision == ReviewDecisionPending || decision == ReviewDecisionApproved || decision == ReviewDecisionRejected
}

func validPublicationState(state PublicationState) bool {
	return state == PublicationStateDraft || state == PublicationStateAvailable ||
		state == PublicationStateSuperseded || state == PublicationStateWithdrawn
}

func ownershipError(valueCorpusID, expectedCorpusID uuid.UUID) error {
	if valueCorpusID != expectedCorpusID {
		return ErrCorpusMismatch
	}
	return ErrInvalidInput
}

func starterOwnershipError(valueCorpusID, expectedCorpusID uuid.UUID) error {
	if valueCorpusID != expectedCorpusID {
		return ErrCorpusMismatch
	}
	return ErrInvalidStarterSelection
}

func ownershipOrSnapshotError(set OpeningSuggestionSet, revision DatasetRevision, snapshot Snapshot) error {
	if set.CorpusID != revision.CorpusID || set.CorpusID != snapshot.CorpusID {
		return ErrCorpusMismatch
	}
	return ErrSnapshotMismatch
}

func findCaseByExternalID(cases []Case, externalID string) Case {
	for _, evaluationCase := range cases {
		if evaluationCase.ExternalID == externalID {
			return evaluationCase
		}
	}
	return Case{}
}
