package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func isolateEnquiryConfigEnv(t *testing.T) {
	t.Helper()

	unsetEnv(
		t,
		"ALLOW_ORIGIN",
		"SUBJECT_PREFIX",
		"REQUIRE_CONSENT",
		"SMTP_HOST",
		"SMTP_PORT",
		"SMTP_FROM",
		"ENQUIRY_TO",
		"SMTP_TO",
		"SUPPORT_TO",
		"SMTP_USER",
		"SMTP_USER_FILE",
		"SMTP_PASS",
		"SMTP_PASS_FILE",
		"INTERNAL_ENQUIRY_SECRET",
		"INTERNAL_ENQUIRY_SECRET_FILE",
	)
}

func TestEnquiryGetSecretPrefersEnvValue(t *testing.T) {
	unsetEnv(t, "TEST_ENQUIRY_SECRET", "TEST_ENQUIRY_SECRET_FILE")

	t.Setenv("TEST_ENQUIRY_SECRET", "  env-secret  ")

	got, err := getSecret("TEST_ENQUIRY_SECRET")
	if err != nil {
		t.Fatalf("getSecret returned error: %v", err)
	}

	if got != "env-secret" {
		t.Fatalf("expected trimmed env secret, got %q", got)
	}
}

func TestEnquiryGetSecretReadsDockerSecretFile(t *testing.T) {
	unsetEnv(t, "TEST_ENQUIRY_SECRET", "TEST_ENQUIRY_SECRET_FILE")

	secretPath := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secretPath, []byte("  file-secret\n"), 0o600); err != nil {
		t.Fatalf("write secret file: %v", err)
	}

	t.Setenv("TEST_ENQUIRY_SECRET_FILE", secretPath)

	got, err := getSecret("TEST_ENQUIRY_SECRET")
	if err != nil {
		t.Fatalf("getSecret returned error: %v", err)
	}

	if got != "file-secret" {
		t.Fatalf("expected trimmed file secret, got %q", got)
	}
}

func TestEnquiryLoadConfigReadsRequiredValues(t *testing.T) {
	isolateEnquiryConfigEnv(t)

	t.Setenv("ALLOW_ORIGIN", "https://finiskitchens.co.uk")
	t.Setenv("SUBJECT_PREFIX", "[Fini's Kitchens]")
	t.Setenv("REQUIRE_CONSENT", "true")
	t.Setenv("SMTP_HOST", "mail.example.test")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_FROM", "Fini's Kitchens <hello@example.test>")
	t.Setenv("ENQUIRY_TO", "owner@example.test")
	t.Setenv("SUPPORT_TO", "support@example.test")
	t.Setenv("SMTP_USER", "smtp-user")
	t.Setenv("SMTP_PASS", "smtp-pass")
	t.Setenv("INTERNAL_ENQUIRY_SECRET", "internal-secret")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	if cfg.AllowOrigin != "https://finiskitchens.co.uk" {
		t.Fatalf("unexpected allow origin: %q", cfg.AllowOrigin)
	}

	if cfg.SubjectPrefix != "[Fini's Kitchens]" {
		t.Fatalf("unexpected subject prefix: %q", cfg.SubjectPrefix)
	}

	if !cfg.RequireConsent {
		t.Fatal("expected RequireConsent true")
	}

	if cfg.EnquiryTo != "owner@example.test" {
		t.Fatalf("unexpected enquiry recipient: %q", cfg.EnquiryTo)
	}

	if cfg.SupportTo != "support@example.test" {
		t.Fatalf("unexpected support recipient: %q", cfg.SupportTo)
	}

	if cfg.InternalEnquirySecret != "internal-secret" {
		t.Fatal("unexpected internal enquiry secret")
	}

	if cfg.SMTP.Host != "mail.example.test" {
		t.Fatalf("unexpected smtp host: %q", cfg.SMTP.Host)
	}

	if cfg.SMTP.Port != 587 {
		t.Fatalf("unexpected smtp port: %d", cfg.SMTP.Port)
	}

	if cfg.SMTP.User != "smtp-user" || cfg.SMTP.Pass != "smtp-pass" {
		t.Fatal("unexpected smtp credentials")
	}
}

func TestEnquiryLoadConfigRequiresSMTPHost(t *testing.T) {
	isolateEnquiryConfigEnv(t)

	t.Setenv("SMTP_FROM", "hello@example.test")
	t.Setenv("ENQUIRY_TO", "owner@example.test")
	t.Setenv("SMTP_USER", "smtp-user")
	t.Setenv("SMTP_PASS", "smtp-pass")
	t.Setenv("INTERNAL_ENQUIRY_SECRET", "internal-secret")

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected missing SMTP_HOST error")
	}

	if !strings.Contains(err.Error(), "SMTP_HOST is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnquiryLoadConfigRejectsInvalidSMTPPort(t *testing.T) {
	isolateEnquiryConfigEnv(t)

	t.Setenv("SMTP_HOST", "mail.example.test")
	t.Setenv("SMTP_PORT", "not-a-port")
	t.Setenv("SMTP_FROM", "hello@example.test")
	t.Setenv("ENQUIRY_TO", "owner@example.test")
	t.Setenv("SMTP_USER", "smtp-user")
	t.Setenv("SMTP_PASS", "smtp-pass")
	t.Setenv("INTERNAL_ENQUIRY_SECRET", "internal-secret")

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected invalid SMTP_PORT error")
	}

	if !strings.Contains(err.Error(), "invalid SMTP_PORT") {
		t.Fatalf("unexpected error: %v", err)
	}
}
