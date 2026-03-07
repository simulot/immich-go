# Retry & Resilience Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `--on-errors=retry` mode that retries transient server errors with exponential backoff and adapts upload concurrency under sustained load.

**Architecture:** Three layers — (1) HTTP-level retry in `immich/call.go` for non-upload API calls, (2) upload-level retry in `immich/upload.go` that re-opens files and rebuilds multipart streams, (3) adaptive concurrency throttle in `app/upload/run.go` that halves on failure and recovers on success.

**Tech Stack:** Go stdlib (`net/http`, `sync`, `time`, `math/rand`), existing `immich/call.go` server call framework, existing `internal/worker` pool.

**Design doc:** `docs/plans/2026-03-07-retry-resilience-design.md`

---

### Task 1: Add `OnErrorsRetry` to the CLI flag

**Files:**
- Modify: `internal/cliFlags/orErrors.go`
- Test: `internal/cliFlags/orErrors_test.go` (create)

**Step 1: Write the test**

Create `internal/cliFlags/orErrors_test.go`:

```go
package cliflags

import "testing"

func TestOnErrorsFlag_SetAndString(t *testing.T) {
	tests := []struct {
		input    string
		expected OnErrorsFlag
		str      string
		wantErr  bool
	}{
		{"stop", OnErrorsStop, "stop", false},
		{"continue", OnErrorsNeverStop, "continue", false},
		{"retry", OnErrorsRetry, "retry", false},
		{"5", OnErrorsFlag(5), "5", false},
		{"invalid", 0, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var f OnErrorsFlag
			err := f.Set(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if f != tt.expected {
				t.Errorf("got %v, want %v", f, tt.expected)
			}
			if f.String() != tt.str {
				t.Errorf("String() = %q, want %q", f.String(), tt.str)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cliFlags/ -run TestOnErrorsFlag -v`
Expected: FAIL — `OnErrorsRetry` undefined

**Step 3: Implement**

Modify `internal/cliFlags/orErrors.go`:

1. Add constant (line 18, before the closing paren):
```go
OnErrorsRetry    = -2
```

2. Add case in `String()` (after line 29, the `OnErrorsNeverStop` case):
```go
case f == OnErrorsRetry:
    return "retry"
```

3. Add case in `Set()` (after the `"continue"` case, line 43):
```go
case "retry":
    *f = OnErrorsRetry
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cliFlags/ -run TestOnErrorsFlag -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cliFlags/orErrors.go internal/cliFlags/orErrors_test.go
git commit -m "feat: add --on-errors=retry flag value"
```

---

### Task 2: Create retry helpers (`immich/retry.go`)

**Files:**
- Create: `immich/retry.go`
- Test: `immich/retry_test.go` (create)

**Step 1: Write the tests**

Create `immich/retry_test.go`:

```go
package immich

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
	"testing"
	"time"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"500 error", callError{status: http.StatusInternalServerError}, true},
		{"502 error", callError{status: http.StatusBadGateway}, true},
		{"503 error", callError{status: http.StatusServiceUnavailable}, true},
		{"504 error", callError{status: http.StatusGatewayTimeout}, true},
		{"408 error", callError{status: http.StatusRequestTimeout}, true},
		{"429 error", callError{status: http.StatusTooManyRequests}, true},
		{"400 error", callError{status: http.StatusBadRequest}, false},
		{"401 error", callError{status: http.StatusUnauthorized}, false},
		{"403 error", callError{status: http.StatusForbidden}, false},
		{"404 error", callError{status: http.StatusNotFound}, false},
		{"409 error", callError{status: http.StatusConflict}, false},
		{"connection refused", fmt.Errorf("connect: %w", syscall.ECONNREFUSED), true},
		{"connection reset", fmt.Errorf("read: %w", syscall.ECONNRESET), true},
		{"EOF", io.EOF, true},
		{"unexpected EOF", io.ErrUnexpectedEOF, true},
		{"net timeout", netTimeoutError{}, true},
		{"wrapped retryable", fmt.Errorf("outer: %w", callError{status: 503}), true},
		{"generic error", errors.New("something"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryable(tt.err)
			if got != tt.expected {
				t.Errorf("isRetryable(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

// netTimeoutError implements net.Error with Timeout() = true
type netTimeoutError struct{}

func (e netTimeoutError) Error() string   { return "timeout" }
func (e netTimeoutError) Timeout() bool   { return true }
func (e netTimeoutError) Temporary() bool { return true }

func TestRetryDelay(t *testing.T) {
	// Attempt 0: ~1s, Attempt 1: ~2s, Attempt 2: ~4s
	for attempt := 0; attempt < 3; attempt++ {
		d := retryDelay(attempt, "")
		base := time.Duration(1<<uint(attempt)) * time.Second
		// Allow jitter: delay should be between base and base + 1s
		if d < base || d > base+time.Second {
			t.Errorf("attempt %d: delay %v outside expected range [%v, %v]", attempt, d, base, base+time.Second)
		}
	}
}

func TestRetryDelay_RetryAfter(t *testing.T) {
	d := retryDelay(0, "5")
	if d < 5*time.Second || d > 6*time.Second {
		t.Errorf("with Retry-After=5, got %v, expected ~5s", d)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./immich/ -run "TestIsRetryable|TestRetryDelay" -v`
Expected: FAIL — `isRetryable` undefined

**Step 3: Implement**

Create `immich/retry.go`:

```go
package immich

import (
	"errors"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"syscall"
	"time"
)

const maxRetries = 3

// retryableStatusCodes is the set of HTTP status codes that are safe to retry.
var retryableStatusCodes = map[int]bool{
	http.StatusRequestTimeout:      true, // 408
	http.StatusTooManyRequests:      true, // 429
	http.StatusInternalServerError:  true, // 500
	http.StatusBadGateway:           true, // 502
	http.StatusServiceUnavailable:   true, // 503
	http.StatusGatewayTimeout:       true, // 504
}

// isRetryable returns true if the error is a transient server or network error
// that is safe to retry. Permission errors (401, 403) and client errors (400, 404, 409)
// are NOT retryable.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for callError with retryable status code
	var ce callError
	if errors.As(err, &ce) {
		return retryableStatusCodes[ce.status]
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// Connection refused / reset
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}

	// EOF errors (server dropped connection)
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	return false
}

// retryDelay calculates the delay before the next retry attempt.
// Uses exponential backoff (1s, 2s, 4s) with random jitter (0-1s).
// If retryAfter is a valid number of seconds (from Retry-After header),
// that value is used as the base instead.
func retryDelay(attempt int, retryAfter string) time.Duration {
	base := time.Duration(1<<uint(attempt)) * time.Second

	if retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
			base = time.Duration(seconds) * time.Second
		}
	}

	jitter := time.Duration(rand.Int64N(int64(time.Second)))
	return base + jitter
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./immich/ -run "TestIsRetryable|TestRetryDelay" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add immich/retry.go immich/retry_test.go
git commit -m "feat: add retry helpers for transient error detection and backoff"
```

---

### Task 3: Add retry to `serverCall.do()` for non-upload API calls

**Files:**
- Modify: `immich/call.go`
- Modify: `immich/client.go`
- Test: `immich/call_test.go` (extend)

**Step 1: Write the test**

Add to `immich/call_test.go`:

```go
func TestCallRetry_TransientError(t *testing.T) {
	attempts := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"Service Unavailable","statusCode":503,"message":"overloaded"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	ic, err := NewImmichClient(server.URL, "key")
	if err != nil {
		t.Fatal(err)
	}
	ic.RetryEnabled = true

	r := map[string]string{}
	err = ic.newServerCall(context.Background(), "test").
		do(getRequest("/test", setAcceptJSON()), responseJSON(&r))
	if err != nil {
		t.Errorf("expected success after retries, got error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestCallRetry_NonRetryableError(t *testing.T) {
	attempts := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"Forbidden","statusCode":403,"message":"no access"}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	ic, err := NewImmichClient(server.URL, "key")
	if err != nil {
		t.Fatal(err)
	}
	ic.RetryEnabled = true

	r := map[string]string{}
	err = ic.newServerCall(context.Background(), "test").
		do(getRequest("/test", setAcceptJSON()), responseJSON(&r))
	if err == nil {
		t.Error("expected error for 403, got nil")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt (no retry for 403), got %d", attempts)
	}
}

func TestCallRetry_Disabled(t *testing.T) {
	attempts := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"Service Unavailable","statusCode":503,"message":"overloaded"}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	ic, err := NewImmichClient(server.URL, "key")
	if err != nil {
		t.Fatal(err)
	}
	// RetryEnabled defaults to false

	r := map[string]string{}
	err = ic.newServerCall(context.Background(), "test").
		do(getRequest("/test", setAcceptJSON()), responseJSON(&r))
	if err == nil {
		t.Error("expected error, got nil")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt (retry disabled), got %d", attempts)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./immich/ -run "TestCallRetry" -v`
Expected: FAIL — `RetryEnabled` not defined

**Step 3: Implement**

3a. Add `RetryEnabled` field to `immich/client.go` (after line 27):

```go
RetryEnabled   bool          // If true, retry transient server errors
```

3b. Replace the `do()` method in `immich/call.go` (lines 220-266) with retry-aware version:

```go
func (sc *serverCall) do(fnRequest requestFunction, opts ...serverResponseOption) error {
	maxAttempts := 1
	if sc.ic.RetryEnabled {
		maxAttempts = maxRetries
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := retryDelay(attempt-1, "")
			// Extract Retry-After from previous callError if available
			var ce callError
			if errors.As(lastErr, &ce) && ce.status == 429 && ce.message != nil {
				// Note: Retry-After from header isn't in callError currently,
				// but the delay function handles the default well.
			}
			sc.ic.log("WARN", "retrying API call", "endpoint", sc.endPoint, "attempt", attempt+1, "delay", delay, "error", lastErr)
			select {
			case <-time.After(delay):
			case <-sc.ctx.Done():
				return sc.ctx.Err()
			}
			// Reset error state for retry
			sc.err = nil
		}

		lastErr = sc.doOnce(fnRequest, opts...)
		if lastErr == nil {
			return nil
		}
		if !isRetryable(lastErr) {
			return lastErr
		}
	}
	return lastErr
}
```

Wait — the `ImmichClient` doesn't have a `log` method. Let me rethink. The existing code doesn't log from the immich layer. We should keep retries silent at this layer and let the caller see the final error. But we need some way to observe retries for debugging.

Actually, looking at the codebase more carefully, the immich client has `apiTraceWriter`. Let's keep it simpler — just log via `fmt.Fprintf` to stderr or skip logging at this layer entirely. The retry is transparent.

Revised `do()`:

```go
func (sc *serverCall) do(fnRequest requestFunction, opts ...serverResponseOption) error {
	maxAttempts := 1
	if sc.ic.RetryEnabled {
		maxAttempts = maxRetries
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := retryDelay(attempt-1, "")
			select {
			case <-time.After(delay):
			case <-sc.ctx.Done():
				return sc.ctx.Err()
			}
			// Reset error state for retry
			sc.err = nil
		}

		lastErr = sc.doOnce(fnRequest, opts...)
		if lastErr == nil {
			return nil
		}
		if !isRetryable(lastErr) {
			return lastErr
		}
	}
	return lastErr
}
```

Extract the current `do()` body into `doOnce()`:

```go
func (sc *serverCall) doOnce(fnRequest requestFunction, opts ...serverResponseOption) error {
	var (
		resp *http.Response
		err  error
	)

	req := fnRequest(sc)
	if sc.err != nil || req == nil {
		return sc.Err(req, nil, nil)
	}

	resp, err = sc.ic.client.Do(req)
	if err != nil {
		sc.err = err
		return sc.Err(req, nil, nil)
	}

	if resp.StatusCode >= 300 {
		msg := ServerErrorMessage{}
		if resp.Body != nil {
			defer resp.Body.Close()
			if isJSON(resp.Header.Get("Content-Type")) {
				if json.NewDecoder(resp.Body).Decode(&msg) == nil {
					return sc.Err(req, resp, &msg)
				}
			}
		}
		return sc.Err(req, resp, &msg)
	}

	for _, opt := range opts {
		if opt != nil {
			_ = sc.joinError(opt(sc, resp))
		}
	}
	if !sc.hasResponseHandler && resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	if sc.err != nil {
		return sc.Err(req, resp, nil)
	}
	return nil
}
```

3c. Add `"time"` and `"errors"` imports to `call.go` if not already present. (`errors` is already imported; `time` needs to be added.)

**Step 4: Run test to verify it passes**

Run: `go test ./immich/ -run "TestCallRetry|TestCall$" -v`
Expected: PASS (both old and new tests)

**Step 5: Commit**

```bash
git add immich/call.go immich/client.go immich/call_test.go
git commit -m "feat: add retry logic to HTTP client for transient server errors"
```

---

### Task 4: Add retry to `uploadAsset()` for file uploads

**Files:**
- Modify: `immich/upload.go`

**Step 1: No isolated test for this** — upload retry requires a full multipart upload flow. We'll verify via integration testing on kapara. The key change is wrapping the upload body in a retry loop.

**Step 2: Implement**

Replace the `uploadAsset` function in `immich/upload.go` (lines 35-120) with a retry-aware version. The retry loop wraps everything from `la.OpenFile()` through the `do()` call, because the `io.Pipe()` body is consumed on each attempt.

```go
func (ic *ImmichClient) uploadAsset(ctx context.Context, la *assets.Asset, endPoint string, replaceID string) (AssetResponse, error) {
	if ic.dryRun {
		return AssetResponse{
			ID:     uuid.NewString(),
			Status: UploadCreated,
		}, nil
	}

	maxAttempts := 1
	if ic.RetryEnabled {
		maxAttempts = maxRetries
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := retryDelay(attempt-1, "")
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return AssetResponse{}, ctx.Err()
			}
		}

		ar, err := ic.uploadAssetOnce(ctx, la, endPoint, replaceID)
		if err == nil {
			return ar, nil
		}
		if !isRetryable(err) {
			return ar, err
		}
		lastErr = err
	}
	return AssetResponse{}, lastErr
}
```

Rename the current `uploadAsset` body to `uploadAssetOnce`:

```go
func (ic *ImmichClient) uploadAssetOnce(ctx context.Context, la *assets.Asset, endPoint string, replaceID string) (AssetResponse, error) {
	// ... existing body from line 43 onwards, unchanged ...
}
```

Add `"time"` import to `upload.go`.

**Step 3: Verify build**

Run: `go build ./...`
Expected: Success

**Step 4: Run all existing tests**

Run: `go test ./immich/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add immich/upload.go
git commit -m "feat: add retry loop to asset uploads with file re-open"
```

---

### Task 5: Create adaptive concurrency throttle

**Files:**
- Create: `internal/worker/throttle.go`
- Test: `internal/worker/throttle_test.go` (create)

**Step 1: Write the test**

Create `internal/worker/throttle_test.go`:

```go
package worker

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestThrottle_BasicConcurrency(t *testing.T) {
	th := NewThrottle(3)
	defer th.Close()

	var running atomic.Int32
	var maxSeen atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			th.Acquire()
			cur := running.Add(1)
			for {
				old := maxSeen.Load()
				if cur <= old || maxSeen.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			running.Add(-1)
			th.Release()
		}()
	}

	wg.Wait()
	if maxSeen.Load() > 3 {
		t.Errorf("max concurrent = %d, want <= 3", maxSeen.Load())
	}
}

func TestThrottle_SetConcurrency(t *testing.T) {
	th := NewThrottle(10)
	defer th.Close()

	if th.Current() != 10 {
		t.Errorf("initial concurrency = %d, want 10", th.Current())
	}

	th.SetConcurrency(5)
	if th.Current() != 5 {
		t.Errorf("after SetConcurrency(5) = %d, want 5", th.Current())
	}

	th.SetConcurrency(0) // should clamp to 1
	if th.Current() != 1 {
		t.Errorf("after SetConcurrency(0) = %d, want 1", th.Current())
	}

	th.SetConcurrency(20)
	if th.Current() != 20 {
		t.Errorf("after SetConcurrency(20) = %d, want 20", th.Current())
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/worker/ -run "TestThrottle" -v`
Expected: FAIL — `NewThrottle` undefined

**Step 3: Implement**

Create `internal/worker/throttle.go`:

```go
package worker

import "sync"

// Throttle controls the level of concurrency using a channel-based semaphore.
// It can be dynamically adjusted at runtime via SetConcurrency.
type Throttle struct {
	sem     chan struct{}
	current int
	max     int
	mu      sync.Mutex
}

// NewThrottle creates a Throttle with the given initial concurrency level.
func NewThrottle(n int) *Throttle {
	if n < 1 {
		n = 1
	}
	t := &Throttle{
		sem:     make(chan struct{}, n),
		current: n,
		max:     n,
	}
	return t
}

// Acquire blocks until a slot is available.
func (t *Throttle) Acquire() {
	t.mu.Lock()
	sem := t.sem
	t.mu.Unlock()
	sem <- struct{}{}
}

// Release frees a slot.
func (t *Throttle) Release() {
	t.mu.Lock()
	sem := t.sem
	t.mu.Unlock()
	<-sem
}

// SetConcurrency changes the concurrency level by replacing the semaphore channel.
// In-flight tasks continue to hold slots on the old channel; new tasks use the new one.
// Minimum concurrency is 1.
func (t *Throttle) SetConcurrency(n int) {
	if n < 1 {
		n = 1
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if n == t.current {
		return
	}
	t.sem = make(chan struct{}, n)
	t.current = n
}

// Current returns the current concurrency level.
func (t *Throttle) Current() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.current
}

// Max returns the original (maximum) concurrency level.
func (t *Throttle) Max() int {
	return t.max
}

// Close is a no-op for compatibility; the throttle doesn't need explicit cleanup.
func (t *Throttle) Close() {}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/worker/ -run "TestThrottle" -v`
Expected: PASS

**Step 5: Run all worker tests**

Run: `go test ./internal/worker/ -v`
Expected: PASS (both Pool and Throttle tests)

**Step 6: Commit**

```bash
git add internal/worker/throttle.go internal/worker/throttle_test.go
git commit -m "feat: add adaptive concurrency throttle"
```

---

### Task 6: Wire retry mode into the client and upload loop

**Files:**
- Modify: `app/upload/run.go` (uploadLoop + handleGroup)
- Modify: `app/upload/upload.go` (or wherever client is initialized)
- Modify: `app/app.go` (ProcessError for retry mode)

**Step 1: Enable `RetryEnabled` on the immich client when `--on-errors=retry`**

Find where the immich client is opened. In `app/upload/upload.go:163`:
```go
err := uc.client.Open(ctx, uc.app)
```

Look at `app/client.go` or wherever `Open` is defined to find where `ImmichClient` is created.

Run: `grep -rn "func.*Open" app/` to find it.

After the client is opened, add:
```go
if uc.app.OnErrors == cliflags.OnErrorsRetry {
    uc.client.Immich.(*immich.ImmichClient).RetryEnabled = true
}
```

Wait — `uc.client.Immich` is an interface. We need to check if it's the real client. Actually, looking at `client.go`, `uc.client.Immich` will be `*ImmichClient` in production. We can use a type assertion.

Better approach: add a method to the interface or set it via the client option. Simplest: just set it after `Open()` in `upload.go:Run()`, around line 168:

```go
if uc.app.OnErrors == cliflags.OnErrorsRetry {
    if ic, ok := uc.client.Immich.(*immich.ImmichClient); ok {
        ic.RetryEnabled = true
    }
}
```

Add import for `cliflags` if not present (check existing imports — it's already imported via `cliflags "github.com/simulot/immich-go/internal/cliFlags"`).

**Step 2: Add `ProcessError` handling for retry mode**

Modify `app/app.go` `ProcessError` method (line 102-124). Add a case for `OnErrorsRetry` after the `OnErrorsNeverStop` case:

```go
} else if app.OnErrors == cliflags.OnErrorsRetry {
    app.Log().Error("Error", "err", err.Error())
    return nil // retry mode: errors that reach here had retries exhausted, continue
```

**Step 3: Wire adaptive concurrency into `uploadLoop`**

Modify `app/upload/run.go`, the `uploadLoop` function (line 615-661).

Replace the function with:

```go
func (uc *UpCmd) uploadLoop(ctx context.Context, groupChan chan *assets.Group) error {
	ctx, cancel := context.WithCancelCause(ctx)

	useAdaptive := uc.app.OnErrors == cliflags.OnErrorsRetry
	throttle := worker.NewThrottle(uc.app.ConcurrentTask)

	var consecutiveSuccesses atomic.Int64

	var wg sync.WaitGroup
	wg.Go(func() {
		workers := worker.NewPool(uc.app.ConcurrentTask)
		defer workers.Stop()
		for {
			select {
			case <-ctx.Done():
				cancel(ctx.Err())
				return
			case g, ok := <-groupChan:
				if !ok {
					return
				}
				if useAdaptive {
					throttle.Acquire()
				}
				workers.Submit(func() {
					if useAdaptive {
						defer throttle.Release()
					}
					err := uc.handleGroup(ctx, g)
					if err != nil {
						if useAdaptive && isServerError(err) {
							consecutiveSuccesses.Store(0)
							cur := throttle.Current()
							newLevel := max(cur/2, 1)
							if newLevel < cur {
								throttle.SetConcurrency(newLevel)
								uc.app.Log().Info("Reducing concurrency due to server errors", "from", cur, "to", newLevel)
							}
							// Brief pause to let server recover
							select {
							case <-time.After(5 * time.Second):
							case <-ctx.Done():
							}
						}
						err = uc.app.ProcessError(err)
						if err != nil {
							cancel(err)
						}
					} else {
						if useAdaptive {
							n := consecutiveSuccesses.Add(1)
							if n%10 == 0 {
								cur := throttle.Current()
								maxLevel := throttle.Max()
								if cur < maxLevel {
									newLevel := min(cur+1, maxLevel)
									throttle.SetConcurrency(newLevel)
									uc.app.Log().Info("Increasing concurrency after consecutive successes", "from", cur, "to", newLevel)
								}
							}
						}
					}
				})
			}
		}
	})

	wg.Wait()
	err := context.Cause(ctx)

	// Cleanup: delete server assets if needed
	if len(uc.deleteServerList) > 0 {
		ids := []string{}
		for _, da := range uc.deleteServerList {
			ids = append(ids, da.ID)
		}
		err := uc.DeleteServerAssets(ctx, ids)
		if err != nil {
			return fmt.Errorf("can't delete server's assets: %w", err)
		}
	}

	return err
}
```

Add a helper function `isServerError` in the same file:

```go
// isServerError checks if an error originated from the server (as opposed to a local error).
// Used by adaptive concurrency to decide whether to reduce concurrency.
func isServerError(err error) bool {
	if err == nil {
		return false
	}
	// Use the retry helper from the immich package to check
	// We check for callError which indicates a server response
	var ce interface{ Error() string }
	return errors.As(err, &ce) && strings.Contains(err.Error(), "http://") || strings.Contains(err.Error(), "https://")
}
```

Wait — that's fragile. Better to just reuse the existing `fileevent` error types or check for the immich `callError` type. But the `immich` package's `callError` is unexported.

Simpler approach: export `IsRetryable` from the immich package and use that:

In `immich/retry.go`, add an exported version:
```go
// IsRetryable is the exported version of isRetryable for use by the upload layer.
func IsRetryable(err error) bool {
	return isRetryable(err)
}
```

Then in `run.go`, replace `isServerError` usage with `immich.IsRetryable`:

```go
if useAdaptive && immich.IsRetryable(err) {
```

Add necessary imports to `run.go`:
- `"sync/atomic"`
- `"time"`
- `"github.com/simulot/immich-go/immich"`
- `"github.com/simulot/immich-go/internal/worker"`

(Check which are already imported and only add missing ones.)

**Step 4: Verify build**

Run: `go build ./...`
Expected: Success

**Step 5: Run all tests**

Run: `go test ./... 2>&1 | tail -30`
Expected: All PASS

**Step 6: Commit**

```bash
git add app/upload/run.go app/upload/upload.go app/app.go immich/retry.go
git commit -m "feat: wire retry mode with adaptive concurrency into upload loop"
```

---

### Task 7: Update flag help text and verify full build

**Files:**
- Modify: `app/app.go` (flag description)
- Modify: `internal/cliFlags/orErrors.go` (RegisterFlags help text)

**Step 1: Update help text**

In `app/app.go` line 48, change:
```go
flags.Var(&app.OnErrors, "on-errors", "What to do when an error occurs (stop, continue, accept N errors at max)")
```
to:
```go
flags.Var(&app.OnErrors, "on-errors", "What to do when an error occurs (stop|continue|retry|N)")
```

In `internal/cliFlags/orErrors.go` line 22, change:
```go
fs.Var(f, prefix+"on-errors", "Action to take on errors, (stop|continue| <n> errors)")
```
to:
```go
fs.Var(f, prefix+"on-errors", "Action to take on errors (stop|continue|retry|N)")
```

**Step 2: Full build and test**

Run: `go build ./... && go test ./...`
Expected: Build succeeds, all tests pass

**Step 3: Commit**

```bash
git add app/app.go internal/cliFlags/orErrors.go
git commit -m "docs: update --on-errors flag help text to include retry option"
```

---

### Task 8: Build, deploy, and test on kapara

**Step 1: Cross-compile**

```bash
GOOS=windows GOARCH=amd64 go build -o immich-go.exe .
```

**Step 2: Kill any running process on kapara**

```bash
ssh -o ProxyJump=admin@contabo-eu yosi@kapara 'taskkill /F /IM immich-go.exe' 2>/dev/null || true
```

**Step 3: Deploy**

```bash
scp -o ProxyJump=admin@contabo-eu immich-go.exe yosi@kapara:'D:\yosi\gilit\immich-go.exe'
```

**Step 4: Test**

Run on kapara with `--on-errors=retry --batch-limit=1` and monitor the log file for:
- Retry WARN messages when server returns 5xx
- Concurrency adjustment INFO messages
- Successful completion despite transient errors

**Step 5: Verify**

Check the log for:
- `"Reducing concurrency"` messages during server pressure
- `"Increasing concurrency"` messages after recovery
- No more abrupt stops from transient 503/500 errors
