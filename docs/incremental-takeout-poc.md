# Incremental Takeout import proof of concept

Tracks [issue #1453](https://github.com/simulot/immich-go/issues/1453). This draft demonstrates local ZIP ingestion with persistent state. It is not a production migration tool.

## What is implemented

The separate `experimental-takeout` command uploads complete media files before their supplemental JSON is available. It stores original paths, checksums, Immich asset IDs and metadata in a private state file. A later invocation can apply an exact-path sidecar's capture date and nonempty description to the original asset without the previous media ZIP. Sidecars can also arrive before media.

The state is scoped to one server URL, authenticated user and operator-chosen export ID. An exclusive lock prevents simultaneous writers. Receipts and metadata are saved through a synced temporary file and durable replacement. The catalog contains no API keys. It contains private filenames and descriptions, so protect it like the original export.

Repeated paths with identical contents reuse their receipts. A changed file at the same original path, conflicting sidecars or renamed/truncated sidecars produce errors instead of guessed associations. Preexisting assets and uploads whose acknowledgement or baseline could not be established are recorded as unowned. The PoC does not enrich those assets automatically.

Before updating metadata, it compares current server values with the saved baseline and desired values. Changed values cause an error instead of overwriting a possible user edit. Failed updates stay pending and can be retried without a ZIP. An update that succeeded remotely but lost its response is safe to retry.

## Immich version requirement

**Minimum supported API baseline for this PoC: Immich v1.133.0.** Use a maintained Immich release at or above that version; the baseline is not a recommendation to downgrade.

The capture-date and description update APIs predate v1.133.0. The limiting dependency is the existing shared uploader: it sends `visibility=timeline` in `POST /api/assets`. The upload DTO first includes `visibility` in [Immich v1.133.0](https://github.com/immich-app/immich/blob/v1.133.0/open-api/immich-openapi-specs.json); it is absent in the preceding [v1.132.0 specification](https://github.com/immich-app/immich/blob/v1.132.0/open-api/immich-openapi-specs.json). This draft does not provide a legacy upload-payload fallback.

The endpoint and field contracts used here were checked in the v1.133.0, v2.0.0 and v3.0.0 specifications: server ping/about/media-types, authenticated user lookup, media upload and asset read/update with `dateTimeOriginal` and `description`. This is a schema-derived compatibility baseline, not a claim that the PoC has been tested against live servers running those releases. Future breaking API changes may require client updates. There is currently no explicit runtime version gate; a reported server version alone does not prove that all required API permissions and behavior are available.

## Try the vertical slice

Build from this branch's repository root with the Go version specified in `go.mod` or newer:

```sh
go build -o /tmp/immich-go-poc .
/tmp/immich-go-poc experimental-takeout --help
```

Use a disposable Immich account and a small test export. Supply the API key through the existing configuration environment variable, `IMMICH_GO_EXPERIMENTAL_TAKEOUT_API_KEY`, or a private config file. Required permissions include asset upload/read/update and the connection checks used by the normal client (`/users/me`, server ping/about/media-types). See the project's [API key guidance](./installation.md) for supported versions.

Process one archive at a time using the same state and export ID:

```sh
/tmp/immich-go-poc experimental-takeout \
  --server=https://immich.example \
  --state=/path/to/private/export-state.json \
  --export-id=my-export-2026-10-09 \
  --acknowledge-incomplete-import \
  /path/to/takeout-001.zip

/tmp/immich-go-poc experimental-takeout \
  --server=https://immich.example \
  --state=/path/to/private/export-state.json \
  --export-id=my-export-2026-10-09 \
  --acknowledge-incomplete-import \
  /path/to/takeout-002.zip
```

A successful invocation reports media receipts, enriched media, unowned duplicates, media awaiting metadata and sidecars awaiting media. ZIP processing finishing is not proof that the full export has been reconciled. Keep the original archives available remotely or on backup storage.

Retry pending updates, or check previously enriched values, without downloading old archives:

```sh
/tmp/immich-go-poc experimental-takeout \
  --server=https://immich.example \
  --state=/path/to/private/export-state.json \
  --export-id=my-export-2026-10-09 \
  --acknowledge-incomplete-import
```

If a process crashes, its `.lock` file deliberately remains. Confirm that no importer is using this exact state before removing the lock and retrying. Never remove a live process's lock. Keep the state file: losing it loses ownership and matching information.

## Boundaries of this draft

- Only exact original-directory matches for `photo.jpg.json` or `photo.jpg.supplemental-metadata.json` with the corresponding JSON title are supported. It does not implement the existing importer's complete naming puzzle solver.
- A nonzero valid capture timestamp is required for enrichment. Empty descriptions do not clear existing descriptions. GPS, favorites, archive status, albums, tags, stacks and relationship reconciliation are not implemented.
- Selection filters are not implemented. Media can arrive before information identifying trashed or partner-shared photos; the acknowledgement flag is required for this reason.
- Server background jobs are not paused. Extraction changes can cause conflicts, and metadata can change after an invocation finishes. A later reconciliation checks for drift; this is not an atomic compare-and-set or a guarantee against concurrent edits. Real Immich job behavior still needs validation.
- The catalog rewrites JSON on every durable operation and scans the accumulated catalog on reconciliation. This is intentionally a small PoC, with quadratic cumulative write/scan cost. Large exports need a transactional store and performance measurements.
- Unknown upload outcomes and baseline failures are handled conservatively, leaving unowned receipts for operator investigation rather than silently claiming complete metadata.
- No downloader, Google authentication, automatic export creation, automatic source deletion, dry-run mode or safe-to-delete archive marker is implemented. Existing import commands keep their default behavior.

## Follow-on design

The issue documents a transactional catalog, full upstream matching, field provenance and user-edit protection, checksummed duplicate occurrence tracking, relationship updates, crash recovery and completion accounting.

For downloading, begin with a user-created Takeout delivered to Google Drive. A downloader can authorize through rclone or an established OAuth library and feed one verified ZIP at a time into this importer. Use the system browser with PKCE and a loopback callback, not an embedded Google login webview. Automatic Google Photos export creation needs its own feasibility and API eligibility investigation.

## Validation handoff

Regression coverage exercises separate ZIP invocations and reopening state, metadata arriving first, retries without old media, duplicate receipts, lost acknowledgements, user-edit conflicts, state scope/locking and invalid inputs. The CLI test uses the real HTTP client against a local test server and verifies multipart media bytes and update payloads.

Local mocked tests establish the mechanism. They do not establish compatibility with a deployed Immich server's asynchronous jobs or complete Takeout fidelity. This PR stays draft pending those checks and broader feature work.

Local validation for this draft:

- `go test -race ./...`: passed, including the real CLI/HTTP flow and early-response duplicate regression.
- Native macOS build and Linux/Windows amd64 cross-builds: passed. Windows runtime filesystem behavior was not exercised.
- `golangci-lint` v2.11.3: full run reports the same 12 findings as unmodified `develop` at `6223e548a586b30dc6d714d3f435f2b9f817e1a7`; no new findings. The pre-existing findings are in unrelated code.
- Independent review: initial portability, duplicate-response and drift findings addressed, with no unresolved findings within this PoC scope.
- Live Immich and Google download/authentication checks: not run.

The uploader regression initially failed with `io: read/write on closed pipe` when the test server returned a checksum duplicate without consuming multipart bytes. It now accepts the confirmed duplicate receipt while preserving genuine transport errors.
