package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type smtpConfig struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

type appConfig struct {
	AllowOrigin string

	SubjectPrefix  string
	RequireConsent bool

	EnquiryTo string
	SupportTo string

	InternalEnquirySecret string

	SMTP smtpConfig
}

func loadConfig() (appConfig, error) {
	allowOrigin := os.Getenv("ALLOW_ORIGIN")
	subjectPrefix := os.Getenv("SUBJECT_PREFIX")
	requireConsent := strings.EqualFold(os.Getenv("REQUIRE_CONSENT"), "true")

	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if host == "" {
		return appConfig{}, errors.New("SMTP_HOST is required")
	}

	port := 587
	if ps := strings.TrimSpace(os.Getenv("SMTP_PORT")); ps != "" {
		p, err := strconv.Atoi(ps)
		if err != nil {
			return appConfig{}, fmt.Errorf("invalid SMTP_PORT %q: %w", ps, err)
		}
		port = p
	}

	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		return appConfig{}, errors.New("SMTP_FROM is required")
	}

	enquiryTo := strings.TrimSpace(os.Getenv("ENQUIRY_TO"))
	if enquiryTo == "" {
		enquiryTo = strings.TrimSpace(os.Getenv("SMTP_TO"))
		if enquiryTo != "" {
			log.Printf("using SMTP_TO as ENQUIRY_TO (legacy env var)")
		}
	}
	if enquiryTo == "" {
		return appConfig{}, errors.New("ENQUIRY_TO (or SMTP_TO) is required")
	}

	supportTo := strings.TrimSpace(os.Getenv("SUPPORT_TO"))
	if supportTo == "" {
		log.Printf("SUPPORT_TO not set – support messages will be sent to ENQUIRY_TO")
	}

	user, err := getSecret("SMTP_USER")
	if err != nil {
		return appConfig{}, err
	}
	pass, err := getSecret("SMTP_PASS")
	if err != nil {
		return appConfig{}, err
	}

	internalSecret, err := getSecret("INTERNAL_ENQUIRY_SECRET")
	if err != nil {
		return appConfig{}, err
	}

	return appConfig{
		AllowOrigin:           allowOrigin,
		SubjectPrefix:         subjectPrefix,
		RequireConsent:        requireConsent,
		EnquiryTo:             enquiryTo,
		SupportTo:             supportTo,
		InternalEnquirySecret: internalSecret,
		SMTP: smtpConfig{
			Host: host,
			Port: port,
			User: user,
			Pass: pass,
			From: from,
		},
	}, nil
}

func getSecret(name string) (string, error) {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v, nil
	}

	if path := strings.TrimSpace(os.Getenv(name + "_FILE")); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("reading secret file %s for %s: %w", path, name, err)
		}
		v := strings.TrimSpace(string(b))
		if v == "" {
			return "", fmt.Errorf("secret %s from file %s is empty", name, path)
		}
		return v, nil
	}

	return "", fmt.Errorf("%s not set (no %s or %s_FILE)", name, name, name)
}
