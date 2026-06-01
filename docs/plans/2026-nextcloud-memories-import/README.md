# Nextcloud Memories Import

## Goal

Define a safe, user-friendly `immich-go` command for migrating a photo library from a Nextcloud instance that uses the Memories app.

The command should feel like a true Memories migration:

1. The user points `immich-go` at a Nextcloud instance.
2. The command authenticates with the source account.
3. The command discovers the effective Memories configuration for that user.
4. The import scope is limited to the Memories library, not the whole Nextcloud filesystem.
5. Albums and supported metadata are recreated in Immich.

## Non-Goals

- Not implementing a generic Nextcloud WebDAV importer for arbitrary folders
- Not migrating people or face assignments
- Not migrating album collaborators, shares, or comments
- Not exposing a public command in shipped docs before the feature exists
- Not changing existing upload command behavior for current sources

## Success Criteria

- The proposed command surface is explicit and consistent with existing `upload` sub-commands
- The source scope is auto-discovered from Memories configuration by default
- Escape hatches are available without turning the command into a generic folder crawler
- Unsupported source data is documented clearly
- Public docs remain accurate until implementation ships

## Proposed Command Surface

Primary command:

```bash
immich-go upload from-nextcloud-memories [options]
```

Optional alias:

```bash
immich-go upload from-nc-memories [options]
```

Rationale:

- `upload` already groups source-specific migrations under sub-commands
- `from-nextcloud-memories` is explicit and self-documenting
- `from-nc-memories` is a useful shorthand, but should remain an alias rather than the primary name

## Scope Model

The command should intentionally have no positional `<source-path>` argument.

Instead, the importer should:

1. Authenticate to Nextcloud
2. Check that the Memories app is installed and reachable
3. Read the effective user configuration
4. Resolve the configured `timeline_path` values
5. Import only files and albums that belong to that Memories library

This keeps the command aligned with user intent. If users want arbitrary folder import from local or remote storage, that is a different feature.

### Default Behavior

- Use the authenticated user's Memories configuration
- Auto-discover all configured `timeline_path` roots
- Preserve supported metadata and album membership
- Recreate albums in Immich when possible

### Escape Hatches

- Limit the run to one or more configured timeline roots
- Run discovery only and print resolved configuration
- Allow continuing with incomplete indexing when the user opts in
- Disable album recreation when the user wants a flat import

### Guardrails

- Fail if the Memories app is missing or disabled
- Fail if `timeline_path` is empty, unknown, or invalid
- Warn or fail when the source library appears partially indexed
- Warn when unsupported source data will be ignored
- Reject arbitrary Nextcloud paths that are outside the configured Memories roots

## Data Mapping

| Source data | Destination support | Notes |
| --- | --- | --- |
| Files | Yes | Imported from the configured Memories library |
| Capture date | Yes | From EXIF or Memories metadata |
| GPS | Yes | From EXIF or Memories metadata |
| Description | Yes | Imported when available |
| Rating | Yes | Imported when available |
| Favorite | Yes | Preserved on destination assets |
| Archived state | Yes | Preserved when detectable from Memories |
| Albums | Yes | Recreated in Immich |
| Tags | Yes | Recreated as Immich tags |
| People / faces | No | Intentionally out of scope |
| Album shares / collaborators | No | No equivalent migration target in this proposal |
| Comments | No | No destination mapping planned |
| Trash state | No | No destination mapping in current upload flow |

## Proposed Source Options

These are draft flags for discussion, not committed API.

| Option | Required | Description |
| --- | :---: | --- |
| `--nextcloud-url` | Y | Source Nextcloud base URL |
| `--nextcloud-user` | Y | Source Nextcloud username |
| `--nextcloud-password` | Y | Source password or app password |
| `--nextcloud-skip-verify-ssl` |  | Skip TLS verification for source |
| `--nextcloud-client-timeout` |  | Timeout for source API calls |
| `--discover-only` |  | Print detected Memories config and exit |
| `--timeline-root` |  | Limit import to configured Memories roots; repeatable |
| `--sync-albums` |  | Recreate Memories albums in Immich; default `true` |
| `--allow-unindexed` |  | Continue even if the source library looks partially indexed |

## Open Questions

1. Should `--allow-unindexed` be a hard opt-in, or should the default be warn-and-continue?
2. Should hidden albums be imported by default even though Immich has no equivalent hidden-album feature?
3. Should discovery print only resolved roots, or also album support and indexing status?
4. Should the command accept both password and app password fields, or document app passwords as the recommended value for `--nextcloud-password`?

## Documentation Strategy

This proposal deliberately does not modify the shipped upload command docs in `docs/commands/` yet.

Reason:

- The command does not exist yet
- The project guidelines require user-facing docs to match implemented behavior
- Draft UX and example docs should live under `docs/plans/` until implementation lands

See `user-docs-draft.md` in this plan folder for the current draft of the future user-facing documentation.