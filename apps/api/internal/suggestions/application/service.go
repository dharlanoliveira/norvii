// Package application coordinates safe opening-suggestion publication.
package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dharlanoliveira/norvii/apps/api/internal/suggestions/domain"
	"github.com/google/uuid"
)

var (
	// ErrInvalidCommand identifies malformed publication input at the application boundary.
	ErrInvalidCommand = errors.New("opening suggestion publication command is invalid")
	// ErrStaleActiveRelease identifies an active snapshot that changed before publication could append.
	ErrStaleActiveRelease = errors.New("opening suggestion active release is stale")
)

// Catalog exposes only the safe catalog metadata needed to publish opening suggestions.
// Implementations must not return answers, legal evidence text, scores, provider data, or prompts.
type Catalog interface {
	DatasetPublication(context.Context, uuid.UUID, uuid.UUID) (domain.DatasetPublication, error)
	StarterCases(context.Context, uuid.UUID, uuid.UUID) ([]domain.StarterCase, error)
	ExpectedEvidence(context.Context, uuid.UUID, uuid.UUID) ([]domain.ExpectedEvidenceRequirement, error)
	ResolvedExpectedEvidence(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) ([]domain.ExpectedEvidenceResolution, error)
}

// ActiveSnapshotReader returns the current immutable release identity for one corpus.
type ActiveSnapshotReader interface {
	ActiveSnapshot(context.Context, uuid.UUID) (domain.Snapshot, error)
}

// ProjectionAppender persists a fully validated append-only projection.
type ProjectionAppender interface {
	AppendProjection(context.Context, domain.Projection) error
}

// PublishCommand fixes publication to a corpus, dataset revision, and snapshot manifest.
type PublishCommand struct {
	CorpusID               uuid.UUID
	DatasetRevisionID      uuid.UUID
	SnapshotID             uuid.UUID
	SnapshotManifestSHA256 domain.SHA256
	SelectionPolicyVersion string
	PublishedBy            string
}

// Validate rejects unsafe or incomplete publication commands before dependency calls.
func (command PublishCommand) Validate() error {
	if command.CorpusID == uuid.Nil || command.DatasetRevisionID == uuid.Nil || command.SnapshotID == uuid.Nil ||
		strings.TrimSpace(command.SelectionPolicyVersion) == "" || strings.TrimSpace(command.PublishedBy) == "" {
		return ErrInvalidCommand
	}
	if err := command.SnapshotManifestSHA256.Validate(); err != nil {
		return fmt.Errorf("%w: snapshot manifest is invalid", ErrInvalidCommand)
	}
	return nil
}

// Service owns publication orchestration while all persistence remains behind application ports.
type Service struct {
	catalog          Catalog
	activeSnapshots  ActiveSnapshotReader
	projectionWriter ProjectionAppender
	newID            func() uuid.UUID
}

// NewService constructs publication use cases around caller-owned ports.
func NewService(
	catalog Catalog,
	activeSnapshots ActiveSnapshotReader,
	projectionWriter ProjectionAppender,
	newID func() uuid.UUID,
) *Service {
	return &Service{
		catalog: catalog, activeSnapshots: activeSnapshots, projectionWriter: projectionWriter, newID: newID,
	}
}

// Publish validates all review, corpus, snapshot, selection, and resolution gates before append.
func (service *Service) Publish(ctx context.Context, command PublishCommand) (domain.Projection, error) {
	if service == nil || service.catalog == nil || service.activeSnapshots == nil || service.projectionWriter == nil || service.newID == nil {
		return domain.Projection{}, ErrInvalidCommand
	}
	if err := command.Validate(); err != nil {
		return domain.Projection{}, err
	}

	requestedSnapshot := domain.Snapshot{
		ID: command.SnapshotID, CorpusID: command.CorpusID, ManifestSHA256: command.SnapshotManifestSHA256,
	}
	activeSnapshot, err := service.activeSnapshots.ActiveSnapshot(ctx, command.CorpusID)
	if err != nil {
		return domain.Projection{}, fmt.Errorf("read opening suggestion active snapshot: %w", err)
	}
	if !sameSnapshot(requestedSnapshot, activeSnapshot) {
		return domain.Projection{}, ErrStaleActiveRelease
	}

	publication, err := service.catalog.DatasetPublication(ctx, command.CorpusID, command.DatasetRevisionID)
	if err != nil {
		return domain.Projection{}, fmt.Errorf("read opening suggestion dataset publication: %w", err)
	}
	starterCases, err := service.catalog.StarterCases(ctx, command.CorpusID, command.DatasetRevisionID)
	if err != nil {
		return domain.Projection{}, fmt.Errorf("read opening suggestion starter cases: %w", err)
	}
	expectedEvidence, err := service.catalog.ExpectedEvidence(ctx, command.CorpusID, command.DatasetRevisionID)
	if err != nil {
		return domain.Projection{}, fmt.Errorf("read opening suggestion expected evidence: %w", err)
	}
	resolvedEvidence, err := service.catalog.ResolvedExpectedEvidence(ctx, command.CorpusID, command.DatasetRevisionID, command.SnapshotID)
	if err != nil {
		return domain.Projection{}, fmt.Errorf("read opening suggestion evidence resolutions: %w", err)
	}

	projection, err := domain.NewProjection(domain.PublicationInput{
		SuggestionSetID: service.newID(), CorpusID: command.CorpusID, DatasetRevisionID: command.DatasetRevisionID,
		Snapshot: requestedSnapshot, ActiveSnapshot: activeSnapshot, DatasetPublication: publication,
		SelectionPolicyVersion: command.SelectionPolicyVersion, PublishedBy: command.PublishedBy,
		StarterCases: starterCases, ExpectedEvidence: expectedEvidence, ResolvedEvidence: resolvedEvidence,
	})
	if err != nil {
		return domain.Projection{}, err
	}

	activeSnapshot, err = service.activeSnapshots.ActiveSnapshot(ctx, command.CorpusID)
	if err != nil {
		return domain.Projection{}, fmt.Errorf("recheck opening suggestion active snapshot: %w", err)
	}
	if !sameSnapshot(requestedSnapshot, activeSnapshot) || !projection.MatchesActiveSnapshot(activeSnapshot) {
		return domain.Projection{}, ErrStaleActiveRelease
	}
	if err := service.projectionWriter.AppendProjection(ctx, projection); err != nil {
		return domain.Projection{}, fmt.Errorf("append opening suggestion projection: %w", err)
	}
	return projection, nil
}

func sameSnapshot(left, right domain.Snapshot) bool {
	return left.Validate() == nil && right.Validate() == nil && left.ID == right.ID &&
		left.CorpusID == right.CorpusID && left.ManifestSHA256 == right.ManifestSHA256
}
