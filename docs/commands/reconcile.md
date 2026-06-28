# Reconcile Command

The `reconcile` command runs post-import convergence workflows that operate on migration state already stored in Immich.

## Syntax

```bash
immich-go reconcile <sub-command> [options]
```

## Sub-commands

| Sub-command | Source | Description |
|-------------|--------|-------------|
| [nextcloud-memories](#nextcloud-memories) | Nextcloud Memories | Rebuild shared-album membership from import-time migration markers |

## nextcloud-memories

Reconcile Nextcloud Memories shared albums after one or more users have already imported their own libraries.

### Migration State Contract

This command consumes migration markers written by `immich-go upload from-nextcloud-memories`:

- **Managed album description block**: owned destination albums carry a machine-managed JSON block embedded in the album description.
- **Synthetic membership tags**: imported assets can be tagged with `immich-go/src/nextcloud-memories/album/<source-album-id>`.

The reconciliation pass matches those two sources of state to add the current user's already-imported assets into the corresponding destination albums.

### Server Connection Options

| Option | Required | Description |
|--------|:--------:|-------------|
| `-s, --server` | Y | Immich server URL |
| `-k, --api-key` | Y | Immich API key for the current user |
| `--skip-verify-ssl` | | Skip SSL certificate verification |
| `--client-timeout` | | Server call timeout |

### Reconciliation Options

| Option | Default | Description |
|--------|---------|-------------|
| `--cleanup-migration-tags` | `false` | Reserved for future cleanup of synthetic membership tags after successful reconciliation |

### Current Behavior

The current implementation:

1. Lists albums visible to the current user.
2. Parses managed Nextcloud Memories state from album descriptions.
3. Resolves synthetic membership tags for source album IDs.
4. Lists current-user assets with those tags.
5. Adds missing assets to matching albums.
6. Reports malformed managed state, missing migration tags, permission failures, and albums with no matching user-owned assets.

### Current Limitations

- Cleanup of migration tags is not implemented yet; `--cleanup-migration-tags` currently returns an error.
- The command only reconciles assets owned by the authenticated Immich user.
- Assets from external libraries are skipped.

### Example

```bash
immich-go reconcile nextcloud-memories \
  --server=http://localhost:2283 \
  --api-key=your-key
```

### Typical Workflow

1. User A imports with `upload from-nextcloud-memories`, including synthetic album membership tags.
2. User B imports their own Nextcloud Memories library into the same Immich instance.
3. Each user runs `reconcile nextcloud-memories` with their own API key.
4. The command adds each user's tagged assets into the shared destination albums they can access.

## See Also

- [Upload Command](upload.md)
- [Command Reference](README.md)
- [Reconcile plan notes](../plans/2026-reconcile-command/README.md)
