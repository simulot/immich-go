# Login Command

Store Immich server credentials in the global configuration file.

## Usage

```bash
immich-go login
```

The login command is fully interactive and prompts for:

1. **Server URL** - Your Immich server address (e.g., `http://192.168.1.100:2283` or `https://immich.example.com`)
2. **API Key** - Your personal API key (found in Immich → User Settings → API Keys)

## What it does

1. Validates server reachability via ping
2. Validates the API key by authenticating against the server
3. Displays the authenticated user email
4. Saves credentials to `~/.config/immich-go/config.yaml` (file mode `0600`)

## Configuration Layering

After login, all commands automatically pick up the saved credentials. The priority order is:

| Priority | Source | Example |
|----------|--------|---------|
| 1 (highest) | CLI flags | `--server=... --api-key=...` |
| 2 | Explicit `--config` file | `--config=myconfig.yaml` |
| 3 | Local config | `./immich-go.yaml` or `./immich-go.toml` |
| 4 | Global config | `~/.config/immich-go/config.yaml` |
| 5 (lowest) | Defaults | - |

This means you can override the global credentials per-directory with a local config file, or per-command with CLI flags.

## Example

```bash
$ immich-go login
Immich server URL (e.g. http://192.168.1.100:2283 or https://immich.example.com): https://photos.example.com
Checking server...
Server is reachable.
Enter your API key (Immich → User Settings → API Keys):
API Key: ********
Validating credentials...
Authenticated as: user@example.com
Credentials saved to /home/user/.config/immich-go/config.yaml

# Now all commands work without --server and --api-key:
$ immich-go upload from-folder /photos
$ immich-go sync down -d /backup
```

## Global Config File

The saved config file at `~/.config/immich-go/config.yaml` looks like:

```yaml
api-key: your-api-key-here
server: https://photos.example.com
```

The file is created with mode `0600` (readable only by the owner) to protect the API key.
