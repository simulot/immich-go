# Technical Details

This document covers the technical aspects of how Immich-Go processes files, handles metadata, and implements various features.

## File Processing

### Supported File Types
Immich-go supports same formats as Immich supports, among them:

#### Image Formats
- **JPEG**: `.jpg`, `.jpeg`
- **HEIC/HEIF**: `.heic`, `.heif` (Apple formats)
- **RAW Formats**: `.dng`, `.cr2`, `.cr3`, `.arw`, `.raf`, `.nef`, `.rw2`, `.orf`
- **Other**: `.png`, `.gif`, `.bmp`, `.tiff`, `.webp`

#### Video Formats
- **Common**: `.mp4`, `.mov`, `.avi`, `.mkv`
- **Mobile**: `.3gp`, `.m4v`
- **Professional**: `.mts`, `.m2ts`

#### Metadata Formats
- **XMP Sidecar**: `.xmp` files
- **Google Photos**: `.json` metadata files
- **Immich Archive**: `.JSON` metadata files

### Banned Files

Automatically excluded file patterns:

| Pattern              | Source          | Description         |
| -------------------- | --------------- | ------------------- |
| `@eaDir/`            | Synology NAS    | Thumbnail directory |
| `@__thumb/`          | Synology NAS    | Thumbnail directory |
| `SYNOFILE_THUMB_*.*` | Synology NAS    | Thumbnail files     |
| `Lightroom Catalog/` | Adobe Lightroom | Catalog directory   |
| `thumbnails/`        | Various         | Generic thumbnails  |
| `.DS_Store`          | macOS           | System metadata     |
| `._*.*`              | macOS           | Resource forks      |
| `.photostructure/`   | PhotoStructure  | Application data    |

### Date Extraction

#### Priority Order
1. **EXIF Metadata**: Primary source from image files
2. **XMP Sidecar**: Secondary source from `.xmp` files
3. **JSON Metadata**: From Google Photos or archive files
4. **Filename Parsing**: Last resort extraction from filenames

#### Filename Date Patterns

| Pattern      | Example                                 | Format                |
| ------------ | --------------------------------------- | --------------------- |
| ISO Format   | `2023-07-15_14-30-25.jpg`               | `YYYY-MM-DD_HH-MM-SS` |
| Timestamp    | `IMG_20230715_143025.jpg`               | `IMG_YYYYMMDD_HHMMSS` |
| Phone Format | `20230715_143025.jpg`                   | `YYYYMMDD_HHMMSS`     |
| Screenshot   | `Screenshot 2023-07-15 at 14.30.25.jpg` | Various patterns      |

#### Date Range Formats

| Input                   | Interpretation                      |
| ----------------------- | ----------------------------------- |
| `2023`                  | January 1, 2023 - December 31, 2023 |
| `2023-07`               | July 1, 2023 - July 31, 2023        |
| `2023-07-15`            | July 15, 2023 (single day)          |
| `2023-01-15,2023-03-15` | Explicit range                      |

## Metadata Handling

### XMP Sidecar Processing

Immich-Go passes XMP files to the Immich server without modification. Immich uses them for:
- **Date/Time Information**: Capture date and timezone
- **GPS Location**: Latitude/longitude coordinates
- **Tags/Keywords**: Hierarchical tag structures
- **Descriptions**: Photo descriptions and titles
- **Technical Data**: Camera settings, lens information

### Google Photos JSON Processing

Google Photos JSON files contain rich metadata extracted and used by Immich-Go:

```json
{
  "title": "IMG_20230715_143025.jpg",
  "description": "Family vacation photo",
  "imageViews": "1",
  "creationTime": {
    "timestamp": "1689424225",
    "formatted": "Jul 15, 2023, 2:30:25 PM UTC"
  },
  "geoData": {
    "latitude": 37.7749,
    "longitude": -122.4194,
    "altitude": 0.0,
    "latitudeSpan": 0.0,
    "longitudeSpan": 0.0
  },
  "people": [
    {
      "name": "John Doe"
    }
  ]
}
```

#### Extracted Information
- **Albums**: Photo album associations
- **GPS Coordinates**: Location data
- **Capture Date**: Original photo timestamp
- **People Tags**: Face recognition names
- **Status Flags**: Favorite, archived, trashed, partner shared

### Archive Metadata Format

Immich-Go generates comprehensive JSON metadata for archived photos:

```json
{
  "fileName": "example.jpg",
  "latitude": 37.7749,
  "longitude": -122.4194,
  "dateTaken": "2023-10-01T12:34:56Z",
  "description": "Golden Gate Bridge view",
  "albums": [
    {
      "title": "San Francisco Trip",
      "description": "Photos from my vacation",
      "start": "2023-10-01T00:00:00Z",
      "end": "2023-10-05T23:59:59Z"
    }
  ],
  "tags": [
    {
      "value": "Location/USA/California/San Francisco"
    },
    {
      "value": "Event/Vacation"
    }
  ],
  "rating": 5,
  "trashed": false,
  "archived": false,
  "favorited": true,
  "fromPartner": false,
  "motion": false,
  "livePhoto": false
}
```

## File Pairing and Stacking

### Burst Detection

#### Time-Based Detection
- **Threshold**: Photos taken within 900ms
- **Algorithm**: Groups consecutive photos below threshold
- **Limitations**: May group unrelated rapid shots

#### Filename-Based Detection

##### Huawei Smartphones
```
Pattern: IMG_YYYYMMDD_HHMMSS_BURSTXXX[_COVER].jpg
Example: IMG_20231014_183246_BURST001_COVER.jpg
         IMG_20231014_183246_BURST002.jpg
```

##### Google Pixel
```
Pattern: PXL_YYYYMMDD_HHMMSSXXX.MOTION-XX.COVER/ORIGINAL.jpg
Example: PXL_20230330_184138390.MOTION-01.COVER.jpg
         PXL_20230330_184138390.MOTION-02.ORIGINAL.jpg
```

##### Samsung
```
Pattern: YYYYMMDD_HHMMSS_XXX.jpg
Example: 20231207_101605_001.jpg
         20231207_101605_002.jpg
```

##### Sony Xperia
```
Pattern: DSC_XXXX_BURSTYYYYMMDDHHMMSSXXX[_COVER].JPG
Example: DSC_0001_BURST20230709220904977.JPG
         DSC_0035_BURST20230709220904977_COVER.JPG
```

##### Nothing Phones
```
Pattern: XXXXXIMG_XXXXX_BURSTXXXXXXXXXXXXX[_COVER].jpg
Example: 00001IMG_00001_BURST1723801037429_COVER.jpg
         00002IMG_00002_BURST1723801037429.jpg
```

### RAW + JPEG Pairing

#### Detection Algorithm
1. **Filename Match**: Same name, different extensions
2. **Directory Location**: Must be in same folder
3. **Timestamp Proximity**: Taken within reasonable timeframe
4. **Size Validation**: RAW significantly larger than JPEG

#### Supported RAW Extensions
- **Canon**: `.cr2`, `.cr3`
- **Nikon**: `.nef`
- **Sony**: `.arw`
- **Fujifilm**: `.raf`
- **Olympus**: `.orf`
- **Panasonic**: `.rw2`
- **Adobe**: `.dng`

#### Example Pairing
```
Files:
- IMG_1234.CR3 (RAW)
- IMG_1234.JPG (JPEG)

Result: Paired for stacking/management
```

### HEIC + JPEG Pairing

Common with Apple devices that shoot in both formats:

```
Files:
- IMG_1234.HEIC (Apple format)
- IMG_1234.JPG (Compatible format)

Detection: Same basename, different extensions
```

### Epson FastFoto Detection

Specialized handling for Epson FastFoto scanner output:

```
Files:
- photo-name.jpg      (Original scan)
- photo-name_a.jpg    (Corrected scan)
- photo-name_b.jpg    (Back of photo)

Stacking: All grouped with corrected (_a) as cover
```

## Upload Processing

### Duplicate Detection

#### Server-Side Deduplication
1. **Checksum Comparison**: SHA-1 hash of file content
2. **Metadata Match**: Filename and timestamp comparison
3. **Size Validation**: File size verification
4. **Skip Logic**: Existing files skipped unless `--overwrite` used

#### Benefits
- **Resumable Uploads**: Interrupted uploads can be safely restarted
- **Multiple Sources**: Same photos from different sources handled gracefully
- **Storage Efficiency**: No unnecessary duplicates on server

### Concurrency Management

#### Upload Workers
- **Default**: Number of CPU cores
- **Range**: 1-20 concurrent uploads
- **Bottlenecks**: Usually network bandwidth, not CPU

#### Performance Characteristics
Based on testing with various configurations:

| Concurrent Uploads | Network Utilization | Server Load | Reliability |
| ------------------ | ------------------- | ----------- | ----------- |
| 1                  | Low                 | Minimal     | Highest     |
| 2-4                | Moderate            | Low         | High        |
| 8-12               | High                | Moderate    | Good        |
| 16+                | Maximum             | High        | Variable    |

#### Resource Usage
- **Memory**: ~10-50MB per concurrent upload
- **Network**: Scales linearly with concurrency
- **CPU**: Minimal impact (I/O bound operation)

## Archive Structure

### Directory Organization

```
archive-folder/
├── 2022/
│   ├── 2022-01/
│   │   ├── IMG_001.jpg
│   │   ├── IMG_001.jpg.JSON
│   │   └── IMG_002.mp4
│   └── 2022-02/
│       └── ...
├── 2023/
│   ├── 2023-01/
│   └── 2023-02/
└── 2024/
    ├── 2024-01/
    └── 2024-02/
```

### Filename Preservation

#### Original Names
- **Primary Files**: Keep original filenames where possible
- **Conflict Resolution**: Append numbers for duplicates (`IMG_001(1).jpg`)
- **Character Sanitization**: Replace filesystem-incompatible characters

#### Metadata Files
- **JSON Sidecar**: `original-name.ext.JSON`
- **XMP Preservation**: Original `.xmp` files copied alongside photos

### Incremental Updates

#### Update Logic
1. **New Files**: Added to appropriate date folders
2. **Existing Files**: Checked for changes, updated if different
3. **Metadata Updates**: JSON files refreshed with current server state
4. **Deletions**: Files not removed from archive (archive-only operation)

### Storage Patterns

#### Temporary Files
- **Location**: System temp or `IMMICHGO_TEMPDIR`
- **Usage**: Minimal temporary storage needed
- **Cleanup**: Automatic cleanup on exit



## See Also

- [Configuration](configuration.md) - All configuration options
- [Commands](commands/) - Command-specific details
- [Best Practices](best-practices.md) - Performance optimization
- [Examples](examples.md) - Practical usage examples