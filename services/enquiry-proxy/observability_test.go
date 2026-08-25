package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewRequestIDFormat(t *testing.T) {
	requestID, err := newRequestID()
	if err != nil {
		t.Fatalf("newRequestID returned error: %v", err)
	}
	if !isValidRequestID(requestID) {
		t.Fatalf("request ID %q is not valid", requestID)
	}
}

func TestProxyObservabilityGeneratesForwardsAndLogsSafeRequestID(t *testing.T) {
	logs := captureProxyLogs(t)

	oldVerifier := turnstileVerifier
	turnstileVerifier = func(context.Context, string, string) error { return nil }
	t.Cleanup(func() { turnstileVerifier = oldVerifier })

	var forwardedRequestID string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardedRequestID = r.Header.Get(requestIDHeader)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	t.Cleanup(upstream.Close)

	handler := newProxyHandler(proxyConfig{
		UpstreamURL:     upstream.URL,
		InternalSecret:  "test-internal-secret",
		TurnstileSecret: "test-turnstile-secret",
		MaxBodyBytes:    64 << 10,
		UpstreamTimeout: 0,
	}, upstream.Client())

	body := `{
		"name": "PRIVACY_TEST_NAME",
		"email": "privacy-test@example.invalid",
		"phone": "PRIVACY_TEST_PHONE",
		"timeline": "PRIVACY_TEST_TIMELINE",
		"message": "PRIVACY_TEST_MESSAGE",
		"page": "https://example.invalid/contact?secret-query-marker=PRIVACY_TEST_QUERY",
		"source": "PRIVACY_TEST_SOURCE",
		"consent": true,
		"captchaToken": "PRIVACY_TEST_CAPTCHA_TOKEN",
		"channel": "marketing"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/enquiry", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(requestIDHeader, "PRIVACY_TEST_EXTERNAL_REQUEST_ID")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}
	requestID := rr.Header().Get(requestIDHeader)
	if !isValidRequestID(requestID) {
		t.Fatalf("response request ID %q is not valid", requestID)
	}
	if requestID == "PRIVACY_TEST_EXTERNAL_REQUEST_ID" {
		t.Fatal("client-supplied request ID was accepted")
	}
	if forwardedRequestID != requestID {
		t.Fatalf("forwarded request ID %q, want %q", forwardedRequestID, requestID)
	}

	if !strings.Contains(logs.String(), "event=enquiry_request service=enquiry-proxy") {
		t.Fatalf("expected proxy request log, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), "request_id="+requestID) || !strings.Contains(logs.String(), "outcome=accepted") {
		t.Fatalf("expected safe request ID and outcome in log, got %q", logs.String())
	}

	for _, prohibited := range []string{
		"PRIVACY_TEST_NAME",
		"privacy-test@example.invalid",
		"PRIVACY_TEST_PHONE",
		"PRIVACY_TEST_TIMELINE",
		"PRIVACY_TEST_MESSAGE",
		"https://example.invalid/contact?secret-query-marker=PRIVACY_TEST_QUERY",
		"PRIVACY_TEST_QUERY",
		"PRIVACY_TEST_SOURCE",
		"PRIVACY_TEST_CAPTCHA_TOKEN",
		"PRIVACY_TEST_EXTERNAL_REQUEST_ID",
	} {
		if strings.Contains(logs.String(), prohibited) {
			t.Fatalf("prohibited request data %q in log %q", prohibited, logs.String())
		}
	}
}

func captureProxyLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	var logs bytes.Buffer
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	oldPrefix := log.Prefix()
	log.SetOutput(&logs)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
		log.SetPrefix(oldPrefix)
	})
	return &logs
}
