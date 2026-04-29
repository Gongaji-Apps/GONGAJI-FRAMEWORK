package httputil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a reusable HTTP client with retry, timeout, and optional logging.
type Client struct {
	baseURL    string
	httpClient *http.Client
	headers    http.Header
	retry      RetryConfig
	logger     Logger
	rng        *rand.Rand
}

// New constructs a Client from cfg.
func New(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}

	headers := http.Header{}
	for k, v := range cfg.Headers {
		headers.Set(k, v)
	}

	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		httpClient: httpClient,
		headers:    headers,
		retry:      cfg.Retry.withDefaults(),
		logger:     cfg.Logger,
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// WithHeader returns a clone of the client with an extra default header.
func (c *Client) WithHeader(key, value string) *Client {
	clone := c.cloneHeaders()
	clone.headers.Set(key, value)
	return clone
}

// WithHeaders returns a clone with multiple headers added.
func (c *Client) WithHeaders(h map[string]string) *Client {
	clone := c.cloneHeaders()
	for k, v := range h {
		clone.headers.Set(k, v)
	}
	return clone
}

// WithAuth returns a clone with an Authorization header set, e.g. WithAuth("Bearer", "abc").
func (c *Client) WithAuth(scheme, token string) *Client {
	value := token
	if scheme != "" {
		value = scheme + " " + token
	}
	return c.WithHeader("Authorization", value)
}

func (c *Client) cloneHeaders() *Client {
	clone := *c
	clone.headers = c.headers.Clone()
	return &clone
}

// Get sends a GET request and decodes a JSON response into result (if non-nil).
func (c *Client) Get(ctx context.Context, path string, result any) error {
	return c.Do(ctx, http.MethodGet, path, nil, result)
}

// Post sends a JSON POST request.
func (c *Client) Post(ctx context.Context, path string, body, result any) error {
	return c.Do(ctx, http.MethodPost, path, body, result)
}

// Put sends a JSON PUT request.
func (c *Client) Put(ctx context.Context, path string, body, result any) error {
	return c.Do(ctx, http.MethodPut, path, body, result)
}

// Patch sends a JSON PATCH request.
func (c *Client) Patch(ctx context.Context, path string, body, result any) error {
	return c.Do(ctx, http.MethodPatch, path, body, result)
}

// Delete sends a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.Do(ctx, http.MethodDelete, path, nil, nil)
}

// PostForm sends an application/x-www-form-urlencoded request.
func (c *Client) PostForm(ctx context.Context, path string, values url.Values, result any) error {
	encoded := values.Encode()
	req, err := c.newRequest(ctx, http.MethodPost, path, []byte(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if result != nil {
		req.Header.Set("Accept", "application/json")
	}
	return c.execute(req, []byte(encoded), result)
}

// Do executes a request with retry. body and result accept:
//   - nil
//   - []byte (sent raw, no Content-Type set)
//   - io.Reader (read fully into memory to enable retry)
//   - any other value (JSON-encoded request, JSON-decoded response)
func (c *Client) Do(ctx context.Context, method, path string, body, result any) error {
	rawBody, contentType, err := encodeBody(body)
	if err != nil {
		return fmt.Errorf("httputil: encode body: %w", err)
	}

	req, err := c.newRequest(ctx, method, path, rawBody)
	if err != nil {
		return err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if result != nil && req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}

	return c.execute(req, rawBody, result)
}

func (c *Client) newRequest(ctx context.Context, method, path string, body []byte) (*http.Request, error) {
	fullURL := c.resolveURL(path)

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reader)
	if err != nil {
		return nil, fmt.Errorf("httputil: new request: %w", err)
	}

	for k, vs := range c.headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	return req, nil
}

func (c *Client) resolveURL(path string) string {
	if c.baseURL == "" {
		return path
	}
	if path == "" {
		return c.baseURL
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return c.baseURL + path
}

func (c *Client) execute(req *http.Request, rawBody []byte, result any) error {
	ctx := req.Context()
	method := req.Method
	urlStr := req.URL.String()

	var lastErr error
	for attempt := 1; attempt <= c.retry.MaxAttempts; attempt++ {
		if attempt > 1 {
			req = c.cloneRequest(req, rawBody)
		}

		c.debug("request", "method", method, "url", urlStr, "attempt", attempt)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("httputil: %s %s: %w", method, urlStr, err)
			c.errorLog("request error", "method", method, "url", urlStr, "attempt", attempt, "err", err)

			if attempt == c.retry.MaxAttempts || !isRetriableErr(err) {
				return lastErr
			}
			if waitErr := sleepCtx(ctx, backoffDelay(c.retry, attempt, c.rng)); waitErr != nil {
				return waitErr
			}
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			err := decodeResponse(resp, result)
			closeBody(resp)
			return err
		}

		body, _ := io.ReadAll(resp.Body)
		closeBody(resp)

		httpErr := &HTTPError{
			Method:     method,
			URL:        urlStr,
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       body,
		}
		lastErr = httpErr

		if attempt == c.retry.MaxAttempts || !shouldRetryStatus(resp.StatusCode) {
			return httpErr
		}

		delay := retryAfterDelay(resp)
		if delay <= 0 {
			delay = backoffDelay(c.retry, attempt, c.rng)
		}
		c.debug("retrying", "method", method, "url", urlStr, "status", resp.StatusCode, "delay", delay)
		if waitErr := sleepCtx(ctx, delay); waitErr != nil {
			return waitErr
		}
	}

	return lastErr
}

func (c *Client) cloneRequest(req *http.Request, body []byte) *http.Request {
	clone := req.Clone(req.Context())
	if body != nil {
		clone.Body = io.NopCloser(bytes.NewReader(body))
		clone.ContentLength = int64(len(body))
	}
	return clone
}

func (c *Client) debug(msg string, fields ...any) {
	if c.logger != nil {
		c.logger.Debug(msg, fields...)
	}
}

func (c *Client) errorLog(msg string, fields ...any) {
	if c.logger != nil {
		c.logger.Error(msg, fields...)
	}
}

func encodeBody(body any) ([]byte, string, error) {
	switch v := body.(type) {
	case nil:
		return nil, "", nil
	case []byte:
		return v, "", nil
	case string:
		return []byte(v), "", nil
	case io.Reader:
		buf, err := io.ReadAll(v)
		if err != nil {
			return nil, "", err
		}
		return buf, "", nil
	default:
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, "", err
		}
		return buf, "application/json", nil
	}
}

func decodeResponse(resp *http.Response, result any) error {
	if result == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	switch v := result.(type) {
	case *[]byte:
		buf, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		*v = buf
		return nil
	default:
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil && err != io.EOF {
			return fmt.Errorf("httputil: decode response: %w", err)
		}
		return nil
	}
}

func closeBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}

// isRetriableErr decides whether a transport-level error is worth retrying.
// Context cancellations are not retried.
func isRetriableErr(err error) bool {
	if err == nil {
		return false
	}
	if ctxErr := ctxErrFromTransport(err); ctxErr != nil {
		return false
	}
	return true
}

func ctxErrFromTransport(err error) error {
	if err == context.Canceled || err == context.DeadlineExceeded {
		return err
	}
	type unwrapper interface{ Unwrap() error }
	for cur := err; cur != nil; {
		if cur == context.Canceled || cur == context.DeadlineExceeded {
			return cur
		}
		u, ok := cur.(unwrapper)
		if !ok {
			break
		}
		cur = u.Unwrap()
	}
	return nil
}
