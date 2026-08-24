package main

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func boolPtr(v bool) *bool {
	return &v
}

func TestValidateAcceptsValidMarketingEnquiry(t *testing.T) {
	in := enquiry{
		Name:     "  Jane   Smith  ",
		Email:    "jane@example.test",
		Timeline: "1-3 months",
		Message:  "I would like to discuss a kitchen project.",
		Consent:  boolPtr(true),
	}

	if err := validate(&in, true); err != nil {
		t.Fatalf("validate returned error: %v", err)
	}

	if in.Name != "Jane Smith" {
		t.Fatalf("expected normalised name, got %q", in.Name)
	}
}

func TestValidateRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name           string
		input          enquiry
		requireConsent bool
		wantErr        string
	}{
		{
			name: "missing name",
			input: enquiry{
				Email:    "jane@example.test",
				Timeline: "1-3 months",
				Message:  "This is a valid length message.",
				Consent:  boolPtr(true),
			},
			requireConsent: true,
			wantErr:        "name is required",
		},
		{
			name: "invalid email",
			input: enquiry{
				Name:     "Jane Smith",
				Email:    "not-an-email",
				Timeline: "1-3 months",
				Message:  "This is a valid length message.",
				Consent:  boolPtr(true),
			},
			requireConsent: true,
			wantErr:        "valid email required",
		},
		{
			name: "missing timeline",
			input: enquiry{
				Name:    "Jane Smith",
				Email:   "jane@example.test",
				Message: "This is a valid length message.",
				Consent: boolPtr(true),
			},
			requireConsent: true,
			wantErr:        "timeline is required",
		},
		{
			name: "short message",
			input: enquiry{
				Name:     "Jane Smith",
				Email:    "jane@example.test",
				Timeline: "1-3 months",
				Message:  "short",
				Consent:  boolPtr(true),
			},
			requireConsent: true,
			wantErr:        "message is too short",
		},
		{
			name: "missing consent when required",
			input: enquiry{
				Name:     "Jane Smith",
				Email:    "jane@example.test",
				Timeline: "1-3 months",
				Message:  "This is a valid length message.",
			},
			requireConsent: true,
			wantErr:        "consent is required",
		},
		{
			name: "false consent when required",
			input: enquiry{
				Name:     "Jane Smith",
				Email:    "jane@example.test",
				Timeline: "1-3 months",
				Message:  "This is a valid length message.",
				Consent:  boolPtr(false),
			},
			requireConsent: true,
			wantErr:        "consent is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(&tt.input, tt.requireConsent)
			if err == nil {
				t.Fatal("expected validation error")
			}

			if err.Error() != tt.wantErr {
				t.Fatalf("got error %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidateDoesNotRequireConsentWhenDisabled(t *testing.T) {
	in := enquiry{
		Name:     "Jane Smith",
		Email:    "jane@example.test",
		Timeline: "1-3 months",
		Message:  "This is a valid length message.",
	}

	if err := validate(&in, false); err != nil {
		t.Fatalf("validate returned error: %v", err)
	}
}

func TestHandleEnquiryRejectsWrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/enquiry", nil)
	rr := httptest.NewRecorder()

	handleEnquiry(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestHandleEnquiryRejectsWrongContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/enquiry", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "text/plain")

	rr := httptest.NewRecorder()

	handleEnquiry(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", rr.Code)
	}
}

func TestHandleEnquiryRejectsUnknownJSONFields(t *testing.T) {
	body := []byte(`{
		"name": "Jane Smith",
		"email": "jane@example.test",
		"timeline": "1-3 months",
		"message": "This is a valid length message.",
		"unexpected": "field"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/enquiry", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	handleEnquiry(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleEnquiryHoneypotReturnsAcceptedWithoutSMTP(t *testing.T) {
	body := []byte(`{
		"name": "Bot Person",
		"email": "bot@example.test",
		"timeline": "1-3 months",
		"message": "This message is long enough.",
		"company": "spam ltd"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/enquiry", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	handleEnquiry(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}

	if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Fatalf("expected ok response, got %q", rr.Body.String())
	}
}

func TestHandleEnquirySuccessLogExcludesCustomerData(t *testing.T) {
	logs := captureEnquiryLogs(t)
	configureEnquiryLoggingTest(t)

	sent := false
	withEnquiryMailSender(t, func(to, subject, plainBody, htmlBody, replyTo string) error {
		sent = true
		if to != "recipient@example.invalid" {
			t.Fatalf("unexpected recipient %q", to)
		}
		if !strings.Contains(plainBody, "PRIVACY_TEST_MESSAGE") {
			t.Fatal("expected email content to remain unchanged")
		}
		return nil
	})

	rr := postEnquiry(t, privacyTestEnquiry("marketing"))
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}
	if !sent {
		t.Fatal("expected enquiry email to be sent")
	}

	assertPrivacySafeEnquiryLog(t, logs.String())
	if !strings.Contains(logs.String(), `enquiry accepted channel="marketing"`) {
		t.Fatalf("expected safe accepted log, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), "duration_ms=") {
		t.Fatalf("expected processing duration in log, got %q", logs.String())
	}
}

func TestHandleEnquiryUnknownChannelLogDoesNotReflectInput(t *testing.T) {
	logs := captureEnquiryLogs(t)
	configureEnquiryLoggingTest(t)
	withEnquiryMailSender(t, func(string, string, string, string, string) error { return nil })

	rr := postEnquiry(t, privacyTestEnquiry("PRIVACY_TEST_UNKNOWN_CHANNEL"))
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}

	assertPrivacySafeEnquiryLog(t, logs.String())
	if !strings.Contains(logs.String(), "unknown channel; treating as marketing") {
		t.Fatalf("expected generic unknown-channel log, got %q", logs.String())
	}
	if strings.Contains(logs.String(), "PRIVACY_TEST_UNKNOWN_CHANNEL") {
		t.Fatalf("unknown channel was reflected in log: %q", logs.String())
	}
}

func TestHandleEnquiryDeliveryFailureLogExcludesCustomerData(t *testing.T) {
	logs := captureEnquiryLogs(t)
	configureEnquiryLoggingTest(t)
	withEnquiryMailSender(t, func(string, string, string, string, string) error {
		return errors.New("delivery failed for privacy-test@example.invalid")
	})

	rr := postEnquiry(t, privacyTestEnquiry("marketing"))
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rr.Code)
	}

	assertPrivacySafeEnquiryLog(t, logs.String())
	if !strings.Contains(logs.String(), "enquiry delivery failed") {
		t.Fatalf("expected generic delivery-failure log, got %q", logs.String())
	}
}

func captureEnquiryLogs(t *testing.T) *bytes.Buffer {
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

func configureEnquiryLoggingTest(t *testing.T) {
	t.Helper()

	oldCfg := cfg
	cfg = appConfig{
		EnquiryTo:      "recipient@example.invalid",
		RequireConsent: true,
	}
	t.Cleanup(func() { cfg = oldCfg })
}

func withEnquiryMailSender(t *testing.T, sender func(string, string, string, string, string) error) {
	t.Helper()

	oldSender := sendEnquiryMail
	sendEnquiryMail = sender
	t.Cleanup(func() { sendEnquiryMail = oldSender })
}

func postEnquiry(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/enquiry", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleEnquiry(rr, req)
	return rr
}

func privacyTestEnquiry(channel string) string {
	return `{
		"name": "PRIVACY_TEST_NAME",
		"email": "privacy-test@example.invalid",
		"phone": "PRIVACY_TEST_PHONE",
		"timeline": "PRIVACY_TEST_TIMELINE",
		"message": "PRIVACY_TEST_MESSAGE",
		"page": "https://example.invalid/contact?secret-query-marker=PRIVACY_TEST_QUERY",
		"source": "PRIVACY_TEST_SOURCE",
		"consent": true,
		"channel": "` + channel + `"
	}`
}

func assertPrivacySafeEnquiryLog(t *testing.T, logs string) {
	t.Helper()

	for _, prohibited := range []string{
		"PRIVACY_TEST_NAME",
		"privacy-test@example.invalid",
		"PRIVACY_TEST_PHONE",
		"PRIVACY_TEST_TIMELINE",
		"PRIVACY_TEST_MESSAGE",
		"https://example.invalid/contact?secret-query-marker=PRIVACY_TEST_QUERY",
		"PRIVACY_TEST_QUERY",
		"PRIVACY_TEST_SOURCE",
	} {
		if strings.Contains(logs, prohibited) {
			t.Fatalf("prohibited customer data %q in log %q", prohibited, logs)
		}
	}
}
