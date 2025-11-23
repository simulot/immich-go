# Release Notes - v0.31.0

This release focuses on richer visibility while uploading assets and a major refresh of the CI/CD system that keeps Immich Go healthy across every platform.

## ✨ New Features

- **Live terminal tracking** – The TUI now exposes separate discovery and processing zones, FileArchived counters, and per-asset size tracking so you can immediately tell what is being scanned, uploaded, or archived.
- **Better metadata insight** – FileProcessor records processed assets (including metadata-only updates) and publishes dedicated events, yielding more accurate progress reporting and easier troubleshooting.

## 🚀 Improvements

- **Globbing resilience** – Folder traversal keeps going when the filesystem throws access errors and now surfaces clearer documentation about supported patterns.
- **Upload ergonomics** – Standardized error-handling flags across upload commands and improved UI layout by right-aligning size columns, making long-running jobs easier to read.

## 🐛 Bug Fixes

- **Album flag conflict** – `--folder-as-album=NONE` no longer clashes with `--into-album`, so you can explicitly skip derived album names while still targeting a destination album.

## 🔧 Internal Changes

- **Revamped CI/CD** – Introduced a two-stage fast-feedback + secure E2E workflow, nightly Immich E2E runs, fork-safe triggering, and far more robust helper scripts (doc-only detection, jq fixes, empty-diff handling, standardized formatting).
- **Safer E2E workflow** – External contributors now rely on an approval gate (or `/run-e2e` comment), and the workflow dispatch code was hardened with better payload validation, environment routing, and completion reporting.
- **Broader coverage** – Added an album-upload E2E test, refreshed dependency stack (tcell v2.11.0, crypto v0.45.0), and removed legacy journal/reporting structures in favor of a unified FileProcessor + new file-event codes.
