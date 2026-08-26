// Command evaluation-worker owns the managed lifecycle for fixed-snapshot evaluation work.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	evaluationagent "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/agent"
	evaluationapplication "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	evaluationpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/postgres"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/config"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/persistence"
)

const (
	workerPollInterval = time.Second
	workerLeasePeriod  = time.Minute
	workerMaxAttempts  = 3
	workerBatchSize    = 4
)

func main() {
	if err := run(os.Args[1:], os.LookupEnv); err != nil {
		slog.Error("Evaluation worker stopped", "error", err)
		os.Exit(1)
	}
}

func run(arguments []string, lookup persistence.EnvironmentLookup) error {
	flags := flag.NewFlagSet("evaluation-worker", flag.ContinueOnError)
	readyFile := flags.String("ready-file", "", "path written after PostgreSQL readiness succeeds")
	if err := flags.Parse(arguments); err != nil {
		return fmt.Errorf("parse evaluation worker arguments: %w", err)
	}
	persistenceConfiguration, err := persistence.LoadConfig(lookup)
	if err != nil {
		return fmt.Errorf("load evaluation worker persistence configuration: %w", err)
	}
	apiConfiguration, err := config.Load(config.LookupEnv(lookup))
	if err != nil {
		return fmt.Errorf("load evaluation worker API configuration: %w", err)
	}
	startupContext, cancelStartup := context.WithTimeout(context.Background(), persistenceConfiguration.Timeout)
	pool, err := persistence.OpenPostgresPool(startupContext, persistenceConfiguration.Postgres)
	cancelStartup()
	if err != nil {
		return fmt.Errorf("open evaluation worker PostgreSQL pool: %w", err)
	}
	defer pool.Close()

	if err := writeReadyFile(*readyFile); err != nil {
		return err
	}
	defer removeReadyFile(*readyFile)

	repository := evaluationpostgres.NewRepository(pool)
	processor, err := evaluationagent.NewProcessor(evaluationagent.NewClient(apiConfiguration.Agent))
	if err != nil {
		return fmt.Errorf("configure evaluation worker processor: %w", err)
	}
	workerID, err := os.Hostname()
	if err != nil || workerID == "" {
		return fmt.Errorf("identify evaluation worker")
	}
	worker, err := evaluationapplication.NewWorker(repository, processor, "evaluation-"+workerID, workerLeasePeriod, workerMaxAttempts, workerBatchSize)
	if err != nil {
		return fmt.Errorf("configure evaluation worker: %w", err)
	}
	slog.Info("Evaluation worker is ready", "executionAdapter", "fixed_snapshot")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := poll(ctx, worker); err != nil {
		slog.Warn("Evaluation worker poll failed", "error", err)
	}
	ticker := time.NewTicker(workerPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := poll(ctx, worker); err != nil && !errors.Is(err, context.Canceled) {
				slog.Warn("Evaluation worker poll failed", "error", err)
			}
		}
	}
}

func poll(ctx context.Context, worker *evaluationapplication.Worker) error {
	_, err := worker.RunOnce(ctx)
	return err
}

func writeReadyFile(path string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create evaluation worker readiness directory: %w", err)
	}
	if err := os.WriteFile(path, []byte("ready\n"), 0o600); err != nil {
		return fmt.Errorf("write evaluation worker readiness marker: %w", err)
	}
	return nil
}

func removeReadyFile(path string) {
	if path != "" && !errors.Is(os.Remove(path), os.ErrNotExist) {
		slog.Warn("Could not remove evaluation worker readiness marker")
	}
}
