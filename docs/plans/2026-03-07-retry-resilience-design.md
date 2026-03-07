# Retry & Resilience Design

## Problem

When the Immich server gets overwhelmed (e.g. 20 concurrent large HEIC uploads to a Synology NAS), it returns errors on all in-flight requests simultaneously. With `--on-errors=stop` (default), a single transient error kills the entire upload run. Even with `--on-errors=continue`, failed files are simply skipped with no retry.

## Solution

A new `--on-errors=retry` mode that adds:
1. HTTP-level retry with exponential backoff for non-upload API calls
2. Upload-level retry (re-open file, rebuild multipart) for asset uploads
3. Adaptive concurrency that reduces worker count under sustained load

## Retryable vs Non-Retryable

**Retryable** (transient server issues):
- 408 Request Timeout
- 429 Too Many Requests (honor `Retry-After` header if present)
- 500 Internal Server Error
- 502 Bad Gateway
- 503 Service Unavailable
- 504 Gateway Timeout
- Network errors: connection refused, reset, timeout, EOF

**Not retryable** (permanent errors):
- 400 Bad Request
- 401 Unauthorized
- 403 Forbidden
- 404 Not Found
- 409 Conflict
- Any other 4xx

## `--on-errors` Flag Values

| Value | Retry transient? | Adaptive concurrency? | On exhausted retries? |
|-------|-----------------|----------------------|----------------------|
| `stop` | No | No | Stop immediately |
| `continue` | No | No | Log and continue |
| `N` | No | No | Stop after N errors |
| `retry` | Yes, 3 attempts | Yes, halve/recover | Log and continue |

## Architecture

### Layer 1: HTTP-level retry (non-upload calls)

**Location:** `immich/call.go` — inside `serverCall.do()`

For all API calls except uploads (album creation, metadata updates, asset queries, tag operations), the `do()` method gets a retry loop:
- On retryable error, sleep with exponential backoff (1s, 2s, 4s + jitter)
- Re-invoke `fnRequest` to rebuild the request from scratch
- Max 3 attempts
- Log each retry at WARN level
- Honor `Retry-After` header on 429

This works because non-upload request bodies (JSON payloads) are built fresh by the `requestFunction` on each call.

### Layer 2: Upload-level retry

**Location:** `immich/upload.go` — inside `uploadAsset()`

Upload requests use `io.Pipe()` to stream multipart data, so the HTTP body is not replayable. Retry must wrap the entire upload sequence:
- Re-open the file via `la.OpenFile()`
- Rebuild the multipart writer and pipe
- Re-send the request

Same backoff and max attempts as Layer 1. On each retry, the previous file handle and goroutine are cleaned up before starting fresh.

### Layer 3: Adaptive concurrency

**Location:** `app/upload/run.go` — in `uploadLoop()`

When retries are exhausted and errors still occur, the system reduces concurrency to ease server pressure:

- **On server error:** halve concurrency (minimum 1), pause 5s
- **On 10 consecutive successes:** increase concurrency by 1, up to original `--concurrent-tasks` value
- **Log** concurrency changes at INFO level

**Implementation:** A `Throttle` (channel-based semaphore) wraps task submission. The worker pool stays at max size, but the throttle gates how many tasks execute concurrently. Adjusting concurrency = adjusting the semaphore.

## File Changes

### New: `immich/retry.go`
- `isRetryable(err error) bool` — inspects `callError` status or network error type
- `retryDelay(attempt int, retryAfter string) time.Duration` — exponential backoff with jitter, honoring Retry-After
- Constants: `maxRetries = 3`, retryable status code set

### Modified: `immich/call.go`
- `do()` accepts a retry flag (enabled when `--on-errors=retry`)
- Retry loop around request+response handling for non-upload calls

### Modified: `immich/upload.go`
- `uploadAsset()` gets outer retry loop wrapping file open through `do()` call
- Needs access to retry config (passed via `ImmichClient` field)

### New: `internal/worker/throttle.go`
- `Throttle` struct: channel-based semaphore
- `Acquire()`, `Release()`, `SetConcurrency(n int)`, `Current() int`
- Thread-safe

### Modified: `internal/cliFlags/orErrors.go`
- Add `OnErrorsRetry` constant
- Parse `"retry"` in `Set()`

### Modified: `app/upload/run.go`
- `uploadLoop`: create `Throttle`, wrap `workers.Submit` with acquire/release
- Track consecutive successes/failures, adjust throttle
- Log concurrency changes

### Modified: `immich/immich.go` (or client struct)
- Add `RetryEnabled bool` field to `ImmichClient` so the HTTP layer knows whether to retry

### Unchanged
- `state.go`, `upload.go` (flags struct), `ui.go`, `noui.go`, `app.go`, `worker/worker.go`
