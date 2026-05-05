package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		key := key
		old, had := os.LookupEnv(key)
		os.Unsetenv(key)

		t.Cleanup(func() {
			if had {
				_ = os.Setenv(key, old)
				return
			}
			_ = os.Unsetenv(key)
		})
	}
}

func isolateProxyConfigEnv(t *testing.T) {
	t.Helper()

	unsetEnv(
		t,
		"ADDR",
		"UPSTREAM_URL",
		"ALLOW_ORIGIN",
		"MAX_BODY_BYTES",
		"INTERNAL_ENQUIRY_SECRET",
		"INTERNAL_ENQUIRY_SECRET_FILE",
		"TURNSTILE_SECRET",
		"TURNSTILE_SECRET_FILE",
	)
}

func TestProxyGetSecretPrefersEnvValue(t *testing.T) {
	unsetEnv(t, "TEST_PROXY_SECRET", "TEST_PROXY_SECRET_FILE")

	t.Setenv("TEST_PROXY_SECRET", "  env-secret  ")

	got, err := getSecret("TEST_PROXY_SECRET")
	if err != nil {
		t.Fatalf("getSecret returned error: %v", err)
	}

	if got != "env-secret" {
		t.Fatalf("expected trimmed env secret, got %q", got)
	}
}

func TestProxyGetSecretReadsDockerSecretFile(t *testing.T) {
	unsetEnv(t, "TEST_PROXY_SECRET", "TEST_PROXY_SECRET_FILE")

	secretPath := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secretPath, []byte("  file-secret\n"), 0o600); err != nil {
		t.Fatalf("write secret file: %v", err)
	}

	t.Setenv("TEST_PROXY_SECRET_FILE", secretPath)

	got, err := getSecret("TEST_PROXY_SECRET")
	if err != nil {
		t.Fatalf("getSecret returned error: %v", err)
	}

	if got != "file-secret" {
		t.Fatalf("expected trimmed file secret, got %q", got)
	}
}

func TestProxyGetSecretFailsWhenMissing(t *testing.T) {
	unsetEnv(t, "TEST_PROXY_SECRET", "TEST_PROXY_SECRET_FILE")

	_, err := getSecret("TEST_PROXY_SECRET")
	if err == nil {
		t.Fatal("expected missing secret error")
	}

	if !strings.Contains(err.Error(), "TEST_PROXY_SECRET not set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProxyLoadConfigDefaults(t *testing.T) {
	isolateProxyConfigEnv(t)

	t.Setenv("UPSTREAM_URL", "http://enquiry:8080/api/enquiry")
	t.Setenv("INTERNAL_ENQUIRY_SECRET", "internal-secret")
	t.Setenv("TURNSTILE_SECRET", "turnstile-secret")
	t.Setenv("ALLOW_ORIGIN", "https://finiskitchens.co.uk,https://finiskitchens.southcoastapps.co.uk")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	if cfg.ListenAddr != ":8080" {
		t.Fatalf("expected default listen addr :8080, got %q", cfg.ListenAddr)
	}

	if cfg.UpstreamURL != "http://enquiry:8080/api/enquiry" {
		t.Fatalf("unexpected upstream url: %q", cfg.UpstreamURL)
	}

	if cfg.InternalSecret != "internal-secret" {
		t.Fatalf("unexpected internal secret")
	}

	if cfg.TurnstileSecret != "turnstile-secret" {
		t.Fatalf("unexpected turnstile secret")
	}

	if cfg.MaxBodyBytes != int64(64<<10) {
		t.Fatalf("expected default max body 64KB, got %d", cfg.MaxBodyBytes)
	}

	if cfg.UpstreamTimeout != 10*time.Second {
		t.Fatalf("expected 10s upstream timeout, got %s", cfg.UpstreamTimeout)
	}
}

func TestProxyLoadConfigRejectsInvalidMaxBody(t *testing.T) {
	isolateProxyConfigEnv(t)

	t.Setenv("UPSTREAM_URL", "http://enquiry:8080/api/enquiry")
	t.Setenv("INTERNAL_ENQUIRY_SECRET", "internal-secret")
	t.Setenv("TURNSTILE_SECRET", "turnstile-secret")
	t.Setenv("MAX_BODY_BYTES", "999")

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected invalid MAX_BODY_BYTES error")
	}

	if !strings.Contains(err.Error(), "MAX_BODY_BYTES out of range") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProxyLoadConfigRejectsMissingRequiredValues(t *testing.T) {
	isolateProxyConfigEnv(t)

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected missing UPSTREAM_URL error")
	}

	if !strings.Contains(err.Error(), "UPSTREAM_URL is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
