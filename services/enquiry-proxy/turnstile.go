package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type turnstileResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

var turnstileHTTPClient = &http.Client{
	Timeout: 5 * time.Second,
}

func verifyTurnstile(ctx context.Context, secret, token string) error {
	secret = strings.TrimSpace(secret)
	token = strings.TrimSpace(token)

	if secret == "" {
		return fmt.Errorf("turnstile not configured")
	}
	if token == "" {
		return fmt.Errorf("missing captcha token")
	}

	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", token)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://challenges.cloudflare.com/turnstile/v0/siteverify",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := turnstileHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("turnstile request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("turnstile http %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var tr turnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return fmt.Errorf("decode turnstile response: %w", err)
	}
	if !tr.Success {
		return fmt.Errorf("turnstile verification failed: %v", tr.ErrorCodes)
	}
	return nil
}
