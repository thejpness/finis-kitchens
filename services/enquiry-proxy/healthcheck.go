package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"
)

// runHealthcheck lets the distroless runtime image health-check itself without
// adding a shell or HTTP client package to the image.
func runHealthcheck() int {
	url := strings.TrimSpace(os.Getenv("HEALTHCHECK_URL"))
	if url == "" {
		url = "http://127.0.0.1:8080/healthz"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 1
	}
	resp, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
