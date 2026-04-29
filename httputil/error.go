package httputil

import (
	"errors"
	"fmt"
)

// HTTPError represents a non-2xx HTTP response.
// Inspect StatusCode and Body to drive caller logic.
type HTTPError struct {
	Method     string
	URL        string
	StatusCode int
	Status     string
	Body       []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("httputil: %s %s -> %s", e.Method, e.URL, e.Status)
}

// AsHTTPError unwraps err into *HTTPError. Returns (nil, false) if err is not one.
func AsHTTPError(err error) (*HTTPError, bool) {
	var he *HTTPError
	if errors.As(err, &he) {
		return he, true
	}
	return nil, false
}
