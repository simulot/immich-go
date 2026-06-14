# Plan

## Goal

Make Nextcloud Memories imports use Memories asset identity as the source of truth so asset metadata, album recreation, and synthetic album-membership tags are applied reliably and repeatably.

## Non-goals

- full post-import reconciliation workflow
- changing the public UX beyond clarifying current flags and behavior
- fixing unrelated album logic for other importers

## Steps

1. Verify and document the source identity contract
   - Confirm which Memories APIs expose the canonical asset ID, canonical file path, and album membership.
   - Record the exact join strategy between DAV/local files and Memories assets.
   - Add or update low-level tests around the Nextcloud client payloads if needed.

2. Replace fragile metadata indexing with asset-identity-first indexing
   - Rework the metadata index so the primary source record is keyed by Memories asset ID.
   - Use path-based lookup only as a helper to resolve a discovered file to the correct Memories asset record.
   - Ensure selected-root filtering works with the canonical source path rather than basename-derived guesses.
   - Add regression tests covering nested timeline paths and live-like filenames.

3. Preserve album membership and synthetic tags through duplicate collapse
   - Audit `AlreadyProcessed`, `SameOnServer`, and related duplicate paths for Nextcloud Memories assets.
   - Ensure all album memberships and synthetic migration tags from equivalent source occurrences are merged onto the chosen Immich asset.
   - Keep the behavior idempotent across reruns.
   - Add focused tests for duplicate source occurrences carrying album membership.

4. Improve diagnostics and fail-closed options
   - Make metadata miss reporting clearly distinguish partial indexing from join failures.
   - Consider tightening or extending `--require-indexed` so identity mismatches are easier to detect during validation runs.
   - Add tests for the expected warnings and failure paths.

5. Update docs and plan tracking
   - Update the active Nextcloud Memories progress notes with the root-cause findings and the new source-identity decision.
   - Review user-facing docs for any wording that overstates current album migration reliability.
