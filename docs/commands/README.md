# Command Reference

Immich-Go uses a hierarchical command structure with global options, commands, sub-commands, and specific options.

## Command Structure

```bash
immich-go [global-options] command sub-command [command-options] [path]
```

## Available Commands

| Command | Description | Sub-commands |
|---------|-------------|--------------|
| [upload](upload.md) | Upload photos/videos to Immich server | from-folder, from-google-photos, from-icloud, from-picasa, from-immich |
| [archive](archive.md) | Export/archive photos to local folder structure | from-folder, from-google-photos, from-icloud, from-picasa, from-immich |
| [stack](stack.md) | Organize related photos into stacks on server | (none) |
| version | Display version information | (none) |

## Global Options

These options work with all commands:

| Option | Default | Description |
|--------|---------|-------------|
| `--config` | Auto-detect | Configuration file (YAML, JSON, or TOML) |
| `-h, --help` | - | Show help information |
| `-l, --log-file` | Auto-generated | Write log messages to specified file |
| `--log-level` | `INFO` | Set logging level: DEBUG, INFO, WARN, ERROR |
| `--log-type` | `TEXT` | Log format: TEXT or JSON |
| `-v, --version` | - | Display current version |

## Configuration File Support

All commands support configuration files to avoid repeating common options:

```bash
# Generate a configuration file
immich-go config generate my-config.yaml

# Use configuration file
immich-go --config my-config.yaml upload from-folder /photos
```

Supported formats: YAML (`.yaml`, `.yml`), JSON (`.json`), TOML (`.toml`)

See [Configuration Guide](../configuration.md) for full details.

### Log File Locations

| OS | Default Path |
|----|-------------|
| Linux | `$HOME/.cache/immich-go/immich-go_YYYY-MM-DD_HH-MI-SS.log` |
| Windows | `%LocalAppData%\immich-go\immich-go_YYYY-MM-DD_HH-MI-SS.log` |
| macOS | `$HOME/Library/Caches/immich-go/immich-go_YYYY-MM-DD_HH-MI-SS.log` |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `IMMICHGO_TEMPDIR` | Temporary directory for Immich-Go operations |

## Quick Examples

```bash
# Generate configuration file
immich-go config generate immich-config.yaml

# Upload from local folder (with config file)
immich-go --config immich-config.yaml upload from-folder /photos

# Upload from local folder (with flags)
immich-go upload from-folder --server=http://localhost:2283 --api-key=your-key /photos

# Archive from server
immich-go archive from-immich --server=http://localhost:2283 --api-key=your-key --write-to-folder=/backup

# Stack photos on server with configuration
immich-go --config immich-config.yaml stack --manage-burst=Stack

# Show version
immich-go version
```

## Detailed Command Documentation

- [Upload Command](upload.md) - Comprehensive upload options and sub-commands
- [Archive Command](archive.md) - Export and archival features  
- [Stack Command](stack.md) - Photo organization and stacking