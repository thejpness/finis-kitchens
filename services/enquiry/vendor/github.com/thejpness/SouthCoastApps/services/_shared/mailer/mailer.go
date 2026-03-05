package mailer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"sync"
	"time"
)

type TLSMode string

const (
	TLSModeAuto     TLSMode = "auto"     // STARTTLS if offered; implicit TLS if port 465
	TLSModeSTARTTLS TLSMode = "starttls" // require STARTTLS
	TLSModeTLS      TLSMode = "tls"      // implicit TLS (SMTPS)
	TLSModeNone     TLSMode = "none"     // plain (not recommended)
)

type Config struct {
	Host string
	Port int
	User string
	Pass string
	From string // may be "Name <email@domain>"

	// Helo: optional. If empty, derived from From domain (preferred) or Host.
	Helo string

	// MessageIDDomain controls the domain used in the Message-ID header.
	// If empty, derived from From domain (preferred) or Host.
	MessageIDDomain string

	Mode    TLSMode
	Timeout time.Duration

	// IdleTimeout controls how long we will keep an SMTP connection open
	// between sends before forcing a reconnect. If zero, defaults to 60s.
	IdleTimeout time.Duration

	MinTLSVersion      uint16
	InsecureSkipVerify bool // keep false in prod
}

type Message struct {
	To      []string
	Subject string
	Text    string
	HTML    string
	ReplyTo string
}

type Client struct {
	cfg Config

	mu       sync.Mutex
	cl       *smtp.Client
	conn     net.Conn
	lastUsed time.Time
	authed   bool
}

func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.Host) == "" {
		return nil, errors.New("mailer: Host required")
	}
	if cfg.Port <= 0 {
		return nil, errors.New("mailer: Port required")
	}
	if strings.TrimSpace(cfg.From) == "" {
		return nil, errors.New("mailer: From required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 8 * time.Second
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = 60 * time.Second
	}
	if cfg.MinTLSVersion == 0 {
		cfg.MinTLSVersion = tls.VersionTLS12
	}
	if cfg.Mode == "" {
		cfg.Mode = TLSModeAuto
	}

	// Portable defaults (so moving repos/domains doesn't require code edits).
	if strings.TrimSpace(cfg.Helo) == "" {
		if d := deriveDomainFromFrom(cfg.From); d != "" {
			cfg.Helo = d
		} else {
			cfg.Helo = cfg.Host
		}
	}
	if strings.TrimSpace(cfg.MessageIDDomain) == "" {
		if d := deriveDomainFromFrom(cfg.From); d != "" {
			cfg.MessageIDDomain = d
		} else {
			cfg.MessageIDDomain = cfg.Host
		}
	}

	return &Client{cfg: cfg}, nil
}

// Close closes any open SMTP connection.
// Safe to call multiple times.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeLocked()
	return nil
}

func (c *Client) Send(ctx context.Context, m Message) (retErr error) {
	if len(m.To) == 0 {
		return errors.New("mailer: missing To")
	}
	if strings.TrimSpace(m.Subject) == "" {
		return errors.New("mailer: missing Subject")
	}
	if strings.TrimSpace(m.Text) == "" && strings.TrimSpace(m.HTML) == "" {
		return errors.New("mailer: missing body (Text or HTML)")
	}

	raw, err := buildMIME(c.cfg.From, c.cfg.MessageIDDomain, m)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(c.cfg.Timeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnectedLocked(deadline); err != nil {
		return err
	}

	// Ensure clean envelope state on a reused connection.
	// If Reset fails, force reconnect.
	if err := c.cl.Reset(); err != nil {
		c.closeLocked()
		if err2 := c.ensureConnectedLocked(deadline); err2 != nil {
			return err2
		}
	}

	// If anything fails mid-flight, close the connection so the next send re-dials cleanly.
	defer func() {
		if retErr != nil {
			c.closeLocked()
		}
	}()

	fromEmail, err := parseEmailAddress(c.cfg.From)
	if err != nil {
		return fmt.Errorf("mailer: from parse: %w", err)
	}

	if err := c.cl.Mail(fromEmail); err != nil {
		return fmt.Errorf("mailer: mail from: %w", err)
	}

	for _, rcpt := range m.To {
		rcptEmail, err := parseEmailAddress(rcpt)
		if err != nil {
			return fmt.Errorf("mailer: rcpt parse: %w", err)
		}
		if err := c.cl.Rcpt(rcptEmail); err != nil {
			return fmt.Errorf("mailer: rcpt to: %w", err)
		}
	}

	w, err := c.cl.Data()
	if err != nil {
		return fmt.Errorf("mailer: data: %w", err)
	}
	if _, err := w.Write([]byte(raw)); err != nil {
		_ = w.Close()
		return fmt.Errorf("mailer: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mailer: data close: %w", err)
	}

	c.lastUsed = time.Now()
	return nil
}

func (c *Client) ensureConnectedLocked(deadline time.Time) error {
	now := time.Now()

	// If we have an existing connection, check if it's too old/idle.
	if c.cl != nil && c.conn != nil {
		idle := now.Sub(c.lastUsed)
		if idle > c.cfg.IdleTimeout {
			c.closeLocked()
		} else {
			_ = c.conn.SetDeadline(deadline)

			// Lightweight liveness check only after some idle time.
			if idle > 15*time.Second {
				if err := c.cl.Noop(); err != nil {
					c.closeLocked()
				}
			}
		}
	}

	// Need to (re)connect?
	if c.cl == nil || c.conn == nil {
		cl, conn, err := c.dialSMTP(deadline)
		if err != nil {
			return err
		}
		c.cl = cl
		c.conn = conn
		c.lastUsed = now
		c.authed = false

		// Optional HELO/EHLO override (we default it in New()).
		if strings.TrimSpace(c.cfg.Helo) != "" {
			if err := c.cl.Hello(sanitizeHeaderValue(c.cfg.Helo)); err != nil {
				c.closeLocked()
				return fmt.Errorf("mailer: helo: %w", err)
			}
		}

		// TLS upgrade if needed/possible
		if _, err := c.maybeTLS(c.cl); err != nil {
			c.closeLocked()
			return err
		}

		// Auth after TLS
		if strings.TrimSpace(c.cfg.User) != "" {
			auth := smtp.PlainAuth("", c.cfg.User, c.cfg.Pass, c.cfg.Host)
			if err := c.cl.Auth(auth); err != nil {
				c.closeLocked()
				return fmt.Errorf("mailer: auth: %w", err)
			}
			c.authed = true
		}
	}

	return nil
}

func (c *Client) closeLocked() {
	if c.cl != nil {
		_ = c.cl.Close()
		c.cl = nil
	}
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.authed = false
}

func (c *Client) dialSMTP(deadline time.Time) (*smtp.Client, net.Conn, error) {
	addr := net.JoinHostPort(c.cfg.Host, fmt.Sprintf("%d", c.cfg.Port))

	timeout := time.Until(deadline)
	if timeout <= 0 {
		return nil, nil, context.DeadlineExceeded
	}
	d := &net.Dialer{Timeout: timeout}

	mode := c.cfg.Mode
	if mode == TLSModeAuto && c.cfg.Port == 465 {
		mode = TLSModeTLS
	}

	if mode == TLSModeTLS {
		tlsCfg := &tls.Config{
			ServerName:         c.cfg.Host,
			MinVersion:         c.cfg.MinTLSVersion,
			InsecureSkipVerify: c.cfg.InsecureSkipVerify,
		}
		conn, err := tls.DialWithDialer(d, "tcp", addr, tlsCfg)
		if err != nil {
			return nil, nil, fmt.Errorf("mailer: tls dial: %w", err)
		}
		_ = conn.SetDeadline(deadline)

		cl, err := smtp.NewClient(conn, c.cfg.Host)
		if err != nil {
			_ = conn.Close()
			return nil, nil, fmt.Errorf("mailer: smtp client: %w", err)
		}
		return cl, conn, nil
	}

	conn, err := d.Dial("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("mailer: dial: %w", err)
	}
	_ = conn.SetDeadline(deadline)

	cl, err := smtp.NewClient(conn, c.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("mailer: smtp client: %w", err)
	}

	return cl, conn, nil
}

func (c *Client) maybeTLS(cl *smtp.Client) (bool, error) {
	mode := c.cfg.Mode
	if mode == TLSModeAuto && c.cfg.Port == 465 {
		// Already TLS in dialSMTP()
		return true, nil
	}
	if mode == TLSModeNone || mode == TLSModeTLS {
		return mode == TLSModeTLS, nil
	}

	ok, _ := cl.Extension("STARTTLS")
	if !ok {
		if mode == TLSModeSTARTTLS {
			return false, errors.New("mailer: STARTTLS required but not supported by server")
		}
		// Auto: proceed without TLS
		return false, nil
	}

	tlsCfg := &tls.Config{
		ServerName:         c.cfg.Host,
		MinVersion:         c.cfg.MinTLSVersion,
		InsecureSkipVerify: c.cfg.InsecureSkipVerify,
	}
	if err := cl.StartTLS(tlsCfg); err != nil {
		return false, fmt.Errorf("mailer: starttls: %w", err)
	}
	return true, nil
}

func buildMIME(from string, msgIDDomain string, m Message) (string, error) {
	fromAddr, err := mail.ParseAddress(sanitizeHeaderValue(from))
	if err != nil {
		return "", fmt.Errorf("mailer: invalid From: %w", err)
	}

	var toAddrs []*mail.Address
	for _, t := range m.To {
		a, err := mail.ParseAddress(sanitizeHeaderValue(t))
		if err != nil {
			return "", fmt.Errorf("mailer: invalid To: %w", err)
		}
		toAddrs = append(toAddrs, a)
	}

	subj := encodeHeader(sanitizeHeaderValue(m.Subject))

	replyTo := ""
	if strings.TrimSpace(m.ReplyTo) != "" {
		ra, err := mail.ParseAddress(sanitizeHeaderValue(m.ReplyTo))
		if err != nil {
			return "", fmt.Errorf("mailer: invalid Reply-To: %w", err)
		}
		replyTo = ra.String()
	}

	msgIDDomain = strings.TrimSpace(msgIDDomain)
	if msgIDDomain == "" {
		msgIDDomain = deriveDomainFromFrom(from)
	}
	if msgIDDomain == "" {
		msgIDDomain = "localhost"
	}

	msgID := fmt.Sprintf("<%s.%d@%s>", randHex(6), time.Now().UnixNano(), msgIDDomain)
	date := time.Now().Format(time.RFC1123Z)

	// Normalize body newlines to CRLF (SMTP-friendly) before QP encoding.
	text := normalizeNewlines(m.Text)
	html := normalizeNewlines(m.HTML)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("From: %s\r\n", fromAddr.String()))
	b.WriteString(fmt.Sprintf("To: %s\r\n", joinAddresses(toAddrs)))
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", subj))
	b.WriteString(fmt.Sprintf("Date: %s\r\n", date))
	b.WriteString(fmt.Sprintf("Message-ID: %s\r\n", msgID))
	b.WriteString("MIME-Version: 1.0\r\n")
	if replyTo != "" {
		b.WriteString(fmt.Sprintf("Reply-To: %s\r\n", replyTo))
	}

	// Single-part (text only)
	if strings.TrimSpace(html) == "" {
		b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		b.WriteString("\r\n")
		b.WriteString(encodeQP(text))
		return b.String(), nil
	}

	// multipart/alternative
	boundary := "bnd-" + randHex(10)
	b.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n", boundary))
	b.WriteString("\r\n")

	// Plain part
	b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	b.WriteString("\r\n")
	b.WriteString(encodeQP(text))
	b.WriteString("\r\n")

	// HTML part
	b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	b.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	b.WriteString("\r\n")
	b.WriteString(encodeQP(html))
	b.WriteString("\r\n")

	b.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	return b.String(), nil
}

func joinAddresses(addrs []*mail.Address) string {
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, a.String())
	}
	return strings.Join(out, ", ")
}

func parseEmailAddress(s string) (string, error) {
	a, err := mail.ParseAddress(sanitizeHeaderValue(s))
	if err != nil {
		return "", err
	}
	return a.Address, nil
}

func sanitizeHeaderValue(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return strings.TrimSpace(s)
}

func encodeHeader(s string) string {
	return mime.QEncoding.Encode("utf-8", s)
}

func encodeQP(s string) string {
	var buf bytes.Buffer
	w := quotedprintable.NewWriter(&buf)
	_, _ = w.Write([]byte(s))
	_ = w.Close()
	// Do NOT rewrite newlines here (QP writer already handles CRLF for soft breaks).
	return buf.String()
}

func randHex(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func deriveDomainFromFrom(from string) string {
	a, err := mail.ParseAddress(sanitizeHeaderValue(from))
	if err != nil {
		return ""
	}
	at := strings.LastIndex(a.Address, "@")
	if at == -1 || at == len(a.Address)-1 {
		return ""
	}
	return a.Address[at+1:]
}

func normalizeNewlines(s string) string {
	// Convert any \r\n or \r to \n, then convert \n -> \r\n.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}
