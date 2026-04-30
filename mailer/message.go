package mailer

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"strings"
	"time"
)

// Message describes an email to send.
//
// At least one of HTMLBody or TextBody must be non-empty. If both are
// provided, the message is sent as multipart/alternative.
type Message struct {
	From        string
	To          []string
	CC          []string
	BCC         []string
	ReplyTo     string
	Subject     string
	TextBody    string
	HTMLBody    string
	Attachments []Attachment
	Headers     map[string]string
}

// Attachment is a file attached to a Message.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// recipients returns To+CC+BCC combined for the SMTP RCPT TO step.
func (m *Message) recipients() []string {
	out := make([]string, 0, len(m.To)+len(m.CC)+len(m.BCC))
	out = append(out, m.To...)
	out = append(out, m.CC...)
	out = append(out, m.BCC...)
	return out
}

// validate checks invariants. Returns nil if the message is sendable.
func (m *Message) validate() error {
	if m.From == "" {
		return errors.New("mailer: From address is required")
	}
	if len(m.To) == 0 {
		return errors.New("mailer: at least one To recipient is required")
	}
	if m.HTMLBody == "" && m.TextBody == "" {
		return errors.New("mailer: HTMLBody or TextBody is required")
	}
	return nil
}

// build assembles the RFC 5322 + MIME message bytes ready for SMTP DATA.
// BCC recipients are intentionally not added to the headers (they remain
// hidden in the SMTP envelope only).
func (m *Message) build() ([]byte, error) {
	var buf bytes.Buffer

	headers := textproto.MIMEHeader{}
	headers.Set("From", m.From)
	headers.Set("To", strings.Join(m.To, ", "))
	if len(m.CC) > 0 {
		headers.Set("Cc", strings.Join(m.CC, ", "))
	}
	if m.ReplyTo != "" {
		headers.Set("Reply-To", m.ReplyTo)
	}
	headers.Set("Subject", encodeSubject(m.Subject))
	headers.Set("Date", time.Now().UTC().Format(time.RFC1123Z))
	headers.Set("MIME-Version", "1.0")
	for k, v := range m.Headers {
		headers.Set(k, v)
	}

	hasAttachments := len(m.Attachments) > 0
	hasBoth := m.HTMLBody != "" && m.TextBody != ""

	switch {
	case hasAttachments:
		mixed := multipart.NewWriter(&buf)
		headers.Set("Content-Type", `multipart/mixed; boundary="`+mixed.Boundary()+`"`)
		writeHeaders(&buf, headers)

		if err := writeBodyPart(mixed, m, hasBoth); err != nil {
			return nil, err
		}
		for _, a := range m.Attachments {
			if err := writeAttachment(mixed, a); err != nil {
				return nil, err
			}
		}
		if err := mixed.Close(); err != nil {
			return nil, err
		}

	case hasBoth:
		alt := multipart.NewWriter(&buf)
		headers.Set("Content-Type", `multipart/alternative; boundary="`+alt.Boundary()+`"`)
		writeHeaders(&buf, headers)

		if err := writeAlternative(alt, m); err != nil {
			return nil, err
		}
		if err := alt.Close(); err != nil {
			return nil, err
		}

	default:
		// single-part: text OR html
		if m.HTMLBody != "" {
			headers.Set("Content-Type", "text/html; charset=UTF-8")
			headers.Set("Content-Transfer-Encoding", "quoted-printable")
			writeHeaders(&buf, headers)
			if err := writeQuotedPrintable(&buf, m.HTMLBody); err != nil {
				return nil, err
			}
		} else {
			headers.Set("Content-Type", "text/plain; charset=UTF-8")
			headers.Set("Content-Transfer-Encoding", "quoted-printable")
			writeHeaders(&buf, headers)
			if err := writeQuotedPrintable(&buf, m.TextBody); err != nil {
				return nil, err
			}
		}
	}

	return buf.Bytes(), nil
}

func writeHeaders(buf *bytes.Buffer, h textproto.MIMEHeader) {
	for k, vs := range h {
		for _, v := range vs {
			fmt.Fprintf(buf, "%s: %s\r\n", k, v)
		}
	}
	buf.WriteString("\r\n")
}

func writeBodyPart(mw *multipart.Writer, m *Message, hasBoth bool) error {
	if hasBoth {
		nested := textproto.MIMEHeader{}
		boundary, err := newBoundary()
		if err != nil {
			return err
		}
		nested.Set("Content-Type", `multipart/alternative; boundary="`+boundary+`"`)
		w, err := mw.CreatePart(nested)
		if err != nil {
			return err
		}
		alt := multipart.NewWriter(w)
		if err := alt.SetBoundary(boundary); err != nil {
			return err
		}
		if err := writeAlternative(alt, m); err != nil {
			return err
		}
		return alt.Close()
	}

	header := textproto.MIMEHeader{}
	if m.HTMLBody != "" {
		header.Set("Content-Type", "text/html; charset=UTF-8")
		header.Set("Content-Transfer-Encoding", "quoted-printable")
		w, err := mw.CreatePart(header)
		if err != nil {
			return err
		}
		return writeQuotedPrintable(w, m.HTMLBody)
	}
	header.Set("Content-Type", "text/plain; charset=UTF-8")
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	w, err := mw.CreatePart(header)
	if err != nil {
		return err
	}
	return writeQuotedPrintable(w, m.TextBody)
}

func writeAlternative(alt *multipart.Writer, m *Message) error {
	// RFC 2046: prefer the richer (HTML) representation; place text/plain first.
	if m.TextBody != "" {
		if err := writePart(alt, "text/plain; charset=UTF-8", m.TextBody); err != nil {
			return err
		}
	}
	if m.HTMLBody != "" {
		if err := writePart(alt, "text/html; charset=UTF-8", m.HTMLBody); err != nil {
			return err
		}
	}
	return nil
}

func writePart(mw *multipart.Writer, contentType, body string) error {
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", contentType)
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	w, err := mw.CreatePart(header)
	if err != nil {
		return err
	}
	return writeQuotedPrintable(w, body)
}

func writeAttachment(mw *multipart.Writer, a Attachment) error {
	contentType := a.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", contentType+`; name="`+a.Filename+`"`)
	header.Set("Content-Disposition", `attachment; filename="`+a.Filename+`"`)
	header.Set("Content-Transfer-Encoding", "base64")
	w, err := mw.CreatePart(header)
	if err != nil {
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(a.Data)
	// 76-char lines per RFC 2045
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		if _, err := io.WriteString(w, encoded[i:end]+"\r\n"); err != nil {
			return err
		}
	}
	return nil
}

func writeQuotedPrintable(w io.Writer, s string) error {
	enc := quotedprintable.NewWriter(w)
	if _, err := enc.Write([]byte(s)); err != nil {
		return err
	}
	return enc.Close()
}

// newBoundary returns a random MIME boundary string.
func newBoundary() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func encodeSubject(subject string) string {
	if isASCII(subject) {
		return subject
	}
	return mime.QEncoding.Encode("UTF-8", subject)
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}
