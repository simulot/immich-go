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
- Immich checksum of the original file (`from-immich` only, see [Running the archive again](#running-the-archive-again))
- GPS coordinates (latitude/longitude)  
- Album associations
- Tags and descriptions
- Rating and favorite status
- Archive/trash status
- The asset's stack on the Immich server (`from-immich` only, see [Stacks](#stacks))

### Example Metadata
```json
{
  "fileName": "example.jpg",
  "checksum": "87AALqWKhRpRT+x3LYq361bAZyU=",
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
  "fromPartner": false,
  "stack": {
    "id": "8f2c4e6a-1b3d-4f5a-9c7e-0d2b4f6a8c1e",
    "primary": true,
    "assetCount": 3
  }
}
```

## Stacks

With `from-immich`, the `.JSON` file of each asset in a stack records that stack: its ID on the
server, whether the asset is the stack's primary, and the number of assets in it. The assets of a
stack can be in different date folders.

When the archive is uploaded with `upload from-folder`, the same stacks are made, with the same
primary, once every file is uploaded. The `--manage-*` options never group or drop these files
(see [upload from-folder](upload.md#file-management)). Running the upload again makes no new
stack. The API key of the upload needs the `stack.read` and `stack.create` permissions.

The archive reads the stacks with the API key's `stack.read` permission. Without it, the archive
logs a warning, and a stack whose assets other than its primary are all archived is not recorded.

## Running the archive again

The archive can be run again into the same folder, for example on a schedule, or after an
interrupted run. It never deletes or overwrites a photo already in the folder. It looks for each
asset under its name in its date folder, then under `name~1.ext`, `name~2.ext`, and so on:

- **The asset is already there**: it is skipped. It is not downloaded or copied again, and no `~N`
  copy is made. Its `.JSON` file is rewritten only when the metadata has changed (an album, a
  description, a rating...).
- **Another file has that name**: the asset is written under the next free `~N` name.

How an asset already there is recognized depends on the source:

- **`from-immich`**: the `.JSON` file holds the asset's Immich checksum, the SHA-1 of the original
  file. The file is the asset when the checksum matches. A `.JSON` file written by an older release
  of immich-go has no checksum: the file is read once to compute its SHA-1, and when it matches, the
  checksum is added to the `.JSON` file.
- **`from-immich`, assets of an external library**: Immich computes their checksum from the file's
  path, not its content. No checksum is written, and a file with the same name and size is taken
  as the asset.
- **Other sources**: a file with the same name and size is taken as the asset.

Immich keeps edits apart from the original. The archive holds the original file of each asset, and
editing a photo in Immich does not change its checksum: the next run does not download it again.

A download is written to a hidden `.<name>.partial` file first. It gets its real name only once it
is complete: it must have the asset's size, when known, and with `from-immich` its SHA-1 must be
the asset's checksum. An interrupted run leaves no truncated photo behind, and the next run
downloads the missing assets. A `.partial` file left by a run that was killed is replaced the next
time that asset is downloaded, and can be deleted.

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

- **Incremental**: Archives can be updated - new photos are added without affecting existing ones (see [Running the archive again](#running-the-archive-again))
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
