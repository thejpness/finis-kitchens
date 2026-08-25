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

type enquiryRequestObservation struct {
	requestID string
	outcome   string
	channel   string
}

type enquiryRequestObservationKey struct{}

type enquiryStatusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *enquiryStatusRecorder) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *enquiryStatusRecorder) Write(body []byte) (int, error) {
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

func enquiryRequestObservationFromContext(ctx context.Context) *enquiryRequestObservation {
	observation, _ := ctx.Value(enquiryRequestObservationKey{}).(*enquiryRequestObservation)
	return observation
}

func enquiryRequestIDFromContext(ctx context.Context) string {
	if observation := enquiryRequestObservationFromContext(ctx); observation != nil {
		return observation.requestID
	}
	return ""
}

func setEnquiryOutcome(r *http.Request, outcome string) {
	if observation := enquiryRequestObservationFromContext(r.Context()); observation != nil {
		observation.outcome = outcome
	}
}

func setEnquiryChannel(r *http.Request, channel string) {
	if observation := enquiryRequestObservationFromContext(r.Context()); observation != nil {
		observation.channel = channel
	}
}

func enquiryOutcomeForStatus(status int) string {
	if status == http.StatusAccepted {
		return "accepted"
	}
	return "rejected"
}

func withEnquiryObservability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/enquiry" {
			next.ServeHTTP(w, r)
			return
		}

		requestID := r.Header.Get(requestIDHeader)
		if !isValidRequestID(requestID) {
			var err error
			requestID, err = newRequestID()
			if err != nil {
				log.Printf("event=enquiry_request service=enquiry outcome=request_id_unavailable status=%d", http.StatusInternalServerError)
				httpError(w, http.StatusInternalServerError, "internal error")
				return
			}
		}

		observation := &enquiryRequestObservation{requestID: requestID}
		r = r.WithContext(context.WithValue(r.Context(), enquiryRequestObservationKey{}, observation))
		w.Header().Set(requestIDHeader, requestID)

		started := time.Now()
		recorder := &enquiryStatusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)

		if observation.outcome == "" {
			observation.outcome = enquiryOutcomeForStatus(recorder.status)
		}

		if observation.channel == "" {
			log.Printf(
				"event=enquiry_request service=enquiry request_id=%s outcome=%s status=%d duration_ms=%d",
				observation.requestID,
				observation.outcome,
				recorder.status,
				time.Since(started).Milliseconds(),
			)
			return
		}

		log.Printf(
			"event=enquiry_request service=enquiry request_id=%s outcome=%s status=%d channel=%q duration_ms=%d",
			observation.requestID,
			observation.outcome,
			recorder.status,
			observation.channel,
			time.Since(started).Milliseconds(),
		)
	})
}
