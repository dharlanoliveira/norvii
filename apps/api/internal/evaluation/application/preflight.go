package application

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	"github.com/google/uuid"
)

var (
	// ErrInvalidPreflightRequest identifies malformed compatibility input.
	ErrInvalidPreflightRequest = errors.New("evaluation compatibility preflight request is invalid")
	// ErrDatasetUnavailable identifies a revision whose latest review is not approved and available.
	ErrDatasetUnavailable = errors.New("evaluation dataset is not available")
	// ErrPreflightCorpusMismatch identifies a revision selected for another corpus.
	ErrPreflightCorpusMismatch = errors.New("evaluation dataset corpus does not match the request")
	// ErrSnapshotIncompatible identifies a source absent from the named immutable snapshot.
	ErrSnapshotIncompatible = errors.New("evaluation snapshot is incompatible with the dataset")
	// ErrLocatorUnresolved identifies a required canonical legal locator absent from the named snapshot.
	ErrLocatorUnresolved = errors.New("evaluation snapshot locator is unresolved")
)

const maxMissingRequirements = 32

var canonicalLocatorPattern = regexp.MustCompile(`^[a-z][a-z-]*:[a-z0-9.-]+(/[a-z][a-z-]*:[a-z0-9.-]+)*$`)

// PreflightRequest fixes evaluation compatibility to one existing dataset revision and snapshot.
type PreflightRequest struct {
	CorpusID          uuid.UUID
	DatasetRevisionID uuid.UUID
	SnapshotID        uuid.UUID
}

// MissingRequirement is a bounded, safe compatibility diagnostic. It never carries document text,
// propositions, prompts, or provider data.
type MissingRequirement struct {
	SourceAlias string `json:"sourceAlias"`
	Locator     string `json:"locator,omitempty"`
	Reason      string `json:"reason"`
}

type missingRequirementGroups struct {
	current    []MissingRequirement
	additional []MissingRequirement
}

// CompatibilityError reports a safe preflight failure and supports errors.Is with its cause.
type CompatibilityError struct {
	cause   error
	missing []MissingRequirement
}

func (err *CompatibilityError) Error() string { return err.cause.Error() }
func (err *CompatibilityError) Unwrap() error { return err.cause }

// MissingRequirements returns a detached bounded copy of safe diagnostics.
func (err *CompatibilityError) MissingRequirements() []MissingRequirement {
	if err == nil {
		return nil
	}
	return slices.Clone(err.missing)
}

// SourceRequirement is the small catalog view required for compatibility checks.
type SourceRequirement struct {
	ID                uuid.UUID
	CorpusID          uuid.UUID
	DatasetRevisionID uuid.UUID
	SourceAlias       string
	CorpusSourceID    *uuid.UUID
}

// ExpectedEvidenceRequirement preserves both the reviewed display locator and the exact canonical
// locator that must resolve. A display locator is never used as a fuzzy matching key.
type ExpectedEvidenceRequirement struct {
	ID                uuid.UUID
	CorpusID          uuid.UUID
	DatasetRevisionID uuid.UUID
	CaseID            uuid.UUID
	SourceAlias       string
	DisplayLocator    string
	CanonicalLocator  string
}

// LocatorRequest asks a persistence adapter to resolve one exact canonical locator in one snapshot.
type LocatorRequest struct {
	DatasetRevisionID uuid.UUID
	CorpusID          uuid.UUID
	SnapshotID        uuid.UUID
	SourceAlias       string
	CanonicalLocator  string
	DisplayLocator    string
}

// ResolvedLocator is immutable provenance returned after an exact snapshot resolution.
type ResolvedLocator struct {
	ExpectedEvidenceID uuid.UUID
	CaseID             uuid.UUID
	CorpusID           uuid.UUID
	SnapshotID         uuid.UUID
	SourceID           uuid.UUID
	SourceRevisionID   uuid.UUID
	DocumentID         uuid.UUID
	UnitID             uuid.UUID
	CanonicalLocator   string
	DisplayLocator     string
	ContentSHA256      string
}

// PreflightCatalog exposes only immutable catalog and snapshot-membership information.
type PreflightCatalog interface {
	DatasetCorpus(context.Context, uuid.UUID) (uuid.UUID, error)
	LatestPublication(context.Context, uuid.UUID, uuid.UUID) (domain.Publication, bool, error)
	SourceRequirements(context.Context, uuid.UUID, uuid.UUID) ([]SourceRequirement, error)
	ExpectedEvidence(context.Context, uuid.UUID, uuid.UUID) ([]ExpectedEvidenceRequirement, error)
	SnapshotContainsSource(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error)
	ResolvePreflightLocator(context.Context, LocatorRequest) (ResolvedLocator, error)
}

// PreflightResult is an all-or-nothing compatibility plan. It intentionally creates neither a run
// nor model work; later tasks may persist this immutable resolution set only after success.
type PreflightResult struct {
	CorpusID          uuid.UUID
	DatasetRevisionID uuid.UUID
	SnapshotID        uuid.UUID
	ResolvedLocators  []ResolvedLocator
}

// PreflightService validates availability, corpus ownership, every source binding and membership,
// and every exact canonical locator before any execution boundary can be reached.
type PreflightService struct{ catalog PreflightCatalog }

// NewPreflightService constructs compatibility checks around caller-owned persistence.
func NewPreflightService(catalog PreflightCatalog) *PreflightService {
	return &PreflightService{catalog: catalog}
}

// Check returns a complete immutable resolution plan or one bounded failure. It has no model,
// agent, run-writer, or queue dependency, so a rejected preflight cannot create model work.
func (service *PreflightService) Check(ctx context.Context, request PreflightRequest) (PreflightResult, error) {
	if service == nil || service.catalog == nil || request.CorpusID == uuid.Nil || request.DatasetRevisionID == uuid.Nil || request.SnapshotID == uuid.Nil {
		return PreflightResult{}, ErrInvalidPreflightRequest
	}
	datasetCorpusID, err := service.catalog.DatasetCorpus(ctx, request.DatasetRevisionID)
	if err != nil {
		return PreflightResult{}, fmt.Errorf("read evaluation dataset corpus: %w", err)
	}
	if datasetCorpusID != request.CorpusID {
		return PreflightResult{}, compatibilityError(ErrPreflightCorpusMismatch, nil)
	}
	publication, found, err := service.catalog.LatestPublication(ctx, request.CorpusID, request.DatasetRevisionID)
	if err != nil {
		return PreflightResult{}, fmt.Errorf("read latest evaluation dataset publication: %w", err)
	}
	if !found || publication.CorpusID != request.CorpusID || publication.DatasetRevisionID != request.DatasetRevisionID ||
		publication.ReviewDecision != domain.ReviewDecisionApproved || publication.PublicationState != domain.PublicationStateAvailable {
		return PreflightResult{}, compatibilityError(ErrDatasetUnavailable, nil)
	}

	sources, err := service.catalog.SourceRequirements(ctx, request.CorpusID, request.DatasetRevisionID)
	if err != nil {
		return PreflightResult{}, fmt.Errorf("read evaluation dataset source requirements: %w", err)
	}
	evidence, err := service.catalog.ExpectedEvidence(ctx, request.CorpusID, request.DatasetRevisionID)
	if err != nil {
		return PreflightResult{}, fmt.Errorf("read evaluation expected evidence: %w", err)
	}
	if len(sources) == 0 {
		return PreflightResult{}, compatibilityError(ErrSnapshotIncompatible, nil)
	}
	if len(evidence) == 0 {
		return PreflightResult{}, compatibilityError(ErrLocatorUnresolved, nil)
	}

	missingSnapshot, err := service.checkSources(ctx, request, sources)
	if err != nil {
		return PreflightResult{}, err
	}
	resolved, missingLocators, err := service.resolveLocators(ctx, request, evidence)
	if err != nil {
		return PreflightResult{}, err
	}
	if len(missingSnapshot) > 0 || len(missingLocators) > 0 {
		missing := appendMissingRequirements(missingRequirementGroups{current: missingSnapshot, additional: missingLocators})
		if len(missingSnapshot) > 0 {
			return PreflightResult{}, compatibilityError(ErrSnapshotIncompatible, missing)
		}
		return PreflightResult{}, compatibilityError(ErrLocatorUnresolved, missing)
	}
	return PreflightResult{
		CorpusID: request.CorpusID, DatasetRevisionID: request.DatasetRevisionID, SnapshotID: request.SnapshotID,
		ResolvedLocators: slices.Clone(resolved),
	}, nil
}

func (service *PreflightService) checkSources(ctx context.Context, request PreflightRequest, sources []SourceRequirement) ([]MissingRequirement, error) {
	missing := make([]MissingRequirement, 0)
	for _, source := range sources {
		if source.ID == uuid.Nil || source.CorpusID != request.CorpusID || source.DatasetRevisionID != request.DatasetRevisionID || strings.TrimSpace(source.SourceAlias) == "" {
			return nil, compatibilityError(ErrPreflightCorpusMismatch, nil)
		}
		if source.CorpusSourceID == nil || *source.CorpusSourceID == uuid.Nil {
			missing = appendMissing(missing, MissingRequirement{SourceAlias: source.SourceAlias, Reason: "The dataset source is not bound to a corpus source."})
			continue
		}
		member, err := service.catalog.SnapshotContainsSource(ctx, request.CorpusID, request.SnapshotID, *source.CorpusSourceID)
		if err != nil {
			return nil, fmt.Errorf("check evaluation snapshot source membership: %w", err)
		}
		if !member {
			missing = appendMissing(missing, MissingRequirement{SourceAlias: source.SourceAlias, Reason: "The required source is not a member of the selected snapshot."})
		}
	}
	return missing, nil
}

func (service *PreflightService) resolveLocators(ctx context.Context, request PreflightRequest, evidence []ExpectedEvidenceRequirement) ([]ResolvedLocator, []MissingRequirement, error) {
	resolved := make([]ResolvedLocator, 0, len(evidence))
	missing := make([]MissingRequirement, 0)
	for _, requirement := range evidence {
		if requirement.ID == uuid.Nil || requirement.CorpusID != request.CorpusID || requirement.DatasetRevisionID != request.DatasetRevisionID ||
			strings.TrimSpace(requirement.SourceAlias) == "" || strings.TrimSpace(requirement.DisplayLocator) == "" {
			return nil, nil, compatibilityError(ErrPreflightCorpusMismatch, nil)
		}
		if !canonicalLocatorPattern.MatchString(requirement.CanonicalLocator) {
			missing = appendMissing(missing, MissingRequirement{SourceAlias: requirement.SourceAlias, Locator: requirement.DisplayLocator, Reason: "The requirement does not have an exact canonical locator."})
			continue
		}
		locator, err := service.catalog.ResolvePreflightLocator(ctx, LocatorRequest{
			DatasetRevisionID: request.DatasetRevisionID, CorpusID: request.CorpusID, SnapshotID: request.SnapshotID,
			SourceAlias: requirement.SourceAlias, CanonicalLocator: requirement.CanonicalLocator, DisplayLocator: requirement.DisplayLocator,
		})
		if err != nil {
			missing = appendMissing(missing, MissingRequirement{SourceAlias: requirement.SourceAlias, Locator: requirement.DisplayLocator, Reason: "The locator did not resolve uniquely."})
			continue
		}
		if locator.CorpusID != request.CorpusID || locator.SnapshotID != request.SnapshotID || locator.CanonicalLocator != requirement.CanonicalLocator || locator.DisplayLocator != requirement.DisplayLocator {
			missing = appendMissing(missing, MissingRequirement{SourceAlias: requirement.SourceAlias, Locator: requirement.DisplayLocator, Reason: "The locator did not resolve uniquely."})
			continue
		}
		locator.ExpectedEvidenceID = requirement.ID
		locator.CaseID = requirement.CaseID
		resolved = append(resolved, locator)
	}
	return resolved, missing, nil
}

func appendMissing(current []MissingRequirement, requirement MissingRequirement) []MissingRequirement {
	if len(current) >= maxMissingRequirements {
		return current
	}
	return append(current, requirement)
}

func appendMissingRequirements(groups missingRequirementGroups) []MissingRequirement {
	if len(groups.additional) > 0 && len(groups.current) >= maxMissingRequirements {
		combined := slices.Clone(groups.current)
		// Preserve both gate categories when each individual check reached the shared bound.
		combined[maxMissingRequirements-1] = groups.additional[0]
		return combined
	}
	for _, requirement := range groups.additional {
		groups.current = appendMissing(groups.current, requirement)
	}
	return groups.current
}

func compatibilityError(cause error, missing []MissingRequirement) *CompatibilityError {
	return &CompatibilityError{cause: cause, missing: slices.Clone(missing)}
}
