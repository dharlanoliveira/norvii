// Command initialize-snapshots creates idempotent initial releases after ingestion is ready.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/persistence"
	snapshotapplication "github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/application"
	snapshotpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/postgres"
	"github.com/google/uuid"
)

var initialCorpusIDs = []uuid.UUID{
	uuid.MustParse("10000000-0000-4000-8000-000000000001"),
	uuid.MustParse("10000000-0000-4000-8000-000000000002"),
}

func main() {
	os.Exit(run())
}

func run() int {
	configuration, err := persistence.LoadConfig(os.LookupEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Snapshot initialization configuration failed: %v\n", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), configuration.Timeout)
	defer cancel()
	pool, err := persistence.OpenPostgresPool(ctx, configuration.Postgres)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Snapshot initialization storage failed: %v\n", err)
		return 1
	}
	defer pool.Close()
	service := snapshotapplication.NewService(snapshotpostgres.NewRepository(pool), uuid.New, time.Now)
	for _, corpusID := range initialCorpusIDs {
		publication, err := service.Initialize(ctx, corpusID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Snapshot initialization failed for %s: %v\n", corpusID, err)
			return 1
		}
		fmt.Printf("Corpus %s active snapshot %s (created: %t).\n", corpusID, publication.Release.SnapshotID, publication.Created)
	}
	return 0
}
