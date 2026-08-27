package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadUsesSafeLocalDefaults(t *testing.T) {
	configuration, err := Load(func(string) (string, bool) { return "", false })

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if configuration.Address != "127.0.0.1:8080" {
		t.Fatalf("Address = %q, want local default", configuration.Address)
	}
	if configuration.MaxRequestBytes != 11*1024*1024 {
		t.Fatalf("MaxRequestBytes = %d, want 11 MiB", configuration.MaxRequestBytes)
	}
	if configuration.ShutdownTimeout != 10*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want 10s", configuration.ShutdownTimeout)
	}
	if configuration.Agent.BaseURL != "http://127.0.0.1:8090" || configuration.Agent.Timeout != 30*time.Second {
		t.Fatalf("Agent defaults = %#v, want bounded agent defaults", configuration.Agent)
	}
	if configuration.Evaluation.RetrievalStrategy != "vector" || configuration.Evaluation.AgentBuild != "norvii-agent-v1" || configuration.Evaluation.MaintainerToken != "" {
		t.Fatalf("Evaluation defaults = %#v", configuration.Evaluation)
	}
}

func TestLoadRejectsInvalidEvaluationFingerprint(t *testing.T) {
	_, err := Load(func(key string) (string, bool) {
		if key == "NORVII_EVALUATION_RETRIEVAL_FINGERPRINT" {
			return "not-a-sha256", true
		}
		return "", false
	})
	if err == nil {
		t.Fatal("Load() error = nil, want invalid evaluation fingerprint")
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	_, err := Load(func(key string) (string, bool) {
		if key == "NORVII_API_PORT" {
			return "70000", true
		}
		return "", false
	})

	if err == nil {
		t.Fatal("Load() error = nil, want invalid port error")
	}
}

func TestLoadNormalizesEvaluationExecutionIdentity(t *testing.T) {
	configuration, err := Load(func(key string) (string, bool) {
		switch key {
		case "NORVII_EVALUATION_AGENT_BUILD":
			return "\x1crelease-2026-08-26\x1f", true
		case "NORVII_CHAT_MODEL":
			return "\x1dchat-model-test\x1e", true
		case "NORVII_EMBEDDING_MODEL":
			return "\x1etext-embedding-3-small\x1d", true
		default:
			return "", false
		}
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if configuration.Evaluation.AgentBuild != "release-2026-08-26" ||
		configuration.Evaluation.ChatModelIdentity != "chat-model-test" ||
		configuration.Evaluation.EmbeddingModelIdentity != "text-embedding-3-small" {
		t.Fatalf("normalized execution identity = %#v", configuration.Evaluation)
	}
}

func TestLoadRejectsBlankEvaluationExecutionIdentity(t *testing.T) {
	for _, key := range []string{"NORVII_EVALUATION_AGENT_BUILD", "NORVII_CHAT_MODEL", "NORVII_EMBEDDING_MODEL"} {
		t.Run(key, func(t *testing.T) {
			_, err := Load(func(received string) (string, bool) {
				if received == key {
					return " \t ", true
				}
				return "", false
			})
			if err == nil || !strings.Contains(err.Error(), key+" must not be empty") {
				t.Fatalf("Load() error = %v, want normalized empty %s error", err, key)
			}
		})
	}
}
