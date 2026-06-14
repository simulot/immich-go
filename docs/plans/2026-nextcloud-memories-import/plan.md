# Nextcloud Memories Import Plan

## Step 1: Add Source Client and Authentication

**Files**:

- `internal/nextcloud/...`
- `adapters/nextcloudmemories/...`

**Dependency choice**:

- Use `github.com/studio-b12/gowebdav` for DAV access
- Keep OCS and Memories integrations in custom `net/http` code

**Changes**:

- Add a minimal Nextcloud client for authenticated DAV and HTTP API calls
- Support source authentication with username plus password or app password
- Add source TLS and timeout configuration
- Normalize user-provided base URLs and derive the DAV root automatically

**Testable**:

- URL normalization
- Request construction
- Auth header handling
- Failure paths for invalid credentials or invalid server URL

**Mergeable**: Yes. Infrastructure only.

---

## Step 2: Discover Memories Configuration

**Files**:

- `adapters/nextcloudmemories/command.go`
- `internal/nextcloud/memories.go`

**Changes**:

- Use the internal custom HTTP client for OCS and Memories discovery calls
- Keep DAV browsing separate from discovery so the app-specific logic stays explicit
- Check that the Memories app is installed and reachable
- Read effective user configuration for Memories
- Resolve configured `timeline_path` values
- Fail early when configuration is empty or invalid

**Testable**:

- Discovery with one root
- Discovery with multiple roots
- Missing Memories app
- Empty or invalid `timeline_path`

**Mergeable**: Yes. Read-only discovery with clear failure modes.

---

## Step 3: Define Command UX and Guardrails

**Files**:

- `adapters/nextcloudmemories/command.go`
- `docs/plans/2026-nextcloud-memories-import/*`

**Changes**:

- Register `from-nextcloud-memories` under `upload`
- Add source flags for URL, user, password, discovery, root limiting, album sync, and indexing override
- Enforce root limits against configured Memories roots only
- Add `--discover-only` mode

**Testable**:

- Flag validation
- Discover-only output flow
- Rejecting non-configured root overrides

**Mergeable**: Yes. Command shell can land before full import, and on this feature branch it does not need to remain hidden solely because follow-up steps are still in progress.

---

## Step 4: Enumerate Source Assets From Memories Scope

**Files**:

- `adapters/nextcloudmemories/browse.go`
- `internal/nextcloud/webdav.go`

**Changes**:

- Enumerate files inside the resolved Memories timeline roots
- Filter to supported media types
- Respect Memories-scoped exclusions where practical
- Build `assets.Asset` values for the shared upload pipeline

**Testable**:

- Enumeration across multiple roots
- Media filtering
- Stable file identity and duplicate handling

**Mergeable**: Yes. File import can work before album recreation.

---

## Step 5: Enrich Assets With Metadata and Flags

**Files**:

- `adapters/nextcloudmemories/metadata.go`
- `internal/nextcloud/memories.go`

**Changes**:

- Read per-file metadata needed for date, GPS, description, rating, favorite, and archive state
- Map that data onto `assets.Metadata` and `assets.Asset`
- Warn or fail when the library appears partially indexed, unless overridden

**Testable**:

- Metadata decoding
- Flag mapping
- Partial-index detection behavior

**Mergeable**: Yes. Improves fidelity without changing destination flow.

---

## Step 6: Recreate Albums

**Files**:

- `adapters/nextcloudmemories/albums.go`

**Changes**:

- Resolve album membership from Memories
- Recreate albums in Immich through the existing upload pipeline
- Document unsupported album semantics such as hidden/shared/collaborative state

**Testable**:

- Album listing and membership mapping
- Repeatable runs without duplicate album creation

**Mergeable**: Yes. Album support can land after base import.

---

## Step 7: Persist Shared-Album Reconstruction State

**Files**:

- `adapters/nextcloudmemories/albums.go`
- `docs/plans/2026-nextcloud-memories-import/shared-album-reconciliation.md`

**Changes**:

- Use Memories `album_id` as the canonical shared-album source key
- Append a managed machine-readable state block to destination album descriptions for owned albums
- Optionally stamp assets with synthetic source album membership tags to support later reconciliation
- Keep this state server-stored so retries do not depend on local manifests

**Testable**:

- State block parsing and replacement
- Tag namespace generation
- Idempotent reruns without duplicate state

**Mergeable**: Yes. This can land before reconciliation if the state format is stable.

---

## Step 8: Restore Shares and Reconcile Shared Albums

**Files**:

- `adapters/nextcloudmemories/albums.go`
- `adapters/nextcloudmemories/reconcile.go`

**Changes**:

- Restore owned-album collaborators when a Nextcloud-to-Immich user mapping is available
- Add a user-driven reconciliation mode that matches synthetic asset tags to accessible destination albums using the persisted source `album_id`
- Add optional cleanup of successfully resolved synthetic tags, disabled by default

**Testable**:

- Share restoration for owned albums
- Reconciliation after owner import
- Repeatable reconciliation runs
- Cleanup behavior gated behind an explicit flag

**Mergeable**: Yes. Share restoration and reconciliation can be introduced after the server-stored state model is settled.

---

## Step 9: Tests and Public Documentation

**Files**:

- `adapters/nextcloudmemories/*_test.go`
- `docs/commands/upload.md`
- `docs/commands/README.md`
- `readme.md`

**Changes**:

- Add focused tests for discovery, enumeration, metadata, and albums
- Keep user-facing docs aligned with the implemented command surface on this feature branch, then ensure merged docs match merged behavior
- Document supported and unsupported source data explicitly

**Testable**:

- `go test` coverage for the new adapter
- Manual dry-run validation against a real Memories instance

**Mergeable**: Yes, once the documented behavior is implemented and tested at the level described by the docs being promoted.