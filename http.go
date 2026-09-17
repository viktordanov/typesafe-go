package typesafe

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	maxResponseBytes  = 8 << 20
	maxErrorBodyBytes = 64 << 10
)

type response struct {
	method    string
	url       string
	status    int
	header    http.Header
	body      []byte
	requestID string
}

func (r *response) invalid(path string, err error) *ResponseValidationError {
	return &ResponseValidationError{
		Method:     r.method,
		URL:        r.url,
		StatusCode: r.status,
		RequestID:  r.requestID,
		Path:       path,
		Err:        err,
		Header:     r.header,
		Body:       bytes.Clone(r.body[:min(len(r.body), maxErrorBodyBytes)]),
	}
}

func (c *Client) send(ctx context.Context, method, path string, body []byte, o requestOptions) (*response, error) {
	u := c.baseURL.JoinPath(path).String()
	policy := c.retry
	if o.retry != nil {
		policy = *o.retry
	}
	timeout := cmp.Or(o.timeout, c.timeout)

	for attempt := 0; ; attempt++ {
		resp, err := c.do(ctx, method, u, body, o.header, attempt, timeout)
		if err == nil || attempt >= policy.MaxRetries || !policy.retryable(err) {
			return resp, err
		}

		var header http.Header
		if apiErr, ok := errors.AsType[*APIError](err); ok {
			header = apiErr.Header
		}
		delay := policy.delay(attempt, header)
		c.logger.LogAttrs(ctx, slog.LevelInfo, "retrying request",
			slog.String("method", method),
			slog.String("url", u),
			slog.Int("retry", attempt+1),
			slog.Duration("delay", delay),
		)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("typesafe: %s %s: %w", method, u, ctx.Err())
		case <-time.After(delay):
		}
	}
}

func (c *Client) do(ctx context.Context, method, u string, body []byte, header http.Header, attempt int, timeout time.Duration) (*response, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(attemptCtx, method, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("typesafe: %w", err)
	}
	if header != nil {
		req.Header = header.Clone()
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if attempt > 0 {
		req.Header.Set("X-Typesafe-Retry-Count", strconv.Itoa(attempt))
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	var (
		data    []byte
		success bool
		limit   = maxErrorBodyBytes
	)
	if err == nil {
		defer resp.Body.Close()
		if success = resp.StatusCode >= 200 && resp.StatusCode <= 299; success {
			limit = maxResponseBytes
		}
		data, err = io.ReadAll(io.LimitReader(resp.Body, int64(limit)+1))
	}
	if err != nil {
		c.logger.LogAttrs(ctx, slog.LevelDebug, "request failed",
			slog.String("method", method),
			slog.String("url", u),
			slog.Int("attempt", attempt+1),
			slog.Duration("duration", time.Since(start)),
			slog.String("error", err.Error()),
		)
		if urlErr, ok := err.(*url.Error); ok {
			err = urlErr.Err
		}
		netErr, isNetErr := errors.AsType[net.Error](err)
		switch {
		case ctx.Err() != nil:
			return nil, fmt.Errorf("typesafe: %s %s: %w", method, u, ctx.Err())
		case attemptCtx.Err() != nil || isNetErr && netErr.Timeout():
			return nil, &TimeoutError{Method: method, URL: u, Timeout: timeout, Err: err}
		default:
			return nil, &ConnectionError{Method: method, URL: u, Err: err}
		}
	}

	r := &response{
		method:    method,
		url:       u,
		status:    resp.StatusCode,
		header:    resp.Header,
		body:      data,
		requestID: resp.Header.Get("X-Typesafe-Request-Id"),
	}
	c.logger.LogAttrs(ctx, slog.LevelDebug, "request completed",
		slog.String("method", method),
		slog.String("url", u),
		slog.Int("status", r.status),
		slog.Int("attempt", attempt+1),
		slog.Duration("duration", time.Since(start)),
		slog.String("request_id", r.requestID),
	)

	if len(data) > limit {
		if success {
			return nil, r.invalid("", fmt.Errorf("body exceeds %d bytes", limit))
		}
		r.body = data[:limit]
	}
	if !success {
		return nil, &APIError{
			StatusCode: r.status,
			Method:     method,
			URL:        u,
			RequestID:  r.requestID,
			Message:    errorMessage(r.body),
			Header:     r.header,
			Body:       r.body,
		}
	}
	return r, nil
}
