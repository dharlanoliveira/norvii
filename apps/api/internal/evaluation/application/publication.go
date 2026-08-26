package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	"github.com/google/uuid"
)

var (
	// ErrInvalidReviewCommand identifies invalid maintainer review input.
	ErrInvalidReviewCommand = errors.New("evaluation dataset review command is invalid")
	// ErrPublicationStore identifies an append-only publication persistence failure.
	ErrPublicationStore = errors.New("evaluation dataset publication persistence failed")
)

// ReviewCommand records one append-only maintainer review decision for a dataset revision.
type ReviewCommand struct {
	CorpusID          uuid.UUID
	DatasetRevisionID uuid.UUID
	ReviewDecision    domain.ReviewDecision
	ReviewerIdentity  string
	ReviewNote        string
	PublicationState  domain.PublicationState
}

// PublicationStore reads the latest review state and appends a new immutable publication record.
type PublicationStore interface {
	LatestPublication(context.Context, uuid.UUID, uuid.UUID) (domain.Publication, bool, error)
	AppendPublication(context.Context, domain.Publication) error
}

// PublicationService owns append-only review lifecycle validation.
type PublicationService struct {
	store PublicationStore
	newID func() uuid.UUID
	now   func() time.Time
}

// NewPublicationService constructs review publication around caller-owned storage.
func NewPublicationService(store PublicationStore, newID func() uuid.UUID, now func() time.Time) *PublicationService {
	return &PublicationService{store: store, newID: newID, now: now}
}

// Review validates the transition and appends a new immutable record. It never updates or
// deletes an earlier review record.
func (service *PublicationService) Review(ctx context.Context, command ReviewCommand) (domain.Publication, error) {
	if service == nil || service.store == nil || service.newID == nil || service.now == nil {
		return domain.Publication{}, ErrPublicationStore
	}
	publication := domain.Publication{
		ID: service.newID(), DatasetRevisionID: command.DatasetRevisionID, CorpusID: command.CorpusID,
		ReviewDecision: command.ReviewDecision, ReviewerIdentity: command.ReviewerIdentity,
		ReviewNote: command.ReviewNote, PublicationState: command.PublicationState, ReviewedAt: service.now().UTC(),
	}
	if command.CorpusID == uuid.Nil || command.DatasetRevisionID == uuid.Nil ||
		strings.TrimSpace(command.ReviewerIdentity) == "" || publication.Validate() != nil {
		return domain.Publication{}, ErrInvalidReviewCommand
	}

	latest, found, err := service.store.LatestPublication(ctx, command.CorpusID, command.DatasetRevisionID)
	if err != nil {
		return domain.Publication{}, fmt.Errorf("read latest evaluation dataset publication: %w", err)
	}
	if found {
		if err := domain.ValidatePublicationTransition(latest.PublicationState, publication.PublicationState); err != nil {
			return domain.Publication{}, err
		}
	}
	if err := service.store.AppendPublication(ctx, publication); err != nil {
		return domain.Publication{}, fmt.Errorf("append evaluation dataset publication: %w", err)
	}
	return publication, nil
}
