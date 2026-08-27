package contract_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type openingSuggestionsFixture struct {
	CorpusID                     string              `json:"corpusId"`
	ActiveSnapshotID             *string             `json:"activeSnapshotId"`
	ActiveSnapshotManifestSHA256 *string             `json:"activeSnapshotManifestSha256"`
	InterfaceLanguage            string              `json:"interfaceLanguage"`
	Suggestions                  []suggestionFixture `json:"suggestions"`
}

type suggestionFixture struct {
	CaseID   string `json:"caseId"`
	Rank     int    `json:"rank"`
	Question string `json:"question"`
}

func TestOpeningSuggestionFixturesMeetThePublicContract(t *testing.T) {
	for _, name := range []string{
		"suggestions-response-en.json",
		"suggestions-response-pt.json",
		"suggestions-response-empty-no-active-snapshot.json",
		"suggestions-response-empty-stale-projection.json",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeOpeningSuggestionsFixture(readOpeningSuggestionFixture(t, name)); err != nil {
				t.Fatalf("fixture validation error = %v", err)
			}
		})
	}
}

func TestOpeningSuggestionInvalidFixturesAreRejected(t *testing.T) {
	for _, name := range []string{
		"suggestions-response-invalid-evaluation-leakage.json",
		"suggestions-response-invalid-rank-order.json",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeOpeningSuggestionsFixture(readOpeningSuggestionFixture(t, name)); err == nil {
				t.Fatal("invalid fixture decoded successfully")
			}
		})
	}
}

func readOpeningSuggestionFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "contracts", "corpus-opening-suggestions", "v1", "fixtures", name)
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return payload
}

func decodeOpeningSuggestionsFixture(payload []byte) (openingSuggestionsFixture, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var fixture openingSuggestionsFixture
	if err := decoder.Decode(&fixture); err != nil {
		return openingSuggestionsFixture{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errorsIsEOF(err) {
		return openingSuggestionsFixture{}, fmt.Errorf("response must contain one JSON value")
	}
	if err := validateOpeningSuggestionsFixture(fixture); err != nil {
		return openingSuggestionsFixture{}, err
	}
	return fixture, nil
}

func errorsIsEOF(err error) bool { return err == io.EOF }

func validateOpeningSuggestionsFixture(fixture openingSuggestionsFixture) error {
	if _, err := uuid.Parse(fixture.CorpusID); err != nil {
		return fmt.Errorf("invalid corpus ID: %w", err)
	}
	if fixture.InterfaceLanguage != "en" && fixture.InterfaceLanguage != "pt" {
		return fmt.Errorf("invalid interface language %q", fixture.InterfaceLanguage)
	}
	if (fixture.ActiveSnapshotID == nil) != (fixture.ActiveSnapshotManifestSHA256 == nil) {
		return fmt.Errorf("active snapshot identity and manifest must be present together")
	}
	if fixture.ActiveSnapshotID != nil {
		if _, err := uuid.Parse(*fixture.ActiveSnapshotID); err != nil {
			return fmt.Errorf("invalid active snapshot ID: %w", err)
		}
		if len(*fixture.ActiveSnapshotManifestSHA256) != 64 || strings.Trim(*fixture.ActiveSnapshotManifestSHA256, "0123456789abcdef") != "" {
			return fmt.Errorf("invalid active snapshot manifest")
		}
	}
	if len(fixture.Suggestions) > 5 {
		return fmt.Errorf("suggestion count exceeds five")
	}
	previousRank := 0
	for _, suggestion := range fixture.Suggestions {
		if suggestion.CaseID == "" || suggestion.Question == "" || suggestion.Rank < 1 || suggestion.Rank > 5 || suggestion.Rank <= previousRank {
			return fmt.Errorf("invalid rank-ordered suggestion")
		}
		previousRank = suggestion.Rank
	}
	return nil
}
