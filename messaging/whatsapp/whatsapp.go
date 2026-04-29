// Package whatsapp provides a WhatsApp gateway client with multi-session
// fallback. It supports two gateway flavors used in production:
//
//   - V1: form-encoded POST endpoints (/message/send, /group/create,
//     /group/add-member). Sessions are simple session names.
//   - V2: JSON POST to the base URL with API-key header. Sessions carry
//     dev_code/app_code pairs. V2 only supports SendMessage.
//
// SendMessage tries each V1 session in order, then each V2 session.
// Group operations are V1-only.
package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	frameworkErr "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/httputil"
)

// Messenger is a generic messaging contract. Other providers (Telegram, etc.)
// can implement this interface.
type Messenger interface {
	SendMessage(ctx context.Context, to, message string) error
}

// SessionV1 is a V1 gateway session identifier (simple session name).
type SessionV1 string

// SessionV2 is a V2 gateway session: dev_code + app_code pair.
type SessionV2 struct {
	DevCode string
	AppCode string
}

// Config configures a WhatsApp Client.
//
// At least one of (BaseURLV1 + SessionsV1) or (BaseURLV2 + SessionsV2 +
// APIKeyV2) must be provided. Group operations require V1 to be configured.
type Config struct {
	BaseURLV1  string
	SessionsV1 []SessionV1

	BaseURLV2  string
	APIKeyV2   string
	SessionsV2 []SessionV2

	Timeout time.Duration   // per-request timeout (default: 30s)
	Logger  httputil.Logger // optional
}

// Client is a WhatsApp gateway client.
type Client struct {
	cfg     Config
	httpV1  *httputil.Client
	httpV2  *httputil.Client
}

// New constructs a Client.
func New(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	c := &Client{cfg: cfg}

	if cfg.BaseURLV1 != "" {
		c.httpV1 = httputil.New(httputil.Config{
			BaseURL: cfg.BaseURLV1,
			Timeout: timeout,
			Logger:  cfg.Logger,
		})
	}
	if cfg.BaseURLV2 != "" {
		headers := map[string]string{}
		if cfg.APIKeyV2 != "" {
			headers["Api_key"] = cfg.APIKeyV2
		}
		c.httpV2 = httputil.New(httputil.Config{
			BaseURL: cfg.BaseURLV2,
			Timeout: timeout,
			Headers: headers,
			Logger:  cfg.Logger,
		})
	}

	return c
}

// SendMessage sends a text message. Tries each V1 session first, then each V2
// session. Returns the last error if all attempts fail.
func (c *Client) SendMessage(ctx context.Context, to, message string) error {
	var lastErr error

	if c.httpV1 != nil {
		for _, session := range c.cfg.SessionsV1 {
			if err := c.sendMessageV1(ctx, session, to, message); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
	}

	if c.httpV2 != nil {
		for _, session := range c.cfg.SessionsV2 {
			if err := c.sendMessageV2(ctx, session, to, message); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
	}

	if lastErr == nil {
		return frameworkErr.NewInternalServerError("[Internal Server Error] Tidak ada session WhatsApp yang dikonfigurasi.")
	}
	return wrapInternal(lastErr, "mengirim Pesan")
}

// SendOTP composes a standard OTP message and sends it. Equivalent to
// SendMessage with a templated body. Use SendMessage directly if you need
// a custom OTP template.
func (c *Client) SendOTP(ctx context.Context, to, fullName, otp string) error {
	message := fmt.Sprintf(`Assalamu'alaikum, %s

Silahkan Gunakan kode OTP berikut *%s* untuk verifikasi akun kamu.

*PENTING:* Jangan bagikan kode OTP ini kepada siapapun, termasuk pihak GoNgaji.

Terima kasih atas kepercayaan Anda kepada kami.

Salam,
[Go Ngaji]`, fullName, otp)
	return c.SendMessage(ctx, to, message)
}

// CreateGroup creates a group via V1 gateway. Returns groupID + invite code.
// Tries each V1 session in order.
func (c *Client) CreateGroup(
	ctx context.Context,
	subject, description string,
	admins, members []string,
) (groupID, inviteCode string, err error) {
	if c.httpV1 == nil || len(c.cfg.SessionsV1) == 0 {
		return "", "", frameworkErr.NewInternalServerError("[Internal Server Error] V1 gateway tidak dikonfigurasi untuk operasi grup.")
	}

	var lastErr error
	for _, session := range c.cfg.SessionsV1 {
		id, code, err := c.createGroupV1(ctx, session, subject, description, admins, members)
		if err == nil {
			return id, code, nil
		}
		lastErr = err
	}
	return "", "", wrapInternal(lastErr, "membuat Grup Whatsapp")
}

// AddGroupMember adds members to a group via V1 gateway. role is the WhatsApp
// group role (e.g. "member", "admin").
func (c *Client) AddGroupMember(
	ctx context.Context,
	groupID string,
	members []string,
	role string,
) error {
	if c.httpV1 == nil || len(c.cfg.SessionsV1) == 0 {
		return frameworkErr.NewInternalServerError("[Internal Server Error] V1 gateway tidak dikonfigurasi untuk operasi grup.")
	}

	var lastErr error
	for _, session := range c.cfg.SessionsV1 {
		if err := c.addGroupMemberV1(ctx, session, groupID, members, role); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return wrapInternal(lastErr, "menambahkan Anggota")
}

// =====================================================================
// V1 — form-encoded
// =====================================================================

type v1Response struct {
	StatusCode int    `json:"status_code"`
	Status     bool   `json:"status"`
	Message    string `json:"message"`
}

type v1GroupCreateResponse struct {
	StatusCode int    `json:"status_code"`
	Status     bool   `json:"status"`
	Message    string `json:"message"`
	Data       struct {
		ID         string `json:"id"`
		InviteCode string `json:"invite_code"`
	} `json:"data"`
}

func (c *Client) sendMessageV1(ctx context.Context, session SessionV1, receiver, message string) error {
	values := url.Values{
		"group":    {"false"},
		"session":  {string(session)},
		"receiver": {receiver},
		"message":  {message},
	}
	var resp v1Response
	if err := c.httpV1.PostForm(ctx, "/message/send", values, &resp); err != nil {
		return err
	}
	return v1Status(&resp)
}

func (c *Client) createGroupV1(
	ctx context.Context,
	session SessionV1,
	subject, description string,
	admins, members []string,
) (string, string, error) {
	values := url.Values{}
	values.Set("session", string(session))
	values.Set("group_name", subject)
	values.Set("group_description", description)
	for _, a := range admins {
		values.Add("group_admin[]", a)
	}
	for _, m := range members {
		values.Add("group_participant[]", m)
	}

	var resp v1GroupCreateResponse
	if err := c.httpV1.PostForm(ctx, "/group/create", values, &resp); err != nil {
		return "", "", err
	}
	if !resp.Status {
		return "", "", v1Failed(resp.StatusCode, resp.Message)
	}
	return resp.Data.ID, resp.Data.InviteCode, nil
}

func (c *Client) addGroupMemberV1(
	ctx context.Context,
	session SessionV1,
	groupID string,
	members []string,
	role string,
) error {
	values := url.Values{}
	values.Set("session", string(session))
	values.Set("group_id", groupID)
	values.Set("group_role", role)
	for _, m := range members {
		values.Add("group_member[]", m)
	}

	var resp v1Response
	if err := c.httpV1.PostForm(ctx, "/group/add-member", values, &resp); err != nil {
		return err
	}
	return v1Status(&resp)
}

func v1Status(resp *v1Response) error {
	if resp.Status {
		return nil
	}
	return v1Failed(resp.StatusCode, resp.Message)
}

func v1Failed(statusCode int, message string) error {
	if statusCode == http.StatusBadRequest {
		return frameworkErr.NewBadRequest(message)
	}
	if message == "" {
		message = "[Internal Server Error] gateway WhatsApp menolak permintaan"
	}
	return frameworkErr.NewInternalServerError(message)
}

// =====================================================================
// V2 — JSON
// =====================================================================

type v2Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func (c *Client) sendMessageV2(ctx context.Context, session SessionV2, receiver, message string) error {
	payload := map[string]any{
		"number":   receiver,
		"dev_code": session.DevCode,
		"message":  message,
		"app_code": session.AppCode,
	}

	var resp v2Response
	// V2 endpoint is the base URL itself; pass empty path.
	if err := c.httpV2.Post(ctx, "", payload, &resp); err != nil {
		return err
	}
	if !resp.Success {
		msg := resp.Message
		if msg == "" {
			msg = "[Internal Server Error] V2 gateway returned success=false"
		}
		return frameworkErr.NewInternalServerError(msg)
	}
	return nil
}

// =====================================================================
// Helpers
// =====================================================================

// wrapInternal returns err unchanged if it is already an *AppError; otherwise
// wraps it as InternalServerError with an Indonesian operation label.
func wrapInternal(err error, operation string) error {
	if err == nil {
		return nil
	}
	var appErr *frameworkErr.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return frameworkErr.NewInternalServerError(
		fmt.Sprintf("[Internal Server Error] Oops! Kami mengalami masalah saat %s.", operation),
	)
}
