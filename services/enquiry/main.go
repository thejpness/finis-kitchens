package main

import (
	"errors"
	"log"
	"net/http"
	"time"
)

var cfg appConfig

func main() {
	var err error
	cfg, err = loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	mux := http.NewServeMux()

	// Enquiries must come through proxy (internal secret required)
	mux.Handle("POST /api/enquiry", requireInternalSecret(cfg.InternalEnquirySecret, http.HandlerFunc(handleEnquiry)))

	// Health stays open
	mux.HandleFunc("GET /healthz", healthHandler)

	addr := env("ADDR", ":8080")

	srv := &http.Server{
		Addr:              addr,
		Handler:           withCommonHeaders(mux),
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	log.Printf("enquiry API listening on %s", addr)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
