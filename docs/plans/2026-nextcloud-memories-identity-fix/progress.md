# Progress

- [x] Step 1: Verify and document the source identity contract
- [x] Step 2: Replace fragile metadata indexing with asset-identity-first indexing
- [x] Step 3: Preserve album membership and synthetic tags through duplicate collapse
- [x] Step 4: Improve diagnostics and fail-closed options
- [x] Step 5: Update docs and plan tracking

## Notes

- 2026-06-13: Live test migration showed a severe metadata join failure. The importer reported `Collected 58998 indexed files from /Photos` but logged `92368` `Nextcloud Memories asset is not indexed; importing without source metadata` warnings during the same run.
- 2026-06-13: Live test migration only created one synthetic `immich-go/src/nextcloud-memories/album/...` tag assignment and barely attempted album creation, which is consistent with widespread metadata join failure.
- 2026-06-13: Current hypothesis is that the metadata index uses a basename/path approximation that does not match the actual discovered DAV/local file paths reliably enough for live Memories libraries.
- 2026-06-13: Separate follow-up concern: duplicate upload paths appear not to merge album membership and synthetic tags robustly when multiple source occurrences collapse onto a single destination asset.
- 2026-06-13: Confirmed from `internal/nextcloud` that the importer can model Memories source identity around `fileid`, with `image/info/<fileid>` returning the canonical `filename` plus album clusters.
- 2026-06-13: Reworked `adapters/nextcloudmemories/metadata.go` so the metadata index is keyed by Memories asset ID first and learns canonical paths from `image/info` hydration.
- 2026-06-13: Added a hybrid warm-up design that starts data transfer immediately while a bounded background queue progressively hydrates `image/info` responses into an exact full-path map.
- 2026-06-13: Reworked shared duplicate handling in `app/upload/run.go` so `AlreadyProcessed`, `SameOnServer`, and `BetterOnServer` merge albums and tags onto the canonical Immich asset before issuing album/tag updates.
- 2026-06-13: Short shared-impact audit suggests the duplicate-membership fix is generally correct for other sources too, because the shared upload pipeline is used by `from-folder`, `from-google-photos`, and `from-immich`, all of which can attach albums and/or tags before deduplication.
- 2026-06-13: Live testing feedback showed the hybrid approach also allowed increasing upload concurrency from 10 to 20 without stressing the Immich server, which supports keeping metadata hydration bounded and decoupled from the hot upload path.
- 2026-06-13: Added a `--sync-tags` toggle for `from-nextcloud-memories` and left it disabled by default so noisy Memories AI/system tags are not migrated unless explicitly requested.
- 2026-06-13: Confirmed rerun behavior is intentionally additive for tags in the shared upload pipeline: album-membership tags are preserved and added, and `--sync-tags` does not try to remove existing Immich tags from previously imported assets.
- 2026-06-13: Live retries currently only cover multipart uploads in `immich/upload.go`; generic Immich JSON operations like `UpdateAsset`, album updates, tag upserts, and copy/delete calls still fail fast on transient `502/503/504` responses.
- 2026-06-13: Follow-up resilience work should move bounded transient retry handling into the shared `immich` request layer so non-upload write operations behave consistently with upload retry policy.
- 2026-06-13: Added INFO-level retry diagnostics for shared Immich requests and multipart uploads so transient retry behavior is visible during live runs without being reported as a terminal failure.
- 2026-06-13: Follow-up resilience work now makes transient Immich retries configurable, increases default patience to 6 attempts with exponential backoff plus jitter, and keeps `--on-errors=stop` semantics focused on unrecoverable failures after retry exhaustion.

## PR Reasoning Notes

- The Memories identity fix is source-specific: `from-nextcloud-memories` was relying on a fragile path/basename join that failed badly on a real library.
- The duplicate-membership fix is shared pipeline correctness work: the old logic could drop relationship metadata whenever a source asset matched an already-known local or server asset.
- This shared fix is expected to benefit any importer that sets `asset.Albums` or `asset.Tags` before entering `app/upload`, including `from-folder`, `from-google-photos`, and `from-immich`.
- The change does not broaden upload selection or alter duplicate detection rules; it preserves metadata that should already have been applied to the canonical destination asset.
- The hybrid warm-up design keeps the importer responsive by preserving the earlier "start transferring quickly" behavior while progressively replacing heuristic lookup with exact path resolution from `image/info/<fileid>`.
- The hybrid approach remains idempotent because exact path learning is cached per Memories `fileid`, duplicate handling stays additive for albums/tags, and ambiguous bootstrap lookup is intentionally conservative.
