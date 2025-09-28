# Configuration Options

This guide covers all configuration options, environment variables, and settings for Immich-Go.

## Environment Variables

### Core Configuration

| Variable           | Description                        | Default               |
| ------------------ | ---------------------------------- | --------------------- |
| `IMMICHGO_TEMPDIR` | Temporary directory for operations | System temp directory |

### Usage Examples
# Configuration File Support

Immich-go now supports configuration files to store API keys, server addresses, and common options. This eliminates the need to specify these values via command-line flags every time you run the application.

## Supported Configuration Formats

Immich-go supports three configuration file formats:

- **YAML** (`.yaml` or `.yml`) - Recommended for readability
- **JSON** (`.json`) - Good for programmatic use
- **TOML** (`.toml`) - Alternative structured format

## Configuration File Locations

### Default Locations

When no `--config` flag is specified, immich-go will look for configuration files in:

1. `$XDG_CONFIG_HOME/immich-go/immich-go.{yaml,json,toml}`
2. `$HOME/.config/immich-go/immich-go.{yaml,json,toml}` (if XDG_CONFIG_HOME is not set)
3. `./immich-go.{yaml,json,toml}` (current directory)

### Custom Configuration File

Use the `--config` flag to specify a custom configuration file:

```bash
immich-go --config /path/to/your/config.yaml upload from-folder /photos
```

## Creating Configuration Files

### Generate Sample Configuration

Create a sample configuration file with all available options:

```bash
# Generate YAML configuration (recommended)
immich-go config generate immich-config.yaml

# Generate JSON configuration
immich-go config generate immich-config.json

# Generate TOML configuration
immich-go config generate immich-config.toml

# Generate in default location
immich-go config generate
```

### Manual Configuration

You can also create a configuration file manually. Here's a complete example in YAML format:

```yaml
# Immich-go Configuration File
server:
  # Immich server URL (required)
  url: "http://your-immich-server:2283"
  
  # API key for accessing Immich (required)
  api_key: "your-api-key-here"
  
  # Admin API key for managing server jobs (optional)
  admin_api_key: "your-admin-api-key-here"
  
  # Skip SSL certificate verification (default: false)
  skip_ssl: false
  
  # Client timeout for API requests (default: 20m)
  client_timeout: "20m"
  
  # Device UUID for uploads (default: hostname)
  device_uuid: ""
  
  # Time zone override (default: system timezone)
  time_zone: ""
  
  # Action to take on server errors: stop, continue, or number of errors
  on_server_errors: "stop"

upload:
  # Number of concurrent upload workers (1-20, default: 4)
  concurrent_uploads: 4
  
  # Enable dry-run mode (no actual uploads)
  dry_run: false
  
  # Always overwrite files on server with local versions
  overwrite: false
  
  # Pause Immich background jobs during uploads
  pause_immich_jobs: true

archive:
  # Date range filter for archiving photos (format: YYYY-MM-DD,YYYY-MM-DD)
  date_range: ""

stack:
  # Manage coupled HEIC and JPEG files
  # Options: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG
  manage_heic_jpeg: "NoStack"
  
  # Manage coupled RAW and JPEG files
  # Options: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG
  manage_raw_jpeg: "NoStack"
  
  # Manage burst photos
  # Options: NoStack, Stack, StackKeepRaw, StackKeepJPEG
  manage_burst: "NoStack"
  
  # Manage Epson FastFoto file groups
  manage_epson_fastfoto: false
  
  # Date range filter for stacking photos (format: YYYY-MM-DD,YYYY-MM-DD)
  date_range: ""

logging:
  # Log level: debug, info, warn, error
  level: "info"
  
  # Log file path (empty means auto-generate)
  file: ""
  
  # Enable API call tracing
  api_trace: false

ui:
  # Disable the user interface
  no_ui: false
```

## Configuration Priority

Settings are resolved in the following priority order (highest to lowest):

1. **Command-line flags** (highest priority)
2. **Environment variables**
3. **Configuration file**
4. **Default values** (lowest priority)

### Example Priority Resolution

Given this configuration file:
```yaml
server:
  url: "http://config-server:2283"
  api_key: "config-key"
```

And these environment variables:
```bash
export IMMICHGO_SERVER_URL="http://env-server:2283"
```

And this command:
```bash
immich-go --server "http://flag-server:2283" upload from-folder /photos
```

The final values will be:
- Server URL: `http://flag-server:2283` (from command-line flag)
- API Key: `config-key` (from configuration file)

## Environment Variables

All configuration options can be overridden with environment variables. The naming convention is:

- Prefix: `IMMICHGO_`
- Convert dots to underscores: `.` → `_`
- Convert hyphens to underscores: `-` → `_`
- Use uppercase

### Environment Variable Examples

| Configuration Key | Environment Variable |
|-------------------|---------------------|
| `server.url` | `IMMICHGO_SERVER_URL` |
| `server.api_key` | `IMMICHGO_SERVER_API_KEY` |
| `server.admin_api_key` | `IMMICHGO_SERVER_ADMIN_API_KEY` |
| `server.skip_ssl` | `IMMICHGO_SERVER_SKIP_SSL` |
| `server.client_timeout` | `IMMICHGO_SERVER_CLIENT_TIMEOUT` |
| `upload.concurrent_uploads` | `IMMICHGO_UPLOAD_CONCURRENT_UPLOADS` |
| `upload.dry_run` | `IMMICHGO_UPLOAD_DRY_RUN` |
| `archive.date_range` | `IMMICHGO_ARCHIVE_DATE_RANGE` |
| `stack.manage_heic_jpeg` | `IMMICHGO_STACK_MANAGE_HEIC_JPEG` |
| `stack.manage_raw_jpeg` | `IMMICHGO_STACK_MANAGE_RAW_JPEG` |
| `stack.manage_burst` | `IMMICHGO_STACK_MANAGE_BURST` |
| `stack.manage_epson_fastfoto` | `IMMICHGO_STACK_MANAGE_EPSON_FASTFOTO` |
| `stack.date_range` | `IMMICHGO_STACK_DATE_RANGE` |
| `logging.level` | `IMMICHGO_LOGGING_LEVEL` |
| `logging.api_trace` | `IMMICHGO_LOGGING_API_TRACE` |

### Setting Environment Variables

```bash
# Linux/macOS
export IMMICHGO_SERVER_URL="http://localhost:2283"
export IMMICHGO_SERVER_API_KEY="your-api-key"
export IMMICHGO_UPLOAD_CONCURRENT_UPLOADS=8

# Windows
set IMMICHGO_SERVER_URL=http://localhost:2283
set IMMICHGO_SERVER_API_KEY=your-api-key
set IMMICHGO_UPLOAD_CONCURRENT_UPLOADS=8
```

## Validating Configuration

Use the `config validate` command to check your configuration:

```bash
# Validate configuration file
immich-go --config your-config.yaml config validate

# Validate with environment variables
IMMICHGO_SERVER_URL="http://localhost:2283" immich-go config validate
```

This command will:
- Show resolved configuration values
- Mask sensitive information (API keys)
- Display which configuration file is being used
- Report any configuration errors

## Security Considerations

### File Permissions

Configuration files may contain sensitive API keys. Ensure proper file permissions:

```bash
# Set restrictive permissions (owner read/write only)
chmod 600 immich-config.yaml
```

### Environment Variables

Be cautious when using environment variables in shared environments:
- Use a `.env` file for local development
- Avoid printing environment variables in logs
- Use secure secret management in production

## Command-Specific Configuration

Each immich-go command supports its own configuration section, allowing you to set command-specific defaults.

### Upload Command Configuration

The `upload` section configures default behavior for all upload operations:

```yaml
upload:
  concurrent_uploads: 8        # Number of simultaneous uploads
  dry_run: false              # Preview uploads without executing
  overwrite: false            # Always overwrite existing files
  pause_immich_jobs: true     # Pause background jobs during upload
```

### Archive Command Configuration

The `archive` section configures default behavior for archiving operations:

```yaml
archive:
  date_range: "2020-01-01,2024-12-31"  # Only archive photos in date range
```

### Stack Command Configuration

The `stack` section configures automatic photo stacking behavior:

```yaml
stack:
  manage_heic_jpeg: "StackCoverHeic"     # Stack HEIC+JPEG pairs, show HEIC
  manage_raw_jpeg: "StackCoverRaw"       # Stack RAW+JPEG pairs, show RAW
  manage_burst: "Stack"                  # Stack burst sequences
  manage_epson_fastfoto: true            # Group Epson FastFoto scans
  date_range: "2023-01-01,2023-12-31"   # Only process photos in date range
```

#### Stack Management Options

| Option | Values | Description |
|--------|--------|-------------|
| `manage_heic_jpeg` | `NoStack`, `KeepHeic`, `KeepJPG`, `StackCoverHeic`, `StackCoverJPG` | How to handle HEIC+JPEG pairs |
| `manage_raw_jpeg` | `NoStack`, `KeepRaw`, `KeepJPG`, `StackCoverRaw`, `StackCoverJPG` | How to handle RAW+JPEG pairs |
| `manage_burst` | `NoStack`, `Stack`, `StackKeepRaw`, `StackKeepJPEG` | How to handle burst photo sequences |

## Migration from Command-Line Flags

If you're currently using command-line flags, you can easily migrate to a configuration file:

### Before (using flags)
```bash
immich-go --server "http://localhost:2283" --api-key "your-key" --concurrent-uploads 8 upload from-folder /photos
```

### After (using config file)
1. Create a configuration file:
```yaml
server:
  url: "http://localhost:2283"
  api_key: "your-key"
upload:
  concurrent_uploads: 8
```

2. Use the simplified command:
```bash
immich-go upload from-folder /photos
```

## Docker Usage

When using immich-go in Docker containers:

### Using Environment Variables (Recommended)
```bash
docker run -e IMMICHGO_SERVER_URL="http://immich:2283" \
           -e IMMICHGO_SERVER_API_KEY="your-key" \
           immich-go upload from-folder /photos
```

### Using Configuration File
```bash
# Mount config file
docker run -v /host/path/to/config.yaml:/config/immich-go.yaml \
           immich-go --config /config/immich-go.yaml upload from-folder /photos
```

## Troubleshooting

### Configuration Not Found
If you see "No configuration file found", either:
- Create a configuration file in the default location
- Specify a configuration file with `--config`
- Use command-line flags or environment variables

### Values Not Being Applied
Check the priority order:
1. Verify command-line flags aren't overriding your config
2. Check for environment variables that might override config values
3. Use `config validate` to see resolved values

### Permission Denied
Ensure the configuration file has proper read permissions:
```bash
chmod 644 immich-config.yaml
```

## Examples

### Basic Setup
```yaml
server:
  url: "http://192.168.1.100:2283"
  api_key: "your-api-key-here"
```

### Development Setup
```yaml
server:
  url: "http://localhost:2283"
  api_key: "dev-api-key"
  skip_ssl: true
upload:
  dry_run: true
  concurrent_uploads: 2
logging:
  level: "debug"
  api_trace: true
```

### Production Setup
```yaml
server:
  url: "https://immich.example.com"
  api_key: "production-api-key"
  client_timeout: "30m"
upload:
  concurrent_uploads: 8
  pause_immich_jobs: true
logging:
  level: "info"
  file: "/var/log/immich-go.log"
```

### Archive-Focused Setup
```yaml
server:
  url: "http://immich.local:2283"
  api_key: "archive-api-key"
archive:
  # Archive photos from last 2 years only
  date_range: "2023-01-01,2024-12-31"
upload:
  # Use conservative settings for archiving
  dry_run: true
  concurrent_uploads: 2
logging:
  level: "info"
```

### Photo Organization Setup
```yaml
server:
  url: "http://immich.local:2283"
  api_key: "organize-api-key"
stack:
  # Automatically stack related photos
  manage_heic_jpeg: "StackCoverHeic"
  manage_raw_jpeg: "StackCoverRaw"
  manage_burst: "Stack"
  manage_epson_fastfoto: true
  # Only process recent photos
  date_range: "2024-01-01,2024-12-31"
logging:
  level: "debug"
```

This configuration system provides a flexible and secure way to manage your immich-go settings across different environments and use cases.

## Legacy Configuration Options

The following sections document additional configuration options available through command-line flags and environment variables.

### Core Environment Variables

| Variable           | Description                        | Default               |
| ------------------ | ---------------------------------- | --------------------- |
| `IMMICHGO_TEMPDIR` | Temporary directory for operations | System temp directory |

### File Processing Configuration

#### File Filtering

| Option                 | Type            | Description              | Example                 |
| ---------------------- | --------------- | ------------------------ | ----------------------- |
| `--include-extensions` | Comma-separated | Extensions to include    | `.jpg,.png,.heic`       |
| `--exclude-extensions` | Comma-separated | Extensions to exclude    | `.gif,.bmp`             |
| `--include-type`       | Single value    | File type filter         | `IMAGE`, `VIDEO`, `all` |
| `--ban-file`           | Pattern         | File patterns to exclude | `*.tmp`, `Thumbs.db`    |

#### Date Processing

| Option             | Default | Description                               |
| ------------------ | ------- | ----------------------------------------- |
| `--date-from-name` | `true`  | Extract date from filename if no metadata |
| `--date-range`     | All     | Filter by capture date                    |

### File Management Configuration

#### Burst Photo Handling

| Option           | Values                                              | Description             |
| ---------------- | --------------------------------------------------- | ----------------------- |
| `--manage-burst` | `NoStack`, `Stack`, `StackKeepRaw`, `StackKeepJPEG` | Burst sequence handling |

#### Coupled File Management

| Option                    | Values                                                              | Description                |
| ------------------------- | ------------------------------------------------------------------- | -------------------------- |
| `--manage-raw-jpeg`       | `NoStack`, `KeepRaw`, `KeepJPG`, `StackCoverRaw`, `StackCoverJPG`   | RAW+JPEG pairs             |
| `--manage-heic-jpeg`      | `NoStack`, `KeepHeic`, `KeepJPG`, `StackCoverHeic`, `StackCoverJPG` | HEIC+JPEG pairs            |
| `--manage-epson-fastfoto` | `true`, `false`                                                     | Epson FastFoto scan groups |

### Album and Tagging Configuration

#### Album Creation

| Option                | Values                   | Description                           |
| --------------------- | ------------------------ | ------------------------------------- |
| `--folder-as-album`   | `NONE`, `FOLDER`, `PATH` | Create albums from folders            |
| `--folder-as-tags`    | `true`, `false`          | Use folder structure as tags          |
| `--album-path-joiner` | String                   | Separator for path-based album names  |
| `--into-album`        | Album name               | Import all files into specified album |

#### Tagging Options

| Option          | Default                | Description                       |
| --------------- | ---------------------- | --------------------------------- |
| `--session-tag` | `false`                | Tag with upload session timestamp |
| `--tag`         | None                   | Custom tags (can be repeated)     |
| `--takeout-tag` | `true` (Google Photos) | Tag with takeout timestamp        |
| `--people-tag`  | `true` (Google Photos) | Tag with people names             |

### Source-Specific Configuration

#### Google Photos Takeout

| Option                    | Default | Description                     |
| ------------------------- | ------- | ------------------------------- |
| `-a, --include-archived`  | `true`  | Import archived photos          |
| `-t, --include-trashed`   | `false` | Import trashed photos           |
| `-p, --include-partner`   | `true`  | Import partner's photos         |
| `-u, --include-unmatched` | `false` | Import files without JSON       |
| `--sync-albums`           | `true`  | Create matching albums          |
| `--from-album-name`       |         | Import from specific album only |

#### iCloud Takeout

| Option       | Default | Description               |
| ------------ | ------- | ------------------------- |
| `--memories` | `false` | Import memories as albums |

#### Immich-to-Immich Transfer

| Option              | Description              |
| ------------------- | ------------------------ |
| `--from-server`     | Source server URL        |
| `--from-api-key`    | Source server API key    |
| `--from-album`      | Transfer specific albums |
| `--from-date-range` | Date filter for source   |

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