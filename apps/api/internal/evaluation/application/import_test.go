package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestDatasetImportServiceValidatesBeforeStoring(t *testing.T) {
	store := &recordingDatasetStore{}
	service := NewDatasetImportService(NewImporter(newMemoryReader(map[string]string{})), store)

	if _, err := service.Import(context.Background(), AssetRequest{}); !errors.Is(err, ErrAssetValidation) {
		t.Fatalf("Import() error = %v, want asset validation error", err)
	}
	if store.calls != 0 {
		t.Fatalf("store calls = %d, want zero after validation failure", store.calls)
	}
}

func TestDatasetImportServiceRejectsMismatchedStoreIdentity(t *testing.T) {
	request := syntheticRequest()
	asset, err := NewImporter(newMemoryReader(map[string]string{
		request.ManifestPath: syntheticManifest(),
		request.JSONLPath:    syntheticCases(1, 1, "", ""),
	})).Validate(request)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	store := &recordingDatasetStore{result: DatasetImportResult{
		RevisionID: uuid.New(), CorpusID: asset.Revision.CorpusID, DatasetKey: asset.Revision.DatasetKey,
	}}
	service := NewDatasetImportService(NewImporter(newMemoryReader(map[string]string{
		request.ManifestPath: syntheticManifest(),
		request.JSONLPath:    syntheticCases(1, 1, "", ""),
	})), store)

	if _, err := service.Import(context.Background(), request); !errors.Is(err, ErrDatasetStore) {
		t.Fatalf("Import() error = %v, want dataset store error", err)
	}
}

type recordingDatasetStore struct {
	calls  int
	result DatasetImportResult
	err    error
}

func (store *recordingDatasetStore) StoreDataset(_ context.Context, asset DatasetAsset) (DatasetImportResult, error) {
	store.calls++
	if store.err != nil {
		return DatasetImportResult{}, store.err
	}
	if store.result.RevisionID == uuid.Nil {
		return DatasetImportResult{
			RevisionID: asset.Revision.ID, CorpusID: asset.Revision.CorpusID, DatasetKey: asset.Revision.DatasetKey,
			Created: true, Sources: len(asset.Sources), Cases: len(asset.Cases), Evidence: len(asset.ExpectedEvidence), Starters: len(asset.StarterSelections),
		}, nil
	}
	return store.result, nil
}
