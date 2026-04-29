package httputil

import (
	"context"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// shouldRetryStatus reports whether a status code is retriable.
// Retries on 408 Request Timeout, 429 Too Many Requests, and any 5xx.
func shouldRetryStatus(code int) bool {
	if code == http.StatusRequestTimeout || code == http.StatusTooManyRequests {
		return true
	}
	return code >= 500 && code <= 599
}

// backoffDelay returns the delay before the given attempt (1-indexed).
// Uses exponential backoff with full-range jitter, capped by MaxDelay.
func backoffDelay(cfg RetryConfig, attempt int, rng *rand.Rand) time.Duration {
	exp := math.Pow(cfg.Multiplier, float64(attempt-1))
	delay := time.Duration(float64(cfg.InitialDelay) * exp)
	if delay > cfg.MaxDelay {
		delay = cfg.MaxDelay
	}
	if cfg.JitterFactor > 0 {
		jitter := (rng.Float64()*2 - 1) * cfg.JitterFactor * float64(delay)
		delay += time.Duration(jitter)
		if delay < 0 {
			delay = 0
		}
	}
	return delay
}

// retryAfterDelay parses the Retry-After header (RFC 7231).
// Supports both delta-seconds and HTTP-date forms. Returns 0 if absent or unparseable.
func retryAfterDelay(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}
	header := resp.Header.Get("Retry-After")
	if header == "" {
		return 0
	}
	if secs, err := strconv.Atoi(header); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(header); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// sleepCtx sleeps for d, returning early if ctx is cancelled.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
