package main

import (
	"context"
	"strings"
	"testing"
)

func TestVerifyTurnstileRejectsMissingSecret(t *testing.T) {
	err := verifyTurnstile(context.Background(), "", "token")
	if err == nil {
		t.Fatal("expected missing secret error")
	}

	if !strings.Contains(err.Error(), "turnstile not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyTurnstileRejectsMissingToken(t *testing.T) {
	err := verifyTurnstile(context.Background(), "secret", "")
	if err == nil {
		t.Fatal("expected missing token error")
	}

	if !strings.Contains(err.Error(), "missing captcha token") {
		t.Fatalf("unexpected error: %v", err)
	}
}
