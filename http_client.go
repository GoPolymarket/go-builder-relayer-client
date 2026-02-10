package relayer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	sdkerrors "github.com/GoPolymarket/go-builder-relayer-client/pkg/errors"
	"github.com/GoPolymarket/go-builder-relayer-client/pkg/logger"
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
	Err        *sdkerrors.SDKError
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("http error: status %d body=%s", e.StatusCode, e.Body)
}

func (e *HTTPError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *HTTPError) Code() sdkerrors.ErrorCode {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Code
}

func httpErrorForStatus(statusCode int) *sdkerrors.SDKError {
	switch statusCode {
	case http.StatusBadRequest:
		return sdkerrors.ErrBadRequest
	case http.StatusUnauthorized, http.StatusForbidden:
		return sdkerrors.ErrUnauthorized
	case http.StatusTooManyRequests:
		return sdkerrors.ErrTooManyRequests
	default:
		if statusCode >= 500 {
			return sdkerrors.ErrInternalServerError
		}
	}
	return nil
}

func parseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, false
	}

	seconds, err := strconv.Atoi(trimmed)
	if err == nil {
		if seconds <= 0 {
			return 0, true
		}
		return time.Duration(seconds) * time.Second, true
	}

	retryTime, err := http.ParseTime(trimmed)
	if err != nil {
		return 0, false
	}
	if !retryTime.After(now) {
		return 0, true
	}
	return retryTime.Sub(now), true
}

func exponentialBackoff(attempt uint) time.Duration {
	exp := attempt
	if exp > maxBackoffExponent {
		exp = maxBackoffExponent
	}
	return defaultBaseDelay * time.Duration(1<<exp)
}

func (c *HTTPClient) Do(ctx context.Context, method, urlStr string, opts *RequestOptions, out interface{}) error {
	if c == nil {
		return fmt.Errorf("http client is nil")
	}
	if ctx == nil {
		ctx = context.Background()
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
	maxAttempts := defaultMaxRetries + 1
	var nextRetryDelay *time.Duration

	for attempt := uint(0); attempt <= defaultMaxRetries; attempt++ {
		if attempt > 0 {
			delay := exponentialBackoff(attempt - 1)
			if nextRetryDelay != nil {
				delay = *nextRetryDelay
				nextRetryDelay = nil
			}
			if err := sleepWithContext(ctx, delay); err != nil {
				return err
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
			if attempt < defaultMaxRetries {
				logger.Warn("http request failed (attempt %d/%d): %v", attempt+1, maxAttempts, err)
			}
			continue // Retry on network errors
		}

		respBytes, err := io.ReadAll(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}

		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			if attempt < defaultMaxRetries {
				logger.Warn("http read response failed (attempt %d/%d): %v", attempt+1, maxAttempts, err)
			}
			continue
		}

		httpErr := &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       string(respBytes),
			Err:        httpErrorForStatus(resp.StatusCode),
		}

		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			lastErr = httpErr
			if attempt < defaultMaxRetries {
				if resp.StatusCode == http.StatusTooManyRequests {
					if retryDelay, ok := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()); ok {
						nextRetryDelay = &retryDelay
					}
				}
				logger.Warn("http retryable status %d (attempt %d/%d)", resp.StatusCode, attempt+1, maxAttempts)
			}
			continue
		}

		if resp.StatusCode >= 400 {
			// Client error (except 429), do not retry.
			return httpErr
		}

		// Success
		if out != nil && len(respBytes) > 0 {
			if err := json.Unmarshal(respBytes, out); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}
		}
		return nil
	}

	if lastErr != nil {
		logger.Error("http request failed after retries: %v", lastErr)
		return fmt.Errorf("max retries exceeded: %w", lastErr)
	}
	return fmt.Errorf("max retries exceeded")
}
