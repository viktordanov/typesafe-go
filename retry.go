package typesafe

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// RetryPolicy controls retries. The zero value disables retries.
type RetryPolicy struct {
	MaxRetries            int
	InitialBackoff        time.Duration
	MaxBackoff            time.Duration
	Jitter                float64
	RetryStatus           func(statusCode int) bool
	RespectRetryAfter     bool
	MaxRetryAfter         time.Duration
	RetryConnectionErrors bool
	RetryTimeouts         bool
}

// DefaultRetryPolicy matches TypeSafe's official SDKs.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:            2,
		InitialBackoff:        500 * time.Millisecond,
		MaxBackoff:            5 * time.Second,
		Jitter:                0.25,
		RetryStatus:           DefaultRetryStatus,
		RespectRetryAfter:     true,
		MaxRetryAfter:         time.Minute,
		RetryConnectionErrors: true,
		RetryTimeouts:         true,
	}
}

// DefaultRetryStatus retries 408, 429, and 5xx.
func DefaultRetryStatus(statusCode int) bool {
	return statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusTooManyRequests ||
		statusCode >= 500 && statusCode <= 599
}

func (p RetryPolicy) validate() error {
	switch {
	case p.MaxRetries < 0:
		return fmt.Errorf("MaxRetries must not be negative, got %d", p.MaxRetries)
	case p.MaxRetries == 0:
		return nil
	case p.InitialBackoff <= 0:
		return fmt.Errorf("InitialBackoff must be positive, got %s", p.InitialBackoff)
	case p.MaxBackoff < p.InitialBackoff:
		return fmt.Errorf("MaxBackoff %s is less than InitialBackoff %s", p.MaxBackoff, p.InitialBackoff)
	case !(p.Jitter >= 0 && p.Jitter <= 1):
		return fmt.Errorf("Jitter must be between 0 and 1, got %v", p.Jitter)
	case p.MaxRetryAfter < 0:
		return fmt.Errorf("MaxRetryAfter must not be negative, got %s", p.MaxRetryAfter)
	}
	return nil
}

func (p RetryPolicy) retryable(err error) bool {
	if apiErr, ok := errors.AsType[*APIError](err); ok {
		if p.RetryStatus == nil {
			return DefaultRetryStatus(apiErr.StatusCode)
		}
		return p.RetryStatus(apiErr.StatusCode)
	}
	if _, ok := errors.AsType[*TimeoutError](err); ok {
		return p.RetryTimeouts
	}
	if _, ok := errors.AsType[*ConnectionError](err); ok {
		return p.RetryConnectionErrors
	}
	return false
}

func (p RetryPolicy) delay(retry int, header http.Header) time.Duration {
	if d, ok := parseRetryAfter(header, time.Now()); ok && p.RespectRetryAfter && d <= p.MaxRetryAfter {
		return d
	}
	backoff := min(float64(p.InitialBackoff)*math.Pow(2, float64(retry)), float64(p.MaxBackoff))
	return time.Duration(backoff * (1 - rand.Float64()*p.Jitter))
}

func parseRetryAfter(h http.Header, now time.Time) (time.Duration, bool) {
	parse := func(v string, unit time.Duration) (time.Duration, bool) {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || !(f >= 0 && f < float64(math.MaxInt64/unit)) {
			return 0, false
		}
		return time.Duration(f * float64(unit)), true
	}
	if d, ok := parse(h.Get("Retry-After-Ms"), time.Millisecond); ok {
		return d, true
	}
	v := h.Get("Retry-After")
	if d, ok := parse(v, time.Second); ok {
		return d, true
	}
	if t, err := http.ParseTime(v); err == nil {
		return max(0, t.Sub(now)), true
	}
	return 0, false
}
