# Phase 2 Performance & Maintainability Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Improve runtime stability, retry/polling performance characteristics, and long-term maintainability while increasing test coverage in critical paths.

**Architecture:** Keep public SDK APIs backward-compatible, introduce optional advanced configuration for HTTP and polling behavior, and expand deterministic unit-test coverage for retry logic, remote signing, encoding, and signer validation. Use incremental refactors with TDD and avoid broad rewrites.

**Tech Stack:** Go 1.24, `testing`, `testify`, `net/http`, GitHub Actions

---

### Task 1: Introduce explicit HTTP client configuration for performance tuning

**Files:**
- Modify: `http_client.go`
- Test: `http_client_config_test.go`
- Modify: `README.md`

**Step 1: Write the failing test**

```go
func TestNewHTTPClientWithConfig_AppliesTimeoutAndTransport(t *testing.T) {
    cfg := HTTPClientConfig{Timeout: 15 * time.Second}
    c := NewHTTPClientWithConfig(nil, cfg)
    require.NotNil(t, c)
    assert.Equal(t, 15*time.Second, c.client.Timeout)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./... -run TestNewHTTPClientWithConfig`
Expected: FAIL because `HTTPClientConfig` and `NewHTTPClientWithConfig` do not exist yet.

**Step 3: Write minimal implementation**

```go
type HTTPClientConfig struct {
    Timeout time.Duration
    MaxIdleConns int
    MaxIdleConnsPerHost int
    IdleConnTimeout time.Duration
}

func NewHTTPClientWithConfig(base *http.Client, cfg HTTPClientConfig) *HTTPClient { /* ... */ }
```

**Step 4: Run test to verify it passes**

Run: `go test ./... -run TestNewHTTPClientWithConfig`
Expected: PASS.

**Step 5: Commit**

```bash
git add http_client.go http_client_config_test.go README.md
git commit -m "feat: add configurable HTTP client performance options"
```

### Task 2: Upgrade retry strategy with jitter and total retry budget

**Files:**
- Modify: `http_client.go`
- Test: `http_client_retry_policy_test.go`

**Step 1: Write the failing tests**

```go
func TestHTTPClientDo_StopsWhenRetryBudgetExceeded(t *testing.T) { /* ... */ }
func TestHTTPClientDo_UsesJitteredDelayWithinBounds(t *testing.T) { /* ... */ }
```

**Step 2: Run test to verify it fails**

Run: `go test ./... -run TestHTTPClientDo_.*Retry`
Expected: FAIL because retry budget and jitter controls are not implemented.

**Step 3: Write minimal implementation**

```go
type RetryPolicy struct {
    MaxRetries uint
    BaseDelay time.Duration
    MaxDelay time.Duration
    JitterRatio float64
    MaxElapsed time.Duration
}
```
- Add deterministic jitter source for testability.
- Keep 429/5xx behavior from phase 1 and route through the new policy.

**Step 4: Run test to verify it passes**

Run: `go test ./... -run TestHTTPClientDo_.*Retry`
Expected: PASS.

**Step 5: Commit**

```bash
git add http_client.go http_client_retry_policy_test.go
git commit -m "feat: add jittered retry policy and retry budget controls"
```

### Task 3: Add configurable polling backoff for `PollUntilState`

**Files:**
- Modify: `client.go`
- Test: `client_poll_backoff_test.go`
- Modify: `README.md`

**Step 1: Write the failing tests**

```go
func TestPollUntilState_UsesExponentialBackoffIntervals(t *testing.T) { /* ... */ }
func TestPollUntilState_RespectsMaxIntervalCap(t *testing.T) { /* ... */ }
```

**Step 2: Run test to verify it fails**

Run: `go test ./... -run TestPollUntilState_.*Backoff`
Expected: FAIL because polling currently uses fixed interval behavior.

**Step 3: Write minimal implementation**

```go
type PollPolicy struct {
    MaxPolls int
    InitialInterval time.Duration
    MaxInterval time.Duration
    Multiplier float64
    JitterRatio float64
}
```
- Keep existing `PollUntilState` signature for compatibility.
- Add optional policy path (`SetPollPolicy` or internal default policy override).

**Step 4: Run test to verify it passes**

Run: `go test ./... -run TestPollUntilState_.*Backoff`
Expected: PASS.

**Step 5: Commit**

```bash
git add client.go client_poll_backoff_test.go README.md
git commit -m "feat: add configurable polling backoff policy"
```

### Task 4: Refactor remote signer request/response to typed models and richer errors

**Files:**
- Modify: `builder_auth.go`
- Test: `builder_auth_remote_test.go`

**Step 1: Write the failing tests**

```go
func TestBuildBuilderHeadersRemote_ParsesTypedResponse(t *testing.T) { /* ... */ }
func TestBuildBuilderHeadersRemote_ReturnsStatusAndBodyOnError(t *testing.T) { /* ... */ }
```

**Step 2: Run test to verify it fails**

Run: `go test ./... -run TestBuildBuilderHeadersRemote`
Expected: FAIL because response parsing/error handling is loosely typed.

**Step 3: Write minimal implementation**

```go
type remoteSignRequest struct {
    Method string `json:"method"`
    Path string `json:"path"`
    Body string `json:"body"`
    Timestamp *int64 `json:"timestamp,omitempty"`
}

type remoteSignResponse struct {
    PolyBuilderAPIKey string `json:"POLY_BUILDER_API_KEY"`
    PolyBuilderPassphrase string `json:"POLY_BUILDER_PASSPHRASE"`
    PolyBuilderSignature string `json:"POLY_BUILDER_SIGNATURE"`
    PolyBuilderTimestamp string `json:"POLY_BUILDER_TIMESTAMP"`
}
```
- Return remote signer status + body excerpt in non-2xx errors.

**Step 4: Run test to verify it passes**

Run: `go test ./... -run TestBuildBuilderHeadersRemote`
Expected: PASS.

**Step 5: Commit**

```bash
git add builder_auth.go builder_auth_remote_test.go
git commit -m "refactor: type remote signer payload and improve error diagnostics"
```

### Task 5: Harden signer input validation and panic-safety

**Files:**
- Modify: `pkg/signer/signer.go`
- Test: `pkg/signer/signer_test.go`

**Step 1: Write the failing tests**

```go
func TestSignTypedData_RejectsNilDomain(t *testing.T) { /* ... */ }
func TestSignTypedData_RejectsEmptyPrimaryType(t *testing.T) { /* ... */ }
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/signer -run TestSignTypedData`
Expected: FAIL because current code dereferences domain directly.

**Step 3: Write minimal implementation**

```go
if domain == nil {
    return nil, errors.New("domain is required")
}
if primaryType == "" {
    return nil, errors.New("primaryType is required")
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/signer -run TestSignTypedData`
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/signer/signer.go pkg/signer/signer_test.go
git commit -m "fix: add signer typed-data input validation"
```

### Task 6: Fill test coverage gaps in utility and encoder internals

**Files:**
- Test: `internal/utils/utils_test.go`
- Test: `internal/encoder/encode_safe_test.go`
- Test: `internal/encoder/encode_proxy_test.go`

**Step 1: Write failing tests for edge cases**

```go
func TestParseBigInt_Invalid(t *testing.T) { /* ... */ }
func TestDecodeHex_Handles0xAndPlain(t *testing.T) { /* ... */ }
func TestSplitAndPackSig_InvalidV(t *testing.T) { /* ... */ }
func TestEncodeProxyTransactionData_InvalidValue(t *testing.T) { /* ... */ }
func TestCreateSafeMultisendTransaction_InvalidData(t *testing.T) { /* ... */ }
```

**Step 2: Run test to verify failures**

Run: `go test ./internal/utils ./internal/encoder`
Expected: Initial FAIL on unhandled edge-path expectations.

**Step 3: Write minimal implementation/fixes if needed**

- Adjust error messages/guards only where test proves a true bug.
- Do not refactor behavior unnecessarily.

**Step 4: Re-run tests**

Run: `go test ./internal/utils ./internal/encoder`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/utils/utils_test.go internal/encoder/encode_safe_test.go internal/encoder/encode_proxy_test.go
git commit -m "test: add edge-case coverage for utils and encoders"
```

### Task 7: Clean up CI maintainability and make coverage checks deterministic

**Files:**
- Modify: `.github/workflows/go.yml`
- Create: `scripts/check-coverage.sh`
- Modify: `README.md`

**Step 1: Write failing CI/local script test path**

- Add a local dry-run command that fails if coverage profile is missing.

```bash
./scripts/check-coverage.sh /tmp/does-not-exist.out
```

Expected: FAIL with clear message.

**Step 2: Implement deterministic coverage checker**

```bash
#!/usr/bin/env bash
set -euo pipefail
profile="${1:-coverage.out}"
threshold="${2:-45.0}"
# parse go tool cover -func output and compare
```

**Step 3: Update workflow to use the new script**

- Remove invalid `matrix.go-version` condition.
- Stop using `continue-on-error: true` for core test job.
- Test full module (`go test -race -coverprofile=coverage.out ./...`).

**Step 4: Run verification locally**

Run: `go test -race -coverprofile=coverage.out ./... && ./scripts/check-coverage.sh coverage.out 45.0`
Expected: PASS.

**Step 5: Commit**

```bash
git add .github/workflows/go.yml scripts/check-coverage.sh README.md
git commit -m "ci: stabilize go workflow and deterministic coverage checks"
```

### Task 8: End-to-end verification and release notes prep

**Files:**
- Modify: `README.md`
- Modify: `docs/builder-auth.md`

**Step 1: Full suite verification**

Run: `go test ./...`
Expected: PASS.

**Step 2: Race verification**

Run: `go test -race ./...`
Expected: PASS.

**Step 3: Coverage verification**

Run: `go test -cover ./...`
Expected: PASS with improved root/internal package coverage.

**Step 4: Documentation updates**

- Add section for advanced HTTP/polling tuning.
- Add remote signer error troubleshooting guidance.

**Step 5: Commit**

```bash
git add README.md docs/builder-auth.md
git commit -m "docs: add phase2 tuning and troubleshooting guidance"
```
