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

Each photo gets a `.JSON` file when its source gives metadata for it: always with `from-immich`,
and with the other sources when the photo comes with metadata (a Google Photos JSON file, an
immich-go `.JSON` file, an iCloud date...). A photo with none, as from a plain folder, gets no
`.JSON` file. The `.JSON` file contains:
- Original filename and capture date
- `checksum`, the SHA-1 of the archived file, and for an asset of an Immich external library,
  `pathChecksum` (see [Running the archive again](#running-the-archive-again))
- GPS coordinates (latitude/longitude)  
- Album associations
- Tags and descriptions
- Rating and favorite status
- Archive/trash status
- `software`, the version of immich-go that wrote it

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
  "fromPartner": false
}
```

## Running the archive again

The archive can be run again into the same folder, for example on a schedule, or after an
interrupted run. It never deletes or overwrites a photo already in the folder. It looks for each
asset under its name in its date folder, then under `name~1.ext`, `name~2.ext`, and so on:

- **The asset is already there**: it is skipped. It is not downloaded or copied again, and no `~N`
  copy is made. Its `.JSON` file is rewritten only when the metadata has changed (an album, a
  description, a rating...), and its `.XMP` file, if any, only when the source's XMP has changed.
- **Another file has that name**: the asset is written under the next free `~N` name. That file
  and its `.JSON` file are left as they are.

### How an asset already there is recognized

A file is the asset only when its content is proven to be, never on its name and size alone. The
`.JSON` file keeps two values for that:

- `checksum`: the SHA-1 of the archived file, base64, whatever the source.
- `pathChecksum`: only for an asset of an Immich external library. Immich computes the checksum of
  such an asset from its file's path, not from its content: `pathChecksum` is that checksum.

A file whose size is not the asset's, when the asset's size is known, is not the asset. A file
whose `.JSON` file has a `pathChecksum` can only be that external-library asset. Then:

| Source | `.JSON` file with a `checksum` | `.JSON` file without a `checksum` (written by an older release), or none |
|---|---|---|
| `from-immich`, timeline asset | The file is the asset when `checksum` is the asset's Immich checksum, the SHA-1 of its original. Nothing is read or downloaded. | The file is read once. It is the asset when its SHA-1 is the asset's Immich checksum. Nothing is downloaded. |
| `from-immich`, external-library asset | The file is the asset when `pathChecksum` is the asset's Immich checksum. Nothing is read or downloaded. | The asset is downloaded once and the file is read once. The file is the asset when both have the same SHA-1. |
| Other sources | The source file is read (a file in a ZIP archive is extracted). The file is the asset when the SHA-1 of the source file is `checksum`. | The source file and the file are read once each. The file is the asset when both have the same SHA-1. |

When the file is the asset, the values its `.JSON` file lacked are added to it: the next run of
`from-immich` reads and downloads nothing for that asset. With the other sources,
each run reads the source file of each asset already archived, and, for a photo with no `.JSON`
file (from a plain folder), the archived file too.

A file cut short after it was archived, whose `.JSON` file has the asset's `checksum` but whose
size is not the asset's, is not the asset when the asset's size is known: the asset is written
under `~1`, and the truncated file stays.

Immich keeps edits apart from the original. The archive holds the original file of each asset, and
editing a photo in Immich does not change its checksum: the next run does not download it again.

### Duplicates

A run looks for an asset only under its name in its date folder. It does not search the rest of
the archive, so:

- Copies such as `name~1.ext` or `name~2.ext`, written by older releases of immich-go for an asset
  already archived, stay where they are and are not reported.
- An asset archived under another name or in another date folder is not found: it is written
  again, and the old copy stays. This happens after a rename in Immich, a change of the file name
  rules, or a change of the capture date or time zone.
- Two source files of `from-folder` with the same name, date and bytes share one archived file.
  The `.JSON` file has the metadata of the one archived last: when their metadata differ, it is
  rewritten on every run.
- A file of an Immich external library archived by an older release (no `checksum` in its `.JSON`
  file) can be taken by a timeline asset with the same name and the same bytes. Its `.JSON` file
  then describes the timeline asset, and the external-library asset is written under `~1`. Nothing
  is lost: the bytes are the same.

These copies cost space, not photos. Restoring the archive does not duplicate them on the server:
`upload from-folder` indexes the server by checksum, so one copy is uploaded and the others are
reported as already there. Removing the copies from the archive is not automated.

### Upgrades of immich-go

The `.JSON` file records the version of immich-go that wrote it (`software`). The first run after
an upgrade of immich-go rewrites every `.JSON` file once. The photos are not written again.

### Interrupted runs

A download is written to a hidden `.<name>.<random>.partial` file first, beside its final name. It
gets its real name only once it is complete: it must have the asset's size, when known, and for an
Immich timeline asset its SHA-1 must be the asset's checksum. Otherwise, or when the rename fails,
the `.partial` file is removed. An interrupted run leaves no truncated photo behind, and the next
run downloads the missing assets. A `.partial` file left by a run that was killed stays: it can be
deleted when no archive is running into that folder.

Each run writes its own `.partial` files. Still, run one archive at a time into a folder: when two
runs write the same name at the same time, each renames its file to that name, and the later one
replaces the other.

## Errors

An asset that cannot be archived (a failed download, a full disk...) is logged as an error.
`--on-errors` says when the run stops: at the first error (`stop`, the default), never
(`continue`), or when there are more than N errors (`--on-errors=N`).

The run exits with a non-zero status, also with `--on-errors=continue`, when:

- an asset was not archived;
- the source counted an error through `--on-errors`, as `from-immich` does when the server fails;
- the source logged a file it could not read as an error, as `from-folder` does for a `.JSON`,
  `.XMP` or album file.

A folder that cannot be read does not change the exit status yet: `from-folder` logs it as an
error (`<folder>: file does not exist`), and `from-google-photos` skips it without a message. A
failure of `from-google-photos` while it walks the takeout as a whole does not change it either,
and is not logged.

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
- **Immich edits are not archived**: Immich keeps edits (non-destructive, since Immich v2.5.0) apart from the original. `from-immich` downloads each asset's original file (`/original`), and the edited version is not downloaded.

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
