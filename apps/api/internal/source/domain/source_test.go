package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewSourceStartsPendingAndBelongsToCorpus(t *testing.T) {
	corpusID := uuid.New()
	source, err := NewSource(uuid.New(), corpusID, " Official text ", KindURL, time.Now())

	if err != nil {
		t.Fatalf("NewSource() error = %v", err)
	}
	if source.CorpusID != corpusID || source.Title != "Official text" {
		t.Fatalf("NewSource() = %+v, want normalized corpus-owned source", source)
	}
	if source.Status != StatusPending || source.Version != 1 {
		t.Fatalf("NewSource() lifecycle = %s/%d, want pending/1", source.Status, source.Version)
	}
}

func TestEnsureCapacityRejectsTwentyExistingSources(t *testing.T) {
	err := EnsureCapacity(MaxSourcesPerCorpus)

	if !errors.Is(err, ErrSourceLimit) {
		t.Fatalf("EnsureCapacity() error = %v, want source limit", err)
	}
}

func TestSourceTransitionsRejectInvalidAndStaleChanges(t *testing.T) {
	source := Source{Status: StatusPending, Version: 2}

	if err := source.MarkReady(2, uuid.New(), time.Now()); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("MarkReady() error = %v, want invalid transition", err)
	}
	if err := source.MarkProcessing(1, time.Now()); !errors.Is(err, ErrStaleState) {
		t.Fatalf("MarkProcessing() error = %v, want stale state", err)
	}
	if err := source.MarkProcessing(2, time.Now()); err != nil {
		t.Fatalf("MarkProcessing() error = %v", err)
	}
	if source.Status != StatusProcessing || source.Version != 3 {
		t.Fatalf("MarkProcessing() lifecycle = %s/%d, want processing/3", source.Status, source.Version)
	}
}

func TestNormalizeURLCanonicalizesSupportedHTTPSOrigins(t *testing.T) {
	normalized, err := NormalizeURL(" HTTPS://Example.COM:443/legal?b=2&a=1#article ")

	if err != nil {
		t.Fatalf("NormalizeURL() error = %v", err)
	}
	if normalized != "https://example.com/legal?a=1&b=2" {
		t.Fatalf("NormalizeURL() = %q, want canonical HTTPS URL", normalized)
	}
}

func TestNormalizeURLLowercasesHostWithNondefaultPort(t *testing.T) {
	normalized, err := NormalizeURL("https://Example.COM:8443/legal")

	if err != nil {
		t.Fatalf("NormalizeURL() error = %v", err)
	}
	if normalized != "https://example.com:8443/legal" {
		t.Fatalf("NormalizeURL() = %q, want lowercase host with preserved port", normalized)
	}
}

func TestNormalizeURLRejectsUnsafeSubmissionShapes(t *testing.T) {
	for _, candidate := range []string{
		"http://example.com/legal",
		"https://user:password@example.com/legal",
		"https:///missing-host",
	} {
		if _, err := NormalizeURL(candidate); !errors.Is(err, ErrInvalidURL) {
			t.Fatalf("NormalizeURL(%q) error = %v, want ErrInvalidURL", candidate, err)
		}
	}
}

func TestNewPDFOriginValidatesSignatureHashAndSafeDeliveryName(t *testing.T) {
	origin, err := NewPDFOrigin(
		`..\unsafe\official "law".pdf`, "application/pdf", []byte("%PDF-generated-test"),
	)

	if err != nil {
		t.Fatalf("NewPDFOrigin() error = %v", err)
	}
	if origin.DeliveryFilename != "official law.pdf" || len(origin.SHA256) != 64 {
		t.Fatalf("PDF origin = %+v, want sanitized delivery name and SHA-256", origin)
	}
	if _, err := NewPDFOrigin("invalid.pdf", "application/pdf", []byte("not-pdf")); !errors.Is(err, ErrUnsupportedContent) {
		t.Fatalf("invalid signature error = %v, want ErrUnsupportedContent", err)
	}
}
