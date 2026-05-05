package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireInternalSecretAllowsValidSecret(t *testing.T) {
	nextCalled := false

	handler := requireInternalSecret("expected-secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/enquiry", nil)
	req.Header.Set(internalSecretHeader, "expected-secret")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}
}

func TestRequireInternalSecretRejectsMissingOrWrongSecret(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		got      string
	}{
		{
			name:     "missing supplied secret",
			expected: "expected-secret",
			got:      "",
		},
		{
			name:     "wrong supplied secret",
			expected: "expected-secret",
			got:      "wrong-secret",
		},
		{
			name:     "empty configured secret rejects even when supplied",
			expected: "",
			got:      "anything",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false

			handler := requireInternalSecret(tt.expected, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusAccepted)
			}))

			req := httptest.NewRequest(http.MethodPost, "/api/enquiry", nil)
			if tt.got != "" {
				req.Header.Set(internalSecretHeader, tt.got)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if nextCalled {
				t.Fatal("next handler should not be called")
			}

			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", rr.Code)
			}
		})
	}
}

func TestCommonHeadersAllowKnownOriginPreflight(t *testing.T) {
	oldCfg := cfg
	cfg = appConfig{
		AllowOrigin: "https://finiskitchens.co.uk,https://finiskitchens.southcoastapps.co.uk",
	}
	t.Cleanup(func() {
		cfg = oldCfg
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/enquiry", nil)
	req.Header.Set("Origin", "https://finiskitchens.co.uk")
	req.Header.Set("Access-Control-Request-Method", "POST")

	rr := httptest.NewRecorder()

	withCommonHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for OPTIONS")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://finiskitchens.co.uk" {
		t.Fatalf("unexpected allow-origin header: %q", got)
	}

	if got := rr.Header().Get("Access-Control-Allow-Methods"); got != "POST, OPTIONS" {
		t.Fatalf("unexpected allow-methods header: %q", got)
	}
}

func TestCommonHeadersDoNotEchoBadOrigin(t *testing.T) {
	oldCfg := cfg
	cfg = appConfig{
		AllowOrigin: "https://finiskitchens.co.uk",
	}
	t.Cleanup(func() {
		cfg = oldCfg
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/enquiry", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Access-Control-Request-Method", "POST")

	rr := httptest.NewRecorder()

	withCommonHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for OPTIONS")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("bad origin should not be echoed, got %q", got)
	}
}

func TestCommonHeadersSetSecurityHeadersOnNonOptionsRequest(t *testing.T) {
	oldCfg := cfg
	cfg = appConfig{
		AllowOrigin: "https://finiskitchens.co.uk",
	}
	t.Cleanup(func() {
		cfg = oldCfg
	})

	req := httptest.NewRequest(http.MethodPost, "/api/enquiry", nil)
	req.Header.Set("Origin", "https://finiskitchens.co.uk")

	rr := httptest.NewRecorder()

	withCommonHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}

	expectedHeaders := map[string]string{
		"Content-Security-Policy": "default-src 'none'",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
	}

	for name, want := range expectedHeaders {
		if got := rr.Header().Get(name); got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
}
