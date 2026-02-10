package relayer

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newResponse(status int, body string, headers map[string]string) *http.Response {
	h := http.Header{}
	for k, v := range headers {
		h.Set(k, v)
	}
	return &http.Response{
		StatusCode: status,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestHTTPClientDo_RetriesOnTooManyRequestsThenSucceeds(t *testing.T) {
	t.Parallel()

	attempts := 0
	base := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return newResponse(http.StatusTooManyRequests, `{"error":"rate limited"}`, map[string]string{"Retry-After": "0"}), nil
		}
		return newResponse(http.StatusOK, `{"ok":true}`, map[string]string{"Content-Type": "application/json"}), nil
	})}

	client := NewHTTPClient(base)

	var out struct {
		OK bool `json:"ok"`
	}
	err := client.Do(context.Background(), http.MethodGet, "https://example.test/tx", nil, &out)
	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
	assert.True(t, out.OK)
}

func TestHTTPClientDo_RetriesTooManyRequestsWithInvalidRetryAfter(t *testing.T) {
	t.Parallel()

	attempts := 0
	base := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return newResponse(http.StatusTooManyRequests, `{"error":"rate limited"}`, map[string]string{"Retry-After": "not-a-valid-value"}), nil
		}
		return newResponse(http.StatusOK, `{"ok":true}`, map[string]string{"Content-Type": "application/json"}), nil
	})}

	client := NewHTTPClient(base)

	var out struct {
		OK bool `json:"ok"`
	}
	err := client.Do(context.Background(), http.MethodGet, "https://example.test/tx", nil, &out)
	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
	assert.True(t, out.OK)
}

func TestHTTPClientDo_TooManyRequestsExhaustsRetries(t *testing.T) {
	t.Parallel()

	attempts := 0
	base := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		return newResponse(http.StatusTooManyRequests, `{"error":"rate limited"}`, map[string]string{"Retry-After": "0"}), nil
	})}

	client := NewHTTPClient(base)

	err := client.Do(context.Background(), http.MethodGet, "https://example.test/tx", nil, nil)
	require.Error(t, err)

	var httpErr *HTTPError
	require.True(t, errors.As(err, &httpErr), "expected HTTPError in wrapped error")
	assert.Equal(t, http.StatusTooManyRequests, httpErr.StatusCode)
	assert.Equal(t, int(defaultMaxRetries)+1, attempts)
}
