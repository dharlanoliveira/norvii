//go:build integration

package integration_test

import (
	"context"
	"os"
	"testing"

	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/persistence"
)

func TestGoRuntimeVerifiesBothStoresWithoutCreatingProductArtifacts(t *testing.T) {
	config, err := persistence.LoadConfig(os.LookupEnv)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()
	postgres, err := persistence.OpenPostgresStore(ctx, config.Postgres)
	if err != nil {
		t.Fatalf("OpenPostgresStore() error = %v", err)
	}
	neo4j, err := persistence.OpenNeo4jStore(config.Neo4j, config.Timeout)
	if err != nil {
		_ = postgres.Close(context.Background())
		t.Fatalf("OpenNeo4jStore() error = %v", err)
	}

	results, err := persistence.NewVerifier(postgres, neo4j).Verify(ctx)

	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if len(results) != 2 || results[0].Store != "PostgreSQL" || results[1].Store != "Neo4j" {
		t.Fatalf("Verify() results = %#v, want both ordered stores", results)
	}
}
