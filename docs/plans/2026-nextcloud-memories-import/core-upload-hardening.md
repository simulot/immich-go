# Core Upload Hardening Note

## Context

During real-world testing of `upload from-nextcloud-memories`, uploads to Immich intermittently failed with transient server-side errors, especially `502 Bad Gateway`, followed by `io: read/write on closed pipe` while multipart request bodies were still being streamed.

The observed failures were not specific to the Nextcloud Memories adapter logic. They were traced to the shared core Immich upload path used by all import sources.

## Scope of the Change

A small core hardening change was introduced in `immich/upload.go` to reduce unnecessary import failures without redesigning the upload transport:

1. Ignore `closed pipe` write errors when Immich has already returned a successful upload response or a duplicate response and the remaining multipart writer goroutine is only observing the server-side early close.
2. Retry a narrow set of transient upload failures for asset uploads:
   - HTTP `502 Bad Gateway`
   - HTTP `503 Service Unavailable`
   - HTTP `504 Gateway Timeout`
   - closed-pipe request aborts consistent with premature upstream termination
3. Keep retries intentionally small and bounded to avoid turning this into a larger transport-policy rewrite.

## Why This Lives in Core

The Nextcloud Memories importer made the issue easy to reproduce because it exercises large, long-running uploads, but the failing behavior is in the shared Immich client and upload implementation rather than in source-specific adapter code.

Keeping the fix in core prevents source-specific workarounds and keeps behavior consistent across import sources.

## Non-Goals

This change does **not** attempt to:

- introduce resumable uploads
- redesign upload scheduling or backpressure
- implement broad retry policies for all API calls
- mask persistent upstream instability

The intent is only to tolerate a limited class of transient upstream failures that are already known to occur during long-running imports.

## Validation

Focused regression tests were added around:

- successful upload responses followed by an early connection close during multipart streaming
- successful retry after transient `502` upload failures
- retry classification for transient upload errors

