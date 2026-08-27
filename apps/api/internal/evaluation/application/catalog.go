package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrDatasetNotFound covers a dataset revision that is absent from the immutable catalog.
	ErrDatasetNotFound = errors.New("evaluation dataset revision was not found")
)

// DatasetRevisionSummary is the immutable identity exposed by the maintainer catalog.
type DatasetRevisionSummary struct {
	ID                            uuid.UUID
	CorpusID                      uuid.UUID
	DatasetKey                    string
	SemanticRevision              string
	Jurisdiction                  string
	ManifestSHA256                string
	JSONLSHA256                   string
	ContentSHA256                 string
	DeclaredSnapshotDate          time.Time
	QueryLanguages                []string
	AuthoritativeEvidenceLanguage string
}

// DatasetReview records the latest append-only review state when one exists.
type DatasetReview struct {
	Decision         string
	PublicationState string
	ReviewedAt       time.Time
}

// DatasetSource describes one manifest authority and its immutable corpus-source binding.
type DatasetSource struct {
	ID               uuid.UUID
	SourceAlias      string
	Title            string
	OfficialURL      string
	IssuingAuthority string
	DocumentType     string
	AuthorityRole    string
	CorpusSourceID   *uuid.UUID
}

// StarterCase identifies a reviewed opening-suggestion candidate without disclosing case content.
type StarterCase struct {
	ID             uuid.UUID
	CaseID         uuid.UUID
	Rank           int
	QueryLanguage  string
	ReviewEligible bool
}

// DatasetCatalogEntry is a maintainer-only immutable catalog projection.
type DatasetCatalogEntry struct {
	Revision DatasetRevisionSummary
	Review   *DatasetReview
	Sources  []DatasetSource
	Starters []StarterCase
}

// Available reports whether the latest publication authorizes evaluation execution.
func (entry DatasetCatalogEntry) Available() bool {
	return entry.Review != nil && entry.Review.Decision == "approved" && entry.Review.PublicationState == "available"
}

// DatasetCatalogStore reads immutable revisions and their manifest-bound inspection records.
type DatasetCatalogStore interface {
	ListDatasetCatalog(context.Context) ([]DatasetCatalogEntry, error)
	GetDatasetCatalog(context.Context, uuid.UUID) (DatasetCatalogEntry, error)
}

// CatalogService composes catalog inspection with the existing side-effect-free preflight check.
type CatalogService struct {
	store     DatasetCatalogStore
	preflight interface {
		Check(context.Context, PreflightRequest) (PreflightResult, error)
	}
}

// NewCatalogService constructs maintainer dataset inspection operations.
func NewCatalogService(store DatasetCatalogStore, preflight interface {
	Check(context.Context, PreflightRequest) (PreflightResult, error)
}) *CatalogService {
	return &CatalogService{store: store, preflight: preflight}
}

// List returns immutable revision identities and their latest review state.
func (service *CatalogService) List(ctx context.Context) ([]DatasetCatalogEntry, error) {
	if service == nil || service.store == nil {
		return nil, ErrDatasetNotFound
	}
	return service.store.ListDatasetCatalog(ctx)
}

// Get returns one immutable revision, its source authority/binding, review, and starter metadata.
func (service *CatalogService) Get(ctx context.Context, datasetRevisionID uuid.UUID) (DatasetCatalogEntry, error) {
	if service == nil || service.store == nil || datasetRevisionID == uuid.Nil {
		return DatasetCatalogEntry{}, ErrDatasetNotFound
	}
	return service.store.GetDatasetCatalog(ctx, datasetRevisionID)
}

// Check verifies a selected immutable snapshot without creating evaluation work.
func (service *CatalogService) Check(ctx context.Context, request PreflightRequest) (PreflightResult, error) {
	if service == nil || service.preflight == nil {
		return PreflightResult{}, ErrInvalidPreflightRequest
	}
	return service.preflight.Check(ctx, request)
}
