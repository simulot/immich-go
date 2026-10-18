# Release Notes - v0.32.0

This release brings full compatibility with Immich V3.0.0. All upload, album, and server
communication paths have been updated to work with the new API while maintaining backward
compatibility with Immich V2.

## ✨ New Features

- **Immich V3 support** – Full compatibility with Immich V3.0.0. The tool now detects the
  server version and adapts its behavior accordingly, ensuring smooth operation regardless
  of which version you're running.
- **Server version detection** – `immich-go` now reads and exposes the server version at
  startup, enabling version-aware API calls.

## 🚀 Improvements

- **Album contents via search** – Album asset retrieval now uses the paginated
  `POST /api/search/metadata` endpoint instead of relying on `GET /api/albums/{id}` to
  return assets. This works with both Immich V2 and V3 and is more reliable for large
  albums.
- **Error handling overhaul** – Server error messages are now parsed according to the
  server version (Zod-based errors for V3, legacy format for V2), providing clearer and
  more actionable error output.


## 🔧 Internal Changes

- **Asset upload field changes** – Removed `deviceId` and `deviceAssetId` from the upload
  payload (dropped in Immich V3's `AssetMediaCreateDto`). These fields are silently
  ignored by V2, so this change is safe across versions.
- **Duration field handling** – The `duration` field has been updated to use `"0"` as a
  plain string value, which is accepted by both Immich V2 and V3.

- **Removed deprecated `replaceAsset`** – The `ReplaceAsset` method and its associated
  `asset.replace` permission have been removed entirely. Use `CopyAsset` instead.
- **Error processing moved to `immich/error.go`** – All server error parsing logic is now
  centralized with a `serverError` interface and separate `serverErrorV2`/`serverErrorV3`
  implementations.
- **Album contents E2E test** – Added `Test_GetImmichAlbums` to verify that album uploads
  and duplicate detection work correctly with the new search-based album retrieval.
