# Reconcile Command

## Goal

Introduce a top-level `immich-go reconcile` command for post-import convergence workflows that operate on destination-side migration state rather than raw source media.

The first use case is Nextcloud Memories shared-album reconstruction:

1. Users import their own assets with `immich-go upload from-nextcloud-memories`.
2. The import stores enough server-side migration state to identify source albums and asset membership.
3. Users later run `immich-go reconcile nextcloud-memories` to add their already-imported assets into matching shared destination albums they can access.

## Non-Goals

- Not replacing `upload` for source ingestion
- Not introducing an admin-wide reconciliation flow in the first iteration
- Not making reconciliation specific to one source in the top-level CLI shape
- Not implementing cleanup of all migration metadata in the first iteration

## Why a Top-Level Command

`upload` in `immich-go` is operation-first and source-specific. Reconciliation is a separate operation:

- it acts on destination Immich state
- it relies on migration markers already written during import
- it should be reusable for future source-specific reconciliation flows

That makes `immich-go reconcile <source>` a better fit than extending `upload from-nextcloud-memories` with more follow-up modes.

## Proposed Command Surface

```bash
immich-go reconcile nextcloud-memories [options]
```

Rationale:

- keeps the existing operation-first CLI structure
- makes reconciliation discoverable as a distinct phase
- leaves room for future source-specific reconcilers

## Implemented Scope

The first functional iteration is now implemented with the following behavior:

1. Connect to Immich as the current user
2. Enumerate accessible destination albums
3. Parse managed Nextcloud Memories album state from descriptions
4. Enumerate the current user's assets that carry synthetic source album membership tags
5. Add missing assets to matching albums when permissions allow
6. Report unmatched source album IDs and permission failures

Additional current behavior details:

- malformed managed album state is reported and skipped
- assets from external libraries are skipped
- already-present album membership is treated idempotently

Current limitation:

- cleanup is scoped to the specific synthetic album-membership tags consumed by reconciliation; it does not attempt broader tag cleanup beyond that contract

## Architectural Direction

- Add `app/reconcile` for the top-level command
- Keep source-specific logic in reconcile-specific packages, starting with `app/reconcile/nextcloudmemories`
- Do not force reconciliation into the existing upload abstraction
- Prefer a dedicated reconciliation interface oriented around destination-side convergence

## Follow-Up Fit With Existing Nextcloud Plan

This command becomes the natural home for the user reconciliation mode already drafted in the Nextcloud Memories import plans.

The import command remains responsible for writing reconciliation state; the reconcile command becomes responsible for consuming it.
