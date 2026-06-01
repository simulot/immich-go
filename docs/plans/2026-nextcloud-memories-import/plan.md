# Nextcloud Memories Import Plan

## Step 1: Add Source Client and Authentication

**Files**:

- `internal/nextcloud/...`
- `adapters/nextcloudmemories/...`

**Changes**:

- Add a minimal Nextcloud client for authenticated WebDAV and HTTP API calls
- Support source authentication with username plus password or app password
- Add source TLS and timeout configuration

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

**Mergeable**: Yes. Command shell can land before full import.

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

## Step 7: Tests and Public Documentation

**Files**:

- `adapters/nextcloudmemories/*_test.go`
- `docs/commands/upload.md`
- `docs/commands/README.md`
- `readme.md`

**Changes**:

- Add focused tests for discovery, enumeration, metadata, and albums
- Move the user-facing docs from draft status into shipped docs only when the command exists
- Document supported and unsupported source data explicitly

**Testable**:

- `go test` coverage for the new adapter
- Manual dry-run validation against a real Memories instance

**Mergeable**: Only when implementation exists.