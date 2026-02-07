#!/bin/bash
set -e
# Get the new API key from the e2eusers.env file
ADMIN_API_KEY=$(grep "E2E_admin@immich.app_APIKEY" internal/e2e/testdata/immich-server/e2eusers.env | cut -d'=' -f2)

# Update the launch.json file with the new API key if it exists
LAUNCH_JSON=.vscode/launch.json
if [ -f "$LAUNCH_JSON" ]; then
	# Use sed in-place; on macOS sed requires an empty extension for -i
	if sed --version >/dev/null 2>&1; then
		sed -i -E "s/--api-key=([A-Za-z0-9]+)/--api-key=$ADMIN_API_KEY/" "$LAUNCH_JSON"
	else
		sed -i "" -E "s/--api-key=([A-Za-z0-9]+)/--api-key=$ADMIN_API_KEY/" "$LAUNCH_JSON"
	fi
	echo "Updated $LAUNCH_JSON with admin API key"
else
	echo "Warning: $LAUNCH_JSON not found — skipping VS Code launch update"
fi
