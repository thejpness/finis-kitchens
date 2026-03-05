package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const internalSecretHeader = "X-Internal-Secret"

// Mirror the upstream schema so DisallowUnknownFields actually does something.
type forwardedEnquiry struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Phone        string `json:"phone,omitempty"`
	Message      string `json:"message"`
	Page         string `json:"page,omitempty"`
	Source       string `json:"source,omitempty"`
	Consent      *bool  `json:"consent,omitempty"`
	HoneyPot     string `json:"company,omitempty"`
	CaptchaToken string `json:"captchaToken,omitempty"`
	Channel      string `json:"channel,omitempty"`
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	upstreamClient := &http.Client{
		Timeout: cfg.UpstreamTimeout,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("POST /api/enquiry", func(w http.ResponseWriter, r *http.Request) {
		// Enforce JSON
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			httpError(w, http.StatusUnsupportedMediaType, "use application/json")
			return
		}

		// Read capped body
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, cfg.MaxBodyBytes))
		if err != nil {
			httpError(w, http.StatusBadRequest, "bad request")
			return
		}

		// Decode strictly into struct
		var payload forwardedEnquiry
		dec := json.NewDecoder(bytes.NewReader(body))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&payload); err != nil {
			httpError(w, http.StatusBadRequest, "invalid json")
			return
		}
		// Ensure there isn't trailing junk / extra JSON values
		var extra any
		if err := dec.Decode(&extra); err != io.EOF {
			httpError(w, http.StatusBadRequest, "invalid json")
			return
		}

		// Verify Turnstile (proxy owns it)
		token := strings.TrimSpace(payload.CaptchaToken)
		if err := verifyTurnstile(r.Context(), cfg.TurnstileSecret, token); err != nil {
			log.Printf("captcha failed: %v", err)
			httpError(w, http.StatusBadRequest, "captcha verification failed")
			return
		}

		// Strip captcha before forwarding
		payload.CaptchaToken = ""

		fwd, err := json.Marshal(payload)
		if err != nil {
			httpError(w, http.StatusBadRequest, "invalid payload")
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, cfg.UpstreamURL, bytes.NewReader(fwd))
		if err != nil {
			httpError(w, http.StatusBadGateway, "upstream error")
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set(internalSecretHeader, cfg.InternalSecret)

		resp, err := upstreamClient.Do(req)
		if err != nil {
			httpError(w, http.StatusBadGateway, "upstream error")
			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	})

	addr := cfg.ListenAddr
	srv := &http.Server{
		Addr:              addr,
		Handler:           withProxyHeaders(cfg, mux),
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	log.Printf("enquiry-proxy listening on %s", addr)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}

type apiError struct {
	Error string `json:"error"`
}

func httpError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(apiError{Error: msg})
}
