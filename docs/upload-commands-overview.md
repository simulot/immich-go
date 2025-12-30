# Upload Commands Overview

This document provides a comprehensive guide to the `immich-go` command-line tool, explaining how its various commands process and upload your media to an Immich server. It covers file selection, metadata handling, grouping, and the intelligent integration with Immich's features.

## 1. The `from-folder` Command: Uploading from Local Folders

This section provides a detailed explanation of how the `immich-go` command-line tool processes and uploads files from a local folder to an Immich server. It covers file selection, metadata handling, grouping, and the various ways `immich-go` interacts with the Immich API.

### File Selection and Filtering

The process begins with `immich-go` scanning the folder you specify. Here’s how it decides which files to upload:

1.  **Initial Scan and Media Detection**: The tool recursively scans your folder and identifies all files. It categorizes them as `image`, `video`, or `other` based on their file extensions. Only images and videos are considered for upload.

2.  **Exclusion of Unwanted Files**:
    *   **Banned Files**: Any file matching a pattern provided with the `--ban-file` flag is immediately discarded. This is useful for excluding specific files you know you don’t want.
    *   **System Files**: Common system files like `.DS_Store` or `thumbs.db` are automatically ignored.

3.  **User-Defined Filtering**: You can further refine the selection with these flags:
    *   `--include-extensions` / `--exclude-extensions`: To limit the upload to specific file types (e.g., only `.jpg` and `.cr2`).
    *   `--date-range`: To upload only photos and videos taken within a specific timeframe.

Any file that makes it through these filters is then prepared for the next stage.

### Metadata, Albums, and Tags

`immich-go` excels at preserving and creating metadata for your assets.

#### Metadata Sources

The tool consults multiple sources for metadata, in the following order of priority:

1.  **`immich-go` JSON Sidecar**: If you’re re-importing assets, `immich-go` may find its own previously generated `.json` sidecar files. These are considered the most reliable source.
2.  **XMP Sidecar**: Standard `.xmp` files, often created by photo editing software, are read next.
3.  **Source-Specific Files**: The tool can parse special metadata files, like the `.csv` files from an iCloud Photo Library takeout or `.picasa.ini` from Picasa.
4.  **Embedded Metadata**: If no sidecar is available, `immich-go` reads EXIF and XMP data embedded directly in the media file.
5.  **Filename Parsing**: As a last resort, if `--date-from-name` is enabled, the tool will try to guess the date from the filename.

#### Album Management

You have several options for organizing your assets into albums:

*   **`--into-album`**: The simplest option. All assets are placed into a single album you specify.
*   **`--folder-as-album`**: A powerful feature that creates albums based on your folder structure. You can use the immediate parent folder name (`FOLDER`) or the full relative path (`PATH`) as the album title.
*   **Source-Specific Albums**: When importing from Picasa (`--album-picasa`) or iCloud (`--memories`), `immich-go` can automatically create albums based on the metadata from those services.

To avoid creating duplicate albums, `immich-go` first fetches your existing album list from the server. It then batches updates to minimize API calls.

#### Tag Management

Tags can be assigned in two main ways:

*   **`--folder-as-tags`**: Uses the folder structure to create tags. For example, a file at `Holidays/Summer 2024/photo.jpg` will be tagged `Holidays/Summer 2024`.
*   **XMP Metadata**: Tags are also read from XMP sidecar files.
*   **`--defer-tags`**: Postpones tag creation until Immich finishes metadata extraction so built-in file tags/keywords aren’t dropped.

`immich-go` creates new tags on the server as needed and efficiently tags assets in batches.

### Uploading and Immich Integration

The final stage is the upload itself and the integration with Immich’s features.

#### The Upload Process

1.  **Duplicate Checking**: Before uploading, `immich-go` builds an index of all assets on your Immich server. For each local file, it calculates a checksum and compares it to the index.
    *   If an identical file (same checksum and size) exists, the local file is skipped.
    *   If `--overwrite` is enabled, `immich-go` can replace a server asset if the local one is of higher quality (e.g., better resolution).

2.  **Concurrent Uploads**: Files are uploaded in parallel to maximize speed. You can configure the number of concurrent workers.

#### Interaction with Immich Features

`immich-go` is more than just an uploader; it intelligently uses Immich’s server-side features:

*   **Stacking**: If you use the `--stack` flag, `immich-go` identifies related files (like RAW+JPEG pairs) and, after uploading them, sends a command to Immich to "stack" them, linking them in the UI.
*   **Background Jobs**: With the `--pause-immich-jobs` flag (which requires an admin API key), `immich-go` can temporarily pause Immich’s background tasks (like thumbnailing and transcoding). This can improve upload performance. The jobs are automatically resumed when the upload is complete.
*   **Asset Replacement**: When `--overwrite` is used, `immich-go` performs a safe replacement. It uploads the new file, copies all metadata (tags, albums, descriptions) from the old asset to the new one, and then deletes the old asset.

#### Error Handling

`immich-go` is designed to be robust:

*   **Journaling**: Most errors (like a corrupted file or a temporary API failure) are logged to a journal file without stopping the entire upload.
*   **Graceful Shutdown**: If you press Ctrl+C, the application attempts to finish any in-progress uploads before exiting.
*   **Unsupported Files**: Files that aren’t recognized as supported media types are simply skipped.

### Specialized `from-folder` Commands

The `from-picasa` and `from-icloud` commands are specialized versions of `from-folder`, tailored to handle the specific structures of those services' backups.

#### The `from-picasa` Command

This command is optimized for folders that have been managed by Google's Picasa software. It functions exactly like `from-folder`, but with one key addition: it automatically looks for `.picasa.ini` files in each directory.

When a `.picasa.ini` file is found, `immich-go` reads it to extract the album name and description, and then associates all the photos in that directory with the corresponding album in Immich.

#### The `from-icloud` Command

This command is designed to handle the structure of an Apple iCloud Photos takeout. It uses a two-pass process to ensure metadata and album information are correctly applied.

1.  **First Pass (Metadata Scan)**: The tool first scans the entire takeout specifically for `.csv` files. It parses these files to build a map of all your photos, linking them to their original creation dates and album memberships (including "Memories," which can be imported as albums using the `--memories` flag).
2.  **Second Pass (Asset Processing)**: The tool then processes the actual media files. For each photo and video, it looks up the information gathered in the first pass and applies the correct creation date and album information. This is crucial because iCloud takeout files often have incorrect file modification dates.

## 2. The `from-google-photos` Command: Migrating Google Photos Takeout

This section explains how the `immich-go` command `from-google-photos` processes a Google Photos Takeout archive and uploads it to an Immich server. The process is designed to preserve as much of your Google Photos organization as possible, including albums, metadata, and tags.

### The Two-Pass Process

`immich-go` uses a sophisticated two-pass approach to handle the complex structure of a Google Photos Takeout.

#### Pass 1: Discovery and Cataloging

First, the tool scans all the files in your Takeout archive (which can be one or more `.zip` files or a decompressed folder).

1.  **File Identification**: It identifies two main types of files:
    *   **Media Files**: Your actual photos and videos.
    *   **JSON Files**: Metadata files that contain information about your media, such as descriptions, locations, and album associations.

2.  **Metadata Parsing**: The content of every JSON file is read and stored in memory. The tool distinguishes between JSONs that describe a photo/video and those that describe an album.

3.  **File Cataloging**: A complete catalog of all media files is created, noting their location within the Takeout archive.

#### The Puzzle Solver: Matching Media with Metadata

Google Photos has a notoriously complex way of naming files, especially when you have edited photos, bursts, or duplicates. A simple 1-to-1 filename match between a media file and its JSON file is often not possible.

`immich-go`'s "puzzle solver" addresses this. It uses a series of increasingly specific rules to correctly associate each media file with its corresponding JSON metadata. This ensures that the original filenames, creation dates, and other details are accurately recovered.

#### Pass 2: Asset Creation and Filtering

Once the puzzle is solved, `immich-go` proceeds to the second pass:

1.  **Asset Creation**: For each successfully matched media file, an `asset` object is created. This object combines the media file itself with the rich metadata from its JSON file.

2.  **Filtering**: You can apply several filters to decide which assets to upload:
    *   `--from-album-name`: To import only a specific album.
    *   `--include-trashed`: To import photos that are in Google's trash.
    *   `--include-archived`: To import photos you had archived in Google Photos.
    *   `--include-partner`: To import photos shared by a partner.
    *   `--include-unmatched`: To import media files that could not be matched with a JSON file (useful for recovering all files, but they will have less metadata).

The assets that pass these filters are then sent to the uploader.

### Mapping Google Photos Features to Immich

`immich-go` does its best to translate your Google Photos organization into Immich features.

#### Album Management

*   **`--sync-albums`**: This is the key feature for album management. When enabled, `immich-go` reads the album definitions from your Takeout and creates corresponding albums in Immich.
*   **Untitled Albums**: By default, albums without a title in Google Photos are ignored. You can include them with `--include-untitled-albums`.
*   **Partner Photos**: Photos from a partner's library can be automatically placed into a specific album using `--partner-shared-album`.

#### Metadata Mapping

The rich metadata from the Google Photos JSON files is mapped to Immich's fields:

*   **Descriptions**: Photo descriptions are preserved.
*   **Creation Date**: The original capture date is used, which is more reliable than the file's modification date.
*   **GPS/Location**: GPS coordinates are extracted and applied to the asset in Immich. If a photo doesn't have GPS data but the album it's in does, `immich-go` will intelligently apply the album's location to the photo.

#### Tagging

`immich-go` can automatically create tags in Immich based on your Google Photos data:

*   **`--people-tag`**: This creates tags for people identified in your photos, using the format `people/Name`.
*   **`--takeout-tag`**: This adds a general tag to all photos from the batch, making it easy to identify them later (e.g., `takeout/takeout-2023-10-27T12:00:00Z`).

### Upload Process

The final upload process is shared with other `immich-go` commands and includes features like:

*   **Duplicate Checking**: Prevents re-uploading files that are already on your Immich server.
*   **Concurrent Uploads**: For faster performance.
*   **Stacking**: RAW+JPEG pairs and bursts can be automatically stacked in Immich.

By combining a robust parsing strategy with intelligent feature mapping, the `from-google-photos` command provides a powerful way to migrate your entire Google Photos library to Immich while keeping your organization intact.

## 3. The `from-immich` Command: Server-to-Server Migration

The `from-immich` command is a powerful tool for selecting assets from one Immich server and piping them to another command, such as `upload` or `archive`. Its primary use case is migrating or copying a subset of your library from one Immich instance to another.

### How it Works

Instead of reading files from a local folder, `from-immich` connects to a *source* Immich server and fetches assets based on the criteria you specify. It then acts as a data source for another `immich-go` command.

For example, to copy all favorite photos from a source server to a destination server, you would use a command like this:

```bash
immich-go upload from-immich \
  --from-server https://source.immich.app --from-api-key <source_key> \
  --from-favorite \
  --server https://destination.immich.app --api-key <destination_key>
```

### Powerful Filtering Capabilities

The strength of `from-immich` lies in its extensive filtering options, allowing you to precisely select which assets to process. All filter flags for this command are prefixed with `--from-`.

*   **By Status**:
    *   `--from-favorite`: Select only favorite assets.
    *   `--from-archived`: Select only archived assets.
    *   `--from-trash`: Select only trashed assets.
    *   `--from-no-album`: Select only assets that are not in any album.

*   **By Metadata**:
    *   `--from-albums`: Select assets from specific album names.
    *   `--from-tags`: Select assets with specific tags.
    *   `--from-people`: Select assets featuring specific people.
    *   `--from-make` / `--from-model`: Filter by camera manufacturer or model.
    *   `--from-country` / `--from-state` / `--from-city`: Filter by location.
    *   `--from-minimal-rating`: Select assets with a rating equal to or higher than the given value.
    *   `--from-date-range`: Select assets within a specific date range.

*   **By Owner**:
    *   `--from-partners`: Include assets shared by your partner.

### Metadata and Album Preservation

When migrating assets, `from-immich` ensures that all metadata is preserved:

*   **Albums and Tags**: The assets' associations with albums and tags are fetched from the source server. When they are uploaded to the destination server, `immich-go` will recreate those albums and tags.
*   **Other Metadata**: Descriptions, GPS coordinates, ratings, and other EXIF/XMP information are all carried over to the destination server.
