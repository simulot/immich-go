# Draft User Docs: from-nextcloud-memories

This is a draft for a planned command. It does not describe shipped behavior yet.

## from-nextcloud-memories

Import photos and videos from a Nextcloud instance that uses the Memories app.

Unlike `from-folder`, this command is designed to migrate the configured Memories library for a user account. It connects to Nextcloud, discovers that user's Memories configuration, and imports the files, albums, and supported metadata from that scope.

### Usage

```bash
immich-go upload from-nextcloud-memories [options]
```

### Why There Is No `<source-path>` Argument

This command is intended to migrate a Memories library, not arbitrary Nextcloud storage.

By default it:

1. Authenticates to Nextcloud
2. Reads the user's Memories configuration
3. Resolves the configured timeline roots
4. Imports only that Memories library

If you want to import arbitrary folders, use `from-folder` after exporting or mounting the source data locally.

## Source Connection Options

| Option | Required | Description |
| --- | :---: | --- |
| `--nextcloud-url` | Y | Nextcloud base URL |
| `--nextcloud-user` | Y | Nextcloud username |
| `--nextcloud-password` | Y | Nextcloud password or app password |
| `--nextcloud-local-dir` |  | Prefer reading asset contents from a local synced Nextcloud copy to improve migration speed |
| `--nextcloud-skip-verify-ssl` |  | Skip TLS verification for the source |
| `--nextcloud-client-timeout` |  | Timeout for source API calls |

`--nextcloud-local-dir` is an optimization, not a source override. The importer still talks to Nextcloud to discover the Memories library and enumerate the configured timeline roots. When a matching file exists in the local synced copy, the importer reads it locally; otherwise it falls back to WebDAV.

## Discovery and Scope Options

| Option | Default | Description |
| --- | --- | --- |
| `--discover-only` | `false` | Print detected Memories configuration and exit |
| `--timeline-root` | all configured roots | Limit import to one or more configured Memories roots; repeatable |
| `--require-indexed` | `false` | Fail if files are found under the selected Memories roots without matching Memories metadata |

## Organization Options

| Option | Default | Description |
| --- | --- | --- |
| `--sync-albums` | `true` | Recreate Memories albums in Immich |

## Supported Data

The hidden draft importer currently preserves:

- files
- capture date
- GPS coordinates
- description
- rating
- favorite flag
- archived flag
- albums
- tags

## Limitations

The hidden draft importer does not preserve:

- people and face assignments
- album collaborators and shares
- comments
- trashed state
- data outside the configured Memories timeline roots

Shared album collaboration semantics are reduced during migration. Album membership is attached per imported file; the importer does not currently pull entire albums as a separate source of truth. When title disambiguation is needed, the current hidden implementation uses a deterministic source-derived suffix.

## Examples

### Basic import

```bash
immich-go upload from-nextcloud-memories \
  --nextcloud-url=https://cloud.example.com \
  --nextcloud-user=alice \
  --nextcloud-password="$NEXTCLOUD_APP_PASSWORD" \
  --server=http://immich.example.com:2283 \
  --api-key="$IMMICH_API_KEY"
```

### Discover the effective Memories configuration first

```bash
immich-go upload from-nextcloud-memories \
  --nextcloud-url=https://cloud.example.com \
  --nextcloud-user=alice \
  --nextcloud-password="$NEXTCLOUD_APP_PASSWORD" \
  --discover-only
```

Example output:

```text
Memories app: available
User: alice
Configured timeline roots:
  - /Photos
  - /Scans
Albums support: enabled
Index status: healthy
```

### Limit the import to one configured timeline root

```bash
immich-go upload from-nextcloud-memories \
  --nextcloud-url=https://cloud.example.com \
  --nextcloud-user=alice \
  --nextcloud-password="$NEXTCLOUD_APP_PASSWORD" \
  --timeline-root=/Photos \
  --server=http://immich.example.com:2283 \
  --api-key="$IMMICH_API_KEY"
```

### Speed up a migration with a local Nextcloud sync

```bash
immich-go upload from-nextcloud-memories \
  --nextcloud-url=https://cloud.example.com \
  --nextcloud-user=alice \
  --nextcloud-password="$NEXTCLOUD_APP_PASSWORD" \
  --nextcloud-local-dir="$HOME/Nextcloud" \
  --timeline-root=/Photos \
  --server=http://immich.example.com:2283 \
  --api-key="$IMMICH_API_KEY"
```

Use this when the Nextcloud desktop client has already staged the source library locally. Discovery and scope selection still come from Memories, but asset bytes are read from the local copy when available, which can make large migrations much faster.

### Import without album recreation

```bash
immich-go upload from-nextcloud-memories \
  --nextcloud-url=https://cloud.example.com \
  --nextcloud-user=alice \
  --nextcloud-password="$NEXTCLOUD_APP_PASSWORD" \
  --sync-albums=false \
  --server=http://immich.example.com:2283 \
  --api-key="$IMMICH_API_KEY"
```

### Fail if the Memories library is only partially indexed

```bash
immich-go upload from-nextcloud-memories \
  --nextcloud-url=https://cloud.example.com \
  --nextcloud-user=alice \
  --nextcloud-password="$NEXTCLOUD_APP_PASSWORD" \
  --require-indexed \
  --server=http://immich.example.com:2283 \
  --api-key="$IMMICH_API_KEY"
```

By default, the importer warns and continues if Nextcloud files exist under the selected Memories roots before Memories has indexed them fully. Use strict mode only when you want the import to fail closed on those gaps.