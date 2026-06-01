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
| `--nextcloud-skip-verify-ssl` |  | Skip TLS verification for the source |
| `--nextcloud-client-timeout` |  | Timeout for source API calls |

## Discovery and Scope Options

| Option | Default | Description |
| --- | --- | --- |
| `--discover-only` | `false` | Print detected Memories configuration and exit |
| `--timeline-root` | all configured roots | Limit import to one or more configured Memories roots; repeatable |
| `--allow-unindexed` | `false` | Continue even if the source library appears partially indexed |

## Organization Options

| Option | Default | Description |
| --- | --- | --- |
| `--sync-albums` | `true` | Recreate Memories albums in Immich |

## Supported Data

The planned importer is expected to preserve:

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

The planned importer is not expected to preserve:

- people and face assignments
- album collaborators and shares
- comments
- trashed state
- data outside the configured Memories timeline roots

Hidden or shared album semantics may be reduced during migration if Immich has no equivalent destination model.

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

### Continue despite partial indexing

```bash
immich-go upload from-nextcloud-memories \
  --nextcloud-url=https://cloud.example.com \
  --nextcloud-user=alice \
  --nextcloud-password="$NEXTCLOUD_APP_PASSWORD" \
  --allow-unindexed \
  --server=http://immich.example.com:2283 \
  --api-key="$IMMICH_API_KEY"
```

Use this only when you understand that some metadata or album relationships may be incomplete on the source side.