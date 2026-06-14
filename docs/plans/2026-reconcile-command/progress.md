# Reconcile Command Progress

## Current Status

**Phase**: Command scaffold

**Last Updated**: 2026-06-14

**Summary**:

A new top-level `reconcile` command is scaffolded to host post-import convergence workflows. The first target is Nextcloud Memories shared-album reconstruction, which is expected to consume server-stored migration state written during import. The current goal is to establish the CLI shape and package structure without coupling reconciliation command code to importer packages.

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
  - Current subcommand returns a clear not-yet-implemented error

- [x] Add focused scaffold tests
  - Added tests for top-level `reconcile` command metadata and missing-subcommand behavior
  - Added tests for `reconcile nextcloud-memories` command metadata and scaffold error behavior

- [x] Keep reconcile command code separate from importer packages
  - Added a reconcile-specific Nextcloud package under `app/reconcile/nextcloudmemories`
  - Kept importer/reconcile coupling at the migration-state contract level

- [ ] Define a reconciliation-specific interface
- [ ] Implement Nextcloud Memories user reconciliation
- [ ] Align plan and user-facing documentation with actual feature readiness
