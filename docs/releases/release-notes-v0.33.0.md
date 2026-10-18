# Release Notes - v0.33.0

**Release Date**: Unreleased

## Overview

This release introduces full support for importing BeReal memories with automatic back/front camera image handling, intelligent stacking, and proper tagging.

---

## ✨ New Features

### BeReal Memories Import (`from-bereal`)

A specialized import command for BeReal memories with complete support for the dual-camera nature of BeReal photos:

- **Automatic Camera Separation**: Separates back (main camera) and front (selfie camera) images into distinct assets
- **Smart Stacking**: Back and front images are automatically stacked together with the back camera as the cover image for viewing
- **Intelligent Tagging**: 
  - Back camera images tagged as `BeReal_Main`
  - Front camera images tagged as `BeReal_Selfie`
- **Metadata Preservation**:
  - Capture date and time extracted from BeReal export
  - GPS location preserved when available
  - Photo captions imported as asset descriptions
- **Album Organization**: All BeReal assets are added to a single `BeReal` album
- **Robust Export Handling**: Handles various BeReal export formats and path variations

**Command**:
```bash
immich-go upload from-bereal [flags] <path>
```

**Usage Example**:
```bash
immich-go upload from-bereal \
  --server=http://your-ip:2283 \
  --api-key=your-api-key \
  /path/to/bereal/export
```

---

## 🚀 Improvements

- **Architecture Enhancements**: Introduced `GroupByDualCamera` grouping type for flexible import source handling
- **Test Infrastructure**: Added E2E tests for BeReal import validation
- **Documentation**: Updated architecture documentation with BeReal as a standard adapter

---

## 🔧 Internal Changes

- Added dual-camera asset grouping support
- Enhanced metadata handling for GPS coordinates and descriptions
- Improved shell script compatibility for macOS and Linux
- Removed unused code and improved code quality

---

## Compatibility

- Same Immich server requirements as the rest of `immich-go` (see the [v0.32.0 release notes](release-notes-v0.32.0.md))
- Works with BeReal export format from official BeReal application
- Compatible with Windows, macOS, and Linux

---

## Installation

Download the latest release for your platform from [GitHub Releases](https://github.com/simulot/immich-go/releases).

---

## Known Limitations

- BeReal captions are imported as asset descriptions only (not as separate note assets)
- All BeReal assets go into a single `BeReal` album (no per-date albums)

---

## Testing

Comprehensive E2E tests verify:
- ✅ Correct asset upload (front and back images)
- ✅ Proper tagging of both camera images
- ✅ Stack creation with correct cover image
- ✅ All assets added to the `BeReal` album

---

## Related Issues

Closes: #1248 — Add support for importing BeReal memories

---

## Contributors

- Implemented by: Kurisudes and Copilot
- Thanks to the Immich community for feedback and testing

---

## Previous Releases

See [Release History](../releases/) for previous versions.
