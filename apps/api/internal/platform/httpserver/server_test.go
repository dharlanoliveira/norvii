package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/config"
	"github.com/google/uuid"
)

func TestServerHealthAndRequestIDMiddleware(t *testing.T) {
	requestID := uuid.MustParse("70000000-0000-4000-8000-000000000001")
	server := New(config.Config{
		Address: "127.0.0.1:0", MaxRequestBytes: 1024, ShutdownTimeout: time.Second,
	}, http.NotFoundHandler(), func() uuid.UUID { return requestID })
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", recorder.Code)
	}
	if recorder.Header().Get("X-Request-ID") != requestID.String() {
		t.Fatalf("X-Request-ID = %q, want generated ID", recorder.Header().Get("X-Request-ID"))
	}
	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", recorder.Header().Get("Content-Type"))
	}
}

func TestServerLogsSafeStructuredRequestOutcomeWithoutQuery(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	requestID := uuid.MustParse("70000000-0000-4000-8000-000000000002")
	server := New(config.Config{
		Address: "127.0.0.1:0", MaxRequestBytes: 1024, ShutdownTimeout: time.Second,
	}, http.NotFoundHandler(), func() uuid.UUID { return requestID })

	server.Handler().ServeHTTP(
		httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz?token=secret", nil),
	)

	logged := output.String()
	for _, expected := range []string{
		`"request_id":"` + requestID.String() + `"`, `"path":"/healthz"`,
		`"status":200`, `"duration_ms":`, `"response_bytes":`,
	} {
		if !strings.Contains(logged, expected) {
			t.Fatalf("structured log %q does not contain %q", logged, expected)
		}
	}
	if strings.Contains(logged, "secret") {
		t.Fatalf("structured log exposed query information: %s", logged)
	}
}

func TestServerRejectsDeclaredBodiesOverLimitBeforeApplicationHandler(t *testing.T) {
	called := false
	application := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	server := New(config.Config{
		Address: "127.0.0.1:0", MaxRequestBytes: 4, ShutdownTimeout: time.Second,
	}, application, uuid.New)
	request := httptest.NewRequest(http.MethodPost, "/corpora", strings.NewReader("12345"))
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if called {
		t.Fatal("oversized request reached the application handler")
	}
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized status = %d, want 413", recorder.Code)
	}
	var envelope map[string]map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error response is not JSON: %v", err)
	}
	if envelope["error"]["code"] != "payload_too_large" {
		t.Fatalf("error code = %v, want payload_too_large", envelope["error"]["code"])
	}
}

func TestServerAppliesSmallerContractLimitToNonMultipartRequests(t *testing.T) {
	called := false
	application := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	server := New(config.Config{
		Address: "127.0.0.1:0", MaxRequestBytes: 11 * 1024 * 1024,
		ShutdownTimeout: time.Second,
	}, application, uuid.New)
	request := httptest.NewRequest(
		http.MethodPost, "/api/v1/corpora", strings.NewReader(strings.Repeat("a", 1024*1024+1)),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if called || recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("non-PDF request = called %t/status %d, want false/413", called, recorder.Code)
	}
}

func TestWriteErrorDoesNotExposeInternalCause(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/corpora", nil)
	request = request.WithContext(withRequestID(request.Context(), uuid.New()))
	recorder := httptest.NewRecorder()

	WriteError(recorder, request, Problem{
		Status: http.StatusInternalServerError,
		Code:   "internal_error", Message: "The service could not complete the request.",
		Cause: errors.New("password=secret database stack"),
	})

	body, err := io.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if strings.Contains(string(body), "secret") || strings.Contains(string(body), "database") {
		t.Fatalf("public error exposed internal cause: %s", body)
	}
}

func TestShutdownOfIdleServerIsSafe(t *testing.T) {
	server := New(config.Config{
		Address: "127.0.0.1:0", MaxRequestBytes: 1024, ShutdownTimeout: time.Second,
	}, http.NotFoundHandler(), uuid.New)

	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}
