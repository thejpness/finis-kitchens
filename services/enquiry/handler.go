package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	emailRx  = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	nameTrim = regexp.MustCompile(`\s+`)
)

type enquiry struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Phone        string `json:"phone,omitempty"`
	Message      string `json:"message"`
	Page         string `json:"page,omitempty"`
	Source       string `json:"source,omitempty"`
	Consent      *bool  `json:"consent,omitempty"`
	HoneyPot     string `json:"company,omitempty"`
	CaptchaToken string `json:"captchaToken,omitempty"` // ignored
	Channel      string `json:"channel,omitempty"`      // "marketing" | "portal" | "support"
}

type apiError struct {
	Error string `json:"error"`
}

func handleEnquiry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		httpError(w, http.StatusUnsupportedMediaType, "use application/json")
		return
	}

	var in enquiry
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		httpError(w, http.StatusBadRequest, "invalid json")
		return
	}
	// Ensure there isn't trailing junk / extra JSON values
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		httpError(w, http.StatusBadRequest, "invalid json")
		return
	}

	// Honeypot: pretend success but do nothing
	if in.HoneyPot != "" {
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		return
	}

	channel := strings.ToLower(strings.TrimSpace(in.Channel))
	if channel == "" {
		channel = "marketing"
	}

	requireConsent := cfg.RequireConsent
	if channel == "support" || channel == "portal" {
		requireConsent = false
	}

	if err := validate(&in, requireConsent); err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}

	to := cfg.EnquiryTo
	subjectPrefix := strings.TrimSpace(cfg.SubjectPrefix)
	if subjectPrefix == "" {
		subjectPrefix = "[Enquiry]"
	}

	switch channel {
	case "support":
		if cfg.SupportTo != "" {
			to = cfg.SupportTo
		}
		subjectPrefix = "[Support]"
	case "portal":
		subjectPrefix = "[Portal]"
	case "marketing":
		// keep configured prefix
	default:
		log.Printf("unknown channel %q, treating as marketing", channel)
		channel = "marketing"
	}

	subject := strings.TrimSpace(fmt.Sprintf("%s New %s message from %s",
		subjectPrefix, channel, safe(in.Name)))

	now := time.Now().UTC().Format(time.RFC3339)

	plain := fmt.Sprintf(
		"New %s message\n\nChannel: %s\nName: %s\nEmail: %s\nPhone: %s\nPage: %s\nSource: %s\nConsent: %v\nTime: %s\n\nMessage:\n%s\n",
		channel,
		channel,
		safe(in.Name), in.Email, safe(in.Phone), safe(in.Page), safe(in.Source),
		boolOr(in.Consent, false), now, in.Message,
	)

	htmlBody := fmt.Sprintf(`
<h2>New %s message</h2>
<p><strong>Channel:</strong> %s<br>
   <strong>Name:</strong> %s<br>
   <strong>Email:</strong> %s<br>
   <strong>Phone:</strong> %s<br>
   <strong>Page:</strong> %s<br>
   <strong>Source:</strong> %s<br>
   <strong>Consent:</strong> %v<br>
   <strong>Time:</strong> %s</p>
<p><strong>Message:</strong><br>%s</p>`,
		esc(channel),
		esc(channel),
		esc(safe(in.Name)), esc(in.Email), esc(safe(in.Phone)), esc(safe(in.Page)),
		esc(safe(in.Source)), boolOr(in.Consent, false),
		esc(now), nl2br(esc(in.Message)),
	)

	if err := sendMail(to, subject, plain, htmlBody, in.Email); err != nil {
		log.Printf("sendMail error: %v", err)
		httpError(w, http.StatusBadGateway, "failed to send")
		return
	}

	log.Printf("enquiry accepted channel=%q name=%q email=%q page=%q source=%q",
		channel, safe(in.Name), in.Email, safe(in.Page), safe(in.Source))

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
}

func validate(in *enquiry, requireConsent bool) error {
	in.Name = strings.TrimSpace(nameTrim.ReplaceAllString(in.Name, " "))
	in.Email = strings.TrimSpace(in.Email)
	in.Message = strings.TrimSpace(in.Message)

	if in.Name == "" || len(in.Name) < 2 {
		return errors.New("name is required")
	}
	if !emailRx.MatchString(in.Email) {
		return errors.New("valid email required")
	}
	if len(in.Message) < 10 {
		return errors.New("message is too short")
	}
	if requireConsent && (in.Consent == nil || !*in.Consent) {
		return errors.New("consent is required")
	}
	return nil
}
