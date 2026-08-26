package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	catalogapplication "github.com/dharlanoliveira/norvii/apps/api/internal/catalog/application"
	cataloghttp "github.com/dharlanoliveira/norvii/apps/api/internal/catalog/http"
	catalogpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/catalog/postgres"
	chatagent "github.com/dharlanoliveira/norvii/apps/api/internal/chat/agent"
	chatapplication "github.com/dharlanoliveira/norvii/apps/api/internal/chat/application"
	chathttp "github.com/dharlanoliveira/norvii/apps/api/internal/chat/http"
	documenthttp "github.com/dharlanoliveira/norvii/apps/api/internal/document/http"
	documentpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/document/postgres"
	evaluationapplication "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	evaluationhttp "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/http"
	evaluationpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/postgres"
	graphapplication "github.com/dharlanoliveira/norvii/apps/api/internal/graphrelease/application"
	graphhttp "github.com/dharlanoliveira/norvii/apps/api/internal/graphrelease/http"
	graphpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/graphrelease/postgres"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/config"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/httpserver"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/persistence"
	snapshotapplication "github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/application"
	snapshothttp "github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/http"
	snapshotpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/postgres"
	sourceapplication "github.com/dharlanoliveira/norvii/apps/api/internal/source/application"
	sourcehttp "github.com/dharlanoliveira/norvii/apps/api/internal/source/http"
	sourcepostgres "github.com/dharlanoliveira/norvii/apps/api/internal/source/postgres"
	suggestionshttp "github.com/dharlanoliveira/norvii/apps/api/internal/suggestions/http"
	suggestionspostgres "github.com/dharlanoliveira/norvii/apps/api/internal/suggestions/postgres"
	"github.com/google/uuid"
)

func main() {
	configuration, err := config.Load(os.LookupEnv)
	if err != nil {
		slog.Error("API configuration is invalid", "error", err)
		os.Exit(1)
	}
	persistenceConfiguration, err := persistence.LoadConfig(os.LookupEnv)
	if err != nil {
		slog.Error("API persistence configuration is invalid", "error", err)
		os.Exit(1)
	}
	startupContext, cancelStartup := context.WithTimeout(
		context.Background(), persistenceConfiguration.Timeout,
	)
	pool, err := persistence.OpenPostgresPool(startupContext, persistenceConfiguration.Postgres)
	cancelStartup()
	if err != nil {
		slog.Error("API canonical storage is unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	application := http.NewServeMux()
	catalogRepository := catalogpostgres.NewRepository(pool)
	catalogService := catalogapplication.NewService(catalogRepository, uuid.New, time.Now)
	cataloghttp.NewHandler(catalogRepository, catalogService).Register(application)
	sourceRepository := sourcepostgres.NewRepository(pool)
	sourceService := sourceapplication.NewService(sourceRepository, uuid.New, time.Now)
	sourcehttp.NewHandler(sourceRepository, sourceService).Register(application)
	documenthttp.NewHandler(documentpostgres.NewRepository(pool)).Register(application)
	snapshotRepository := snapshotpostgres.NewRepository(pool)
	snapshothttp.NewHandler(snapshotapplication.NewService(snapshotRepository, uuid.New, time.Now)).Register(application)
	graphRepository := graphpostgres.NewRepository(pool)
	graphhttp.NewHandler(graphapplication.NewService(graphRepository)).Register(application)
	suggestionshttp.NewHandler(suggestionspostgres.NewRepository(pool)).Register(application)
	chathttp.NewHandler(chatapplication.NewService(snapshotRepository, chatagent.NewClient(configuration.Agent))).Register(application)
	evaluationRepository := evaluationpostgres.NewRepository(pool)
	evaluationPreflight := evaluationapplication.NewPreflightService(evaluationRepository)
	evaluationRunnable := evaluationapplication.RunnableConfiguration{
		Retrieval: evaluationapplication.RetrievalConfiguration{Strategy: configuration.Evaluation.RetrievalStrategy, Fingerprint: configuration.Evaluation.RetrievalFingerprint},
		Identity:  evaluationapplication.ExecutionIdentity{AgentBuild: configuration.Evaluation.AgentBuild, ChatModelIdentity: configuration.Evaluation.ChatModelIdentity, EmbeddingModelIdentity: configuration.Evaluation.EmbeddingModelIdentity},
	}
	evaluationhttp.NewHandler(
		evaluationapplication.NewRunService(evaluationPreflight, evaluationRepository, evaluationRunnable),
		evaluationhttp.NewTokenAuthorizer(configuration.Evaluation.MaintainerToken),
		evaluationapplication.NewCatalogService(evaluationRepository, evaluationPreflight),
	).WithComparison(evaluationapplication.NewComparisonService(evaluationRepository)).Register(application)
	server := httpserver.New(configuration, application, uuid.New)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.ListenAndServe() }()
	slog.Info("Norvii API started", "address", configuration.Address)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Norvii API stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(
			context.Background(), configuration.ShutdownTimeout,
		)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			slog.Error("Norvii API graceful shutdown failed", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("Norvii API stopped", "time", time.Now().UTC())
}
