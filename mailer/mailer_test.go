package mailer

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"
)

// fakeSender captures the send call for assertions.
type fakeSender struct {
	from string
	to   []string
	body []byte
	err  error
}

func (s *fakeSender) send(ctx context.Context, cfg Config, from string, to []string, msg []byte) error {
	s.from = from
	s.to = append([]string(nil), to...)
	s.body = append([]byte(nil), msg...)
	return s.err
}

func newTestMailer(s *fakeSender) *Mailer {
	return &Mailer{
		cfg:    Config{From: "noreply@example.com"},
		sendFn: s.send,
	}
}

// ---------- validation ----------

func TestSend_RequiresFrom(t *testing.T) {
	m := &Mailer{cfg: Config{}, sendFn: (&fakeSender{}).send}
	err := m.Send(context.Background(), Message{To: []string{"x@x.com"}, HTMLBody: "<p>x</p>"})
	if err == nil {
		t.Fatal("expected error when From is empty")
	}
}

func TestSend_RequiresTo(t *testing.T) {
	m := newTestMailer(&fakeSender{})
	err := m.Send(context.Background(), Message{HTMLBody: "<p>x</p>"})
	if err == nil {
		t.Fatal("expected error when To is empty")
	}
}

func TestSend_RequiresBody(t *testing.T) {
	m := newTestMailer(&fakeSender{})
	err := m.Send(context.Background(), Message{To: []string{"x@x.com"}})
	if err == nil {
		t.Fatal("expected error when both bodies empty")
	}
}

// ---------- envelope ----------

func TestSend_DefaultFromUsedWhenMessageFromEmpty(t *testing.T) {
	s := &fakeSender{}
	m := newTestMailer(s)

	if err := m.Send(context.Background(), Message{
		To:       []string{"a@x.com"},
		Subject:  "Hi",
		HTMLBody: "<p>hello</p>",
	}); err != nil {
		t.Fatal(err)
	}
	if s.from != "noreply@example.com" {
		t.Errorf("from = %q, want default", s.from)
	}
}

func TestSend_RecipientsIncludeBCCInEnvelope(t *testing.T) {
	s := &fakeSender{}
	m := newTestMailer(s)

	if err := m.Send(context.Background(), Message{
		To:       []string{"a@x.com"},
		CC:       []string{"b@x.com"},
		BCC:      []string{"c@x.com"},
		Subject:  "x",
		HTMLBody: "<p>x</p>",
	}); err != nil {
		t.Fatal(err)
	}
	want := []string{"a@x.com", "b@x.com", "c@x.com"}
	if len(s.to) != 3 {
		t.Fatalf("recipients = %v, want %v", s.to, want)
	}
	for i, r := range want {
		if s.to[i] != r {
			t.Errorf("recipient[%d] = %q, want %q", i, s.to[i], r)
		}
	}
}

// ---------- body composition ----------

func TestSend_HTMLOnly_SinglePartHeaders(t *testing.T) {
	s := &fakeSender{}
	m := newTestMailer(s)
	if err := m.Send(context.Background(), Message{
		To:       []string{"a@x.com"},
		Subject:  "Test",
		HTMLBody: "<p>Hello</p>",
	}); err != nil {
		t.Fatal(err)
	}

	headers, body := splitHeaderBody(t, s.body)
	if got := headers.Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Errorf("Content-Type = %q, want text/html…", got)
	}
	if got := headers.Get("Content-Transfer-Encoding"); got != "quoted-printable" {
		t.Errorf("CTE = %q", got)
	}
	if !strings.Contains(body, "Hello") {
		t.Errorf("body missing 'Hello': %q", body)
	}
}

func TestSend_BothBodies_MultipartAlternative(t *testing.T) {
	s := &fakeSender{}
	m := newTestMailer(s)
	if err := m.Send(context.Background(), Message{
		To:       []string{"a@x.com"},
		Subject:  "T",
		TextBody: "plain text",
		HTMLBody: "<b>html</b>",
	}); err != nil {
		t.Fatal(err)
	}

	headers, body := splitHeaderBody(t, s.body)
	mediaType, params, err := mime.ParseMediaType(headers.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	if mediaType != "multipart/alternative" {
		t.Errorf("media type = %q", mediaType)
	}

	parts := readParts(t, body, params["boundary"])
	if len(parts) != 2 {
		t.Fatalf("got %d parts, want 2", len(parts))
	}
	if !strings.HasPrefix(parts[0].contentType, "text/plain") {
		t.Errorf("part[0] content-type = %q", parts[0].contentType)
	}
	if !strings.HasPrefix(parts[1].contentType, "text/html") {
		t.Errorf("part[1] content-type = %q", parts[1].contentType)
	}
}

func TestSend_WithAttachment_MultipartMixed(t *testing.T) {
	s := &fakeSender{}
	m := newTestMailer(s)

	pdfData := []byte("%PDF-1.4 fake")
	if err := m.Send(context.Background(), Message{
		To:       []string{"a@x.com"},
		Subject:  "Invoice",
		HTMLBody: "<p>see attached</p>",
		Attachments: []Attachment{
			{Filename: "invoice.pdf", ContentType: "application/pdf", Data: pdfData},
		},
	}); err != nil {
		t.Fatal(err)
	}

	headers, body := splitHeaderBody(t, s.body)
	mediaType, params, err := mime.ParseMediaType(headers.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	if mediaType != "multipart/mixed" {
		t.Errorf("media type = %q, want multipart/mixed", mediaType)
	}
	parts := readParts(t, body, params["boundary"])
	if len(parts) != 2 {
		t.Fatalf("got %d parts, want 2 (body + attachment)", len(parts))
	}
	att := parts[1]
	if !strings.Contains(att.contentType, "application/pdf") {
		t.Errorf("attachment Content-Type = %q", att.contentType)
	}
	if !strings.Contains(att.disposition, `filename="invoice.pdf"`) {
		t.Errorf("disposition missing filename: %q", att.disposition)
	}
}

// ---------- subject encoding ----------

func TestSubject_NonASCIIIsQEncoded(t *testing.T) {
	s := &fakeSender{}
	m := newTestMailer(s)
	if err := m.Send(context.Background(), Message{
		To:       []string{"a@x.com"},
		Subject:  "Halo Dunia — emoji 🎉",
		HTMLBody: "<p>x</p>",
	}); err != nil {
		t.Fatal(err)
	}
	headers, _ := splitHeaderBody(t, s.body)
	subj := headers.Get("Subject")
	if !strings.HasPrefix(subj, "=?UTF-8?q?") {
		t.Errorf("subject not Q-encoded: %q", subj)
	}
}

func TestSubject_ASCIIIsNotEncoded(t *testing.T) {
	s := &fakeSender{}
	m := newTestMailer(s)
	if err := m.Send(context.Background(), Message{
		To:       []string{"a@x.com"},
		Subject:  "plain ascii",
		HTMLBody: "<p>x</p>",
	}); err != nil {
		t.Fatal(err)
	}
	headers, _ := splitHeaderBody(t, s.body)
	if got := headers.Get("Subject"); got != "plain ascii" {
		t.Errorf("subject = %q, want plain", got)
	}
}

// ---------- helpers ----------

type parsedPart struct {
	contentType string
	disposition string
	body        []byte
}

func splitHeaderBody(t *testing.T, raw []byte) (mail.Header, string) {
	t.Helper()
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("parse: %v\n---\n%s", err, raw)
	}
	body, _ := io.ReadAll(msg.Body)
	return msg.Header, string(body)
}

func readParts(t *testing.T, body, boundary string) []parsedPart {
	t.Helper()
	mr := multipart.NewReader(strings.NewReader(body), boundary)
	var out []parsedPart
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("next part: %v", err)
		}
		var data []byte
		switch part.Header.Get("Content-Transfer-Encoding") {
		case "quoted-printable":
			data, err = io.ReadAll(quotedprintable.NewReader(part))
		default:
			data, err = io.ReadAll(part)
		}
		if err != nil {
			t.Fatalf("read part: %v", err)
		}
		out = append(out, parsedPart{
			contentType: part.Header.Get("Content-Type"),
			disposition: part.Header.Get("Content-Disposition"),
			body:        data,
		})
	}
	return out
}
