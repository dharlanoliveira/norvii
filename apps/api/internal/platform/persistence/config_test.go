package persistence

import (
	"strings"
	"testing"
	"time"
)

func TestLoadConfigAcceptsCompleteEnvironment(t *testing.T) {
	environment := validEnvironment()

	config, err := LoadConfig(environment.lookup)

	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.Postgres.Port != 5432 {
		t.Fatalf("Postgres.Port = %d, want 5432", config.Postgres.Port)
	}
	if config.Neo4j.URI != "neo4j://localhost:7687" {
		t.Fatalf("Neo4j.URI = %q, want local Bolt URI", config.Neo4j.URI)
	}
	if config.Timeout != 5*time.Second {
		t.Fatalf("Timeout = %s, want 5s", config.Timeout)
	}
}

func TestLoadConfigRejectsInvalidEnvironmentWithoutDisclosingSecrets(t *testing.T) {
	const postgresSecret = "postgres-secret-value"
	const neo4jSecret = "neo4j-secret-value"

	tests := []struct {
		name        string
		variable    string
		value       string
		wantMessage string
	}{
		{name: "missing host", variable: "NORVII_POSTGRES_HOST", value: "", wantMessage: "NORVII_POSTGRES_HOST"},
		{name: "invalid postgres port", variable: "NORVII_POSTGRES_PORT", value: "70000", wantMessage: "NORVII_POSTGRES_PORT"},
		{name: "invalid neo4j URI", variable: "NORVII_NEO4J_URI", value: "http://localhost:7687", wantMessage: "NORVII_NEO4J_URI"},
		{name: "invalid timeout", variable: "NORVII_PERSISTENCE_TIMEOUT_SECONDS", value: "11", wantMessage: "NORVII_PERSISTENCE_TIMEOUT_SECONDS"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			environment := validEnvironment()
			environment["NORVII_POSTGRES_PASSWORD"] = postgresSecret
			environment["NORVII_NEO4J_PASSWORD"] = neo4jSecret
			environment[test.variable] = test.value

			_, err := LoadConfig(environment.lookup)
			if err == nil {
				t.Fatal("LoadConfig() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), test.wantMessage) {
				t.Fatalf("LoadConfig() error = %q, want variable name %q", err, test.wantMessage)
			}
			if strings.Contains(err.Error(), postgresSecret) || strings.Contains(err.Error(), neo4jSecret) {
				t.Fatalf("LoadConfig() error disclosed a secret: %q", err)
			}
		})
	}
}

type testEnvironment map[string]string

func (environment testEnvironment) lookup(key string) (string, bool) {
	value, found := environment[key]
	return value, found
}

func validEnvironment() testEnvironment {
	return testEnvironment{
		"NORVII_POSTGRES_HOST":               "localhost",
		"NORVII_POSTGRES_PORT":               "5432",
		"NORVII_POSTGRES_DATABASE":           "norvii",
		"NORVII_POSTGRES_USER":               "norvii",
		"NORVII_POSTGRES_PASSWORD":           "local-postgres-secret",
		"NORVII_NEO4J_URI":                   "neo4j://localhost:7687",
		"NORVII_NEO4J_USER":                  "neo4j",
		"NORVII_NEO4J_PASSWORD":              "local-neo4j-secret",
		"NORVII_NEO4J_DATABASE":              "neo4j",
		"NORVII_PERSISTENCE_TIMEOUT_SECONDS": "5",
	}
}
