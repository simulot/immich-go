# Manage Command

The `manage` command provides tools for managing resources on your Immich server. Unlike `upload` or `archive`, it operates entirely on existing server-side data.

## Syntax

```bash
immich-go manage <sub-command> [options]
```

## Sub-commands

| Sub-command | Description |
|-------------|-------------|
| [people-album-sync](#people-album-sync) | Sync photos of specific people into albums |

---

## people-album-sync

Reads a YAML config file that maps albums to people and ensures each album contains all photos of its mapped people. The command is **additive only** — it never removes photos from albums or deletes anything.

### Syntax

```bash
immich-go manage people-album-sync --config=<path> [options]
```

### Required Options

| Option | Required | Description |
|--------|:--------:|-------------|
| `--config` | Y | Path to the manage YAML config file |
| `-s, --server` | Y | Immich server URL |
| `-k, --api-key` | Y | Your API key |

### Connection Options

| Option | Default | Description |
|--------|---------|-------------|
| `--skip-verify-ssl` | `false` | Skip SSL certificate verification |
| `--client-timeout` | `20m` | Server call timeout |
| `--api-trace` | `false` | Enable API call tracing |

### Config File Format

The config file is a standalone YAML file (separate from the main `immich-go.toml` configuration). It has a `people-album-sync` section containing a list of album mappings:

```yaml
people-album-sync:
  albums:
    - album: "Family Gatherings"
      mode: "all"
      people:
        names:
          - "Alice"
          - "Bob"

    - album: "Kids"
      mode: "any"
      people:
        ids:
          - "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
          - "ffffffff-1111-2222-3333-444444444444"
```

### Album Mapping Fields

| Field | Required | Description |
|-------|:--------:|-------------|
| `album` | Y* | Album name (created if it doesn't exist) |
| `album_id` | Y* | Album UUID (use instead of `album` to target an existing album by ID) |
| `people` | Y | People selector (see below) |
| `mode` | | `"all"` (default) or `"any"` — controls how multiple people are matched |

\* Specify exactly one of `album` or `album_id`.

### People Selector

People can be identified by name, by Immich UUID, or both:

```yaml
people:
  names:
    - "Alice"
    - "Bob"
  ids:
    - "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
```

### Mode

| Mode | Behavior |
|------|----------|
| `all` (default) | Only photos where **all** listed people appear together are added to the album |
| `any` | Photos where **any** listed person appears are added, including solo shots |

**Example**: Given people Alice, Bob, and Charlie:
- `mode: "all"` — only photos containing all three together
- `mode: "any"` — any photo containing Alice, Bob, or Charlie (or any combination)

### How It Works

1. Reads the YAML config file
2. For each album mapping:
   - Resolves people names to Immich person UUIDs (if using `names`)
   - Finds or creates the target album
   - Searches for matching photos based on the `mode`
   - Adds any missing photos to the album (in batches of 500)
3. Never removes photos from albums or deletes anything

Multiple mappings can target the same album. For example, you can combine `mode: "any"` and `mode: "all"` entries for the same album to build up the desired set of photos.

## Examples

### Basic Usage — Group Photos

Ensure an album contains all photos where both parents appear together:

```yaml
# family-sync.yaml
people-album-sync:
  albums:
    - album: "Mom & Dad"
      mode: "all"
      people:
        names:
          - "Mom"
          - "Dad"
```

```bash
immich-go manage people-album-sync \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --config=family-sync.yaml
```

### Any Mode — Solo and Group Shots

Collect all photos of any child into one album:

```yaml
# kids-sync.yaml
people-album-sync:
  albums:
    - album: "All Kids Photos"
      mode: "any"
      people:
        names:
          - "Emma"
          - "Jack"
          - "Lily"
```

```bash
immich-go manage people-album-sync \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --config=kids-sync.yaml
```

### Mixed Modes — Complex Album Rules

Build a "Family" album that includes any photo of the kids plus all photos where both parents appear:

```yaml
# family-complete.yaml
people-album-sync:
  albums:
    - album: "Family"
      mode: "any"
      people:
        names:
          - "Emma"
          - "Jack"
    - album: "Family"
      mode: "all"
      people:
        names:
          - "Mom"
          - "Dad"
```

### Using Person IDs

Reference people by their Immich UUIDs instead of names:

```yaml
people-album-sync:
  albums:
    - album: "Shared"
      mode: "any"
      people:
        ids:
          - "b1ba8aca-1a36-4b4b-a6e7-871afa2bfa2b"
          - "97d06b61-b01f-4f19-a931-69c33468cf9b"
```

### Debug Run

Use logging to verify behavior before running for real:

```bash
immich-go --log-level=DEBUG \
  manage people-album-sync \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --api-trace \
  --config=family-sync.yaml
```

## Tips

- **Start small**: Test with a single album mapping before adding more.
- **Use names for readability**: `names` are easier to maintain than UUIDs, but require that the person is named in Immich.
- **Use IDs for stability**: Person UUIDs won't break if you rename someone in Immich.
- **Run repeatedly**: The command is idempotent — running it again skips photos already in the album.
- **Automate with cron**: Since the command is additive and idempotent, it's safe to run on a schedule.

## See Also

- [Upload Command](upload.md) - Upload photos to Immich
- [Stack Command](stack.md) - Organize photos into stacks
- [Examples](../examples.md) - More usage examples
