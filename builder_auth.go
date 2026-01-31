package relayer

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/GoPolymarket/go-builder-relayer-client/pkg/types"
)

const (
	// #nosec G101 -- header names, not hardcoded credentials.
	HeaderPolyBuilderAPIKey = "POLY_BUILDER_API_KEY"
	// #nosec G101 -- header names, not hardcoded credentials.
	HeaderPolyBuilderPassphrase = "POLY_BUILDER_PASSPHRASE"
	HeaderPolyBuilderSignature  = "POLY_BUILDER_SIGNATURE"
	HeaderPolyBuilderTimestamp  = "POLY_BUILDER_TIMESTAMP"
)

const builderTimestampMillisThreshold = int64(1_000_000_000_000)

func normalizeBuilderTimestamp(timestamp int64) int64 {
	if timestamp == 0 {
		return time.Now().UnixMilli()
	}
	if timestamp < builderTimestampMillisThreshold {
		return timestamp * 1000
	}
	return timestamp
}

// BuilderCredentials represents builder attribution credentials.
type BuilderCredentials struct {
	Key        string
	Secret     string
	Passphrase string
}

// BuilderHTTPDoer executes HTTP requests for remote signing.
type BuilderHTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// BuilderRemoteConfig configures a remote signing service.
type BuilderRemoteConfig struct {
	Host       string
	Token      string
	HTTPClient BuilderHTTPDoer
}

// BuilderConfig holds configuration for local or remote builder attribution.
type BuilderConfig struct {
	Local  *BuilderCredentials
	Remote *BuilderRemoteConfig
}

// IsValid returns true if the configuration has sufficient credentials.
func (c *BuilderConfig) IsValid() bool {
	if c == nil {
		return false
	}
	if c.Local != nil {
		return c.Local.Key != "" && c.Local.Secret != "" && c.Local.Passphrase != ""
	}
	if c.Remote != nil {
		return c.Remote.Host != ""
	}
	return false
}

// Headers returns the attribution headers for a given request.
func (c *BuilderConfig) Headers(ctx context.Context, method, path string, body *string, timestamp int64) (http.Header, error) {
	if c == nil {
		return nil, types.ErrMissingBuilderConfig
	}
	if c.Local != nil {
		return buildBuilderHeadersLocal(c.Local, method, path, body, timestamp)
	}
	if c.Remote != nil {
		return buildBuilderHeadersRemote(ctx, c.Remote, method, path, body, timestamp)
	}
	return nil, types.ErrMissingBuilderConfig
}

func buildBuilderHeadersLocal(creds *BuilderCredentials, method, path string, body *string, timestamp int64) (http.Header, error) {
	if creds == nil || creds.Key == "" || creds.Secret == "" || creds.Passphrase == "" {
		return nil, types.ErrMissingBuilderConfig
	}
	timestamp = normalizeBuilderTimestamp(timestamp)
	message := fmt.Sprintf("%d%s%s", timestamp, method, path)
	if body != nil && *body != "" {
		message += *body
	}
	// HMAC signature
	sig, err := SignHMAC(creds.Secret, message)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Set(HeaderPolyBuilderAPIKey, creds.Key)
	headers.Set(HeaderPolyBuilderPassphrase, creds.Passphrase)
	headers.Set(HeaderPolyBuilderTimestamp, fmt.Sprintf("%d", timestamp))
	headers.Set(HeaderPolyBuilderSignature, sig)
	return headers, nil
}

func buildBuilderHeadersRemote(ctx context.Context, remote *BuilderRemoteConfig, method, path string, body *string, timestamp int64) (http.Header, error) {
	if remote == nil || remote.Host == "" {
		return nil, types.ErrMissingBuilderConfig
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var normalizedTimestamp int64
	if timestamp != 0 {
		normalizedTimestamp = normalizeBuilderTimestamp(timestamp)
	}
	payload := map[string]interface{}{
		"method": method,
		"path":   path,
		"body":   "",
	}
	if body != nil {
		payload["body"] = *body
	}
	if normalizedTimestamp != 0 {
		payload["timestamp"] = normalizedTimestamp
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal builder payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, remote.Host, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("builder request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if remote.Token != "" {
		req.Header.Set("Authorization", "Bearer "+remote.Token)
	}

	client := remote.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("builder request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("builder signer error: status %d", resp.StatusCode)
	}

	var rawHeaders map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&rawHeaders); err != nil {
		return nil, fmt.Errorf("decode builder headers: %w", err)
	}

	get := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := rawHeaders[k]; ok && v != "" {
				return v
			}
		}
		return ""
	}

	builderKey := get(HeaderPolyBuilderAPIKey, "poly_builder_api_key", "POLY_BUILDER_API_KEY")
	builderPass := get(HeaderPolyBuilderPassphrase, "poly_builder_passphrase", "POLY_BUILDER_PASSPHRASE")
	builderSig := get(HeaderPolyBuilderSignature, "poly_builder_signature", "POLY_BUILDER_SIGNATURE")
	builderTs := get(HeaderPolyBuilderTimestamp, "poly_builder_timestamp", "POLY_BUILDER_TIMESTAMP")

	if builderKey == "" || builderPass == "" || builderSig == "" || builderTs == "" {
		return nil, fmt.Errorf("invalid builder headers response")
	}

	headers := http.Header{}
	headers.Set(HeaderPolyBuilderAPIKey, builderKey)
	headers.Set(HeaderPolyBuilderPassphrase, builderPass)
	headers.Set(HeaderPolyBuilderSignature, builderSig)
	headers.Set(HeaderPolyBuilderTimestamp, builderTs)
	return headers, nil
}

// SignHMAC calculates the HMAC-SHA256 signature used for builder attribution.
func SignHMAC(secret string, message string) (string, error) {
	decodedSecret, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}

	h := hmac.New(sha256.New, decodedSecret)
	h.Write([]byte(message))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return signature, nil
}

func decodeSecret(secret string) ([]byte, error) {
	decoded, err := base64.URLEncoding.DecodeString(secret)
	if err == nil {
		return decoded, nil
	}
	decoded, err = base64.RawURLEncoding.DecodeString(secret)
	if err == nil {
		return decoded, nil
	}
	decoded, err = base64.StdEncoding.DecodeString(secret)
	if err == nil {
		return decoded, nil
	}
	decoded, err = base64.RawStdEncoding.DecodeString(secret)
	if err == nil {
		return decoded, nil
	}
	return nil, fmt.Errorf("invalid base64 secret: %w", err)
}
