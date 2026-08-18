package config

import (
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
