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
		fmt.Fprintf(os.Stderr, "Persistence verification configuration failed: %v\n", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	postgres, err := persistence.OpenPostgresStore(ctx, config.Postgres)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PostgreSQL connectivity failed: %v\n", err)
		return 1
	}
	neo4j, err := persistence.OpenNeo4jStore(config.Neo4j, config.Timeout)
	if err != nil {
		closePostgres(postgres)
		fmt.Fprintf(os.Stderr, "Neo4j connectivity failed: %v\n", err)
		return 1
	}

	results, err := persistence.NewVerifier(postgres, neo4j).Verify(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Persistence verification failed: %v\n", err)
		return 1
	}
	for _, result := range results {
		fmt.Printf("%s connectivity verified in %d ms.\n", result.Store, result.Duration.Milliseconds())
	}
	fmt.Println("Persistence verification succeeded.")
	return 0
}

func closePostgres(postgres *persistence.PostgresStore) {
	ctx, cancel := context.WithTimeout(context.Background(), persistenceCloseTimeout)
	defer cancel()
	if err := postgres.Close(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "PostgreSQL connection did not close cleanly.")
	}
}
