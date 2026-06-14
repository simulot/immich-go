# Nextcloud Memories Import Progress

## Current Status

**Phase**: Command supports source metadata, album mapping, owned-album share restoration, and live-tested lazy metadata loading

**Last Updated**: 2026-06-04

**Summary**:

The command shape, scope model, guardrails, and draft user-facing documentation have been outlined. The internal Nextcloud client layer uses `gowebdav` for basic DAV access plus custom `net/http` for non-DAV APIs. Discovery is implemented, and the command can enumerate supported media from the selected Memories timeline roots and hand those assets to the existing upload pipeline. Enumeration uses DAV directory walking. The command also supports reading file contents from a local synced directory instead of WebDAV while still using server discovery for Memories scope validation. Upload preparation reuses a single cached source read for checksum calculation and upload streaming, which removes an avoidable second fetch for non-local sources such as WebDAV. Source-side metadata mapping is now wired through Memories day and image-info APIs, including capture date, GPS, description, rating, favorite, archive state, tags, and album membership. After live testing against a real Nextcloud Memories instance, metadata enrichment was changed from eager full-library image-info hydration to a lazy per-asset lookup model so imports no longer stall before Browse() can emit assets. The importer also now tolerates observed live payload variants such as tag arrays and numeric album `shared` flags, and surfaces preparation progress through normal `INFO`/`DEBUG` logging plus concise terminal messages. Because development is happening on a feature branch, there is no longer a plan-level requirement to keep the command or matching docs hidden before merge; the remaining gate is implementation readiness.

Shared-album reconstruction now has its first implemented restore path: owned albums store source state on the destination, the importer can read Nextcloud DAV collaborator metadata, and reruns restore mapped album collaborators idempotently through the Immich album-user APIs.

The follow-up user reconciliation phase is now being split into a dedicated top-level `immich-go reconcile` command so post-import shared-album convergence does not overload `upload from-nextcloud-memories`.

Live migration review on 2026-06-13 found that album reconstruction and synthetic album-membership tagging are not yet reliable on real libraries. A follow-up plan in `docs/plans/2026-nextcloud-memories-identity-fix/` now tracks correcting the source identity model and duplicate-membership preservation before this importer should be considered ready for album migration.
---

## Step Tracking

- [x] Define the command direction and scope model
  - Import should target the configured Memories library, not arbitrary Nextcloud folders
  - Auto-discovery is the default behavior

- [x] Define escape hatches and guardrails
  - Drafted root-limiting, discovery-only, and indexing override options
  - Drafted failure behavior for missing app and invalid configuration

- [x] Draft user-facing command documentation
  - Added a planned command reference and examples in `user-docs-draft.md`
  - Kept draft docs in the plan folder initially while the command surface was still changing

- [x] Scaffold command UX and validation
  - Added a `from-nextcloud-memories` command shell
  - Added source flag registration, normalization, and validation for the planned UX contract
  - Added tests for command metadata, flag validation, and explicit not-implemented failure paths
  - Added a human-readable scaffold summary so the CLI UX can be reviewed before discovery exists

- [x] Choose dependency strategy and add client layer
  - Added `internal/nextcloud` as the home for the source-side client abstraction
  - Chose `gowebdav` for DAV access and custom `net/http` for OCS and Memories endpoints
  - Added tests for URL normalization, request construction, and OCS header behavior

- [x] Implement source authentication and discovery
  - Added OCS and Memories discovery helpers in `internal/nextcloud`
  - Wired `--discover-only` to validate DAV access and print detected source configuration
  - Added tests for discovery responses, empty timeline roots, and command output

- [x] Implement file enumeration from configured Memories roots
  - Added a read-only DAV-backed filesystem wrapper in `internal/nextcloud`
  - Added `Browse()` for the Nextcloud Memories adapter so imports now emit uploadable assets
  - Filters supported media, ignores common banned/useless files, and deduplicates overlapping selected roots

- [x] Drop the unverified WebDAV `SEARCH` path
  - Removed the untested recursive DAV `SEARCH` implementation and its adapter fast path
  - Enumeration now uses the standard DAV directory walk only
  - Kept the local synced directory optimization for fast file reads during migration

- [x] Allow local synced directory as preferred file source
  - Added `--nextcloud-local-dir` to prefer a local sync directory when opening asset contents while keeping Memories as the source of truth
  - Enumeration still comes from Nextcloud discovery and DAV-backed browsing, so local files are an optimization path rather than an override
  - Added focused tests for the layered local-first file source behavior

### 2026-06-02: No Short Alias

**Decision**: Keep the command name as `from-nextcloud-memories` without a `from-nc-memories` alias.

**Rationale**:

- The full name is explicit enough for copy-paste use
- Shared examples and support instructions are clearer with one canonical spelling
- The local copy optimization is the more valuable usability improvement

- [x] Reuse cached source reads across checksum and upload
  - Centralized asset cache creation so checksum and upload share the same cached representation
  - Avoids a second source fetch for non-local readers such as Nextcloud WebDAV
  - Added regression tests that assert a single source open across checksum and upload flows

- [x] Implement metadata and album mapping
  - Added Memories timeline and per-file image-info API helpers in `internal/nextcloud`
  - Built a source metadata index keyed by the file paths returned from Memories image-info
  - Mapped capture date, GPS, description, rating, favorite, archive state, tags, and album membership onto imported assets
  - Added partial-index detection so DAV-enumerated files without Memories metadata warn and continue by default, with `--require-indexed` available for fail-closed imports
  - Shared albums are renamed with an owner suffix when needed to avoid album title collisions in Immich

- [x] Make metadata enrichment robust for live Memories libraries
  - Switched from eager full-library `image/info` hydration to lazy per-asset metadata lookup during browse
  - Kept metadata enrichment best-effort so source-side API incompatibilities no longer collapse imports into `0 assets found`
  - Added compatibility handling for observed live payload variants including array-shaped `tags` and numeric album `shared` flags
  - Added preparation progress logging that is visible in both the log file and normal no-UI terminal runs
  - Verified the revised importer with real dry-run and non-dry-run test migrations against a live Nextcloud Memories source and test Immich destination

- [x] Add focused tests for discovery and base enumeration
  - Added DAV filesystem tests for path handling and file reads
  - Added adapter browse tests for media filtering and overlapping-root deduplication

- [x] Draft shared-album reconstruction follow-up
  - Persist source album identity on destination albums with a managed description block
  - Persist source album membership on assets with synthetic tags behind an opt-in flag
  - Add a repeatable user reconciliation step
  - Keep migration-tag cleanup opt-in and disabled by default

- [x] Start shared-album reconstruction state in the importer
  - Owned source albums now stamp a managed description block on recreated destination albums
  - Added a hidden `--tag-album-membership` opt-in flag to stamp synthetic source album membership tags on imported assets
  - Added focused tests for the description block format and synthetic tag generation

- [x] Restore owned-album collaborators when explicit user mappings are provided
  - Added Immich album-user client support for reading album users, adding users, and updating roles
  - Added Nextcloud DAV collaborator lookup for owned Memories albums via `nc:collaborators`
  - Added repeatable `--user-map` flags so source user IDs can be mapped to destination Immich user IDs
  - The upload pipeline now asks source adapters for desired album users and restores only missing or mismatched collaborators on reruns
  - Added focused tests for the DAV collaborator lookup, user-map parsing, adapter share resolution, and upload-side album-user restoration

- [ ] Align command visibility and public docs with actual feature readiness on this branch

### 2026-06-02: Album Membership Uses Source Metadata

**Decision**: Resolve album membership from per-file Memories image-info responses instead of building a separate album-first traversal.

**Rationale**:

- It keeps DAV enumeration as the single source of asset discovery
- It avoids a second source of truth for file selection and duplicate handling
- The shared upload pipeline already recreates albums once asset membership is populated
- It makes partial indexing visible immediately because missing image-info means missing source metadata

### 2026-06-04: Metadata Hydration Must Not Block Asset Enumeration

**Decision**: Keep Memories metadata enrichment, but switch from eager full-library `image/info` hydration to lazy per-asset loading during browse.

**Rationale**:

- live testing showed that eager hydration can front-load tens of thousands of `image/info` requests before the first asset is emitted
- this made the importer appear hung even when DAV enumeration itself was healthy
- lazy loading preserves metadata fidelity for imported assets while restoring the earlier "enumerate first" behavior of the DAV-backed importer
- best-effort enrichment is safer for real Memories deployments that may return payload variants not covered by the original typed structs

### 2026-06-13: Memories Asset Identity Must Drive Metadata Mapping

**Decision**: Treat the Nextcloud Memories asset ID as the canonical source identity, and use file paths only to resolve discovered files to that source asset.

**Rationale**:

- live migration evidence showed the current metadata join misses most discovered files in real libraries
- album membership and synthetic migration tags belong to the Memories asset object, not to a basename-derived path guess
- upload deduplication can only preserve album membership correctly if source occurrences resolve to stable source asset records first
- diagnostics should distinguish source indexing gaps from importer join failures

### 2026-06-03: Shared Albums Should Prefer Server-Stored Migration State

**Decision**: Shared-album reconstruction should prefer state persisted in the destination Immich server rather than local manifests.

**Rationale**:

- user-scoped imports may be run from different machines
- retries should not depend on preserving local files between runs
- support and troubleshooting are simpler when the destination server remains the source of truth for migration state
- the current Immich API has no dedicated custom album metadata field, so the practical draft uses a managed block in album descriptions plus synthetic asset tags

**Follow-Up Draft**:

- use Memories `album_id` as the canonical source album key
- append a versioned machine-readable block to owned destination album descriptions
- optionally tag assets with source album membership tags to enable later reconciliation
- restore owned-album shares when user mapping is available
- add reconciliation with cleanup disabled by default so repeated runs remain safe

---

## Decisions

### 2026-06-06: Feature Branch Can Expose In-Progress Docs And Command Surface

**Decision**: On this feature branch, the command and matching user-facing docs do not need to stay hidden purely because the work is still in progress.

**Rationale**:

- the branch will only be merged once the feature is ready
- keeping the real command surface and docs visible makes iteration easier
- the important guardrail is merge readiness, not temporary branch-local visibility

### 2026-06-01: Public Docs Stay Accurate

**Decision**: Do not add `from-nextcloud-memories` to `docs/commands/upload.md` yet.

**Rationale**:

- The command is not implemented
- The project guidelines require user-facing docs to track shipped behavior
- Draft UX docs belong in `docs/plans/` until code and tests exist

### 2026-06-01: No Positional Source Path

**Decision**: The proposed command should not take a positional `<source-path>`.

**Rationale**:

- A Memories migration should import the configured Memories library
- Requiring manual folder selection would make this a generic Nextcloud importer instead
- Existing `from-folder` already covers the generic import case

### 2026-06-01: `timeline_path` Defines Scope

**Decision**: Default import scope should be derived from the effective Memories `timeline_path` configuration.

**Rationale**:

- `timeline_path` is the actual Memories library scope
- `folders_path` is a UI navigation root, not the library definition
- Importing outside `timeline_path` would violate user expectations

### 2026-06-01: Command Started Hidden During Early Scaffolding

**Decision**: The scaffold command started hidden while discovery and browsing were still missing.

**Rationale**:

- It allowed iterative work on command UX in the real CLI surface
- It reduced confusion while the command was only a stub
- This is now superseded by the 2026-06-06 feature-branch decision

### 2026-06-01: Use `gowebdav` Only For DAV

**Decision**: Use `github.com/studio-b12/gowebdav` for DAV operations, while keeping OCS and Memories calls in `internal/nextcloud` with custom `net/http` code.

**Rationale**:

- DAV is the stable and reusable part of the source integration
- OCS and Memories endpoints still need importer-specific request shaping and error handling
- This avoids overcommitting to a niche Nextcloud client dependency that still would not cover Memories properly
- It keeps the dependency surface small and aligns with the project's preference for minimal external libraries