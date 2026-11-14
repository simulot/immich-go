#!/bin/bash
# Copy the e2e admin API key into the local VS Code launch configuration.
#
# The file is optional: the script exits 0 when it is absent. The file must be
# git-ignored, so the key cannot be committed by accident. The key is never
# printed.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ROOT}/internal/e2e/testdata/immich-server/e2eusers.env"
LAUNCH="${ROOT}/.vscode/launch.json"

if [ ! -f "${LAUNCH}" ]; then
  echo "skipped: ${LAUNCH} does not exist"
  exit 0
fi

if git -C "${ROOT}" ls-files --error-unmatch -- .vscode/launch.json >/dev/null 2>&1; then
  echo "refused: .vscode/launch.json is tracked by git; it must not receive an API key" >&2
  exit 1
fi

if ! git -C "${ROOT}" check-ignore -q .vscode/launch.json; then
  echo "refused: .vscode/launch.json is not git-ignored" >&2
  exit 1
fi

if [ ! -f "${ENV_FILE}" ]; then
  echo "skipped: ${ENV_FILE} does not exist"
  exit 0
fi

KEY="$(sed -n 's/^E2E_admin@immich\.app_APIKEY=//p' "${ENV_FILE}" | head -n 1)"
if [ -z "${KEY}" ]; then
  echo "skipped: no admin API key in ${ENV_FILE}"
  exit 0
fi

# Escape the characters that are special in a sed replacement.
ESCAPED="$(printf '%s' "${KEY}" | sed 's/[&\\]/\\&/g')"

sed -i.bak -E "s|(--api-key=)[^\"]*|\1${ESCAPED}|g" "${LAUNCH}"
rm -f "${LAUNCH}.bak"

echo "updated ${LAUNCH}"
