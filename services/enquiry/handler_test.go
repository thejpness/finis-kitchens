package main

import (
	"bytes"
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
