package httputil

import (
	"net/http"
	"time"
)

// Logger is an optional interface for observability hooks.
// Provide an implementation in Config.Logger to enable logging.
type Logger interface {
	Debug(msg string, fields ...any)
	Error(msg string, fields ...any)
}

// Config controls Client behavior.
type Config struct {
	BaseURL    string
	Timeout    time.Duration
	Headers    map[string]string
	Retry      *RetryConfig
	Logger     Logger
	HTTPClient *http.Client
}

// RetryConfig controls retry behavior. The zero value triggers defaults.
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	JitterFactor float64
}

const (
	defaultTimeout      = 60 * time.Second
	defaultMaxAttempts  = 3
	defaultInitialDelay = 200 * time.Millisecond
	defaultMaxDelay     = 5 * time.Second
	defaultMultiplier   = 2.5
	defaultJitterFactor = 0.3
)

func (r *RetryConfig) withDefaults() RetryConfig {
	if r == nil {
		return RetryConfig{
			MaxAttempts:  defaultMaxAttempts,
			InitialDelay: defaultInitialDelay,
			MaxDelay:     defaultMaxDelay,
			Multiplier:   defaultMultiplier,
			JitterFactor: defaultJitterFactor,
		}
	}
	out := *r
	if out.MaxAttempts <= 0 {
		out.MaxAttempts = defaultMaxAttempts
	}
	if out.InitialDelay <= 0 {
		out.InitialDelay = defaultInitialDelay
	}
	if out.MaxDelay <= 0 {
		out.MaxDelay = defaultMaxDelay
	}
	if out.Multiplier <= 1 {
		out.Multiplier = defaultMultiplier
	}
	if out.JitterFactor < 0 {
		out.JitterFactor = 0
	}
	return out
}
