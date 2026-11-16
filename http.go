package adsbv2

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ErrRateLimited matches, under errors.Is, a *RateLimitError: the upstream
// answered 429.
var ErrRateLimited = errors.New("adsbv2: rate limited")

// RateLimitError is a 429 from the upstream, with the delay it asked for.
type RateLimitError struct {
	// RetryAfter is the Retry-After delay, or zero when none was sent.
	RetryAfter time.Duration
}

// Error implements error.
func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("adsbv2: rate limited, retry after %s", e.RetryAfter)
	}
	return "adsbv2: rate limited"
}

// Unwrap lets errors.Is(err, ErrRateLimited) match.
func (e *RateLimitError) Unwrap() error {
	return ErrRateLimited
}

// StatusError is a response that was neither a 200 nor a 429.
type StatusError struct {
	StatusCode int
	// Body is the start of the response body (at most 512 bytes).
	Body string
}

// Error implements error.
func (e *StatusError) Error() string {
	return fmt.Sprintf("adsbv2: unexpected status %d: %s", e.StatusCode, e.Body)
}

// Decode reads a v2 response, mapping a non-200 to an error as CheckStatus
// does. It doesn't close the body.
func Decode(res *http.Response) (*Response, error) {
	if err := CheckStatus(res); err != nil {
		return nil, err
	}

	var out Response
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("adsbv2: unable to decode response: %w", err)
	}

	return &out, nil
}

// CheckStatus maps a non-200 response to an error: a 429 to a
// *RateLimitError, anything else to a *StatusError carrying the start of
// the body. It returns nil for a 200.
func CheckStatus(res *http.Response) error {
	if res.StatusCode == http.StatusTooManyRequests {
		return &RateLimitError{RetryAfter: retryAfter(res.Header.Get("Retry-After"), time.Now())}
	}

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return &StatusError{StatusCode: res.StatusCode, Body: strings.TrimSpace(string(body))}
	}

	return nil
}

// retryAfter reads a Retry-After header, which is either a number of
// seconds or an HTTP date.
func retryAfter(v string, now time.Time) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if s, err := strconv.Atoi(v); err == nil {
		if s > 0 {
			return time.Duration(s) * time.Second
		}
		return 0
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := t.Sub(now); d > 0 {
			return d.Round(time.Second)
		}
	}
	return 0
}
