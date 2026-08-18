package contract_test

import (
	"os"
	"path/filepath"
	"testing"

	platformcontracts "github.com/dharlanoliveira/norvii/apps/api/internal/platform/contracts"
)

func TestCorpusListFixtureDecodesThroughProviderContract(t *testing.T) {
	payload := readFixture(t, "corpus-list.json")

	corpora, err := platformcontracts.DecodeCorpusList(payload)

	if err != nil {
		t.Fatalf("DecodeCorpusList() error = %v", err)
	}
	if len(corpora) != 2 || corpora[0].ID == "" || corpora[0].SourceCount != 1 {
		t.Fatalf("DecodeCorpusList() = %+v, want two canonical corpora", corpora)
	}
}

func TestErrorFixtureDecodesOnlyStablePublicFields(t *testing.T) {
	payload := readFixture(t, "error.json")

	envelope, err := platformcontracts.DecodeError(payload)

	if err != nil {
		t.Fatalf("DecodeError() error = %v", err)
	}
	if envelope.Error.Code != "stale_state" || envelope.Error.RequestID == "" {
		t.Fatalf("DecodeError() = %+v, want stale_state and request ID", envelope)
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "contracts", "corpus-ingestion", "v1", "fixtures", name)
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return payload
}
