# Incremental Takeout import proof of concept

Tracks [issue #1453](https://github.com/simulot/immich-go/issues/1453) and [draft PR #1454](https://github.com/simulot/immich-go/pull/1454). This draft demonstrates local ZIP ingestion with persistent state. It is not a production migration tool.

## Filename safety and incremental enrichment

Immich stores `originalFileName` during upload, and its update DTO has no filename field. A wrong provisional name cannot be repaired through the metadata update API. This is visible in the pinned [v3.3.5 upload service](https://github.com/immich-app/immich/blob/v3.3.5/server/src/services/asset-media.service.ts) and [update DTO](https://github.com/immich-app/immich/blob/v3.3.5/server/src/dtos/asset.dto.ts).

The `experimental-takeout` command therefore waits for an unambiguous sidecar association before uploading a new logical asset. It uses the sidecar's original title with upstream filename normalization for the upload, rather than assuming the archived filename is correct. An exact sidecar-stem association can restore a longer original title even when Google shortened the archived filename. Unsupported or ambiguous naming remains deferred and reported per item so unrelated files can proceed. This PoC does not implement the normal importer's complete matching puzzle solver. Media checksum/read failures are also recorded per source: healthy items continue and their progress is saved before the command returns the collected errors. A repaired or re-downloaded item can be retried without re-uploading healthy receipts.

The default mode retains metadata and source-path/checksum records. Optional durable staging also retains unresolved media, so a later sidecar can trigger upload without the earlier ZIP. This keeps the filename requirement without relying on a download URL remaining usable. Input ZIPs are never deleted automatically.

## Keeping archive downloads to one pass

Enable staging with `--staging-dir` and a positive `--max-staging-bytes` limit. The importer keeps one checksum-addressed payload per logical asset, regardless of how many year/album copies it sees. If media arrives first, its bytes wait locally. If metadata arrives first, only its catalog record waits. When the pair is complete and the canonical name is resolved, the cached media can upload automatically, including after restart. Managed payload copies are released only after a remote upload receipt is durably recorded; user ZIPs and unrelated files are never cleaned up.

Use a dedicated, initially empty staging directory. Its ownership marker binds it to the exact state-file location; keep both locations fixed. Later invocations restore the persisted directory and cap when the flags are omitted. To increase capacity, pass the same directory with a larger limit. Decreasing an established cap is rejected without changing its saved value or payloads. Arbitrary files and symlinks in that directory cause errors and are preserved. Missing or corrupted cached media is reported before upload; restoring the cache or re-presenting the original ZIP repairs it.

Staging currently requires a filesystem supporting private POSIX directories and directory syncing. Windows explicitly rejects staging until ACL privacy and durable publication are implemented; catalog-only import remains available there. This avoids presenting POSIX permission checks as Windows ACL protection.

With sufficient staging capacity, supported cataloged media can be processed by downloading each source archive once. No reused download URL or second media pass is required. This does not certify unsupported export content or unresolvable associations as complete. The current PoC still takes local ZIP paths; it does not implement the downloader itself.

The maximum pending set depends on archive ordering and unresolved matching. In the worst case it can approach the entire library's unique media size. Plan for the current downloaded ZIP, the pending uncompressed payloads, catalog and working space. The configured limit bounds retained staging payloads, not all disk usage. Guarded upload workspace is excluded from that cap but obeys the free-space reserve.

### Free-space thresholds and two-pass fallback

The CLI reports **WARNING at 50 GB free or less**, **DANGER at 20 GB or less**, and preserves a **10 GB reserve** before storing more media. GB is decimal: 1 GB = 1,000,000,000 bytes. Checks use filesystem blocks available to the current user on the staging volume, not just the configured retained-media cap. They run before admitting a complete payload and again during the write. Warnings are emitted as thresholds change during the invocation.

If a pending payload would cross the reserve or exceed the retained-media cap, the importer persistently switches to a catalog/second-pass workflow. It preserves all complete cached media, stops adding pending payloads, and continues cataloging the remaining archives. Later metadata can still release cached assets through uploads. On the second pass, it requests only source archives containing resolved media that is neither cached nor already uploaded. It does not attempt another download itself. Fallback persists across restarts; increasing the cap does not reset that choice.

ZIP uploads also need temporary seekable media. In staging mode, that workspace is created on the guarded volume and checked against the same 10 GB reserve; the shared uploader reads it as a regular file without making another whole-media copy. If even that workspace cannot fit, the media remains pending and the CLI explains that space must be freed before its replay can upload. Already staged files can upload without another full copy.

These checks control importer media writes. Input ZIP downloads are still manual, so account for their size separately before downloading the next archive. Other programs can also consume disk space between checks; the importer cannot reserve the filesystem against external writers. Probe, read, write and cleanup failures remain explicit errors. Keep the current ZIP after any such error.

### Exact next-archive guidance

Create an ordered list of the export's ZIP filenames, one per line, and supply it with `--archive-list=./takeout-archives.txt`:

```text
takeout-001.zip
takeout-002.zip
takeout-003.zip
```

The list and archive progress persist in the catalog. After each invocation, `Next:` identifies the next unscanned filename, an earlier filename to replay after the catalog pass, or a retry that needs no download because the media is already cached. Download URLs are not stored. Completed archives retain a SHA-256 identity of their compressed bytes. A held, failed archive can be replaced by a repaired download with the same filename only when its previously readable media and raw sidecars are preserved; healthy upload receipts survive that repair.

Cached bytes are verified before the plan promises a ZIP-free retry. If a payload is missing or corrupt, the plan names its recorded source ZIP for restore/re-presentation; it does not repeat an impossible reconcile instruction. Without archive provenance, it reports that the filename is unknown and requires restoring the original cache or supplying the source. Repair and retry preserve existing upload receipts and uncertain ownership.

For example, media from `takeout-001.zip` may be left uncached after fallback. If `takeout-002.zip` supplies its sidecar and `takeout-003.zip` is still unscanned, the next request remains `takeout-003.zip`. Once all listed archives have been cataloged, the plan can request `takeout-001.zip` for its missing media. Re-present a retained local copy if available; otherwise obtain a fresh URL for that archive and download it again.

Without the complete archive list, the importer cannot name an unseen ZIP or determine that the first pass has finished. It asks for that inventory instead of guessing. Unsupported or unresolved items remain visible after every known archive has been scanned; a repeated download is not presented as a repair for an unproven association.

Staging also returns an error when a supported media entry cannot be cataloged safely, such as duplicate ZIP entry paths, unsafe paths or conflicting bytes at one original path. Healthy items still proceed. Keep that source ZIP for investigation; the importer does not invent a metadata association or claim those skipped bytes were cached. Unsupported media extensions remain outside this PoC's scope.

Without staging, deferred media must still be re-presented or downloaded again after metadata discovery.

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

Process one ZIP at a time with the same state, export ID and staging directory. This example allows up to 20 GiB of staged payloads:

```sh
./immich-go-poc experimental-takeout \
  --server=https://immich.example \
  --state=./import-state/export-state.json \
  --export-id=my-export \
  --acknowledge-incomplete-import \
  --archive-list=./takeout-archives.txt \
  --staging-dir=./import-state/pending \
  --max-staging-bytes=21474836480 \
  ./takeout/takeout-001.zip

./immich-go-poc experimental-takeout \
  --server=https://immich.example \
  --state=./import-state/export-state.json \
  --export-id=my-export \
  --acknowledge-incomplete-import \
  --archive-list=./takeout-archives.txt \
  --staging-dir=./import-state/pending \
  --max-staging-bytes=21474836480 \
  ./takeout/takeout-002.zip
```

If ZIP 001 contains media without its sidecar, the first invocation retains that media in staging. If ZIP 002 supplies the sidecar, the next invocation uploads the staged bytes with the original name. ZIP 001 is no longer needed locally for that upload after its successful staging invocation; keep the original export or backup available until the overall migration is verified. Never discard a source ZIP after an error.

Later metadata for an already uploaded asset can be retried without any source ZIP:

```sh
./immich-go-poc experimental-takeout \
  --server=https://immich.example \
  --state=./import-state/export-state.json \
  --export-id=my-export \
  --acknowledge-incomplete-import \
  --staging-dir=./import-state/pending \
  --max-staging-bytes=21474836480
```

The output distinguishes logical assets and source occurrences, pending uploads/metadata, unowned assets, deferred items and staging usage. Review the reasons for deferred items; finishing ZIP processing is not proof of a fully reconciled export. Keep original archives available remotely or on backup storage. No command here deletes them or certifies them safe to delete.

## State compatibility and recovery

The staging catalog uses version 3. Version 2 checksum-based catalogs migrate additively, preserving receipts, ownership and source occurrences. Version 1 per-path state lacks an authoritative canonical name and may describe media already uploaded with irreversible provisional names. It is refused explicitly instead of silently migrating or changing ownership. Keep that old state and original sources for investigation; use a separate state file and disposable account to test the new PoC. Reusing a fresh state against an existing library does not make existing assets owned.

A crash deliberately leaves the exclusive state `.lock` and, when enabled, the staging directory's `.immich-go.lock`. Confirm that no importer is using that exact state and directory before removing stale locks. Never remove a live process's lock or discard its state to bypass a conflict. Restart recovers fully published payloads and removes only unfinished temporary payloads belonging to this cache; it never evicts complete pending media.

## Boundaries and follow-on work

- Only supported unambiguous sidecar associations establish names. Missing, unsafe, truncated or duplicate-index associations that cannot be proven remain deferred; no fuzzy guess is uploaded.
- A valid nonzero capture timestamp is required for date/description patching with the shared client. Empty descriptions do not clear server values. GPS, favorites, archive status, albums, tags and stacks are not reconciled.
- Selection-filter parity is not implemented. The acknowledgement flag remains required; the PoC does not provide the normal importer's trash/partner filtering guarantees.
- Server jobs are not paused. Concurrent edits and asynchronous extraction still require live-server validation.
- The JSON catalog is rewritten on durable operations and accumulated state is rescanned. Archive planning adds compressed-content hashing and an inventory read. Large-export performance and transactional storage remain follow-on work.
- No downloader, OAuth, automatic export creation, source cleanup, dry-run mode or safe-to-delete marker is implemented. Other import commands retain their existing behavior.

A downloader can eventually feed a user-created Google Drive Takeout into this flow using rclone or an established OAuth library. Use the system browser with PKCE and a loopback callback, not an embedded Google login webview. For one-download operation, it must respect staging backpressure and preserve the current archive on errors. Catalog-only mode still requires retrieval of deferred media. Automatic Google Photos export creation remains a separate feasibility investigation.

## Review and validation handoff

The maintainer's [matching review](https://github.com/simulot/immich-go/pull/1454#issuecomment-6400152762) exposed two correctness gaps in the initial draft: immutable provisional names and per-path ownership losing metadata from duplicate occurrences. The revised design addresses both through code and regression coverage.

The branch is rebased onto `main`. Its existing duplicate closed-pipe fix from [upstream PR #1391](https://github.com/simulot/immich-go/pull/1391) is preserved; this draft no longer modifies the shared uploader to duplicate that fix.

Regression evidence includes the real CLI/HTTP naming gate and canonical multipart filename, same bytes in year/album paths in both arrival orders, metadata only on a duplicate occurrence, restart/idempotence, failed-update retries, uncertain ownership, per-item deferral, conflicts, locking and state-version refusal. Storage tests inject deterministic available-byte measurements and cover threshold boundaries, projected/per-chunk reserve stops, retained-cap fallback, unfinished-file accounting, guarded upload workspace and cached uploads below the reserve. Archive and CLI/HTTP tests verify exact next-file prompts, replay after a catalog pass, missing/corrupt-cache recovery, source CRC repairs and provenance preservation. Tests use local fake clients or HTTP fixtures. They do not establish real Immich job behavior, complete Takeout fidelity or Windows runtime filesystem behavior.

Local validation of naming, staging, reserve/fallback and archive planning:

- `go test -p 2 -parallel 2 -race ./...`: passed, including importer, real CLI/HTTP, source/cache repair, archive planning and duplicate-upload regressions.
- `golangci-lint` v2.15.0: zero issues. No checks were disabled for this implementation.
- Native macOS and Linux/Windows/FreeBSD amd64 builds: passed. CLI help exposes the retention and archive-list options. Cross-builds do not establish runtime filesystem behavior on those platforms; Windows staging is explicitly unsupported.
- Independent reviews reproduced healthy-progress, failed-temporary-file accounting, cap-reduction and damaged-cache guidance defects. Their fixes have regression coverage; this draft still requires maintainer design review and live-server validation.
- Physical ENOSPC/power-loss tests, representative-export performance measurements and live Immich background-job checks were not performed.
