# Architecture

This document describes the high-level architecture of `immich-go`, how its components interact, and where to extend it with new features.

## Overview

`immich-go` is a layered application that imports photos from diverse sources and uploads them to an Immich server. It follows a **source-agnostic pipeline**: read files and metadata from an adapter → normalize to a common `Asset` model → upload to Immich → apply tags and albums.

```
┌─────────────────────────────────────────────────────────────┐
│  CLI Layer (app/)                                           │
│  - Commands: upload, archive, stack                         │
│  - Flag parsing & user interaction                          │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│  Adapter Layer (adapters/)                                  │
│  - folder: local directory reader                           │
│  - googlePhotos: Google Takeout parser                       │
│  - icloud: iCloud reader                                    │
│  - fromimmich: server-to-server                             │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│  Asset Model (internal/assets/)                             │
│  - Unified Asset struct with metadata                       │
│  - Metadata extraction & merging                            │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│  Processing Layer (internal/)                               │
│  - File type detection (filetypes)                          │
│  - EXIF parsing (exif)                                      │
│  - Concurrency/job queue (worker)                           │
│  - Duplicate detection (assettracker)                       │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│  Immich Client (immich/)                                    │
│  - HTTP client wrapper                                      │
│  - API endpoints (upload, tag, album, etc.)                 │
└────────────────────┬────────────────────────────────────────┘
                     │
              Immich Server
```

## Layer Details

### 1. CLI Layer (`app/`)

**Responsibility**: Parse user input, orchestrate workflows, display results.

**Key files**:
- `root/rootCmd.go` — Entry point, global flags
- `upload/upload.go` — Upload command logic
- `archive/archiveCmd.go` — Export/archive command
- `stack/stack.go` — Photo stacking command

**Pattern**: Each command is a Cobra command with its own flow:
1. Validate flags and authentication
2. Create an adapter or client
3. Fetch/process assets
4. Call processing layer functions
5. Report results

### 2. Adapter Layer (`adapters/`)

**Responsibility**: Read assets and metadata from diverse sources, normalize to the internal `Asset` model.

**Adapters**:
- **`folder/`** — Reads local filesystem
  - Discovers files recursively
  - Creates albums from directory names
  - Pairs XMP sidecar files
  
- **`googlePhotos/`** — Parses Google Takeout exports
  - Extracts metadata from `.json` files
  - Groups photos with their JSON companions
  - Reconstructs albums and people tags
  
- **`icloud/`** — Handles iCloud exports
  - Reads CSV metadata
  - Parses iCloud-specific file structures
  - Reconstructs album info
  
- **`fromimmich/`** — Fetches from another Immich server
  - Uses Immich API to list assets
  - Applies server-side filtering
  - Re-uploads to target server

**Interface** (implicit):
Each adapter is responsible for:
1. Discovering assets (filenames, paths)
2. Opening/reading file content
3. Extracting metadata (date, location, description, tags, albums)
4. Merging into the normalized `Asset` model

### 3. Asset Model (`internal/assets/`)

**Responsibility**: Unified representation of a photo/video with all metadata.

**Key type**: `Asset`
```go
type Asset struct {
    File              fshelper.FSAndName  // filesystem reference
    OriginalFileName  string              // name on original source
    ID                string              // Immich ID (after upload)
    Checksum          string              // SHA1 hash (from Immich)
    
    // Core metadata
    CaptureDate       time.Time           // when photo was taken
    Description       string              // user description
    Favorite          bool                // marked as favorite
    Archived          bool                // marked as archived
    Trashed           bool                // marked as trashed
    Rating            int                 // star rating (0-5)
    
    // Organization
    Albums            []Album             // album memberships
    Tags              []Tag               // hierarchical tags
    Visibility        Visibility          // archive/timeline/hidden/locked
    
    // Location
    Latitude, Longitude float64           // GPS coordinates
    
    // Metadata sources (for precedence/debugging)
    FromSideCar       *Metadata           // extracted from XMP
    FromSourceFile    *Metadata           // extracted from EXIF
    FromApplication   *Metadata           // from JSON/CSV metadata
}
```

**Related types**:
- `Tag` — hierarchical tag (`Name`, `Value`)
- `Album` — album membership (`Title`, `Description`)
- `Metadata` — extracted metadata bundle

**Usage in pipeline**:
1. Adapters populate `Asset` fields
2. Processing layer enriches/validates
3. Upload layer sends to Immich
4. Client applies tags/albums after upload

### 4. Processing Layer (`internal/`)

**Responsibility**: Enrich, validate, and deduplicate assets before upload.

**Key packages**:

- **`filetypes/`** — Detect media type from extension
  - Maps `.jpg` → `image`, `.mp4` → `video`
  - Validates against Immich-supported types

- **`exif/`** — Parse EXIF metadata from files
  - Extracts date taken, GPS, camera model
  - Falls back to filename if EXIF missing

- **`worker/`** — Concurrency and job queue
  - Manages goroutine pool for parallelism
  - Handles upload rate limiting and retries

- **`assettracker/`** — Duplicate detection
  - Tracks uploaded assets by checksum
  - Prevents re-uploading same file

- **`filenames/`** — Parse dates from filename patterns
  - ISO format: `2023-07-15_14-30-25.jpg`
  - Phone format: `IMG_20230715_143025.jpg`

- **`filters/`** — Asset filtering logic
  - Date range filters
  - Album/tag matching

- **`fshelper/`** — Filesystem abstraction
  - Unified API for reading from zip, directory, etc.

### 5. Immich Client (`immich/`)

**Responsibility**: HTTP communication with Immich API.

**Key types**:

- `ImmichClient` — Main API wrapper
  ```go
  type ImmichClient struct {
      baseURL     string
      apiKey      string
      http        *http.Client
      dryRun      bool
      DeviceUUID  string
  }
  ```

- `Asset` — Immich's API asset representation
- `Tag`, `Album` — Immich data models
- Responses: `AssetResponse`, `TagAssetsResponse`, etc.

**Core operations**:

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `AssetUpload(ctx, asset)` | `POST /assets` | Upload a single asset |
| `UpsertTags(ctx, tagNames)` | `PUT /tags` | Create/get tags by name |
| `TagAssets(ctx, tagID, assetIDs)` | `PUT /tags/{id}/assets` | Apply tag to assets |
| `GetAlbums(ctx)` | `GET /albums` | List albums on server |
| `CreateAlbum(ctx, album)` | `POST /albums` | Create album |
| `AddAssetsToAlbum(ctx, albumID, assetIDs)` | `PUT /albums/{id}/assets` | Add assets to album |
| `DeleteAssets(ctx, assetIDs, force)` | `DELETE /assets` | Remove assets |

**Special features**:
- **Dry-run mode** (`dryRun=true`): Simulates API calls, returns synthetic IDs
- **Checksum optimization**: Sends SHA1 header to detect duplicates on server
- **Multipart upload**: Streams large files with metadata in single request
- **Sidecar support**: Optionally includes XMP files with asset

---

## Data Flow: Upload Example

Here's how a photo flows through the system:

```
1. User runs:
   immich-go upload from-folder --server=... --api-key=... /photos

2. CLI (app/upload/)
   ├─ Parse flags, authenticate
   └─ Create folder adapter

3. Adapter (adapters/folder/)
   ├─ Walk filesystem recursively
   ├─ Find JPEG: /photos/2023/vacation.jpg
   ├─ Find XMP sidecar: /photos/2023/vacation.xmp
   └─ Populate Asset:
       {
         File: fshelper.FSName(dirfs, "vacation.jpg"),
         OriginalFileName: "vacation.jpg",
         CaptureDate: parse from XMP or EXIF,
         Tags: parse from XMP keywords,
         Albums: ["2023", "vacation"]
       }

4. Processing (internal/)
   ├─ filetypes: confirm ".jpg" → image
   ├─ exif: extract EXIF date, GPS if available
   ├─ assettracker: compute checksum, check for duplicates
   └─ worker: queue asset for upload

5. Upload (app/upload/)
   ├─ For each asset:
   │   ├─ Client.AssetUpload(asset)
   │   │   └─ POST /assets with multipart: file + sidecar + metadata
   │   ├─ Store returned asset ID
   │   └─ Fetch or create albums/tags
   │       ├─ Client.UpsertTags(["2023", "vacation"])
   │       ├─ Client.TagAssets(tagID, [assetID])
   │       └─ Client.AddAssetsToAlbum(albumID, [assetID])
   │
   └─ Report: "Uploaded 1 asset, 2 tags, 1 album"

6. Immich Server
   ├─ Stores asset
   ├─ Generates thumbnails
   ├─ Indexes metadata
   └─ User sees photo in timeline
```

---

## Extension Points: Adding BeReal Support

When adding a new import source (e.g., BeReal), you need to:

### 1. Create an Adapter (`adapters/bereal/`)

```go
package bereal

import "github.com/simulot/immich-go/internal/assets"

type BeRealAdapter struct {
    sourceDir string  // path to BeReal export
    fsys      fs.FS   // filesystem abstraction
}

func NewBeRealAdapter(dir string) (*BeRealAdapter, error) {
    // Initialize and validate directory structure
}

func (ba *BeRealAdapter) DiscoverAssets(ctx context.Context) (chan *assets.Asset, error) {
    // Scan BeReal directory structure
    // Parse main + selfie image pairs
    // Extract metadata (capture time, location, etc.)
    // Yield assets on channel
    
    // For each memory:
    // 1. Create Asset for main image
    //    a. Add tag: "BeReal_Main"
    // 2. Create Asset for selfie
    //    a. Add tag: "BeReal_Selfie"
    // 3. Group in album: "BeReal"
    // 4. Set CaptureDate from BeReal metadata
}
```

### 2. Wire into CLI Command

In `app/upload/run.go` or new command:
```go
case "from-bereal":
    adapter, err := bereal.NewBeRealAdapter(sourceDir)
    if err != nil {
        return err
    }
    // Process assets like existing adapters
```

### 3. Use Immich Client for Tagging

Already implemented — use existing methods:
```go
client.UpsertTags(ctx, []string{"BeReal_Main", "BeReal_Selfie"})
client.TagAssets(ctx, tagID, []string{assetID})
client.CreateAlbum(ctx, &Album{Title: "BeReal"})
client.AddAssetsToAlbum(ctx, albumID, []string{assetID})
```

Or use the new helper method you added:
```go
result, err := client.ImportBeRealMemory(ctx, mainAsset, selfieAsset)
```

---

## Key Patterns

### 1. Context Usage

All I/O operations accept `context.Context` for cancellation and timeouts:
```go
func (ba *BeRealAdapter) DiscoverAssets(ctx context.Context) (chan *assets.Asset, error)
func (ic *ImmichClient) AssetUpload(ctx context.Context, la *assets.Asset) (AssetResponse, error)
```

### 2. Dry-Run Mode

All destructive operations check `ImmichClient.dryRun`:
```go
if ic.dryRun {
    return AssetResponse{ID: uuid.NewString(), Status: UploadCreated}, nil
}
```

Users can test workflows safely with `--dry-run` flag.

### 3. Asset Metadata Merging

`Asset.UseMetadata(md *Metadata)` prioritizes sources:
```go
func (a *Asset) UseMetadata(md *Metadata) *Metadata {
    // Metadata fields overwrite only if currently zero/empty
    a.Description = md.Description
    a.CaptureDate = md.DateTaken
    a.MergeAlbums(md.Albums)   // append without duplicates
    a.MergeTags(md.Tags)
    return md
}
```

### 4. Filesystem Abstraction

All adapters use `fshelper.FSAndName` (wraps `fs.FS` + filename):
- Works with directories, zip files, remote URLs equally
- Lazy-loads file content (not all in memory at once)

### 5. Tag Hierarchy

Tags are hierarchical strings: `"Location/USA/California/SF"`
```go
type Tag struct {
    ID    string  // Immich tag ID
    Name  string  // leaf name: "SF"
    Value string  // full path: "Location/USA/California/SF"
}

func (a *Asset) AddTag(tag string) {
    // Adds tag with leaf name auto-extracted
}
```

---

## File Organization Summary

```
immich-go/
│
├── main.go                          Entry point
├── app/                             Commands layer
│   ├── root/                        Root command
│   ├── upload/                      Upload command + orchestration
│   ├── archive/                     Archive command
│   └── stack/                       Stacking command
│
├── adapters/                        Input adapters (source readers)
│   ├── folder/                      Local folder reader
│   ├── googlePhotos/                Google Takeout parser
│   ├── icloud/                      iCloud reader
│   ├── fromimmich/                  Server-to-server
│   └── shared/                      Common utilities (banned files, stack logic)
│
├── immich/                          Immich API client
│   ├── client.go                    Main client & HTTP wrapper
│   ├── asset.go                     Asset operations
│   ├── upload.go                    Multipart upload logic
│   ├── tag.go                       Tagging operations
│   ├── album.go                     Album operations
│   ├── call.go                      HTTP call abstraction
│   ├── bereal.go                    ← Your new BeReal import helper
│   └── ... (other endpoints)
│
├── internal/                        Internal utilities & processing
│   ├── assets/                      Asset model & metadata
│   │   ├── asset.go                 Asset struct
│   │   ├── tag.go                   Tag struct
│   │   ├── album.go                 Album struct
│   │   └── metadata.go              Metadata extraction
│   │
│   ├── exif/                        EXIF parsing
│   ├── filetypes/                   Media type detection
│   ├── filenames/                   Filename date parsing
│   ├── filters/                     Filtering logic
│   ├── fshelper/                    Filesystem abstraction
│   ├── worker/                      Job queue & concurrency
│   ├── assettracker/                Duplicate detection
│   ├── ui/                          User interface (progress bars)
│   └── ... (other utilities)
│
└── docs/                            Documentation
    ├── README.md                    Doc hub
    ├── technical.md                 File formats & metadata
    ├── architecture.md              ← This file
    ├── commands/                    Command reference
    ├── best-practices.md            Performance tips
    └── ...
```

---

## Running Tests

The project uses Go's standard testing + specialized test helpers:

```bash
# Unit tests
go test ./...

# E2E tests (requires running Immich server)
go test -tags=e2e ./internal/e2e/...

# Specific package
go test ./adapters/folder -v
```

See `docs/test.md` for detailed testing guidelines.

---

## Next Steps for BeReal Feature

1. **Finalize data format**: How does BeReal export provide the main + selfie images? (directory structure, metadata format, etc.)
2. **Create `adapters/bereal/` package**: Implement discovery and metadata extraction
3. **Test parsing**: Unit tests for BeReal metadata parsing
4. **Wire into CLI**: Add `upload from-bereal` command
5. **Add E2E tests**: End-to-end test with sample BeReal data
6. **Update docs**: Document BeReal import in `docs/commands/upload.md`

The `immich.ImportBeRealMemory()` helper you created can be used by the upload orchestration logic to simplify the tagging workflow.
