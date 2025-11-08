#!/bin/bash

# Immich API Monitor Script
# This script checks for changes in the Immich OpenAPI specifications
# and can be run manually or as part of a development workflow

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
MONITOR_DIR="$PROJECT_ROOT/.github/immich-api-monitor"
TEMP_FILE="$MONITOR_DIR/temp-specs.json"
BASELINE_FILE="$MONITOR_DIR/immich-openapi-specs-baseline.json"
COMMIT_FILE="$MONITOR_DIR/last-checked-commit.txt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 Immich API Monitor${NC}"
echo "========================================"

# Create monitoring directory if it doesn't exist
mkdir -p "$MONITOR_DIR"

# Download current specs
echo -e "${YELLOW}📥 Downloading current Immich OpenAPI specs...${NC}"
if ! curl -s -f -H "Accept: application/vnd.github.raw" \
    "https://api.github.com/repos/immich-app/immich/contents/open-api/immich-openapi-specs.json?ref=main" \
    -o "$TEMP_FILE"; then
    echo -e "${RED}❌ Failed to download OpenAPI specs${NC}"
    exit 1
fi

# Check if baseline exists
if [ ! -f "$BASELINE_FILE" ]; then
    echo -e "${YELLOW}📋 No baseline found, creating initial baseline...${NC}"
    cp "$TEMP_FILE" "$BASELINE_FILE"
    
    # Get current commit
    CURRENT_COMMIT=$(curl -s https://api.github.com/repos/immich-app/immich/commits/main | jq -r '.sha' 2>/dev/null || echo "unknown")
    echo "$CURRENT_COMMIT" > "$COMMIT_FILE"
    
    echo -e "${GREEN}✅ Baseline created successfully${NC}"
    rm -f "$TEMP_FILE"
    exit 0
fi

# Compare files
echo -e "${YELLOW}🔄 Comparing with baseline...${NC}"

if cmp -s "$BASELINE_FILE" "$TEMP_FILE"; then
    echo -e "${GREEN}✅ No changes detected${NC}"
    rm -f "$TEMP_FILE"
    exit 0
fi

echo -e "${RED}🚨 Changes detected!${NC}"
echo ""

# Show version information if available
OLD_VERSION=$(jq -r '.info.version // "unknown"' "$BASELINE_FILE" 2>/dev/null || echo "unknown")
NEW_VERSION=$(jq -r '.info.version // "unknown"' "$TEMP_FILE" 2>/dev/null || echo "unknown")

echo -e "${BLUE}Version Information:${NC}"
echo "  Previous version: $OLD_VERSION"
echo "  New version: $NEW_VERSION"
echo ""

# Show commit information
CURRENT_COMMIT=$(curl -s https://api.github.com/repos/immich-app/immich/commits/main | jq -r '.sha' 2>/dev/null || echo "unknown")
LAST_COMMIT=$(cat "$COMMIT_FILE" 2>/dev/null || echo "unknown")

echo -e "${BLUE}Commit Information:${NC}"
echo "  Current Immich commit: ${CURRENT_COMMIT:0:7}"
echo "  Last checked commit: ${LAST_COMMIT:0:7}"
echo ""

# --- function to display changed endpoints ---
view_changed_endpoints() {
    echo -e "${BLUE}🔎 Analyzing API endpoint changes...${NC}"
    
    # Extract paths from both files
    mapfile -t old_paths < <(jq -r '.paths | keys[]' "$BASELINE_FILE")
    mapfile -t new_paths < <(jq -r '.paths | keys[]' "$TEMP_FILE")

    # Find added, removed and common paths
    added_paths=()
    removed_paths=()
    common_paths=()

    for path in "${new_paths[@]}"; do
        found=0
        for old_path in "${old_paths[@]}"; do
            if [[ "$path" == "$old_path" ]]; then
                found=1
                common_paths+=("$path")
                break
            fi
        done
        if [[ $found -eq 0 ]]; then
            added_paths+=("$path")
        fi
    done

    for old_path in "${old_paths[@]}"; do
        found=0
        for path in "${new_paths[@]}"; do
            if [[ "$old_path" == "$path" ]]; then
                found=1
                break
            fi
        done
        if [[ $found -eq 0 ]]; then
            removed_paths+=("$old_path")
        fi
    done

    # Find newly deprecated endpoints
    deprecated_paths=()
    for path in "${common_paths[@]}"; do
        mapfile -t methods < <(jq -r ".paths[\"$path\"] | keys[]" "$TEMP_FILE")
        for method in "${methods[@]}"; do
            new_deprecated=$(jq -r ".paths[\"$path\"].\"$method\".deprecated" "$TEMP_FILE")
            old_deprecated=$(jq -r ".paths[\"$path\"].\"$method\".deprecated" "$BASELINE_FILE")
            if [[ "$new_deprecated" == "true" && "$old_deprecated" != "true" ]]; then
                deprecated_paths+=("$method: $path")
            fi
        done
    done


    if [ ${#added_paths[@]} -gt 0 ]; then
        echo -e "${GREEN}✨ Added endpoints:${NC}"
        for path in "${added_paths[@]}"; do
            echo "  - $path"
        done
    fi

    if [ ${#removed_paths[@]} -gt 0 ]; then
        echo -e "${RED}🗑️ Removed endpoints:${NC}"
        for path in "${removed_paths[@]}"; do
            echo "  - $path"
        done
    fi

    if [ ${#deprecated_paths[@]} -gt 0 ]; then
        echo -e "${YELLOW}⚠️ Newly deprecated endpoints:${NC}"
        for path in "${deprecated_paths[@]}"; do
            echo "  - $path"
        done
    fi

    if [ ${#added_paths[@]} -eq 0 ] && [ ${#removed_paths[@]} -eq 0 ] && [ ${#deprecated_paths[@]} -eq 0 ]; then
        echo -e "${YELLOW}Endpoints are the same, but their content may have changed.${NC}"
    fi
    echo ""
}


# Prompt for action
echo -e "${YELLOW}What would you like to do?${NC}"
echo "1) View detailed diff"
echo "2) View changed endpoints"
echo "3) Update baseline (accept changes)"
echo "4) View OpenAPI spec in browser"
echo "5) Exit without updating"
echo ""

read -p "Enter your choice (1-5): " choice

case $choice in
    1)
        echo -e "${BLUE}📊 Detailed diff:${NC}"
        if command -v jq >/dev/null 2>&1; then
            # Pretty print JSON and diff
            echo "--- Baseline (old)"
            echo "+++ Current (new)"
            diff -u <(jq . "$BASELINE_FILE" 2>/dev/null || cat "$BASELINE_FILE") \
                    <(jq . "$TEMP_FILE" 2>/dev/null || cat "$TEMP_FILE") || true
        else
            # Plain text diff
            diff -u "$BASELINE_FILE" "$TEMP_FILE" || true
        fi
        echo ""
        read -p "Update baseline? (y/N): " update
        if [[ $update =~ ^[Yy]$ ]]; then
            cp "$TEMP_FILE" "$BASELINE_FILE"
            echo "$CURRENT_COMMIT" > "$COMMIT_FILE"
            echo -e "${GREEN}✅ Baseline updated${NC}"
        fi
        ;;
    2)
        view_changed_endpoints
        ;;
    3)
        cp "$TEMP_FILE" "$BASELINE_FILE"
        echo "$CURRENT_COMMIT" > "$COMMIT_FILE"
        echo -e "${GREEN}✅ Baseline updated${NC}"
        ;;
    4)
        echo -e "${BLUE}🌐 Opening Immich OpenAPI specs in browser...${NC}"
        if command -v xdg-open >/dev/null 2>&1; then
            xdg-open "https://github.com/immich-app/immich/blob/main/open-api/immich-openapi-specs.json"
        elif command -v open >/dev/null 2>&1; then
            open "https://github.com/immich-app/immich/blob/main/open-api/immich-openapi-specs.json"
        else
            echo "Please visit: https://github.com/immich-app/immich/blob/main/open-api/immich-openapi-specs.json"
        fi
        ;;
    5)
        echo -e "${YELLOW}👋 Exiting without updating baseline${NC}"
        ;;
    *)
        echo -e "${RED}❌ Invalid choice${NC}"
        ;;
esac

# Clean up
rm -f "$TEMP_FILE"

echo -e "${BLUE}📋 Monitoring Summary:${NC}"
echo "  Baseline file: $BASELINE_FILE"
echo "  Last commit file: $COMMIT_FILE"
echo "  Monitor directory: $MONITOR_DIR"