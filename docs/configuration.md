# Configuration Options

This guide covers all configuration options, environment variables, and settings for Immich-Go.

## Environment Variables

### Core Configuration

| Variable           | Description                        | Default               |
| ------------------ | ---------------------------------- | --------------------- |
| `IMMICHGO_TEMPDIR` | Temporary directory for operations | System temp directory |

### Usage Examples
```bash
# Use custom temp directory
export IMMICHGO_TEMPDIR=/fast-storage/tmp
immich-go upload from-folder --server=... --api-key=... /photos

# Windows
set IMMICHGO_TEMPDIR=D:\temp
immich-go.exe upload from-folder --server=... --api-key=... C:\photos
```

## Global Options

Available for all commands:

### Logging Configuration

| Option           | Values                           | Default        | Description              |
| ---------------- | -------------------------------- | -------------- | ------------------------ |
| `--log-level`    | `DEBUG`, `INFO`, `WARN`, `ERROR` | `INFO`         | Logging verbosity        |
| `--log-type`     | `TEXT`, `JSON`                   | `TEXT`         | Log output format        |
| `-l, --log-file` | File path                        | Auto-generated | Custom log file location |

#### Log File Locations (Default)

| Operating System | Path                                                               |
| ---------------- | ------------------------------------------------------------------ |
| **Linux**        | `$HOME/.cache/immich-go/immich-go_YYYY-MM-DD_HH-MI-SS.log`         |
| **Windows**      | `%LocalAppData%\immich-go\immich-go_YYYY-MM-DD_HH-MI-SS.log`       |
| **macOS**        | `$HOME/Library/Caches/immich-go/immich-go_YYYY-MM-DD_HH-MI-SS.log` |

#### Logging Examples
```bash
# Debug mode with custom log file
immich-go --log-level=DEBUG --log-file=/tmp/debug.log upload from-folder ...

# JSON logging for automated parsing
immich-go --log-type=JSON --log-level=INFO upload from-folder ...
```

## Connection Configuration

### Server Settings

| Option          | Required | Description       | Example                     |
| --------------- | :------: | ----------------- | --------------------------- |
| `-s, --server`  |    Y     | Immich server URL | `http://192.168.1.100:2283` |
| `-k, --api-key` |    Y     | User API key      | `your-api-key-here`         |


### SSL/TLS Options

| Option              | Default                      | Description                   |
| ------------------- | ---------------------------- | ----------------------------- |
| `--skip-verify-ssl` | `false`                      | Skip certificate verification |
| `--client-timeout`  | `20m` (upload), `5m` (other) | Request timeout               |

#### SSL Examples
```bash
# Self-signed certificate
immich-go --skip-verify-ssl upload from-folder --server=https://immich.local ...

# Increase timeout for large files
immich-go upload from-folder --client-timeout=60m --server=... --api-key=... /large-videos
```

## Performance Configuration

### Upload Concurrency

| Option                 | Default   | Range | Description             |
| ---------------------- | --------- | ----- | ----------------------- |
| `--concurrent-uploads` | CPU cores | 1-20  | Parallel upload workers |

#### Concurrency Guidelines

| Connection Type   | Recommended Value | Notes                        |
| ----------------- | :---------------: | ---------------------------- |
| **Gigabit LAN**   |       8-16        | High bandwidth, stable       |
| **Fast Internet** |        4-8        | Good bandwidth, some latency |
| **Slow/Unstable** |        1-2        | Conservative, reliable       |
| **NAS/Low Power** |        2-4        | Limited server resources     |

### Server Load Management

| Option                | Default | Description                                   |
| --------------------- | ------- | --------------------------------------------- |
| `--pause-immich-jobs` | `true`  | Pause server jobs during upload               |
| `--on-server-errors`  | `stop`  | Error handling: `stop`, `continue`, or tolerated number of errors |

#### Performance Examples
```bash
# High-performance upload
immich-go upload from-folder \
  --concurrent-uploads=16 \
  --client-timeout=30m \
  --pause-immich-jobs=true \
  --server=... --api-key=... /photos

# Conservative upload for unstable connections
immich-go upload from-folder \
  --concurrent-uploads=1 \
  --client-timeout=60m \
  --on-server-errors=continue \
  --server=... --api-key=... /photos
```

## File Processing Configuration

### File Filtering

| Option                 | Type            | Description              | Example                 |
| ---------------------- | --------------- | ------------------------ | ----------------------- |
| `--include-extensions` | Comma-separated | Extensions to include    | `.jpg,.png,.heic`       |
| `--exclude-extensions` | Comma-separated | Extensions to exclude    | `.gif,.bmp`             |
| `--include-type`       | Single value    | File type filter         | `IMAGE`, `VIDEO`, `all` |
| `--ban-file`           | Pattern         | File patterns to exclude | `*.tmp`, `Thumbs.db`    |

#### Built-in Banned Files
- `@eaDir/`, `@__thumb/`
- `SYNOFILE_THUMB_*.*`
- `Lightroom Catalog/`
- `thumbnails/`
- `.DS_Store`, `._*.*`
- `.photostructure/`

### Date Processing

| Option             | Default | Description                               |
| ------------------ | ------- | ----------------------------------------- |
| `--date-from-name` | `true`  | Extract date from filename if no metadata |
| `--time-zone`      | System  | Override timezone for operations          |
| `--date-range`     | All     | Filter by capture date                    |

#### Date Range Formats

| Format                  | Example                 | Description    |
| ----------------------- | ----------------------- | -------------- |
| `YYYY`                  | `2023`                  | Entire year    |
| `YYYY-MM`               | `2023-07`               | Specific month |
| `YYYY-MM-DD`            | `2023-07-15`            | Specific day   |
| `YYYY-MM-DD,YYYY-MM-DD` | `2023-01-15,2023-03-15` | Date range     |

## File Management Configuration

### Burst Photo Handling

| Option           | Values                                              | Description             |
| ---------------- | --------------------------------------------------- | ----------------------- |
| `--manage-burst` | `NoStack`, `Stack`, `StackKeepRaw`, `StackKeepJPEG` | Burst sequence handling |

### Coupled File Management

| Option                    | Values                                                              | Description                |
| ------------------------- | ------------------------------------------------------------------- | -------------------------- |
| `--manage-raw-jpeg`       | `NoStack`, `KeepRaw`, `KeepJPG`, `StackCoverRaw`, `StackCoverJPG`   | RAW+JPEG pairs             |
| `--manage-heic-jpeg`      | `NoStack`, `KeepHeic`, `KeepJPG`, `StackCoverHeic`, `StackCoverJPG` | HEIC+JPEG pairs            |
| `--manage-epson-fastfoto` | `true`, `false`                                                     | Epson FastFoto scan groups |

### Sidecar Files

| Option                   | Default | Description            |
| ------------------------ | ------- | ---------------------- |
| `--ignore-sidecar-files` | `false` | Skip XMP sidecar files |

## Album and Tagging Configuration

### Album Creation

| Option                | Values                   | Description                           |
| --------------------- | ------------------------ | ------------------------------------- |
| `--folder-as-album`   | `NONE`, `FOLDER`, `PATH` | Create albums from folders            |
| `--folder-as-tags`    | `true`, `false`          | Use folder structure as tags          |
| `--album-path-joiner` | String                   | Separator for path-based album names  |
| `--into-album`        | Album name               | Import all files into specified album |

### Tagging Options

| Option          | Default                | Description                       |
| --------------- | ---------------------- | --------------------------------- |
| `--session-tag` | `false`                | Tag with upload session timestamp |
| `--tag`         | None                   | Custom tags (can be repeated)     |
| `--takeout-tag` | `true` (Google Photos) | Tag with takeout timestamp        |
| `--people-tag`  | `true` (Google Photos) | Tag with people names             |

#### Tagging Examples
```bash
# Hierarchical tags
immich-go upload from-folder \
  --tag="Events/Wedding" \
  --tag="People/Family" \
  --session-tag \
  --server=... --api-key=... /wedding-photos

# Album from folder structure
immich-go upload from-folder \
  --folder-as-album=PATH \
  --album-path-joiner=" - " \
  --server=... --api-key=... /Photos/2023/Summer/
# Creates album: "Photos - 2023 - Summer"
```

## User Interface Configuration

### Display Options

| Option        | Default | Description                         |
| ------------- | ------- | ----------------------------------- |
| `--no-ui`     | `false` | Disable interactive progress UI     |
| `--api-trace` | `false` | Show detailed API calls             |
| `--dry-run`   | `false` | Simulate operations without changes |

### Device Identification

| Option          | Default      | Description              |
| --------------- | ------------ | ------------------------ |
| `--device-uuid` | `$LOCALHOST` | Custom device identifier |

## Source-Specific Configuration

### Google Photos Takeout

| Option                    | Default | Description                     |
| ------------------------- | ------- | ------------------------------- |
| `-a, --include-archived`  | `true`  | Import archived photos          |
| `-t, --include-trashed`   | `false` | Import trashed photos           |
| `-p, --include-partner`   | `true`  | Import partner's photos         |
| `-u, --include-unmatched` | `false` | Import files without JSON       |
| `--sync-albums`           | `true`  | Create matching albums          |
| `--from-album-name`       |      | Import from specific album only |

### iCloud Takeout

| Option       | Default | Description               |
| ------------ | ------- | ------------------------- |
| `--memories` | `false` | Import memories as albums |

### Immich-to-Immich Transfer

| Option              | Description              |
| ------------------- | ------------------------ |
| `--from-server`     | Source server URL        |
| `--from-api-key`    | Source server API key    |
| `--from-album`      | Transfer specific albums |
| `--from-date-range` | Date filter for source   |

## Configuration File Support

Currently, Immich-Go uses command-line options and environment variables. Configuration files are not supported but may be added in future versions.

### Workaround: Shell Scripts
Create reusable configuration with shell scripts:

```bash
#!/bin/bash
# immich-config.sh

IMMICH_SERVER="http://192.168.1.100:2283"
IMMICH_API_KEY="your-api-key"
COMMON_OPTS="--concurrent-uploads=8 --manage-raw-jpeg=StackCoverRaw"

case "$1" in
  upload-photos)
    immich-go upload from-folder $COMMON_OPTS --server=$IMMICH_SERVER --api-key=$IMMICH_API_KEY "$2"
    ;;
  backup)
    immich-go archive from-immich --server=$IMMICH_SERVER --api-key=$IMMICH_API_KEY --write-to-folder="$2"
    ;;
  *)
    echo "Usage: $0 {upload-photos|backup} <path>"
    ;;
esac
```

## Troubleshooting Configuration

### Common Issues

1. **API Key Problems**: Ensure key has all required permissions
2. **SSL Errors**: Use `--skip-verify-ssl` for self-signed certificates
3. **Timeout Issues**: Increase `--client-timeout` for large files
4. **Performance**: Adjust `--concurrent-uploads` based on network/server capacity

### Debug Configuration
```bash
# Maximum debugging
immich-go --log-level=DEBUG --log-file=/tmp/debug.log --api-trace \
  upload from-folder --dry-run \
  --server=... --api-key=... /test-folder
```

## See Also

- [Installation Guide](installation.md) - API key setup
- [Commands](commands/) - Command-specific options
- [Technical Details](technical.md) - File processing details
- [Best Practices](best-practices.md) - Optimization tips