package main

import (
	"encoding/json"
	"html"
	"net/http"
	"os"
	"strings"
)

func withCommonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && isAllowedOrigin(origin, cfg.AllowOrigin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")

			reqHdrs := r.Header.Get("Access-Control-Request-Headers")
			if strings.TrimSpace(reqHdrs) != "" {
				w.Header().Set("Access-Control-Allow-Headers", reqHdrs)
				w.Header().Add("Vary", "Access-Control-Request-Headers")
			} else {
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			}

			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Max-Age", "600")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

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

func httpError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(apiError{Error: msg})
}

func nl2br(s string) string {
	// tolerate CRLF input
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\n", "<br>")
}

func esc(s string) string {
	return html.EscapeString(s)
}

func boolOr(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func safe(s string) string {
	return strings.TrimSpace(s)
}

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}
