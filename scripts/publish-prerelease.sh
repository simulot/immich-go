#!/bin/bash

# Prerelease Publishing Script
# This script creates a prerelease based on the develop branch
# Tag format: v{major}.{minor+1}.{patch}-{short_commit}
# Where patch increments for each prerelease after the last stable

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Immich Go Prerelease Publisher${NC}"
echo "====================================="

cd "$PROJECT_ROOT"

# Check for required tokens
if [ -z "$GITHUB_TOKEN" ]; then
    echo -e "${YELLOW}🔑 GITHUB_TOKEN not set, trying to get from GitHub CLI...${NC}"
    if command -v gh >/dev/null 2>&1; then
        # Check if gh auth token command is available (requires newer gh version)
        if gh auth --help 2>/dev/null | grep -q "token"; then
            GITHUB_TOKEN=$(gh auth token 2>/dev/null || echo "")
            if [ -z "$GITHUB_TOKEN" ]; then
                echo -e "${RED}❌ Could not get token from GitHub CLI${NC}"
                echo -e "${YELLOW}💡 Please ensure you're logged in with 'gh auth login'${NC}"
                exit 1
            fi
            echo -e "${GREEN}✅ Got token from GitHub CLI${NC}"
            export GITHUB_TOKEN
        else
            echo -e "${RED}❌ GitHub CLI version too old to get token automatically${NC}"
            echo -e "${YELLOW}💡 Please either:${NC}"
            echo "  1. Upgrade GitHub CLI: sudo apt update && sudo apt install gh"
            echo "  2. Or set GITHUB_TOKEN manually: export GITHUB_TOKEN=your_token_here"
            echo "     Get a token at: https://github.com/settings/tokens (scopes: repo, workflow)"
            exit 1
        fi
    else
        echo -e "${RED}❌ GITHUB_TOKEN environment variable is not set and GitHub CLI is not available${NC}"
        echo -e "${YELLOW}💡 Please either:${NC}"
        echo "  1. Install GitHub CLI and run 'gh auth login'"
        echo "  2. Or set GITHUB_TOKEN manually: export GITHUB_TOKEN=your_token_here"
        echo "     Get a token at: https://github.com/settings/tokens (scopes: repo, workflow)"
        exit 1
    fi
fi

# Pull latest changes
echo -e "${YELLOW}📥 Pulling latest changes from develop...${NC}"
git pull origin develop

# Get latest stable release tag (without prerelease suffix)
latest_stable=$(git tag --list 'v*' --sort=-version:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -1)

if [ -z "$latest_stable" ]; then
    echo -e "${RED}❌ No stable release tags found${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Latest stable release: $latest_stable${NC}"

# Parse version (remove 'v' prefix)
version=${latest_stable#v}
IFS='.' read -r major minor patch <<< "$version"

# Calculate next version (increment minor, reset patch)
next_minor=$((minor + 1))
next_version="$major.$next_minor"

echo -e "${BLUE}📊 Next version base: v$next_version.0${NC}"

# Find existing prerelease tags for the next version
existing_prereleases=$(git tag --list "v$next_version.*" --sort=-version:refname | grep -E "v$next_version\.[0-9]+-")

next_patch=0
if [ -n "$existing_prereleases" ]; then
    # Extract patch numbers and find the highest
    highest_patch=$(echo "$existing_prereleases" | sed -E "s/v$next_version\.([0-9]+)-.*/\1/" | sort -n | tail -1)
    next_patch=$((highest_patch + 1))
    echo -e "${YELLOW}📈 Found existing prereleases, next patch: $next_patch${NC}"
fi

# Get short commit hash
short_commit=$(git rev-parse --short HEAD)

# Create tag
tag="v$next_version.$next_patch-$short_commit"

echo -e "${YELLOW}🏷️  Creating prerelease tag: $tag${NC}"

# Confirm before proceeding
echo -e "${BLUE}Ready to create prerelease:${NC}"
echo "  Tag: $tag"
echo "  Based on commit: $short_commit"
echo "  Branch: $current_branch"
echo ""
read -p "Continue? (y/N): " confirm
if [[ ! $confirm =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}👋 Aborted${NC}"
    exit 0
fi

# Create and push tag
git tag "$tag"
git push origin "$tag"

# Run GoReleaser
echo -e "${YELLOW}🔨 Building and publishing prerelease...${NC}"
goreleaser release --clean

echo -e "${GREEN}✅ Prerelease published successfully!${NC}"
echo -e "${BLUE}📋 Release details:${NC}"
echo "  Tag: $tag"
echo "  URL: https://github.com/simulot/immich-go/releases/tag/$tag"
echo ""
echo -e "${YELLOW}💡 Note: Binaries are attached to the GitHub release${NC}"