package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewCorpusNormalizesValidatedMetadata(t *testing.T) {
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	corpus, err := NewCorpus(uuid.New(), Draft{
		Name:         "  Privacy Law  ",
		Description:  "  Official legal materials.  ",
		Language:     LanguageEnglish,
		Jurisdiction: "  European Union  ",
	}, now)

	if err != nil {
		t.Fatalf("NewCorpus() error = %v", err)
	}
	if corpus.Name != "Privacy Law" || corpus.Jurisdiction != "European Union" {
		t.Fatalf("NewCorpus() did not normalize metadata: %+v", corpus)
	}
	if corpus.Status != StatusEnabled || corpus.Version != 1 {
		t.Fatalf("NewCorpus() lifecycle = %s/%d, want enabled/1", corpus.Status, corpus.Version)
	}
}

func TestNewCorpusRejectsUnsupportedLanguageWithSafeTypedError(t *testing.T) {
	_, err := NewCorpus(uuid.New(), Draft{
		Name: "Privacy", Description: "Legal materials", Language: "fr", Jurisdiction: "France",
	}, time.Now())

	var domainError *Error
	if !errors.As(err, &domainError) || domainError.Code != ErrorInvalidInput {
		t.Fatalf("NewCorpus() error = %v, want invalid_input domain error", err)
	}
	if domainError.Internal != nil {
		t.Fatal("public domain error retained an internal cause")
	}
}

func TestCorpusLifecycleUsesOptimisticVersion(t *testing.T) {
	corpus := Corpus{Status: StatusEnabled, Version: 3}

	if err := corpus.Disable(2, time.Now()); !errors.Is(err, ErrStaleState) {
		t.Fatalf("Disable() error = %v, want stale state", err)
	}
	if err := corpus.Disable(3, time.Now()); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if corpus.Status != StatusDisabled || corpus.Version != 4 {
		t.Fatalf("Disable() lifecycle = %s/%d, want disabled/4", corpus.Status, corpus.Version)
	}
	if err := corpus.Enable(4, time.Now()); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
}
