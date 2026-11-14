# E2E Immich instance

Scripts that bring up a disposable Immich instance for the end-to-end tests.

The instance has its own compose project (`immich-e2e`) and its own container
names. It never touches another Immich install on the machine. Its API listens
on `127.0.0.1` only, because the test credentials are well known.

The containers run **rootless** by default: they use the developer's uid/gid,
drop `NET_RAW` and set `no-new-privileges`, as in Immich's
`docker/docker-compose.rootless.yml`. The bind mounts stay owned by the
developer, so `e2e-cleanup.sh` does not need `sudo`.

## Prerequisites

- Docker with Compose v2, or Podman with a Compose provider
- `curl`
- Go, to build the user creation tool
- Network access to `github.com/immich-app/immich` for the compose file and images

## Container engine

The scripts use Docker by default and switch to Podman when no Docker daemon
answers. `E2E_ENGINE` forces one engine:

| `E2E_ENGINE` | Behaviour |
|---|---|
| unset or `auto` | Docker when `docker info` succeeds, Podman otherwise |
| `docker` | Docker only |
| `podman` | Podman only |

The scripts ask the engine (`docker info`, `podman info`), never the client
version. `docker compose version` succeeds while the daemon is down, so it
cannot prove that an engine runs.

Podman needs a Compose provider: `podman compose` runs `docker-compose` or
`podman-compose` found in `PATH`. Podman 5.0 or newer is recommended.

Podman also needs two things on the host, or the stack cannot start:

- Rootless mode with a user id range: the `uidmap` package for `newuidmap`, and
  an entry for the user in `/etc/subuid` and `/etc/subgid`. Without them Podman
  falls back to a single-id mapping, and unpacking an image layer fails on a
  file owned by another uid (`requested 0:42 for /etc/gshadow`).
- Its API socket, because the Compose provider talks to it:
  `systemctl --user enable --now podman.socket`, or `podman system service
  --time=0` when the machine has no systemd user session.

The scripts call `podman compose` directly, so both must be in place before
`e2e-start.sh` runs.

## Quick start

```bash
# Provision the instance, create the test users, and wait for the API
scripts/e2e/e2e-start.sh

# Run the e2e tests
e2e_url=http://localhost:2283 go test -v -tags=e2e -timeout=30m ./internal/e2e/client/...

# Reset the data between runs (users and API keys are kept)
scripts/e2e/e2e-reset.sh

# Remove the instance and its volumes
scripts/e2e/e2e-cleanup.sh
```

The tests read the credentials from
`internal/e2e/testdata/immich-server/e2eusers.env`. The directory is
git-ignored, so the file cannot be committed. The scripts never print the
passwords or the API keys.

## Options

| Option | Environment variable | Default |
|---|---|---|
| `--install-dir DIR` | `E2E_INSTALL_DIR` | `internal/e2e/testdata/immich-server` |
| `--port PORT` | `E2E_PORT` | `2283` |
| `--version TAG` | `E2E_VERSION` | `v3.2.4` |
| `--force` (provision) | | refuse to replace an existing instance |
| `--no-rootless` (provision) | `E2E_ROOTLESS=false` | containers run as the current uid/gid |
| | `E2E_UID`, `E2E_GID` | `id -u`, `id -g` |
| | `E2E_ENGINE` | `auto`: Docker first, then Podman |
| `--yes` (cleanup) | | ask for confirmation |

`e2e_start.sh` passes its arguments to `e2e-provision.sh`.

The compose file and the images both come from the pinned release tag, so they
cannot drift apart. To test another Immich version:

```bash
scripts/e2e/e2e-start.sh --version v3.3.3
```

## What each script does

| Script | Purpose |
|---|---|
| `e2e-start.sh` | Calls `e2e-provision.sh`, then updates `.vscode/launch.json` when it exists |
| `e2e-provision.sh` | Downloads the compose file for the pinned release, writes a compose override, pulls the images, starts the services, waits for the API, creates the admin user |
| `e2e-reset.sh` | Stops the server, deletes assets, albums, stacks, tags and people from the database, restarts the server |
| `e2e-cleanup.sh` | Removes the e2e containers and the volumes of the e2e project, then removes the install directory |
| `lib/e2e-provision-users.sh` | Creates `admin@immich.app` and its API key through the Go tool in `internal/e2e/e2eUtils/cmd/createUser` |
| `lib/common.sh` | Shared settings and helpers, sourced by the other scripts |

## Safety

- Every compose call uses `-p immich-e2e` and explicit `-f` files. The scripts
  never act on the `immich` project of a real installation.
- The container names are prefixed with the project name.
- `e2e-cleanup.sh` removes only the volumes attached to the e2e project, selected
  by their `com.docker.compose.project` label. It does not run
  `<engine> system prune` and does not remove volumes by name pattern.
- `e2e-reset.sh` resolves the database container from the e2e project. It has no
  fallback to other Postgres containers.
- `e2e-provision.sh` refuses to delete an existing instance without `--force`.
- `e2e-cleanup.sh` asks for confirmation unless `--yes` is given.
- Files created by the containers may be owned by root. The cleanup script then
  prints the `sudo rm -rf` command instead of running `sudo` itself.

## Troubleshooting

Replace `docker` with `podman` in the commands below when `E2E_ENGINE=podman`.

```bash
# Status of the e2e containers
docker compose -p immich-e2e ps

# Logs of the server
docker compose -p immich-e2e logs --tail=100 immich-server

# Check the API
curl -s http://localhost:2283/api/server/ping
```
