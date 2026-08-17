package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	driverconfig "github.com/neo4j/neo4j-go-driver/v6/neo4j/config"
)

// Neo4jStore verifies the graph projection through the production Bolt driver.
type Neo4jStore struct {
	driver   neo4j.Driver
	database string
}

// OpenNeo4jStore constructs a bounded authenticated graph-store driver.
func OpenNeo4jStore(config Neo4jConfig, timeout time.Duration) (*Neo4jStore, error) {
	driver, err := neo4j.NewDriver(
		config.URI,
		neo4j.BasicAuth(config.User, config.Password, ""),
		func(driverConfig *driverconfig.Config) {
			driverConfig.SocketConnectTimeout = timeout
			driverConfig.ConnectionAcquisitionTimeout = timeout
			driverConfig.MaxTransactionRetryTime = 0
			driverConfig.MaxConnectionPoolSize = 2
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create Neo4j driver: %w", err)
	}
	return &Neo4jStore{driver: driver, database: config.Database}, nil
}

// Name returns the safe diagnostic identity for the graph projection store.
func (store *Neo4jStore) Name() string {
	return "Neo4j"
}

// Verify executes a constant read-only Cypher query over Bolt.
func (store *Neo4jStore) Verify(ctx context.Context) error {
	result, err := neo4j.ExecuteQuery(
		ctx,
		store.driver,
		"RETURN 1 AS ready",
		nil,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(store.database),
		neo4j.ExecuteQueryWithReadersRouting(),
	)
	if err != nil {
		return fmt.Errorf("execute graph readiness query: %w", err)
	}
	if len(result.Records) != 1 {
		return fmt.Errorf("graph readiness query returned an unexpected record count")
	}
	return nil
}

// Close releases the Neo4j driver and its connection pool.
func (store *Neo4jStore) Close(ctx context.Context) error {
	return store.driver.Close(ctx)
}
