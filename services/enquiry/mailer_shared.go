package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/thejpness/SouthCoastApps/services/_shared/mailer"
)

var (
	enquiryOnce    sync.Once
	enquiryClient  *mailer.Client
	enquiryInitErr error
)

func ensureEnquiryMailer() (*mailer.Client, error) {
	enquiryOnce.Do(func() {
		mode := mailer.TLSModeSTARTTLS
		if cfg.SMTP.Port == 465 {
			mode = mailer.TLSModeTLS
		}

		c, err := mailer.New(mailer.Config{
			Host:        cfg.SMTP.Host,
			Port:        cfg.SMTP.Port,
			User:        cfg.SMTP.User,
			Pass:        cfg.SMTP.Pass,
			From:        cfg.SMTP.From,
			Mode:        mode,
			Timeout:     8 * time.Second,
			IdleTimeout: 60 * time.Second,

			// Helo + MessageIDDomain now default from SMTP_FROM domain (portable across sites)
			// Helo:            "",
			// MessageIDDomain: "",
		})
		if err != nil {
			enquiryInitErr = err
			return
		}
		enquiryClient = c
	})
	return enquiryClient, enquiryInitErr
}

func sendMail(to, subject, plainBody, htmlBody, replyTo string) error {
	c, err := ensureEnquiryMailer()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	t0 := time.Now()
	err = c.Send(ctx, mailer.Message{
		To:      []string{to},
		Subject: subject,
		Text:    plainBody,
		HTML:    htmlBody,
		ReplyTo: replyTo,
	})
	log.Printf("[enquiry] smtp_ms=%d to=%q err=%v", time.Since(t0).Milliseconds(), to, err)

	return err
}
