# Agent Team Optimization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Improve reliability and test coverage for relayer request/retry/polling behavior while preserving existing public APIs.

**Architecture:** Keep the SDK API stable but harden internal behavior in three focused areas: transient HTTP retries, context propagation in proxy gas estimation, and cancellation-aware polling sleeps. Add targeted unit tests for each change first (RED), then minimal implementation (GREEN), then cleanup.

**Tech Stack:** Go 1.24, `net/http`, `testing`, `testify`

---

### Task 1: Retry transient 429 responses with `Retry-After`

**Files:**
- Modify: `http_client.go`
- Test: `http_client_test.go`

**Step 1: Write the failing tests**

Add tests for:
1. 429 response followed by 200 should retry and succeed.
2. 429 response with invalid `Retry-After` should fall back to exponential delay and still retry.
3. 429 exhaustion should return `HTTPError` with status 429.

**Step 2: Run test to verify it fails**

Run: `go test ./... -run TestHTTPClientDo`
Expected: FAIL because 429 is currently treated as non-retry client error.

**Step 3: Write minimal implementation**

In `HTTPClient.Do`:
- Treat 429 as retryable like 5xx.
- Parse `Retry-After` (seconds or HTTP-date) into a delay.
- Keep existing behavior for other 4xx (do not retry).

**Step 4: Run test to verify it passes**

Run: `go test ./... -run TestHTTPClientDo`
Expected: PASS.

**Step 5: Commit**

```bash
git add http_client.go http_client_test.go
git commit -m "test: add and implement retry logic for 429 responses"
```

### Task 2: Propagate request context into proxy gas estimation

**Files:**
- Modify: `internal/builder/builder_proxy.go`
- Modify: `client.go`
- Test: `client_proxy_context_test.go`

**Step 1: Write the failing test**

Add integration-style test ensuring `Execute` propagates request context through `BuildProxyTransactionRequest` into signer `EstimateGas` (use a context value and fake signer).

**Step 2: Run test to verify it fails**

Run: `go test ./... -run TestExecuteProxy_PropagatesRequestContextToGasEstimate`
Expected: FAIL because current implementation uses `context.Background()` when estimating proxy gas.

**Step 3: Write minimal implementation**

- Change `BuildProxyTransactionRequest` signature to accept `context.Context`.
- Pass the caller context to `getGasLimit`.
- Update call site in `executeProxyTransactions` to pass `ctx`.

**Step 4: Run test to verify it passes**

Run: `go test ./... -run TestExecuteProxy_PropagatesRequestContextToGasEstimate`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/builder/builder_proxy.go client_proxy_context_test.go client.go
git commit -m "test: propagate execute context to proxy gas estimation"
```

### Task 3: Make polling sleep cancellation-aware and skip final unnecessary sleep

**Files:**
- Modify: `client.go`
- Test: `client_poll_test.go`

**Step 1: Write the failing tests**

Add tests for:
1. `PollUntilState` with `maxPolls=1` does not sleep after last poll.
2. `PollUntilState` exits promptly when context is canceled during sleep.

**Step 2: Run test to verify it fails**

Run: `go test ./... -run TestPollUntilState`
Expected: FAIL because polling always sleeps after each iteration and uses non-cancelable sleep.

**Step 3: Write minimal implementation**

- Add a `sleepFn` to `RelayClient` for testability.
- Default sleep implementation should select on `ctx.Done()` vs timer.
- In `PollUntilState`, sleep only when another poll attempt remains.

**Step 4: Run test to verify it passes**

Run: `go test ./... -run TestPollUntilState`
Expected: PASS.

**Step 5: Commit**

```bash
git add client.go client_poll_test.go
git commit -m "test: make poll loop cancellation-aware and avoid extra delay"
```

### Task 4: Full verification

**Files:**
- Verify all touched files

**Step 1: Run complete suite**

Run: `go test ./...`
Expected: PASS all packages.

**Step 2: Optional static checks**

Run: `go test -race ./...`
Expected: PASS (if runtime permits).

**Step 3: Final review**

Ensure behavior changes are documented in PR notes and tests clearly describe expected semantics.

**Step 4: Commit final polish if needed**

```bash
git add -A
git commit -m "refactor: improve relayer retry/polling robustness with tests"
```
