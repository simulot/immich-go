# Archive Command

The `archive` command exports photos and videos from various sources to a local folder structure organized by date. The destination folder isn't wiped out before the operation, so it's possible to add new photos to an existing archive.


## Syntax

```bash
immich-go archive <sub-command> --write-to-folder=<destination> [options] <source>
```

## Output Structure

Photos are organized chronologically:
```
destination-folder/
├── 2022/
│   ├── 2022-01/
│   │   ├── photo01.jpg
│   │   └── photo01.jpg.JSON    # Metadata file
│   └── 2022-02/
│       ├── photo02.jpg
│       └── photo02.jpg.JSON
├── 2023/
│   ├── 2023-03/
│   └── 2023-04/
└── 2024/
    ├── 2024-05/
    └── 2024-06/
```

## Required Options

| Option | Description |
|--------|-------------|
| `--write-to-folder` | Destination folder for archived photos |

## Sub-commands

All `upload` sub-commands are available for `archive`:

| Sub-command | Source | Description |
|-------------|--------|-------------|
| `from-folder` | Local filesystem | Archive from local folders or ZIP archives |
| `from-google-photos` | Google Takeout | Archive from Google Photos takeout |
| `from-icloud` | iCloud export | Archive from iCloud takeout |
| `from-picasa` | Picasa | Archive from Picasa collections |
| `from-immich` | Immich server | Archive from Immich server |

## Metadata Files

Each photo gets a corresponding `.JSON` file containing:
- Original filename and capture date
- GPS coordinates (latitude/longitude)  
- Album associations
- Tags and descriptions
- Rating and favorite status
- Archive/trash status

### Example Metadata
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
      "description": "Photos from my trip"
    }
  ],
  "tags": [
    { "value": "USA/California/San Francisco" }
  ],
  "rating": 5,
  "trashed": false,
  "archived": false,
  "favorited": true,
  "fromPartner": false
}
```

## Examples

### Archive from Immich Server
```bash
# Archive all photos from server

immich-go archive from-immich \
  --from-server=http://localhost:2283 \
  --from-api-key=your-key \
  --write-to-folder=/backup/photos

# Archive specific date range
immich-go archive from-immich \
  --from-server=http://localhost:2283 \
  --from-api-key=your-key \
  --from-date-range=2023 \
  --write-to-folder=/backup/2023-photos

# Archive specific album
immich-go archive from-immich \
  --from-server=http://localhost:2283 \
  --from-api-key=your-key \
  --from-albums="Family" \
  --write-to-folder=/backup/albums
```

### Archive Google Photos Takeout
```bash
# Create organized archive from takeout
immich-go archive from-google-photos \
  --write-to-folder=/organized-photos \
  /path/to/takeout-*.zip

# Archive a specific album
immich-go archive from-google-photos \
  --from-album-name="Summer Vacation" \
  --write-to-folder=/vacations \
  /path/to/takeout
```

### Archive Local Folders
```bash
# Reorganize existing photos by date
immich-go archive from-folder \
  --write-to-folder=/organized \
  /messy/photo/folders

```

## Use Cases

### 1. Server Backup
Create a complete backup of your Immich server:
```bash
immich-go archive from-immich \
  --from-server=http://localhost:2283 \
  --from-api-key=your-key \
  --write-to-folder=/complete-backup
```

### 2. Migration Preparation  
Prepare photos for migration to another system:
```bash
immich-go archive from-immich \
  --from-server=http://localhost:2283 \
  --from-api-key=your-key \
  --write-to-folder=/migration-ready
```

### 3. Photo Organization
Transform messy folder structures into organized archives:
```bash
immich-go archive from-folder \
  --write-to-folder=/organized \
  /chaotic/photo/collection
```

### 4. Selective Archival
Archive specific content based on criteria:
```bash
# Archive by year
immich-go archive from-immich \
  --from-date-range=2022-01-01,2022-12-31 \
  --write-to-folder=/archive-2022 \
  --from-server=http://localhost:2283 \
  --from-api-key=your-key \

# Archive by album
immich-go archive from-immich \
  --from-albums="Professional Photos" \
  --write-to-folder=/work-archive \
  --from-server=http://localhost:2283 \
  --from-api-key=your-key \
```

## Important Notes

- **Incremental**: Archives can be updated - new photos are added without affecting existing ones
- **Metadata Preservation**: JSON files ensure no metadata is lost
- **Cross-Platform**: Archived photos can be imported to any compatible system
- **Space Efficient**: No unnecessary duplication during incremental updates

## Options Reference

### Connection Options (for from-immich)
Same as [upload command](upload.md#server-connection-options).

### Filtering Options  
Same filtering options as corresponding upload sub-commands:
- File type filtering (`--include-type`, `--include-extensions`)
- Date range filtering (`--date-range`, `--from-date-range`)
- Album filtering (`--from-albums`)

## See Also

- [Upload Command](upload.md) - For option details
- [Technical Details](../technical.md) - Metadata formats
- [Examples](../examples.md) - More use cases
