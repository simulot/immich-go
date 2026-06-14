# Reconcile Command Plan

## Step 1: Add Top-Level Command Scaffold

**Files**:

- `app/root/rootCmd.go`
- `app/reconcile/...`

**Changes**:

- Add a new top-level `reconcile` command
- Register it from the root command
- Keep the base command operation-first, mirroring `upload`, `archive`, and `stack`

**Testable**:

- Root command exposes `reconcile`
- Base command help and no-args behavior are correct

**Mergeable**: Yes. CLI structure only.

---

## Step 2: Add Source-Specific Reconciliation Subcommand Scaffold

**Files**:

- `app/reconcile/reconcile.go`
- `app/reconcile/nextcloudmemories/...`

**Changes**:

- Add `reconcile nextcloud-memories`
- Keep the command in reconcile-specific packages rather than importer packages
- Keep the command read-only or scaffolded until reconciliation behavior is implemented

**Testable**:

- Command metadata
- Flag registration
- Expected not-yet-implemented or scaffold summary behavior

**Mergeable**: Yes. Surface definition can land before full behavior.

---

## Step 3: Define Reconciliation Interface

**Files**:

- `app/reconcile/...`
- `immich/...` as needed

**Changes**:

- Introduce a reconciliation-specific interface rather than overloading upload interfaces
- Keep it small and explicit
- Align it with post-import destination-side workflows

**Testable**:

- Build-only validation through command wiring

**Mergeable**: Yes. Internal architecture only.

---

## Step 4: Implement Nextcloud Memories User Reconciliation

**Files**:

- `app/reconcile/...`
- `app/reconcile/nextcloudmemories/...`
- `immich/...` as needed

**Changes**:

- Query accessible albums and parse managed description blocks
- Query current-user assets with synthetic migration tags
- Add missing assets to matching albums idempotently
- Summarize matched albums, added assets, unresolved album IDs, and permission failures

**Testable**:

- Description parsing
- Tag decoding
- Idempotent album membership updates
- Unresolved album reporting

**Mergeable**: Yes.

---

## Step 5: Documentation Alignment

**Files**:

- `docs/plans/2026-reconcile-command/*`
- user-facing docs if command behavior becomes ready for branch visibility

**Changes**:

- Track scope and readiness of the new top-level command
- Keep the Nextcloud import plan consistent with the split between import and reconciliation

**Testable**:

- Doc review

**Mergeable**: Yes.
