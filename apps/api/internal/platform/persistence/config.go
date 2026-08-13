package persistence

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	minimumPort           = 1
	maximumPort           = 65535
	minimumTimeoutSeconds = 1
	maximumTimeoutSeconds = 10
)

// EnvironmentLookup retrieves one environment variable without coupling configuration to process state.
type EnvironmentLookup func(string) (string, bool)

// Config contains validated persistence settings for one runtime process.
type Config struct {
	Postgres PostgresConfig
	Neo4j    Neo4jConfig
	Timeout  time.Duration
}

// PostgresConfig contains discrete PostgreSQL connection fields so credentials never enter a URL.
type PostgresConfig struct {
	Host     string
	Port     uint16
	Database string
	User     string
	Password string
}

// Neo4jConfig contains the Bolt connection settings for the local graph projection.
type Neo4jConfig struct {
	URI      string
	User     string
	Password string
	Database string
}

// LoadConfig validates the version 1 local persistence environment contract.
func LoadConfig(lookup EnvironmentLookup) (Config, error) {
	postgresPort, err := requiredInteger(lookup, "NORVII_POSTGRES_PORT", minimumPort, maximumPort)
	if err != nil {
		return Config{}, err
	}
	timeoutSeconds, err := requiredInteger(
		lookup,
		"NORVII_PERSISTENCE_TIMEOUT_SECONDS",
		minimumTimeoutSeconds,
		maximumTimeoutSeconds,
	)
	if err != nil {
		return Config{}, err
	}

	postgresHost, err := requiredValue(lookup, "NORVII_POSTGRES_HOST")
	if err != nil {
		return Config{}, err
	}
	postgresDatabase, err := requiredValue(lookup, "NORVII_POSTGRES_DATABASE")
	if err != nil {
		return Config{}, err
	}
	postgresUser, err := requiredValue(lookup, "NORVII_POSTGRES_USER")
	if err != nil {
		return Config{}, err
	}
	postgresPassword, err := requiredValue(lookup, "NORVII_POSTGRES_PASSWORD")
	if err != nil {
		return Config{}, err
	}
	neo4jURI, err := requiredNeo4jURI(lookup)
	if err != nil {
		return Config{}, err
	}
	neo4jUser, err := requiredValue(lookup, "NORVII_NEO4J_USER")
	if err != nil {
		return Config{}, err
	}
	neo4jPassword, err := requiredValue(lookup, "NORVII_NEO4J_PASSWORD")
	if err != nil {
		return Config{}, err
	}
	neo4jDatabase, err := requiredValue(lookup, "NORVII_NEO4J_DATABASE")
	if err != nil {
		return Config{}, err
	}

	return Config{
		Postgres: PostgresConfig{
			Host:     postgresHost,
			Port:     uint16(postgresPort),
			Database: postgresDatabase,
			User:     postgresUser,
			Password: postgresPassword,
		},
		Neo4j: Neo4jConfig{
			URI:      neo4jURI,
			User:     neo4jUser,
			Password: neo4jPassword,
			Database: neo4jDatabase,
		},
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}, nil
}

func requiredValue(lookup EnvironmentLookup, name string) (string, error) {
	value, found := lookup(name)
	if !found || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("configuration variable %s is required", name)
	}
	return value, nil
}

func requiredInteger(lookup EnvironmentLookup, name string, minimum int, maximum int) (int, error) {
	value, err := requiredValue(lookup, name)
	if err != nil {
		return 0, err
	}
	parsed, parseErr := strconv.Atoi(value)
	if parseErr != nil || parsed < minimum || parsed > maximum {
		return 0, fmt.Errorf("configuration variable %s must be an integer from %d through %d", name, minimum, maximum)
	}
	return parsed, nil
}

func requiredNeo4jURI(lookup EnvironmentLookup) (string, error) {
	value, err := requiredValue(lookup, "NORVII_NEO4J_URI")
	if err != nil {
		return "", err
	}
	parsed, parseErr := url.Parse(value)
	if parseErr != nil || parsed.Scheme != "neo4j" || parsed.Host == "" || parsed.User != nil {
		return "", fmt.Errorf("configuration variable NORVII_NEO4J_URI must be a credential-free neo4j URI")
	}
	return value, nil
}
