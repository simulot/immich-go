# Sync Command

Bidirectional synchronization between a local directory and your Immich server.

## Usage

```bash
immich-go sync down [flags]    # Server → Local
immich-go sync up [flags]      # Local → Server
```

## Sub-commands

| Sub-command | Direction | Description |
|-------------|-----------|-------------|
| `down` | Server → Local | Download assets from server to local directory |
| `up` | Local → Server | Upload local assets to server |

## Shared Flags

These flags apply to both `sync down` and `sync up`:

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--directory` | `-d` | (required) | Local directory for sync operations |
| `--delete` | | `false` | Delete assets that no longer exist on the source side |
| `--no-recursive` | | `false` | Do not recurse into subdirectories |
| `--force` | | `false` | Force operation even on state mismatch or large deletions |
| `--dry-run` | | `false` | Simulate all actions (no downloads, uploads, or deletions) |

Both sub-commands also accept all [server connection flags](upload.md#server-options) (`--server`, `--api-key`, etc.). If you have used `immich-go login`, these are picked up automatically from the global config.

## How Sync Works

### State Tracking

Sync maintains a state file at `.immich-sync/state.json` inside the sync directory. This tracks:

- Which assets have been synced (by SHA1 checksum)
- Server asset IDs for each synced file
- Local file paths and sizes
- Server URL and user ID (to prevent accidental cross-server operations)

The state is saved in batches (every 50 assets) and on graceful shutdown.

### Sync Down (Server → Local)

1. Fetches all assets from the server
2. Compares checksums against the local state
3. Downloads missing or changed assets to `YYYY/YYYY-MM/filename` directory structure
4. With `--delete`: removes local files that are tracked in state but no longer on the server

### Sync Up (Local → Server)

1. Fetches all server assets (checksum index)
2. Walks the local directory, computing SHA1 checksums
3. Uploads files not found on the server
4. With `--delete`: removes server assets that are tracked in state but no longer exist locally

### Duplicate Detection

Both directions use SHA1 checksums (base64-encoded, matching Immich's format) to detect duplicates. Files already present on the target are skipped.

## Delete Safeguards

The `--delete` flag has multiple safety mechanisms:

- **Only tracked assets**: Only deletes files/assets that appear in the sync state. Assets not previously synced are never touched.
- **Server/user validation**: If the state was created with a different server or user, sync refuses to run (unless `--force`).
- **Confirmation for bulk deletes**: If more than 10 deletions are planned, interactive confirmation is required (bypass with `--force`).
- **Dry-run**: `--dry-run --delete` shows what would be deleted without actually deleting anything.

## Graceful Shutdown (Ctrl+C)

- **First Ctrl+C**: Stops processing after the current asset, saves state, releases lock, exits cleanly.
- **Second Ctrl+C**: Immediately saves state and exits.

The next sync run picks up where it left off — assets already in the state are skipped.

## Lock File

A lock file (`.immich-sync/lock`) prevents concurrent syncs on the same directory. If a sync was interrupted ungracefully (e.g., `kill -9`), you may need to manually remove the lock file.

## Examples

```bash
# First time: login once
immich-go login

# Download all server photos to local backup
immich-go sync down -d /media/backup/immich

# Preview what would be downloaded
immich-go sync down -d /media/backup/immich --dry-run

# Upload local photos to server
immich-go sync up -d /media/photos

# Two-way sync with deletion (careful!)
immich-go sync down -d /media/backup --delete
immich-go sync up -d /media/photos --delete

# Force past a server mismatch warning
immich-go sync down -d /media/backup --force
```

## Directory Structure

Downloaded files are organized by EXIF date:

```
/media/backup/immich/
├── .immich-sync/
│   ├── state.json      # Sync state
│   └── lock            # Lock file (only during sync)
├── 2024/
│   ├── 2024-01/
│   │   ├── IMG_001.jpg
│   │   └── IMG_002.jpg
│   └── 2024-06/
│       └── vacation.mp4
├── 2025/
│   └── 2025-12/
│       └── christmas.jpg
└── no-date/
    └── screenshot.png
```
