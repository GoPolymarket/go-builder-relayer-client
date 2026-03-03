package relayer

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fakeHTTPDoer struct {
	resp *http.Response
	err  error
}

func (f fakeHTTPDoer) Do(_ *http.Request) (*http.Response, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

func TestBuilderHeadersRemoteFallbackLocalOnRequestError(t *testing.T) {
	cfg := &BuilderConfig{
		Local: &BuilderCredentials{
			Key:        "local-key",
			Secret:     "c2VjcmV0", // base64("secret")
			Passphrase: "local-pass",
		},
		Remote: &BuilderRemoteConfig{
			Host:       "https://remote-signer.test/sign",
			HTTPClient: fakeHTTPDoer{err: errors.New("remote down")},
		},
		RemoteFallbackLocal: true,
	}

	headers, err := cfg.Headers(context.Background(), http.MethodPost, "/v2/order", nil, 0)
	if err != nil {
		t.Fatalf("expected local fallback success, got error: %v", err)
	}
	if got := headers.Get(HeaderPolyBuilderAPIKey); got != "local-key" {
		t.Fatalf("expected local key, got %q", got)
	}
	if got := headers.Get(HeaderPolyBuilderPassphrase); got != "local-pass" {
		t.Fatalf("expected local passphrase, got %q", got)
	}
	if headers.Get(HeaderPolyBuilderSignature) == "" || headers.Get(HeaderPolyBuilderTimestamp) == "" {
		t.Fatalf("expected local signature and timestamp")
	}
}

func TestBuilderHeadersRemoteFallbackLocalPrefersRemoteWhenHealthy(t *testing.T) {
	cfg := &BuilderConfig{
		Local: &BuilderCredentials{
			Key:        "local-key",
			Secret:     "c2VjcmV0",
			Passphrase: "local-pass",
		},
		Remote: &BuilderRemoteConfig{
			Host: "https://remote-signer.test/sign",
			HTTPClient: fakeHTTPDoer{
				resp: &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(`{
						"POLY_BUILDER_API_KEY":"remote-key",
						"POLY_BUILDER_PASSPHRASE":"remote-pass",
						"POLY_BUILDER_SIGNATURE":"remote-sig",
						"POLY_BUILDER_TIMESTAMP":"1730000000000"
					}`)),
				},
			},
		},
		RemoteFallbackLocal: true,
	}

	headers, err := cfg.Headers(context.Background(), http.MethodPost, "/v2/order", nil, 0)
	if err != nil {
		t.Fatalf("expected remote success, got error: %v", err)
	}
	if got := headers.Get(HeaderPolyBuilderAPIKey); got != "remote-key" {
		t.Fatalf("expected remote key, got %q", got)
	}
	if got := headers.Get(HeaderPolyBuilderPassphrase); got != "remote-pass" {
		t.Fatalf("expected remote passphrase, got %q", got)
	}
}

func TestBuilderHeadersRemoteFallbackLocalReturnsCombinedErrorWhenBothFail(t *testing.T) {
	cfg := &BuilderConfig{
		Local: &BuilderCredentials{
			Key:        "local-key",
			Secret:     "!!!",
			Passphrase: "local-pass",
		},
		Remote: &BuilderRemoteConfig{
			Host:       "https://remote-signer.test/sign",
			HTTPClient: fakeHTTPDoer{err: errors.New("remote down")},
		},
		RemoteFallbackLocal: true,
	}

	_, err := cfg.Headers(context.Background(), http.MethodPost, "/v2/order", nil, 0)
	if err == nil {
		t.Fatalf("expected error when remote and local both fail")
	}
	if !strings.Contains(err.Error(), "remote fallback local") {
		t.Fatalf("expected combined fallback error, got %v", err)
	}
}
