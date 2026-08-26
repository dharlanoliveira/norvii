// Package config loads and validates API process configuration.
package config

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHost                        = "127.0.0.1"
	defaultPort                        = 8080
	defaultMaxRequestBytes       int64 = 11 * 1024 * 1024
	defaultShutdownSeconds             = 10
	defaultReadHeaderSeconds           = 5
	defaultReadSeconds                 = 15
	defaultWriteSeconds                = 30
	defaultIdleSeconds                 = 60
	defaultAgentTimeoutSeconds         = 30
	defaultEvaluationStrategy          = "vector"
	defaultEvaluationFingerprint       = "4a24773ff594172e714cb08099af9525839b5c16c0ec09da62bfae7612102523"
	defaultEvaluationAgentBuild        = "norvii-agent-v1"
	executionIdentityTrimCutset        = " \t\n\v\f\r\x1c\x1d\x1e\x1f"
)

var evaluationFingerprintPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// LookupEnv resolves one environment variable without coupling configuration to process state.
type LookupEnv func(string) (string, bool)

// Config contains the bounded settings required by the API HTTP server.
type Config struct {
	Address           string
	MaxRequestBytes   int64
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	Agent             AgentConfig
	Evaluation        EvaluationConfig
}

// AgentConfig contains the internal Python orchestration endpoint settings.
type AgentConfig struct {
	BaseURL string
	Timeout time.Duration
}

// EvaluationConfig is the runnable identity accepted by the managed evaluator. It is frozen
// into a run before its immutable ledger is created.
type EvaluationConfig struct {
	RetrievalStrategy      string
	RetrievalFingerprint   string
	AgentBuild             string
	ChatModelIdentity      string
	EmbeddingModelIdentity string
	MaintainerToken        string
}

// Load reads API configuration, applies local defaults, and rejects unsafe values.
func Load(lookup LookupEnv) (Config, error) {
	host := stringValue(lookup, "NORVII_API_HOST", defaultHost)
	if host == "" {
		return Config{}, fmt.Errorf("NORVII_API_HOST must not be empty")
	}
	port, err := integerValue(lookup, "NORVII_API_PORT", defaultPort, 1, 65535)
	if err != nil {
		return Config{}, err
	}
	maxRequestBytes, err := integer64Value(
		lookup,
		"NORVII_API_MAX_REQUEST_BYTES",
		defaultMaxRequestBytes,
		1,
	)
	if err != nil {
		return Config{}, err
	}
	shutdownSeconds, err := integerValue(
		lookup,
		"NORVII_API_SHUTDOWN_TIMEOUT_SECONDS",
		defaultShutdownSeconds,
		1,
		300,
	)
	if err != nil {
		return Config{}, err
	}
	readHeaderSeconds, err := durationSeconds(
		lookup, "NORVII_API_READ_HEADER_TIMEOUT_SECONDS", defaultReadHeaderSeconds,
	)
	if err != nil {
		return Config{}, err
	}
	readSeconds, err := durationSeconds(lookup, "NORVII_API_READ_TIMEOUT_SECONDS", defaultReadSeconds)
	if err != nil {
		return Config{}, err
	}
	writeSeconds, err := durationSeconds(lookup, "NORVII_API_WRITE_TIMEOUT_SECONDS", defaultWriteSeconds)
	if err != nil {
		return Config{}, err
	}
	idleSeconds, err := durationSeconds(lookup, "NORVII_API_IDLE_TIMEOUT_SECONDS", defaultIdleSeconds)
	if err != nil {
		return Config{}, err
	}
	agentTimeout, err := durationSeconds(
		lookup, "NORVII_AGENT_TIMEOUT_SECONDS", defaultAgentTimeoutSeconds,
	)
	if err != nil {
		return Config{}, err
	}
	evaluationStrategy := stringValue(lookup, "NORVII_EVALUATION_RETRIEVAL_STRATEGY", defaultEvaluationStrategy)
	if evaluationStrategy != "vector" && evaluationStrategy != "hybrid" {
		return Config{}, fmt.Errorf("NORVII_EVALUATION_RETRIEVAL_STRATEGY must be vector or hybrid")
	}
	evaluationFingerprint := stringValue(lookup, "NORVII_EVALUATION_RETRIEVAL_FINGERPRINT", defaultEvaluationFingerprint)
	if !evaluationFingerprintPattern.MatchString(evaluationFingerprint) {
		return Config{}, fmt.Errorf("NORVII_EVALUATION_RETRIEVAL_FINGERPRINT must be a lowercase SHA-256 value")
	}
	agentBuild, err := normalizedExecutionIdentityValue(
		lookup, "NORVII_EVALUATION_AGENT_BUILD", defaultEvaluationAgentBuild,
	)
	if err != nil {
		return Config{}, err
	}
	chatModel, err := normalizedExecutionIdentityValue(
		lookup, "NORVII_CHAT_MODEL", "gpt-4o-mini",
	)
	if err != nil {
		return Config{}, err
	}
	embeddingModel, err := normalizedExecutionIdentityValue(
		lookup, "NORVII_EMBEDDING_MODEL", "text-embedding-3-small",
	)
	if err != nil {
		return Config{}, err
	}
	return Config{
		Address:           net.JoinHostPort(host, strconv.Itoa(port)),
		MaxRequestBytes:   maxRequestBytes,
		ShutdownTimeout:   time.Duration(shutdownSeconds) * time.Second,
		ReadHeaderTimeout: readHeaderSeconds,
		ReadTimeout:       readSeconds,
		WriteTimeout:      writeSeconds,
		IdleTimeout:       idleSeconds,
		Agent: AgentConfig{
			BaseURL: stringValue(lookup, "NORVII_AGENT_BASE_URL", "http://127.0.0.1:8090"),
			Timeout: agentTimeout,
		},
		Evaluation: EvaluationConfig{
			RetrievalStrategy: evaluationStrategy, RetrievalFingerprint: evaluationFingerprint,
			AgentBuild: agentBuild, ChatModelIdentity: chatModel, EmbeddingModelIdentity: embeddingModel,
			MaintainerToken: stringValue(lookup, "NORVII_EVALUATION_MAINTAINER_TOKEN", ""),
		},
	}, nil
}

func durationSeconds(lookup LookupEnv, key string, fallback int) (time.Duration, error) {
	seconds, err := integerValue(lookup, key, fallback, 1, 300)
	if err != nil {
		return 0, err
	}
	return time.Duration(seconds) * time.Second, nil
}

func stringValue(lookup LookupEnv, key, fallback string) string {
	if value, ok := lookup(key); ok {
		return value
	}
	return fallback
}

func normalizedExecutionIdentityValue(lookup LookupEnv, key, fallback string) (string, error) {
	value := strings.Trim(stringValue(lookup, key, fallback), executionIdentityTrimCutset)
	if value == "" {
		return "", fmt.Errorf("%s must not be empty", key)
	}
	return value, nil
}

func integerValue(lookup LookupEnv, key string, fallback, minimum, maximum int) (int, error) {
	value, ok := lookup(key)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum || parsed > maximum {
		return 0, fmt.Errorf("%s must be an integer between %d and %d", key, minimum, maximum)
	}
	return parsed, nil
}

func integer64Value(lookup LookupEnv, key string, fallback, minimum int64) (int64, error) {
	value, ok := lookup(key)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < minimum {
		return 0, fmt.Errorf("%s must be an integer greater than or equal to %d", key, minimum)
	}
	return parsed, nil
}
