# Reconcile Command Progress

## Current Status

**Phase**: First functional implementation

**Last Updated**: 2026-06-28

**Summary**:

A new top-level `reconcile` command now includes a functional implementation for Nextcloud Memories shared-album reconstruction. The command consumes server-stored migration state written during import via managed album description blocks and synthetic album-membership tags, adds the current user's matching assets into destination albums they can access, and can optionally remove those same album-membership tags from all reconciled current-user assets for each matched source album.

## Step Tracking

- [x] Evaluate CLI shape for reconciliation
  - Chosen direction: add `immich-go reconcile <source>` as a top-level operation
  - Rationale: reconciliation is destination-side post-import work, not another upload mode

- [x] Add top-level command scaffold
  - Added `app/reconcile` and registered `reconcile` in the root command
  - Base command currently requires a source subcommand and returns a clear error otherwise

- [x] Add `reconcile nextcloud-memories` scaffold
  - Added `nextcloud-memories` under `reconcile`
  - Added initial reconciliation-only flag surface with `--cleanup-migration-tags`
  - Current subcommand now opens an Immich connection and runs reconciliation logic

- [x] Add focused scaffold tests
  - Added tests for top-level `reconcile` command metadata and missing-subcommand behavior
  - Added tests for `reconcile nextcloud-memories` command metadata and scaffold error behavior

- [x] Keep reconcile command code separate from importer packages
  - Added a reconcile-specific Nextcloud package under `app/reconcile/nextcloudmemories`
  - Kept importer/reconcile coupling at the migration-state contract level

- [x] Define a reconciliation-specific interface
  - Added focused internal interfaces for album/tag access and tagged-asset enumeration
  - Kept command wiring separate from reconciliation logic for testability

- [x] Implement Nextcloud Memories user reconciliation
  - Parses managed album description state
  - Resolves source album IDs to synthetic migration tags
  - Enumerates current-user tagged assets and skips external-library assets
  - Adds missing assets to matching albums and reports unresolved/malformed cases

- [x] Align plan and user-facing documentation with actual feature readiness
  - Added `docs/commands/reconcile.md`
  - Updated command reference to include `reconcile`

- [x] Implement synthetic tag cleanup
  - Added Immich client support for `DELETE /tags/{id}/assets`
  - Cleanup now removes the specific synthetic album-membership tag consumed by reconciliation
  - Cleanup includes already-present current-user assets in the matched destination album set, not only newly added assets
