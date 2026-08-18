// Package domain defines source lifecycle and capacity invariants.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MaxSourcesPerCorpus bounds POC storage and processing cost.
const MaxSourcesPerCorpus = 20

// MaxOriginBytes bounds each registered PDF and acquired URL origin to 10 MiB.
const MaxOriginBytes = 10 * 1024 * 1024

// Kind identifies the immutable source origin type.
type Kind string

const (
	// KindPDF identifies a preserved uploaded PDF.
	KindPDF Kind = "pdf"
	// KindURL identifies an external HTTPS origin.
	KindURL Kind = "url"
)

// Status identifies the observable processing lifecycle.
type Status string

const (
	// StatusPending awaits a worker claim.
	StatusPending Status = "pending"
	// StatusProcessing has an active bounded attempt.
	StatusProcessing Status = "processing"
	// StatusReady has a published latest document.
	StatusReady Status = "ready"
	// StatusFailed records a safe attempt failure.
	StatusFailed Status = "failed"
)

var (
	// ErrSourceLimit prevents unbounded corpus growth in the POC.
	ErrSourceLimit = errors.New("a corpus supports at most 20 sources")
	// ErrStaleState identifies an optimistic concurrency conflict.
	ErrStaleState = errors.New("the source changed; reload and retry")
	// ErrInvalidTransition identifies an unsupported lifecycle transition.
	ErrInvalidTransition = errors.New("the source lifecycle transition is invalid")
	// ErrInvalidInput identifies malformed source metadata.
	ErrInvalidInput = errors.New("the source metadata is invalid")
	// ErrInvalidURL identifies a URL that cannot be registered for safe acquisition.
	ErrInvalidURL = errors.New("only an absolute HTTPS URL without credentials is supported")
	// ErrDuplicateSource identifies an origin already registered in the same corpus.
	ErrDuplicateSource = errors.New("the source origin is already registered in this corpus")
	// ErrCorpusUnavailable avoids distinguishing absent and disabled corpus identities.
	ErrCorpusUnavailable = errors.New("the corpus is unavailable")
	// ErrPayloadTooLarge identifies an origin beyond the POC byte limit.
	ErrPayloadTooLarge = errors.New("the source origin exceeds the supported size")
	// ErrUnsupportedContent identifies a non-PDF upload on the PDF route.
	ErrUnsupportedContent = errors.New("the source content is unsupported")
)

// PDFOrigin contains validated immutable upload metadata and bytes.
type PDFOrigin struct {
	OriginalFilename  string
	DeliveryFilename  string
	DeclaredMediaType string
	DetectedMediaType string
	ByteSize          int64
	SHA256            string
	Content           []byte
}

// NewPDFOrigin validates and hashes one bounded PDF upload.
func NewPDFOrigin(filename, declaredMediaType string, content []byte) (PDFOrigin, error) {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return PDFOrigin{}, ErrInvalidInput
	}
	if len(content) == 0 || len(content) > MaxOriginBytes {
		return PDFOrigin{}, ErrPayloadTooLarge
	}
	if len(content) < 5 || string(content[:5]) != "%PDF-" {
		return PDFOrigin{}, ErrUnsupportedContent
	}
	digest := sha256.Sum256(content)
	deliveryFilename := path.Base(strings.ReplaceAll(filename, "\\", "/"))
	deliveryFilename = strings.Map(func(character rune) rune {
		if character < 32 || character == 127 || character == '"' {
			return -1
		}
		return character
	}, deliveryFilename)
	if deliveryFilename == "" || deliveryFilename == "." {
		deliveryFilename = "document.pdf"
	}
	return PDFOrigin{
		OriginalFilename: filename, DeliveryFilename: deliveryFilename,
		DeclaredMediaType: declaredMediaType, DetectedMediaType: "application/pdf",
		ByteSize: int64(len(content)), SHA256: hex.EncodeToString(digest[:]),
		Content: append([]byte(nil), content...),
	}, nil
}

// NormalizeURL creates a stable duplicate key without performing network acquisition.
func NormalizeURL(submitted string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(submitted))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", ErrInvalidURL
	}
	if !strings.EqualFold(parsed.Scheme, "https") || parsed.User != nil || parsed.Hostname() == "" {
		return "", ErrInvalidURL
	}
	port := parsed.Port()
	if port != "" && port != "443" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return "", ErrInvalidURL
		}
	}
	host := strings.ToLower(parsed.Hostname())
	if port != "" && port != "443" {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	parsed.Scheme = "https"
	parsed.Host = host
	parsed.Fragment = ""
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	query := parsed.Query()
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

// Source is a stable corpus-owned origin identity and mutable lifecycle projection.
type Source struct {
	ID                    uuid.UUID
	CorpusID              uuid.UUID
	Title                 string
	Kind                  Kind
	Status                Status
	LatestFailureCategory string
	LatestReadyDocumentID uuid.UUID
	Version               int
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// NewSource validates and creates a pending corpus-owned source.
func NewSource(id, corpusID uuid.UUID, title string, kind Kind, now time.Time) (Source, error) {
	title = strings.TrimSpace(title)
	if id == uuid.Nil || corpusID == uuid.Nil {
		return Source{}, ErrInvalidInput
	}
	if title == "" {
		return Source{}, ErrInvalidInput
	}
	if kind != KindPDF && kind != KindURL {
		return Source{}, ErrInvalidInput
	}
	now = now.UTC()
	return Source{
		ID:        id,
		CorpusID:  corpusID,
		Title:     title,
		Kind:      kind,
		Status:    StatusPending,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// EnsureCapacity rejects a new source when the corpus reached its POC limit.
func EnsureCapacity(existingSourceCount int) error {
	if existingSourceCount < 0 {
		return errors.New("existing source count must not be negative")
	}
	if existingSourceCount >= MaxSourcesPerCorpus {
		return ErrSourceLimit
	}
	return nil
}

// MarkProcessing claims a pending source for one bounded attempt.
func (source *Source) MarkProcessing(expectedVersion int, now time.Time) error {
	return source.transition(expectedVersion, StatusPending, StatusProcessing, now)
}

// MarkReady publishes a document from an active attempt.
func (source *Source) MarkReady(
	expectedVersion int,
	documentID uuid.UUID,
	now time.Time,
) error {
	if documentID == uuid.Nil {
		return errors.New("ready document identifier is required")
	}
	if err := source.transition(expectedVersion, StatusProcessing, StatusReady, now); err != nil {
		return err
	}
	source.LatestReadyDocumentID = documentID
	source.LatestFailureCategory = ""
	return nil
}

// MarkFailed records a safe category while retaining any prior ready document.
func (source *Source) MarkFailed(expectedVersion int, category string, now time.Time) error {
	category = strings.TrimSpace(category)
	if category == "" {
		return errors.New("failure category is required")
	}
	if err := source.transition(expectedVersion, StatusProcessing, StatusFailed, now); err != nil {
		return err
	}
	source.LatestFailureCategory = category
	return nil
}

// Retry returns a failed source to pending for an explicit attempt.
func (source *Source) Retry(expectedVersion int, now time.Time) error {
	return source.transition(expectedVersion, StatusFailed, StatusPending, now)
}

// Reprocess returns a ready source to pending while retaining its latest document.
func (source *Source) Reprocess(expectedVersion int, now time.Time) error {
	return source.transition(expectedVersion, StatusReady, StatusPending, now)
}

func (source *Source) transition(
	expectedVersion int,
	from Status,
	to Status,
	now time.Time,
) error {
	if source.Version != expectedVersion {
		return ErrStaleState
	}
	if source.Status != from {
		return ErrInvalidTransition
	}
	source.Status = to
	source.Version++
	source.UpdatedAt = now.UTC()
	return nil
}
