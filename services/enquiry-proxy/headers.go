package main

import (
	"net/http"
	"strings"
)

func withProxyHeaders(cfg proxyConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS: browser hygiene (not a security boundary)
		origin := r.Header.Get("Origin")
		if origin != "" && isAllowedOrigin(origin, cfg.AllowOrigin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")

			// Echo requested headers if present, otherwise default.
			reqHdrs := r.Header.Get("Access-Control-Request-Headers")
			if strings.TrimSpace(reqHdrs) != "" {
				w.Header().Set("Access-Control-Allow-Headers", reqHdrs)
				w.Header().Add("Vary", "Access-Control-Request-Headers")
			} else {
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			}

			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Max-Age", "600") // cache preflight 10m
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Basic API hygiene headers
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")

		next.ServeHTTP(w, r)
	})
}

func isAllowedOrigin(origin, allowList string) bool {
	if strings.TrimSpace(allowList) == "" {
		return false
	}
	for _, o := range strings.Split(allowList, ",") {
		if strings.TrimSpace(o) == origin {
			return true
		}
	}
	return false
}
