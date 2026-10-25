# Incremental Takeout import proof of concept

Tracks [issue #1453](https://github.com/simulot/immich-go/issues/1453) and [draft PR #1454](https://github.com/simulot/immich-go/pull/1454). This draft demonstrates local ZIP ingestion with persistent state. It is not a production migration tool.

## Filename safety and incremental enrichment

Immich stores `originalFileName` during upload, and its update DTO has no filename field. A wrong provisional name cannot be repaired through the metadata update API. This is visible in the pinned [v3.3.5 upload service](https://github.com/immich-app/immich/blob/v3.3.5/server/src/services/asset-media.service.ts) and [update DTO](https://github.com/immich-app/immich/blob/v3.3.5/server/src/dtos/asset.dto.ts).

The `experimental-takeout` command therefore waits for an unambiguous sidecar association before uploading a new logical asset. It uses the sidecar's original title with upstream filename normalization for the upload, rather than assuming the archived filename is correct. An exact sidecar-stem association can restore a longer original title even when Google shortened the archived filename. Unsupported or ambiguous naming remains deferred and reported per item so unrelated files can proceed. This PoC does not implement the normal importer's complete matching puzzle solver. Media checksum/read failures are also recorded per source: healthy items continue and their progress is saved before the command returns the collected errors. A repaired or re-downloaded item can be retried without re-uploading healthy receipts.

The importer retains metadata and source-path/checksum records. It does not spool unresolved media bytes or retain ZIPs. If a sidecar arrives after an unnamed media item, provide that media ZIP again after metadata discovery, downloading it again if necessary. Running through an export and then re-presenting deferred media trades network traffic for bounded media storage. A single pass with one local ZIP and no ability to retrieve deferred media cannot guarantee original names.

Capture dates and descriptions can be enriched after the canonical name is established. A later source occurrence can supply more metadata without the old media ZIP or another upload.

## One asset, multiple source occurrences

A logical asset is identified by its checksum and owns one upload receipt, canonical name, baseline and applied metadata. The catalog separately records every original year/album path. The same bytes at two paths reuse the logical asset; they do not become an independently unowned duplicate merely because the second path was discovered later.

Metadata is reconciled across all known occurrences. Missing fields do not clear known fields, and conflicts are reported instead of silently selecting an occurrence. Sidecars associated with another copy can enrich the existing owned asset. A genuinely preexisting asset or an upload with an uncertain acknowledgement/baseline remains unowned and is not modified automatically. Pending metadata for such assets remains visible.

The state is scoped to one server URL, authenticated user and operator-chosen export ID. An exclusive lock prevents simultaneous writers. State is saved through a synced private temporary file and durable replacement. The catalog contains no API keys, but does contain private filenames and descriptions.

Before updating, the importer compares server values with its baseline and desired metadata. Changed values cause a conflict instead of overwriting a possible user edit. Failed updates remain pending and can be retried without media. Reconciliation also detects drift after a previously acknowledged update. This does not make read/update atomic or eliminate background-job races.

## Immich version requirement

**Minimum API baseline: Immich v1.133.0.** Use a maintained Immich release rather than downgrading to that baseline.

The capture-date and description update APIs predate v1.133.0. The existing shared uploader sends `visibility=timeline` in `POST /api/assets`; the upload DTO includes `visibility` in [v1.133.0](https://github.com/immich-app/immich/blob/v1.133.0/open-api/immich-openapi-specs.json), whereas it is absent in [v1.132.0](https://github.com/immich-app/immich/blob/v1.132.0/open-api/immich-openapi-specs.json). The draft has no legacy payload fallback or explicit version gate.

Endpoint and field contracts were checked in the v1.133.0, v2.0.0 and v3.0.0 specifications. The immutable-name constraint was checked against v3.3.5. These are source/schema checks, not live-server compatibility tests. Future breaking changes may require client updates.

## Try the vertical slice

Build from the branch's repository root using the Go version in `go.mod` or newer:

```sh
go build -o ./immich-go-poc .
./immich-go-poc experimental-takeout --help
```

Use a disposable Immich account and a small export. Supply the API key through `IMMICH_GO_EXPERIMENTAL_TAKEOUT_API_KEY` or a private config file. Required access includes asset upload/read/update and normal client connection checks (`/users/me`, server ping/about/media-types). See [API key guidance](./installation.md).

Process one ZIP at a time with the same state and export ID:

```sh
./immich-go-poc experimental-takeout \
  --server=https://immich.example \
  --state=./import-state/export-state.json \
  --export-id=my-export \
  --acknowledge-incomplete-import \
  ./takeout/takeout-001.zip

./immich-go-poc experimental-takeout \
  --server=https://immich.example \
  --state=./import-state/export-state.json \
  --export-id=my-export \
  --acknowledge-incomplete-import \
  ./takeout/takeout-002.zip
```

If ZIP 001 had media without a sidecar and ZIP 002 supplied that sidecar, the first invocation defers the media and the second records metadata. Re-present ZIP 001 to perform the upload with the original name:

```sh
./immich-go-poc experimental-takeout \
  --server=https://immich.example \
  --state=./import-state/export-state.json \
  --export-id=my-export \
  --acknowledge-incomplete-import \
  ./takeout/takeout-001.zip
```

Later metadata for an already uploaded asset can be retried without any source ZIP:

```sh
./immich-go-poc experimental-takeout \
  --server=https://immich.example \
  --state=./import-state/export-state.json \
  --export-id=my-export \
  --acknowledge-incomplete-import
```

The output distinguishes logical assets and source occurrences, pending uploads/metadata, unowned assets and deferred items. Review the reasons for deferred items; finishing ZIP processing is not proof of a fully reconciled export. Keep original archives available remotely or on backup storage. No command here deletes them or certifies them safe to delete.

## State compatibility and recovery

The revised catalog uses a new state version. The old per-path state lacks an authoritative canonical name and may describe media already uploaded with irreversible provisional names. It is refused explicitly instead of silently migrating or changing ownership. Keep the old state and original sources for investigation; use a separate state file and disposable account to test the new PoC. Reusing a fresh state against an existing library does not make existing assets owned.

A crash deliberately leaves the exclusive `.lock` file. Confirm that no importer is using that exact state before removing the stale lock. Never remove a live process's lock or discard its state to bypass a conflict.

## Boundaries and follow-on work

- Only supported unambiguous sidecar associations establish names. Missing, unsafe, truncated or duplicate-index associations that cannot be proven remain deferred; no fuzzy guess is uploaded.
- A valid nonzero capture timestamp is required for date/description patching with the shared client. Empty descriptions do not clear server values. GPS, favorites, archive status, albums, tags and stacks are not reconciled.
- Selection-filter parity is not implemented. The acknowledgement flag remains required; the PoC does not provide the normal importer's trash/partner filtering guarantees.
- Server jobs are not paused. Concurrent edits and asynchronous extraction still require live-server validation.
- The JSON catalog is rewritten on durable operations and accumulated state is rescanned. Large-export performance and transactional storage remain follow-on work.
- No downloader, OAuth, automatic export creation, source cleanup, dry-run mode or safe-to-delete marker is implemented. Other import commands retain their existing behavior.

A downloader can eventually feed a user-created Google Drive Takeout into this flow using rclone or an established OAuth library. Use the system browser with PKCE and a loopback callback, not an embedded Google login webview. It must be able to retrieve deferred media again. Automatic Google Photos export creation remains a separate feasibility investigation.

## Review and validation handoff

The maintainer's [matching review](https://github.com/simulot/immich-go/pull/1454#issuecomment-6400152762) exposed two correctness gaps in the initial draft: immutable provisional names and per-path ownership losing metadata from duplicate occurrences. The revised design addresses both through code and regression coverage.

The branch is rebased onto `main`. Its existing duplicate closed-pipe fix from [upstream PR #1391](https://github.com/simulot/immich-go/pull/1391) is preserved; this draft no longer modifies the shared uploader to duplicate that fix.

Regression evidence includes the real CLI/HTTP naming gate and canonical multipart filename, same bytes in year/album paths in both arrival orders, metadata only on a duplicate occurrence, restart/idempotence, failed-update retries, uncertain ownership, per-item deferral, conflicts, locking and state-version refusal. Tests use local fake clients or HTTP fixtures. They do not establish real Immich job behavior, complete Takeout fidelity or Windows runtime filesystem behavior.

Local validation after the review fixes:

- `go test -p 2 -parallel 2 -race ./...`: passed, including importer, real CLI/HTTP and duplicate-upload regressions.
- `golangci-lint` v2.15.0: zero issues, matching the clean `main` baseline at `bc54ad2`.
- Native macOS and Linux/Windows amd64 builds: passed. Windows runtime behavior remains untested.
- Independent re-review: the corrupt-media progress finding was reproduced, fixed and regression-tested; no remaining correctness findings within this PoC scope.
