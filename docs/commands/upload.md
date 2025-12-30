# Upload Command

The `upload` command transfers photos and videos from various sources to your Immich server.

## Syntax

```bash
immich-go upload <sub-command> [options] <source-path>
```

## Sub-commands

| Sub-command                               | Source           | Description                                |
| ----------------------------------------- | ---------------- | ------------------------------------------ |
| [from-folder](#from-folder)               | Local filesystem | Upload from local folders or ZIP archives  |
| [from-google-photos](#from-google-photos) | Google Takeout   | Upload from Google Photos takeout archives |
| [from-icloud](#from-icloud)               | iCloud export    | Upload from iCloud takeout                 |
| [from-picasa](#from-picasa)               | Picasa           | Upload from Picasa photo collections       |
| [from-immich](#from-immich)               | Immich server    | Transfer between Immich servers            |

## Server Connection Options

All upload sub-commands require these connection parameters:

| Option              | Required | Description                                       |
| ------------------- | :------: | ------------------------------------------------- |
| `-s, --server`      |    Y     | Immich server URL (e.g., `http://localhost:2283`) |
| `-k, --api-key`     |    Y     | Your API key                                      |
| `--skip-verify-ssl` |          | Skip SSL certificate verification                 |
| `--client-timeout`  |          | Server call timeout (default: `20m`)              |

## Upload Behavior Options

| Option                | Default   | Description                                                         |
| --------------------- | --------- | ------------------------------------------------------------------- |
| `--dry-run`           | `false`   | Simulate upload without actual transfers                            |
| `--concurrent-tasks`  | CPU cores | Number of parallel tasks (1-20)                                     |
| `--overwrite`         | `false`   | Replace existing files on server                                    |
| `--pause-immich-jobs` | `true`    | Pause server jobs during upload                                     |
| `--on-errors`         | `stop`    | Action on errors: `stop`, `continue`, or tolerated number of errors |

## Tagging and Organization

| Option          | Default      | Description                                                  |
| --------------- | ------------ | ------------------------------------------------------------ |
| `--session-tag` | `false`      | Tag with upload session timestamp                            |
| `--defer-tags`  | `false`      | Apply tags only after metadata extraction has completed to keep embedded file tags/keywords intact |
| `--tag`         | -            | Add custom tags (can be used multiple times)                 |
| `--device-uuid` | `$LOCALHOST` | Set device identifier                                        |

## User Interface

| Option        | Default | Description             |
| ------------- | ------- | ----------------------- |
| `--no-ui`     | `false` | Disable interactive UI  |
| `--api-trace` | `false` | Enable API call tracing |

---

## from-folder

Upload photos from local folders, including ZIP archives.

### Usage
```bash
immich-go upload from-folder [options] <folder-path>
```

### Specific Options

| Option                   | Default | Description                                             |
| ------------------------ | ------- | ------------------------------------------------------- |
| `--recursive`            | `true`  | Process subfolders                                      |
| `--date-from-name`       | `true`  | Extract date from filename if no metadata               |
| `--ignore-sidecar-files` | `false` | Skip XMP sidecar files                                  |

### File Filtering

| Option                 | Default                                  | Description                                                     |
| ---------------------- | ---------------------------------------- | --------------------------------------------------------------- |
| `--include-extensions` | `all`                                    | Comma-separated extensions to include                           |
| `--exclude-extensions` | -                                        | Comma-separated extensions to exclude                           |
| `--include-type`       | `all`                                    | File type filter: `IMAGE`, `VIDEO`, or `all`                    |
| `--ban-file`           | [See list](../technical.md#banned-files) | Exclude files by pattern                                        |
| `--date-range`         | -                                        | Date range filter (see [formats](../technical.md#date-formats)) |

### Album Management

| Option                | Default | Description                                             |
| --------------------- | ------- | ------------------------------------------------------- |
| `--folder-as-album`   | `NONE`  | Create albums from folders: `FOLDER`, `PATH`, or `NONE` |
| `--folder-as-tags`    | `false` | Use folder structure as tags                            |
| `--album-path-joiner` | `" / "` | String for joining folder names in album titles         |
| `--album-picasa`      | `false` | Use Picasa album names from `.picasa.ini` files         |
| `--into-album`        | -       | Put all photos into specified album                     |

### File Management

| Option                    | Values                                                              | Description                                                |
| ------------------------- | ------------------------------------------------------------------- | ---------------------------------------------------------- |
| `--manage-burst`          | `NoStack`, `Stack`, `StackKeepRaw`, `StackKeepJPEG`                 | [Burst photo handling](../technical.md#burst-detection)    |
| `--manage-raw-jpeg`       | `NoStack`, `KeepRaw`, `KeepJPG`, `StackCoverRaw`, `StackCoverJPG`   | [RAW+JPEG handling](../technical.md#raw-jpeg-management)   |
| `--manage-heic-jpeg`      | `NoStack`, `KeepHeic`, `KeepJPG`, `StackCoverHeic`, `StackCoverJPG` | [HEIC+JPEG handling](../technical.md#heic-jpeg-management) |
| `--manage-epson-fastfoto` | `false`                                                             | Handle Epson FastFoto scanned photos                       |

### Examples
```bash
# Basic folder upload
immich-go upload from-folder --server=http://localhost:2283 --api-key=your-key /path/to/photos

# Create albums from folder structure
immich-go upload from-folder --folder-as-album=FOLDER --server=http://localhost:2283 --api-key=your-key /photos

# Stack RAW+JPEG files
immich-go upload from-folder --manage-raw-jpeg=StackCoverRaw --server=http://localhost:2283 --api-key=your-key /photos

# Filter by date and file type
immich-go upload from-folder --date-range=2023 --include-type=IMAGE --server=http://localhost:2283 --api-key=your-key /photos
```

---

## from-google-photos

Upload from Google Photos Takeout archives.

### Usage
```bash
immich-go upload from-google-photos [options] <takeout-path>
```

### Takeout Handling

| Option                    | Default | Description                        |
| ------------------------- | ------- | ---------------------------------- |
| `-u, --include-unmatched` | `false` | Import files without JSON metadata |
| `-a, --include-archived`  | `true`  | Import archived photos             |
| `-t, --include-trashed`   | `false` | Import trashed photos              |
| `-p, --include-partner`   | `true`  | Import partner's photos            |

### Album Options

| Option                      | Default | Description                          |
| --------------------------- | ------- | ------------------------------------ |
| `--sync-albums`             | `true`  | Create albums matching Google Photos |
| `--include-untitled-albums` | `false` | Include photos from untitled albums  |
| `--from-album-name`         | -       | Import only from specified album     |
| `--partner-shared-album`    | -       | Album name for partner photos        |

### Tagging

| Option          | Default | Description                     |
| --------------- | ------- | ------------------------------- |
| `--takeout-tag` | `true`  | Tag with takeout timestamp      |
| `--people-tag`  | `true`  | Tag with people names from JSON |

### File Management
Same options as `from-folder` for burst, RAW/JPEG, and HEIC/JPEG management.

### Examples
```bash
# Basic Google Photos import
immich-go upload from-google-photos --server=http://localhost:2283 --api-key=your-key /path/to/takeout-*.zip

# Import including unmatched files
immich-go upload from-google-photos --include-unmatched --server=http://localhost:2283 --api-key=your-key /takeout

# Import from specific album only
immich-go upload from-google-photos --from-album-name="Vacation 2023" --server=http://localhost:2283 --api-key=your-key /takeout
```

---

## from-icloud

Upload from iCloud takeout archives.

### Usage
```bash
immich-go upload from-icloud [options] <icloud-path>
```

### Specific Options

| Option       | Default | Description                      |
| ------------ | ------- | -------------------------------- |
| `--memories` | `false` | Import iCloud memories as albums |

### Examples
```bash
# Basic iCloud import
immich-go upload from-icloud --server=http://localhost:2283 --api-key=your-key /path/to/icloud-export

# Include memories as albums  
immich-go upload from-icloud --memories --server=http://localhost:2283 --api-key=your-key /path/to/icloud-export
```

---

## from-picasa

Upload from Picasa photo collections.

### Usage
```bash
immich-go upload from-picasa [options] <picasa-path>
```

Uses same options as `from-folder` with automatic Picasa metadata detection.

---

## from-immich

Transfer photos between Immich servers.

### Usage
```bash
immich-go upload from-immich [source-options] [destination-options]
```

### Source Server Options

  | Option                   | Description                      |
  | ------------------------ | -------------------------------- |
  | `--from-server`          | Source Immich server URL         |
  | `--from-api-key`         | Source server API key            |
  | `--from-client-timeout`  | Source server timeout            |
  | `--from-skip-verify-ssl` | Skip SSL verification for source |

### Source Filtering


  | Option                  | Description                  |
  | ----------------------- | ---------------------------- |
  | `--from-date-range`     | Date range filter for source |
  | `--from-archived`       | Include archived assets      |
  | `--from-trash`          | Include trashed assets       |
  | `--from-favorite`       | Include only favorite assets |
  | `--from-minimal-rating` | Minimum rating filter        |


### Examples
```bash
# Transfer all photos between servers
immich-go upload from-immich \
  --from-server=http://old-server:2283 --from-api-key=old-key \
  --server=http://new-server:2283 --api-key=new-key

# Transfer images with a rating of 3 or above
immich-go upload from-immich \
  --from-server=http://old-server:2283 --from-api-key=old-key --from-minimal-rating=3 \
  --server=http://new-server:2283 --api-key=new-key

# Transfer photos from a specific date range
immich-go upload from-immich \
  --from-server=http://old-server:2283 --from-api-key=old-key --from-date-range=2023-01-01,2023-06-30 \
  --server=http://new-server:2283 --api-key=new-key
```



## Performance Tips

- **Concurrent Tasks**: Start with default (CPU cores), adjust based on network/server capacity
- **Large Files**: Increase `--client-timeout` for large video files
- **Network Issues**: Use lower `--concurrent-tasks` for unstable connections
- **Server Load**: Enable `--pause-immich-jobs` during large uploads

## See Also

- [Configuration Options](../configuration.md)
- [Technical Details](../technical.md)
- [Best Practices](../best-practices.md)
