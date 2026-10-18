#!/bin/bash
# Bring up the e2e Immich instance.
#
# Arguments are passed to e2e-provision.sh. See `e2e-start.sh --help`.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"${SCRIPT_DIR}/e2e-provision.sh" "$@"

# Optional: point the local VS Code launch file at the new admin API key.
# The updater skips when the file is absent, and refuses to touch a tracked file.
if ! "${SCRIPT_DIR}/../update-launch-with-key.sh"; then
  echo "warning: .vscode/launch.json was not updated; the instance is up" >&2
fi
