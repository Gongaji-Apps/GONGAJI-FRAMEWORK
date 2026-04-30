// Package fcm wraps Firebase Cloud Messaging for sending push notifications
// to single devices, multiple tokens, or topics.
//
// Initialization accepts a service-account JSON either inline or via a file
// path. With neither set, Application Default Credentials are used.
//
// Example:
//
//	c, err := fcm.New(ctx, fcm.Config{
//	    ProjectID:      "my-project",
//	    CredentialJSON: []byte(os.Getenv("FCM_SERVICE_ACCOUNT")),
//	})
//
//	_, err = c.Send(ctx, deviceToken, fcm.Notification{
//	    Title:  "Pesanan Diterima",
//	    Body:   "Pesanan #1234 telah diterima.",
//	    Module: "TRANSACTION",
//	    Type:   "ORDER_RECEIVED",
//	    Route:  "/orders/1234",
//	})
package fcm

import (
	"context"
	"errors"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// MaxTokensPerBatch is the FCM-imposed cap on tokens per multicast call.
// SendMulticast automatically chunks larger lists.
const MaxTokensPerBatch = 500

// Config controls FCM client construction.
type Config struct {
	ProjectID      string
	CredentialJSON []byte // raw service account JSON content
	CredentialFile string // path to service account JSON file
}

// Notification carries the fields used to build an FCM message.
//
// Title and Body are surfaced as the user-visible notification.
// Module / Type / Route are merged into the data payload (under those keys)
// so client apps can route deep-links. Data is merged last and may override.
type Notification struct {
	Title  string
	Body   string
	Module string
	Type   string
	Route  string
	Data   map[string]string
}

// Client wraps a *messaging.Client.
type Client struct {
	msg messagingClient
}

// messagingClient is the subset of *messaging.Client that this package uses.
// It exists to enable hermetic testing.
type messagingClient interface {
	Send(ctx context.Context, message *messaging.Message) (string, error)
	SendEachForMulticast(ctx context.Context, message *messaging.MulticastMessage) (*messaging.BatchResponse, error)
}

// New constructs a Client.
func New(ctx context.Context, cfg Config) (*Client, error) {
	var opts []option.ClientOption
	switch {
	case len(cfg.CredentialJSON) > 0:
		opts = append(opts, option.WithCredentialsJSON(cfg.CredentialJSON))
	case cfg.CredentialFile != "":
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialFile))
	}

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.ProjectID}, opts...)
	if err != nil {
		return nil, fmt.Errorf("fcm: new firebase app: %w", err)
	}
	msg, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("fcm: messaging client: %w", err)
	}
	return &Client{msg: msg}, nil
}

// Send delivers a notification to a single device token.
// Returns the FCM message ID on success.
func (c *Client) Send(ctx context.Context, token string, notif Notification) (string, error) {
	if token == "" {
		return "", errors.New("fcm: token is required")
	}
	msg := &messaging.Message{
		Token:        token,
		Notification: buildNotification(notif),
		Data:         buildData(notif),
	}
	return c.msg.Send(ctx, msg)
}

// SendToTopic delivers a notification to all subscribers of a topic.
func (c *Client) SendToTopic(ctx context.Context, topic string, notif Notification) (string, error) {
	if topic == "" {
		return "", errors.New("fcm: topic is required")
	}
	msg := &messaging.Message{
		Topic:        topic,
		Notification: buildNotification(notif),
		Data:         buildData(notif),
	}
	return c.msg.Send(ctx, msg)
}

// MulticastResult aggregates results across one or more underlying batch
// calls (SendEachForMulticast caps at MaxTokensPerBatch tokens per call).
type MulticastResult struct {
	SuccessCount int
	FailureCount int
	Errors       []TokenError // one entry per failed token
}

// TokenError pairs a failed token with its error.
type TokenError struct {
	Token string
	Err   error
}

// SendMulticast delivers a notification to up to MaxTokensPerBatch tokens
// per underlying call, chunking larger token lists transparently.
func (c *Client) SendMulticast(
	ctx context.Context,
	tokens []string,
	notif Notification,
) (*MulticastResult, error) {
	if len(tokens) == 0 {
		return nil, errors.New("fcm: tokens is empty")
	}

	notification := buildNotification(notif)
	data := buildData(notif)

	result := &MulticastResult{}

	for _, batch := range chunkTokens(tokens, MaxTokensPerBatch) {
		msg := &messaging.MulticastMessage{
			Tokens:       batch,
			Notification: notification,
			Data:         data,
		}
		resp, err := c.msg.SendEachForMulticast(ctx, msg)
		if err != nil {
			return result, fmt.Errorf("fcm: send multicast: %w", err)
		}
		result.SuccessCount += resp.SuccessCount
		result.FailureCount += resp.FailureCount
		for i, sr := range resp.Responses {
			if !sr.Success {
				result.Errors = append(result.Errors, TokenError{
					Token: batch[i],
					Err:   sr.Error,
				})
			}
		}
	}
	return result, nil
}

// =====================================================================
// Pure helpers (no network — testable independently)
// =====================================================================

func buildNotification(n Notification) *messaging.Notification {
	if n.Title == "" && n.Body == "" {
		return nil
	}
	return &messaging.Notification{Title: n.Title, Body: n.Body}
}

func buildData(n Notification) map[string]string {
	out := make(map[string]string)
	if n.Module != "" {
		out["module"] = n.Module
	}
	if n.Type != "" {
		out["type"] = n.Type
	}
	if n.Route != "" {
		out["route"] = n.Route
	}
	for k, v := range n.Data {
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func chunkTokens(tokens []string, size int) [][]string {
	if size <= 0 {
		size = MaxTokensPerBatch
	}
	if len(tokens) == 0 {
		return nil
	}
	out := make([][]string, 0, (len(tokens)+size-1)/size)
	for i := 0; i < len(tokens); i += size {
		end := i + size
		if end > len(tokens) {
			end = len(tokens)
		}
		out = append(out, tokens[i:end])
	}
	return out
}
