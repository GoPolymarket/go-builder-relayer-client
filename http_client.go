package relayer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultMaxRetries  = uint(3)
	defaultBaseDelay   = 500 * time.Millisecond
	maxBackoffExponent = uint(10)
)

type RequestOptions struct {
	Headers http.Header
	Params  map[string]string
	Body    []byte
}

type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient(client *http.Client) *HTTPClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPClient{client: client}
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("http error: status %d body=%s", e.StatusCode, e.Body)
}

func (c *HTTPClient) Do(ctx context.Context, method, urlStr string, opts *RequestOptions, out interface{}) error {
	if c == nil {
		return fmt.Errorf("http client is nil")
	}
	if opts == nil {
		opts = &RequestOptions{}
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}

	if len(opts.Params) > 0 {
		q := parsed.Query()
		for k, v := range opts.Params {
			q.Set(k, v)
		}
		parsed.RawQuery = q.Encode()
	}

	var lastErr error

	for attempt := uint(0); attempt <= defaultMaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter could be better, but simple exponential is fine for now
			exp := attempt - 1
			if exp > maxBackoffExponent {
				exp = maxBackoffExponent
			}
			delay := defaultBaseDelay * time.Duration(1<<exp)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		// Re-create body reader for each attempt
		var bodyReader io.Reader
		if len(opts.Body) > 0 {
			bodyReader = bytes.NewReader(opts.Body)
		}

		req, err := http.NewRequestWithContext(ctx, method, parsed.String(), bodyReader)
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}

		if req.Header == nil {
			req.Header = http.Header{}
		}
		req.Header.Set("Accept", "application/json")
		if len(opts.Body) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		for k, values := range opts.Headers {
			for _, v := range values {
				req.Header.Add(k, v)
			}
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue // Retry on network errors
		}

		respBytes, err := io.ReadAll(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}

		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = &HTTPError{StatusCode: resp.StatusCode, Body: string(respBytes)}
			continue // Retry on server errors
		}

		if resp.StatusCode >= 400 {
			// Client error, do not retry
			return &HTTPError{StatusCode: resp.StatusCode, Body: string(respBytes)}
		}

		// Success
		if out != nil && len(respBytes) > 0 {
			if err := json.Unmarshal(respBytes, out); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}
		}
		return nil
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}
