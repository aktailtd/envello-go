// Package envello is a minimal Go client for the Envello transactional
// email API (https://envello.dev). It targets the API's real
// request/response shapes, not an invented contract, so keep this in sync
// with the API if those shapes change.
package envello

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultBaseURL is used when no WithBaseURL option is given.
const DefaultBaseURL = "https://api.envello.dev"

// Client is a minimal HTTP client for the Envello API. Construct one with
// NewClient.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// Option configures a Client at construction time.
type Option func(*Client)

// WithBaseURL overrides the API base URL, e.g. for local dev
// ("http://localhost:3000") or self-hosted deployments.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = baseURL }
}

// WithHTTPClient overrides the underlying *http.Client (e.g. to set a
// custom transport, or to inject a fake one in tests).
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) { c.httpClient = httpClient }
}

// NewClient builds a client for the Envello transactional email API.
// apiKey is an "env_live_..." or "env_test_..." API key.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// APIError is returned whenever the Envello API responds with a non-2xx
// status code.
type APIError struct {
	StatusCode int    // HTTP status code
	Code       string // machine-readable `error` field from the response body, e.g. "rate_limited"
	Message    string // human-readable `message` field, if present
	Body       []byte // raw response body
}

func (e *APIError) Error() string {
	return fmt.Sprintf("envello: API error (status %d, code %q): %s", e.StatusCode, e.Code, e.Message)
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// do sends a JSON request and decodes a JSON response into out (if
// non-nil). extraHeaders (may be nil) are set on the request in addition
// to the standard auth/content-type/user-agent headers. It returns
// *APIError for any non-2xx response.
func (c *Client) do(ctx context.Context, method, path string, body any, out any, extraHeaders map[string]string) error {
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("envello: marshal request: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("envello: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "envello-go/0.1.0")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("envello: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("envello: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var parsed errorResponse
		_ = json.Unmarshal(respBody, &parsed) // best-effort; body may not be JSON
		return &APIError{
			StatusCode: resp.StatusCode,
			Code:       parsed.Error,
			Message:    parsed.Message,
			Body:       respBody,
		}
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("envello: decode response: %w", err)
		}
	}
	return nil
}
