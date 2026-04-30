// Package mailer provides an SMTP email client with HTML body, attachments,
// and TLS/STARTTLS support.
//
// Built on stdlib net/smtp + crypto/tls — zero new external dependencies.
package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"time"
)

// Config controls SMTP connection behavior.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string // default From address (overridable per Message)
	UseTLS   bool   // implicit TLS (port 465). false uses STARTTLS.
	Timeout  time.Duration
}

const defaultTimeout = 30 * time.Second

// Mailer sends emails over SMTP.
type Mailer struct {
	cfg    Config
	sendFn sendFunc
}

// sendFunc is the dependency-injection seam used by tests.
type sendFunc func(ctx context.Context, cfg Config, from string, to []string, msg []byte) error

// New constructs a Mailer using the standard SMTP transport.
func New(cfg Config) *Mailer {
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	return &Mailer{cfg: cfg, sendFn: smtpSend}
}

// Send transmits a single Message.
func (m *Mailer) Send(ctx context.Context, msg Message) error {
	if msg.From == "" {
		msg.From = m.cfg.From
	}
	if err := msg.validate(); err != nil {
		return err
	}

	body, err := msg.build()
	if err != nil {
		return fmt.Errorf("mailer: build message: %w", err)
	}

	return m.sendFn(ctx, m.cfg, msg.From, msg.recipients(), body)
}

// SendHTML is a convenience wrapper for a single recipient HTML email.
func (m *Mailer) SendHTML(ctx context.Context, to, subject, htmlBody string) error {
	return m.Send(ctx, Message{
		To:       []string{to},
		Subject:  subject,
		HTMLBody: htmlBody,
	})
}

// smtpSend dials the SMTP server, authenticates, and submits the message.
// Honors ctx for the dial step; subsequent SMTP I/O respects cfg.Timeout.
func smtpSend(ctx context.Context, cfg Config, from string, to []string, msg []byte) error {
	if cfg.Host == "" {
		return errors.New("mailer: SMTP Host is required")
	}
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	dialer := &net.Dialer{Timeout: cfg.Timeout}

	var conn net.Conn
	var err error
	if cfg.UseTLS {
		tlsConfig := &tls.Config{ServerName: cfg.Host}
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("mailer: dial %s: %w", addr, err)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(cfg.Timeout)
	}
	_ = conn.SetDeadline(deadline)

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("mailer: smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if !cfg.UseTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsConfig := &tls.Config{ServerName: cfg.Host}
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("mailer: STARTTLS: %w", err)
			}
		}
	}

	if cfg.Username != "" {
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("mailer: auth: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("mailer: MAIL FROM: %w", err)
	}
	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("mailer: RCPT TO %s: %w", addr, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mailer: DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("mailer: write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mailer: close body: %w", err)
	}

	return client.Quit()
}
