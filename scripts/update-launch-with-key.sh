#!/bin/bash
set -e
# Get the new API key from the e2eusers.env file
ADMIN_API_KEY=$(grep "E2E_admin@immich.app_APIKEY" internal/e2e/testdata/immich-server/e2eusers.env | cut -d'=' -f2)

# Update the launch.json file with the new API key
sed -i -E "s/--api-key=([A-Za-z0-9]+)/--api-key=$ADMIN_API_KEY/" .vscode/launch.json
