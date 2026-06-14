# Shared Album Reconstruction Draft

## Goal

Define a follow-up migration mode for `from-nextcloud-memories` that can restore shared albums incrementally across multiple user-scoped imports without depending on local state.

The model should fit `immich-go`'s existing user-first workflow:

1. Each user imports their own assets into their own Immich account.
2. Users who own albums recreate those albums in Immich and restore collaborators when user mapping is available.
3. A reconciliation run can later add the current user's assets into already-shared destination albums by matching persisted source-side album identity.

## Non-Goals

- Not replacing the single-user import path with an admin-only migration flow
- Not requiring a local manifest or shared filesystem state as the primary source of truth
- Not guaranteeing immediate completeness of shared albums after one user's import
- Not implementing admin reconciliation in the first iteration of this feature
- Not hiding migration state perfectly from destination users if Immich has no native metadata field for it

## Why Server-Stored State

Local state creates avoidable operational risk for a migration workflow:

- users may lose the file between runs
- different users may run imports on different machines
- retries and partial runs become harder to reason about
- support and troubleshooting become more complex

Because of that, the preferred design is to persist enough migration state directly in the destination Immich server, even if that means using less elegant fields such as album descriptions and synthetic tags.

## Canonical Source Key

The canonical source key for shared-album reconstruction is the source Memories numeric `album_id`.

Why:

- it is stable across collaborators on the same Nextcloud instance
- it survives album renames better than `owner/name`
- it is already exposed by the Memories APIs used for album cluster membership

Supporting metadata should still include:

- source owner UID
- source album name at import time
- schema version

## State Model

### Album State

Albums created by the owning user should carry a machine-readable state block appended to the destination album description.

Human description remains first. The migration state is appended after a blank line using explicit delimiters.

Example:

```text
Summer trip to Norway

--- immich-go:nextcloud-memories:v1 ---
{"source":"nextcloud-memories","schema_version":1,"album_id":12345,"owner_uid":"alice","album_name":"Roadtrip"}
--- /immich-go ---
```

Requirements:

- preserve any existing human description text
- replace only the managed machine block on reruns
- detect and reject conflicting album identity for the same destination album
- ignore malformed blocks with a warning instead of failing the whole run

Initial fields:

- `source`
- `schema_version`
- `album_id`
- `owner_uid`
- `album_name`

### Asset State

Assets can carry source album membership through synthetic tags when explicitly enabled.

Recommended namespace:

- `immich-go/src/nextcloud-memories/album/<album_id>`

Properties:

- one tag per source album membership
- idempotent on reruns
- user-visible but queryable through existing Immich tag APIs
- sufficient for later reconciliation without local state

This feature should be opt-in during import because it introduces migration-only tags into the destination library.

## Proposed Modes

### Mode 1: User Import

Import the authenticated user's own Memories-scoped assets and metadata.

Behavior:

- upload the user's assets
- recreate album membership for the user's visible library
- recreate albums owned by the importing user
- optionally append the album state block to owned destination albums
- optionally tag assets with source album membership tags
- restore shares on owned albums when user mapping is available

### Mode 2: User Reconciliation

Re-run after one or more user imports to add the current user's assets into already-existing shared albums the user can access.

Behavior:

- list accessible destination albums
- parse machine blocks from album descriptions
- build `source album_id -> destination album` for albums accessible to the current user
- inspect the current user's assets for synthetic membership tags
- add assets into matching accessible albums when missing
- report unresolved source album IDs as warnings

This mode is intended to be safe and repeatable. It should converge gradually as more album owners complete their imports.

### Mode 3: Admin Reconciliation

This remains a separate follow-up feature.

Expected responsibilities:

- validate user mapping at the server level
- repair incomplete or out-of-order migrations
- reconcile viewer-only limitations where user runs cannot add assets
- optionally clean up migration state across the whole instance

## Reconciliation Algorithm

For the current user:

1. Query accessible albums from Immich.
2. Parse the managed description block from each album.
3. Build a lookup keyed by source `album_id`.
4. Query the current user's assets.
5. Read synthetic membership tags from each asset.
6. For each membership tag:
   - if a matching accessible album exists, ensure the asset is in that album
   - if no matching accessible album exists, record a warning
7. Summarize:
   - albums matched
   - assets added
   - unresolved album IDs
   - permission failures

This run must be idempotent.

## Ordering Model

Shared-album completeness becomes eventual rather than strictly ordered.

Rules:

- users may import their own assets before album owners import their albums
- owners should import before collaborators expect reconciliation to succeed for that album
- reconciliation can be run multiple times safely
- the owner does not need to run first for asset upload correctness, only for shared-album convergence

## Share Restoration

Owned-album import should restore collaborators when a Nextcloud-to-Immich user mapping is available.

Initial scope should prefer direct user collaborators only.

Deferred items:

- group collaborators
- public album links
- richer permission translation beyond the minimum viable owner/editor/viewer mapping

Important limitation:

- if a user only has viewer access to a destination album, that user cannot perform self-service reconciliation for assets missing from that album

This is a strong reason to keep admin reconciliation as a later companion feature.

## Cleanup

Cleanup should exist, but it should not be enabled by default.

Recommended shape:

- `--cleanup-migration-tags=false` on the reconciliation command or step

Behavior when enabled:

- remove only synthetic asset membership tags that were successfully resolved during the current reconciliation run
- leave unresolved tags intact so later runs still have the necessary source state
- do not remove album description state by default

Why conservative cleanup is safer:

- reconciliation is expected to run multiple times
- owners and collaborators may import out of order
- premature cleanup would destroy the only server-stored source of truth

Album description cleanup should be treated as a separate, later action with stronger safeguards.

## Open Questions

1. Should album state blocks be hidden behind an opt-in flag initially, or always enabled for owned-album import once the feature ships?
2. Should description-block parsing tolerate multiple legacy block formats, or only the current versioned one?
3. Should reconciliation query all current-user assets, or only assets tagged with the migration namespace?
4. Should unresolved album IDs be emitted as structured output for later admin reconciliation?
5. Should migration-tag cleanup be limited to albums that are currently shared with the user, or all successfully matched albums?