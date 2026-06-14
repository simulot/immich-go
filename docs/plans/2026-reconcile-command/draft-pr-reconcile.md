# Draft PR body: Reconcile command scaffold

## Summary

Add a top-level `immich-go reconcile` command and the first source-specific entry point for `reconcile nextcloud-memories`.

This PR is opened as a **draft** to describe the reconciliation feature track, establish the command shape, and document the migration-state contract it is expected to use.

Related:
- #1366

## Overview

This PR starts a separate post-import workflow for convergence tasks that operate on migration state already stored in Immich.

The current implementation establishes the CLI entry points, initial flags, and planning/docs for the first reconciliation target: Nextcloud Memories shared-album convergence.

## Key design choices

- **Top-level operation**: reconciliation is exposed as `immich-go reconcile <source>` rather than as another upload mode.
- **Destination-side workflow**: reconciliation operates on migration state already written to Immich instead of re-reading the source library.
- **Server-stored state contract**: the command is designed around managed album description blocks and synthetic membership tags written during import.
- **Explicit cleanup**: cleanup remains opt-in and conservative because reconciliation is expected to be rerun as imports converge.

## Implementation

- added top-level `immich-go reconcile`
- added `reconcile nextcloud-memories`
- added initial reconciliation-specific flag surface with `--cleanup-migration-tags`
- added command tests for top-level and source-specific scaffold behavior
- added plan and progress docs for the reconciliation track

## Scope and non-goals

Included now:

- top-level reconciliation command shape
- source-specific Nextcloud Memories reconciliation entry point
- initial flag surface
- tests and planning docs

Not included yet:

- actual user reconciliation logic
- admin/global reconciliation
- automatic cleanup behavior
- broad migration-state repair tools

## Limitations

- the current command is scaffold-level only and returns clear not-yet-implemented errors
- no album-state parsing or convergence behavior is implemented yet
- no cleanup behavior is implemented yet, even though the flag surface exists

## Intended feature behavior

The first functional implementation is expected to:

1. enumerate destination albums the current user can access
2. parse managed Nextcloud Memories album state from descriptions
3. enumerate the current user's assets carrying synthetic album-membership tags
4. add missing assets to matching destination albums
5. report unresolved albums and permission limitations

## Relationship to the importer PR

This feature depends on the importer writing stable migration state, but remains separate because it is a distinct destination-side workflow.

The importer PR remains responsible for:

- raw source ingestion
- metadata mapping
- album recreation
- writing the migration markers consumed here

## Testing

Added focused scaffold tests for:

- top-level `reconcile` command metadata and missing-subcommand behavior
- `reconcile nextcloud-memories` command metadata and scaffold error behavior

Remaining draft work:

- define a reconciliation-specific interface
- implement user reconciliation behavior
- add behavior tests around album-state parsing and asset-to-album convergence
- align docs with the eventual supported workflow

## Plan references

Relevant plan notes reviewed while shaping this PR:

- `docs/plans/2026-reconcile-command/README.md`
- `docs/plans/2026-reconcile-command/plan.md`
- `docs/plans/2026-reconcile-command/progress.md`
- `docs/plans/2026-nextcloud-memories-import/shared-album-reconciliation.md`

## Follow-up

Expected implementation work includes:

- reconciliation-specific interface design
- Nextcloud Memories user reconciliation logic
- structured reporting for unresolved album IDs and permission failures
- conservative cleanup behavior guarded by explicit flags
