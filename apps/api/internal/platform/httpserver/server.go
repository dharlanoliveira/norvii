// Package httpserver provides bounded HTTP process behavior and safe public errors.
package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/config"
	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-ID"
const maxNonPDFRequestBytes int64 = 1024 * 1024

type requestIDContextKey struct{}

// Problem describes a safe public error and an optional private cause.
type Problem struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
	Cause   error
}

// Server owns the bounded standard-library HTTP server.
type Server struct {
	httpServer *http.Server
}

// New composes health, request identity, recovery, and request-size middleware.
func New(
	configuration config.Config,
	application http.Handler,
	newRequestID func() uuid.UUID,
) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(writer http.ResponseWriter, _ *http.Request) {
		WriteJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("/", application)

	handler := requestIdentityMiddleware(
		newRequestID,
		loggingMiddleware(
			recoveryMiddleware(bodyLimitMiddleware(configuration.MaxRequestBytes, mux)),
		),
	)
	return &Server{httpServer: &http.Server{
		Addr:              configuration.Address,
		Handler:           handler,
		ReadHeaderTimeout: configuration.ReadHeaderTimeout,
		ReadTimeout:       configuration.ReadTimeout,
		WriteTimeout:      configuration.WriteTimeout,
		IdleTimeout:       configuration.IdleTimeout,
	}}
}

type responseObserver struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (observer *responseObserver) WriteHeader(status int) {
	if observer.status != 0 {
		return
	}
	observer.status = status
	observer.ResponseWriter.WriteHeader(status)
}

func (observer *responseObserver) Write(content []byte) (int, error) {
	if observer.status == 0 {
		observer.WriteHeader(http.StatusOK)
	}
	written, err := observer.ResponseWriter.Write(content)
	observer.bytes += written
	return written, err
}

func (observer *responseObserver) Unwrap() http.ResponseWriter { return observer.ResponseWriter }

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		observer := &responseObserver{ResponseWriter: writer}
		next.ServeHTTP(observer, request)
		status := observer.status
		if status == 0 {
			status = http.StatusOK
		}
		requestID, _ := request.Context().Value(requestIDContextKey{}).(uuid.UUID)
		slog.Info(
			"HTTP request completed",
			"request_id", requestID.String(), "method", request.Method,
			"path", request.URL.Path, "status", status,
			"duration_ms", time.Since(started).Milliseconds(), "response_bytes", observer.bytes,
		)
	})
}

// Handler exposes the fully composed handler for tests and in-process serving.
func (server *Server) Handler() http.Handler { return server.httpServer.Handler }

// ListenAndServe starts the configured HTTP listener.
func (server *Server) ListenAndServe() error { return server.httpServer.ListenAndServe() }

// Shutdown gracefully stops accepting requests and drains active handlers.
func (server *Server) Shutdown(ctx context.Context) error { return server.httpServer.Shutdown(ctx) }

// WriteError serializes the stable safe error envelope without its private cause.
func WriteError(writer http.ResponseWriter, request *http.Request, problem Problem) {
	requestID, _ := request.Context().Value(requestIDContextKey{}).(uuid.UUID)
	payload := map[string]any{"error": map[string]any{
		"code": problem.Code, "message": problem.Message, "requestId": requestID.String(),
	}}
	if len(problem.Fields) > 0 {
		payload["error"].(map[string]any)["fields"] = problem.Fields
	}
	WriteJSON(writer, problem.Status, payload)
}

func requestIdentityMiddleware(newRequestID func() uuid.UUID, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestID := newRequestID()
		writer.Header().Set(requestIDHeader, requestID.String())
		next.ServeHTTP(writer, request.WithContext(withRequestID(request.Context(), requestID)))
	})
}

func bodyLimitMiddleware(maxBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestLimit := maxBytes
		if !strings.HasPrefix(request.Header.Get("Content-Type"), "multipart/form-data") &&
			requestLimit > maxNonPDFRequestBytes {
			requestLimit = maxNonPDFRequestBytes
		}
		if request.ContentLength > requestLimit {
			WriteError(writer, request, Problem{
				Status: http.StatusRequestEntityTooLarge, Code: "payload_too_large",
				Message: "The request body exceeds the supported size.",
			})
			return
		}
		request.Body = http.MaxBytesReader(writer, request.Body, requestLimit)
		next.ServeHTTP(writer, request)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				WriteError(writer, request, Problem{
					Status: http.StatusInternalServerError, Code: "internal_error",
					Message: "The service could not complete the request.",
					Cause:   fmt.Errorf("recovered panic: %v", recovered),
				})
			}
		}()
		next.ServeHTTP(writer, request)
	})
}

func withRequestID(ctx context.Context, requestID uuid.UUID) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// WriteJSON serializes one stable UTF-8 JSON response.
func WriteJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(value); err != nil && !errors.Is(err, http.ErrHandlerTimeout) {
		return
	}
}
