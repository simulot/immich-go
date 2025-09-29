# Configuration File

The configuration file is a TOML file. By default, `immich-go` looks for a file named `immich-go.toml` in the current directory.

## Global settings

```toml
# Immich server URL
server = "http://immich:2283"
# Immich API key
api-key = "..."
# Log level (DEBUG|INFO|WARN|ERROR)
log-level = "INFO"
```

## Command specific settings

Settings for specific commands are nested under keys corresponding to the command path.

### `upload` command

```toml
[upload]
# Number of concurrent upload workers
concurrent = 2
# Create albums for assets
create-albums = true
```

### `upload from-folder` command

```toml
[upload.from-folder]
# Use the folder name as the album name
folder-as-album = "FOLDER"
```
