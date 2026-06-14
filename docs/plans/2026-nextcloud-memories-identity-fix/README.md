# Nextcloud Memories Identity Fix

## Goal

Repair `upload from-nextcloud-memories` so source metadata, album reconstruction, and synthetic album-membership tags are driven by stable Nextcloud Memories asset identity instead of fragile path/basename joins.

## Problem Summary

Live migration evidence shows the importer currently discovers far more files than it can match to Memories metadata. As a result, most assets are imported without album membership or synthetic migration tags. The current upload deduplication flow also does not reliably preserve album/tag membership when multiple source occurrences collapse onto a single destination asset.

## Scope

This plan covers:

- source-side asset identity modeling for Nextcloud Memories
- reliable mapping from discovered files to Memories asset records
- preservation of album membership and synthetic tags through upload deduplication
- focused tests and documentation updates for the corrected behavior

## Non-goals

- implementing the full reconcile command
- redesigning generic upload behavior for non-Memories sources
- broad refactors outside the Nextcloud Memories importer and the minimal shared upload paths it depends on
