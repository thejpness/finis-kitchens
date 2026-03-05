package main

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

const internalSecretHeader = "X-Internal-Secret"

func requireInternalSecret(expected string, next http.Handler) http.Handler {
	expected = strings.TrimSpace(expected)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get(internalSecretHeader)

		if expected == "" || got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
			httpError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}
