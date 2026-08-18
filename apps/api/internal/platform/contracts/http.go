// Package contracts validates stable public payloads at module boundaries.
package contracts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

// CorpusResponse is the versioned HTTP corpus representation.
type CorpusResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Language     string `json:"language"`
	Jurisdiction string `json:"jurisdiction"`
	Status       string `json:"status"`
	SourceCount  int    `json:"sourceCount"`
	Version      int    `json:"version"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// PublicError contains only stable safe failure fields.
type PublicError struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"requestId"`
}

// ErrorEnvelope is the stable public failure shape.
type ErrorEnvelope struct {
	Error PublicError `json:"error"`
}

// DecodeCorpusList strictly decodes and validates a v1 corpus list payload.
func DecodeCorpusList(payload []byte) ([]CorpusResponse, error) {
	var corpora []CorpusResponse
	if err := decodeStrict(payload, &corpora); err != nil {
		return nil, err
	}
	for index, corpus := range corpora {
		if err := validateCorpus(corpus); err != nil {
			return nil, fmt.Errorf("validate corpus at index %d: %w", index, err)
		}
	}
	return corpora, nil
}

// DecodeError strictly decodes and validates a v1 public error payload.
func DecodeError(payload []byte) (ErrorEnvelope, error) {
	var envelope ErrorEnvelope
	if err := decodeStrict(payload, &envelope); err != nil {
		return ErrorEnvelope{}, err
	}
	validCodes := map[string]bool{
		"invalid_input": true, "payload_too_large": true, "unsafe_url": true,
		"unsupported_content": true, "duplicate_source": true, "stale_state": true,
		"not_found": true, "unavailable": true, "acquisition_failed": true,
		"extraction_failed": true, "publication_failed": true, "internal_error": true,
	}
	if !validCodes[envelope.Error.Code] {
		return ErrorEnvelope{}, fmt.Errorf("unsupported public error code %q", envelope.Error.Code)
	}
	if envelope.Error.Message == "" {
		return ErrorEnvelope{}, fmt.Errorf("public error message is required")
	}
	if _, err := uuid.Parse(envelope.Error.RequestID); err != nil {
		return ErrorEnvelope{}, fmt.Errorf("parse public error request ID: %w", err)
	}
	return envelope, nil
}

func validateCorpus(corpus CorpusResponse) error {
	if _, err := uuid.Parse(corpus.ID); err != nil {
		return fmt.Errorf("parse corpus ID: %w", err)
	}
	if corpus.Name == "" || corpus.Description == "" || corpus.Jurisdiction == "" {
		return fmt.Errorf("corpus metadata must not be empty")
	}
	if corpus.Language != "en" && corpus.Language != "pt" {
		return fmt.Errorf("unsupported corpus language %q", corpus.Language)
	}
	if corpus.Status != "enabled" && corpus.Status != "disabled" {
		return fmt.Errorf("unsupported corpus status %q", corpus.Status)
	}
	if corpus.SourceCount < 0 || corpus.Version < 1 {
		return fmt.Errorf("corpus counts and version must be nonnegative")
	}
	if _, err := time.Parse(time.RFC3339, corpus.CreatedAt); err != nil {
		return fmt.Errorf("parse corpus creation time: %w", err)
	}
	if _, err := time.Parse(time.RFC3339, corpus.UpdatedAt); err != nil {
		return fmt.Errorf("parse corpus update time: %w", err)
	}
	return nil
}

func decodeStrict(payload []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode v1 contract payload: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("decode v1 contract payload: trailing JSON value")
	}
	return nil
}
