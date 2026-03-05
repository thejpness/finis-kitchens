package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type proxyConfig struct {
	ListenAddr  string
	UpstreamURL string

	AllowOrigin string

	InternalSecret string

	TurnstileSecret string

	MaxBodyBytes int64

	UpstreamTimeout time.Duration
}

func loadConfig() (proxyConfig, error) {
	upstream := strings.TrimSpace(os.Getenv("UPSTREAM_URL"))
	if upstream == "" {
		return proxyConfig{}, fmt.Errorf("UPSTREAM_URL is required")
	}

	internalSecret, err := getSecret("INTERNAL_ENQUIRY_SECRET")
	if err != nil {
		return proxyConfig{}, err
	}

	// REQUIRED: fail fast if not configured (prevents “broken form” behaviour)
	turnstileSecret, err := getSecret("TURNSTILE_SECRET")
	if err != nil {
		return proxyConfig{}, err
	}

	addr := strings.TrimSpace(os.Getenv("ADDR"))
	if addr == "" {
		addr = ":8080"
	}

	allowOrigin := strings.TrimSpace(os.Getenv("ALLOW_ORIGIN"))

	maxBody := int64(64 << 10) // 64KB default
	if v := strings.TrimSpace(os.Getenv("MAX_BODY_BYTES")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return proxyConfig{}, fmt.Errorf("invalid MAX_BODY_BYTES %q: %w", v, err)
		}
		// guardrails: 1KB min, 1MB max
		if n < 1024 || n > (1<<20) {
			return proxyConfig{}, fmt.Errorf("MAX_BODY_BYTES out of range (1024..1048576): %d", n)
		}
		maxBody = n
	}

	timeout := 10 * time.Second

	return proxyConfig{
		ListenAddr:      addr,
		UpstreamURL:     upstream,
		AllowOrigin:     allowOrigin,
		InternalSecret:  internalSecret,
		TurnstileSecret: turnstileSecret,
		MaxBodyBytes:    maxBody,
		UpstreamTimeout: timeout,
	}, nil
}

// getSecret returns NAME from the environment or from NAME_FILE (Docker secrets style).
func getSecret(name string) (string, error) {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v, nil
	}

	if path := strings.TrimSpace(os.Getenv(name + "_FILE")); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("reading secret file %s for %s: %w", path, name, err)
		}
		v := strings.TrimSpace(string(b))
		if v == "" {
			return "", fmt.Errorf("secret %s from file %s is empty", name, path)
		}
		return v, nil
	}

	return "", fmt.Errorf("%s not set (no %s or %s_FILE)", name, name, name)
}
