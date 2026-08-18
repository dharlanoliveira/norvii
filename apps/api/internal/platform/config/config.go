// Package config loads and validates API process configuration.
package config

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

const (
	defaultHost                    = "127.0.0.1"
	defaultPort                    = 8080
	defaultMaxRequestBytes   int64 = 11 * 1024 * 1024
	defaultShutdownSeconds         = 10
	defaultReadHeaderSeconds       = 5
	defaultReadSeconds             = 15
	defaultWriteSeconds            = 30
	defaultIdleSeconds             = 60
)

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
	return Config{
		Address:           net.JoinHostPort(host, strconv.Itoa(port)),
		MaxRequestBytes:   maxRequestBytes,
		ShutdownTimeout:   time.Duration(shutdownSeconds) * time.Second,
		ReadHeaderTimeout: readHeaderSeconds,
		ReadTimeout:       readSeconds,
		WriteTimeout:      writeSeconds,
		IdleTimeout:       idleSeconds,
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
