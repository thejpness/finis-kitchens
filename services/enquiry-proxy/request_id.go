package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"
)

const (
	requestIDHeader = "X-Request-ID"
	requestIDBytes  = 16
)

type proxyRequestObservation struct {
	requestID string
	outcome   string
}

type proxyRequestObservationKey struct{}

type proxyStatusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *proxyStatusRecorder) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *proxyStatusRecorder) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func newRequestID() (string, error) {
	bytes := make([]byte, requestIDBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func isValidRequestID(value string) bool {
	if len(value) != requestIDBytes*2 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func proxyRequestObservationFromContext(ctx context.Context) *proxyRequestObservation {
	observation, _ := ctx.Value(proxyRequestObservationKey{}).(*proxyRequestObservation)
	return observation
}

func proxyRequestIDFromContext(ctx context.Context) string {
	if observation := proxyRequestObservationFromContext(ctx); observation != nil {
		return observation.requestID
	}
	return ""
}

func setProxyOutcome(r *http.Request, outcome string) {
	if observation := proxyRequestObservationFromContext(r.Context()); observation != nil {
		observation.outcome = outcome
	}
}

func proxyOutcomeForStatus(status int) string {
	if status >= http.StatusInternalServerError {
		return "upstream_failed"
	}

	switch status {
	case http.StatusAccepted:
		return "accepted"
	case http.StatusUnsupportedMediaType:
		return "invalid_content_type"
	default:
		return "rejected"
	}
}

func withProxyObservability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/enquiry" {
			next.ServeHTTP(w, r)
			return
		}

		requestID, err := newRequestID()
		if err != nil {
			log.Printf("event=enquiry_request service=enquiry-proxy outcome=request_id_unavailable status=%d", http.StatusInternalServerError)
			httpError(w, http.StatusInternalServerError, "internal error")
			return
		}

		observation := &proxyRequestObservation{requestID: requestID}
		r = r.WithContext(context.WithValue(r.Context(), proxyRequestObservationKey{}, observation))
		w.Header().Set(requestIDHeader, requestID)

		started := time.Now()
		recorder := &proxyStatusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)

		if observation.outcome == "" {
			observation.outcome = proxyOutcomeForStatus(recorder.status)
		}

		log.Printf(
			"event=enquiry_request service=enquiry-proxy request_id=%s outcome=%s status=%d duration_ms=%d",
			observation.requestID,
			observation.outcome,
			recorder.status,
			time.Since(started).Milliseconds(),
		)
	})
}
