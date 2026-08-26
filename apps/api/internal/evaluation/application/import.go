package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	"github.com/google/uuid"
)

// ErrDatasetStore identifies a failed immutable dataset persistence operation.
var ErrDatasetStore = errors.New("evaluation dataset persistence failed")

// DatasetImportResult is the safe identity and count summary of one imported asset.
// It deliberately excludes questions, answers, evidence propositions, and source URLs.
type DatasetImportResult struct {
	RevisionID uuid.UUID
	CorpusID   uuid.UUID
	DatasetKey string
	Created    bool
	Sources    int
	Cases      int
	Evidence   int
	Starters   int
}

// DatasetStore persists one fully validated immutable asset. Implementations must make
// an identical corpus/content hash idempotent and must not publish or modify snapshots.
type DatasetStore interface {
	StoreDataset(context.Context, DatasetAsset) (DatasetImportResult, error)
}

// DatasetImportService coordinates deterministic validation and immutable persistence.
type DatasetImportService struct {
	validator *Importer
	store     DatasetStore
}

// NewDatasetImportService constructs an import application service around caller-owned boundaries.
func NewDatasetImportService(validator *Importer, store DatasetStore) *DatasetImportService {
	return &DatasetImportService{validator: validator, store: store}
}

// Import validates one local asset and persists the resulting immutable revision.
func (service *DatasetImportService) Import(ctx context.Context, request AssetRequest) (DatasetImportResult, error) {
	if service == nil || service.validator == nil || service.store == nil {
		return DatasetImportResult{}, ErrDatasetStore
	}
	asset, err := service.validator.Validate(request)
	if err != nil {
		return DatasetImportResult{}, err
	}
	result, err := service.store.StoreDataset(ctx, asset)
	if err != nil {
		return DatasetImportResult{}, fmt.Errorf("store validated evaluation dataset: %w", err)
	}
	if err := validateImportResult(result, asset.Revision); err != nil {
		return DatasetImportResult{}, err
	}
	return result, nil
}

func validateImportResult(result DatasetImportResult, revision domain.DatasetRevision) error {
	if result.RevisionID != revision.ID || result.CorpusID != revision.CorpusID || result.DatasetKey != revision.DatasetKey ||
		result.Sources < 0 || result.Cases < 0 || result.Evidence < 0 || result.Starters < 0 {
		return ErrDatasetStore
	}
	return nil
}
