#!/bin/bash
# Shared configuration and helpers for the e2e scripts.
# Source this file, do not run it.

set -euo pipefail

# Resolve directories from this file, so the scripts work from any working directory.
E2E_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
E2E_SCRIPTS_DIR="$(cd "${E2E_LIB_DIR}/.." && pwd)"
E2E_PROJECT_ROOT="$(cd "${E2E_SCRIPTS_DIR}/../.." && pwd)"

# The e2e stack has its own compose project. Never reuse "immich": a developer's
# real Immich uses that project name and the same container names, so composing
# in the wrong project acts on the wrong containers and volumes.
E2E_PROJECT="${E2E_PROJECT:-immich-e2e}"

# Immich release used for both the compose file and the images.
# Override it to test the version a user reported.
E2E_VERSION="${E2E_VERSION:-v3.2.4}"

# Loopback port served by the e2e instance.
E2E_PORT="${E2E_PORT:-2283}"

# Rootless containers: the services run as the developer's uid/gid, so the bind
# mounts stay owned by the developer and no sudo is needed to clean up.
# Same hardening as Immich's docker-compose.rootless.yml.
E2E_ROOTLESS="${E2E_ROOTLESS:-true}"
E2E_UID="${E2E_UID:-$(id -u)}"
E2E_GID="${E2E_GID:-$(id -g)}"

# Installation directory: compose file, data and credentials.
# It is git-ignored through `internal/e2e/testdata` in .gitignore.
E2E_INSTALL_DIR="${E2E_INSTALL_DIR:-${E2E_PROJECT_ROOT}/internal/e2e/testdata/immich-server}"

export E2E_PROJECT E2E_VERSION E2E_PORT E2E_INSTALL_DIR
export E2E_ROOTLESS E2E_UID E2E_GID

e2e_info() { printf '==> %s\n' "$*"; }

e2e_die() { printf 'error: %s\n' "$*" >&2; exit 1; }

e2e_require_cmd() {
  command -v "$1" >/dev/null 2>&1 || e2e_die "'$1' is required but was not found in PATH"
}

# e2e_engine prints the container engine to use: docker or podman.
# E2E_ENGINE forces one. The default, auto, prefers docker, which CI and most
# contributors have, and falls back to podman.
e2e_engine() {
  case "${E2E_ENGINE:-auto}" in
    docker|podman) printf '%s\n' "${E2E_ENGINE}" ;;
    auto)
      if docker info >/dev/null 2>&1; then
        printf 'docker\n'
      elif podman info >/dev/null 2>&1; then
        printf 'podman\n'
      else
        e2e_die "no container engine is reachable: 'docker info' and 'podman info' both failed"
      fi
      ;;
    *) e2e_die "E2E_ENGINE must be auto, docker or podman (got '${E2E_ENGINE}')" ;;
  esac
}

# e2e_require_engine checks the engine and its Compose v2 front end.
# The check asks the engine, not the client: an installed client proves nothing,
# because 'docker compose version' succeeds while the daemon is down.
e2e_require_engine() {
  local engine
  engine="$(e2e_engine)" || exit 1
  e2e_require_cmd "${engine}"
  "${engine}" info >/dev/null 2>&1 ||
    e2e_die "'${engine} info' failed: the engine is not usable"
  "${engine}" compose version >/dev/null 2>&1 ||
    e2e_die "'${engine} compose' (Compose v2) is required"
}

# e2e_compose runs the engine's Compose v2 front end against the e2e instance only.
e2e_compose() {
  local engine files=(-f "${E2E_INSTALL_DIR}/docker-compose.yml")
  engine="$(e2e_engine)" || exit 1
  if [ -f "${E2E_INSTALL_DIR}/docker-compose.e2e.yml" ]; then
    files+=(-f "${E2E_INSTALL_DIR}/docker-compose.e2e.yml")
  fi
  E2E_PROJECT="${E2E_PROJECT}" E2E_PORT="${E2E_PORT}" \
    E2E_UID="${E2E_UID}" E2E_GID="${E2E_GID}" \
    "${engine}" compose --project-directory "${E2E_INSTALL_DIR}" -p "${E2E_PROJECT}" "${files[@]}" "$@"
}

e2e_immich_url() { printf 'http://localhost:%s' "${E2E_PORT}"; }

# e2e_wait_ready waits until the Immich API answers. Returns 1 on timeout.
e2e_wait_ready() {
  local timeout elapsed url
  timeout="${1:-180}"
  elapsed=0
  url="$(e2e_immich_url)"
  while ! curl -sf "${url}/api/server/ping" >/dev/null 2>&1; do
    if [ "${elapsed}" -ge "${timeout}" ]; then
      return 1
    fi
    printf '    waiting for %s ... %ss/%ss\n' "${url}" "${elapsed}" "${timeout}"
    sleep 2
    elapsed=$((elapsed + 2))
  done
}
