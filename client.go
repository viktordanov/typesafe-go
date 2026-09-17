package typesafe

import (
	"cmp"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	Version        = "0.1.0"
	DefaultBaseURL = "https://api.typesafe.ai"
	DefaultModel   = "jev-latest"
	DefaultTimeout = 10 * time.Second // per attempt
)

type Config struct {
	APIKey       string // required
	BaseURL      string // defaults to DefaultBaseURL
	DefaultModel string // defaults to DefaultModel
}

// Client is a TypeSafe API client. It is safe for concurrent use.
type Client struct {
	apiKey       string
	baseURL      *url.URL
	defaultModel string
	httpClient   *http.Client
	retry        RetryPolicy
	timeout      time.Duration
	logger       *slog.Logger
	userAgent    string
}

// New returns a client for cfg.
func New(cfg Config, options ...Option) (*Client, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("%w: API key is required", ErrInvalidConfig)
	}
	baseURL, err := url.Parse(cmp.Or(strings.TrimSpace(cfg.BaseURL), DefaultBaseURL))
	if err != nil || (baseURL.Scheme != "http" && baseURL.Scheme != "https") || baseURL.Host == "" ||
		baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return nil, fmt.Errorf("%w: base URL must be an absolute http(s) URL without a query or fragment", ErrInvalidConfig)
	}

	c := &Client{
		apiKey:       apiKey,
		baseURL:      baseURL,
		defaultModel: cmp.Or(strings.TrimSpace(cfg.DefaultModel), DefaultModel),
		httpClient:   &http.Client{},
		retry:        DefaultRetryPolicy(),
		timeout:      DefaultTimeout,
		logger:       slog.Default(),
		userAgent:    "typesafe-go/" + Version,
	}
	for _, option := range options {
		if err := option(c); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
		}
	}
	c.logger = c.logger.WithGroup("typesafe_client")

	httpClient := *c.httpClient
	if httpClient.CheckRedirect == nil {
		httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	c.httpClient = &httpClient
	return c, nil
}

type Option func(*Client) error

// WithHTTPClient sets the HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) error {
		if httpClient == nil {
			return errors.New("HTTP client is nil")
		}
		c.httpClient = httpClient
		return nil
	}
}

func WithRetryPolicy(policy RetryPolicy) Option {
	return func(c *Client) error {
		if err := policy.validate(); err != nil {
			return err
		}
		c.retry = policy
		return nil
	}
}

// WithTimeout bounds each attempt.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) error {
		if timeout <= 0 {
			return fmt.Errorf("timeout must be positive, got %s", timeout)
		}
		c.timeout = timeout
		return nil
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) Option {
	return func(c *Client) error {
		if logger == nil {
			return errors.New("logger is nil")
		}
		c.logger = logger
		return nil
	}
}

// WithUserAgent appends product to the User-Agent.
func WithUserAgent(product string) Option {
	return func(c *Client) error {
		if strings.TrimSpace(product) == "" {
			return errors.New("user agent is blank")
		}
		c.userAgent += " " + strings.TrimSpace(product)
		return nil
	}
}

type RequestOption func(*requestOptions) error

type requestOptions struct {
	model   string
	timeout time.Duration
	retry   *RetryPolicy
	header  http.Header
}

func applyRequestOptions(options []RequestOption) (requestOptions, error) {
	var o requestOptions
	var errs []error
	for _, option := range options {
		if err := option(&o); err != nil {
			errs = append(errs, err)
		}
	}
	return o, errors.Join(errs...)
}

// WithModel sets the model for SystemOne.
func WithModel(model string) RequestOption {
	return func(o *requestOptions) error {
		if strings.TrimSpace(model) == "" {
			return invalid("options", "model is blank")
		}
		o.model = strings.TrimSpace(model)
		return nil
	}
}

func WithRequestTimeout(timeout time.Duration) RequestOption {
	return func(o *requestOptions) error {
		if timeout <= 0 {
			return invalid("options", "timeout must be positive, got %s", timeout)
		}
		o.timeout = timeout
		return nil
	}
}

func WithRequestRetryPolicy(policy RetryPolicy) RequestOption {
	return func(o *requestOptions) error {
		if err := policy.validate(); err != nil {
			return &ValidationError{Path: "options", Err: err}
		}
		o.retry = &policy
		return nil
	}
}

// WithRequestHeader sets a header.
func WithRequestHeader(key, value string) RequestOption {
	return func(o *requestOptions) error {
		if o.header == nil {
			o.header = make(http.Header)
		}
		o.header.Set(key, value)
		return nil
	}
}
