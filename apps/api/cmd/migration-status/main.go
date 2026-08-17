package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/persistence"
)

const persistenceCloseTimeout = 2 * time.Second

func main() {
	os.Exit(run())
}

func run() int {
	config, err := persistence.LoadConfig(os.LookupEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PostgreSQL migration configuration failed: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()
	migrator, err := persistence.OpenPostgresMigrator(ctx, config.Postgres)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PostgreSQL migration status initialization failed: %v\n", err)
		return 1
	}
	defer closeMigrator(migrator)

	status, err := persistence.NewMigrationService(migrator).Status(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PostgreSQL migration status failed: %v\n", err)
		return 1
	}
	fmt.Printf("PostgreSQL migration version %d is current.\n", status.CurrentVersion)
	return 0
}

func closeMigrator(migrator *persistence.PostgresMigrator) {
	ctx, cancel := context.WithTimeout(context.Background(), persistenceCloseTimeout)
	defer cancel()
	if err := migrator.Close(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "PostgreSQL migration connection did not close cleanly.")
	}
}
