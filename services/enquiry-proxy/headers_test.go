package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxyIsAllowedOriginSupportsCommaSeparatedList(t *testing.T) {
	allowList := "https://finiskitchens.co.uk, https://finiskitchens.southcoastapps.co.uk"

	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{
			name:   "production apex allowed",
			origin: "https://finiskitchens.co.uk",
			want:   true,
		},
		{
			name:   "staging allowed",
			origin: "https://finiskitchens.southcoastapps.co.uk",
			want:   true,
		},
		{
			name:   "unknown origin rejected",
			origin: "https://evil.example",
			want:   false,
		},
		{
			name:   "empty origin rejected",
			origin: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAllowedOrigin(tt.origin, allowList)
			if got != tt.want {
				t.Fatalf("isAllowedOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestProxyHeadersAllowKnownOriginPreflight(t *testing.T) {
	cfg := proxyConfig{
		AllowOrigin: "https://finiskitchens.co.uk,https://finiskitchens.southcoastapps.co.uk",
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/enquiry", nil)
	req.Header.Set("Origin", "https://finiskitchens.co.uk")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")

	rr := httptest.NewRecorder()

	withProxyHeaders(cfg, next).ServeHTTP(rr, req)

	if nextCalled {
		t.Fatal("next handler should not be called for OPTIONS")
	}

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://finiskitchens.co.uk" {
		t.Fatalf("unexpected allow-origin header: %q", got)
	}

	if got := rr.Header().Get("Access-Control-Allow-Methods"); got != "POST, OPTIONS" {
		t.Fatalf("unexpected allow-methods header: %q", got)
	}

	if got := rr.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type" {
		t.Fatalf("unexpected allow-headers header: %q", got)
	}

	if got := rr.Header().Get("Access-Control-Max-Age"); got != "600" {
		t.Fatalf("unexpected max-age header: %q", got)
	}
}

func TestProxyHeadersDoNotEchoBadOrigin(t *testing.T) {
	cfg := proxyConfig{
		AllowOrigin: "https://finiskitchens.co.uk",
	}

	req := httptest.NewRequest(http.MethodOptions, "/api/enquiry", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Access-Control-Request-Method", "POST")

	rr := httptest.NewRecorder()

	withProxyHeaders(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for OPTIONS")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("bad origin should not be echoed, got %q", got)
	}
}

func TestProxyHeadersSetSecurityHeadersOnNonOptionsRequest(t *testing.T) {
	cfg := proxyConfig{
		AllowOrigin: "https://finiskitchens.co.uk",
	}

	req := httptest.NewRequest(http.MethodPost, "/api/enquiry", nil)
	req.Header.Set("Origin", "https://finiskitchens.co.uk")

	rr := httptest.NewRecorder()

	withProxyHeaders(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://finiskitchens.co.uk" {
		t.Fatalf("unexpected allow-origin header: %q", got)
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
