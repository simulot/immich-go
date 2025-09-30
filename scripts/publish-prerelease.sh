#!/bin/bash
# filepath: scripts/publish-prerelease.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Publishing Prerelease from develop branch${NC}"
echo "========================================"

# Check if we're on the develop branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "develop" ]; then
    echo -e "${YELLOW}⚠️  Switching to develop branch...${NC}"
    git checkout develop
fi

# Pull latest changes
echo -e "${YELLOW}📥 Pulling latest changes...${NC}"
git pull origin develop

# Check if working directory is clean
if [ -n "$(git status --porcelain)" ]; then
    echo -e "${RED}❌ Working directory is not clean. Please commit or stash changes.${NC}"
    exit 1
fi

# Get the current version from the latest tag or use default
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.28.0")
echo -e "${BLUE}Latest stable version: ${LATEST_TAG}${NC}"

# Increment version for next release (develop branch work)
# Extract version numbers and increment minor version
if [[ $LATEST_TAG =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    MAJOR=${BASH_REMATCH[1]}
    MINOR=${BASH_REMATCH[2]}
    PATCH=${BASH_REMATCH[3]}
    # Increment minor version for next development cycle
    NEXT_MINOR=$((MINOR + 1))
    NEXT_VERSION="v${MAJOR}.${NEXT_MINOR}.0"
else
    # Fallback if tag format doesn't match
    NEXT_VERSION="v0.29.0"
fi

echo -e "${BLUE}Next development version: ${NEXT_VERSION}${NC}"

# Generate prerelease version based on current date and commit
DATE=$(date +"%Y%m%d")
SHORT_COMMIT=$(git rev-parse --short HEAD)
PRERELEASE_VERSION="${NEXT_VERSION}-alpha.${DATE}.${SHORT_COMMIT}"

echo -e "${YELLOW}📋 Creating prerelease: ${PRERELEASE_VERSION}${NC}"

# Create and push the tag
git tag -a "$PRERELEASE_VERSION" -m "Prerelease $PRERELEASE_VERSION from develop branch

Built from commit: ${SHORT_COMMIT}
Date: $(date -u)

This prerelease includes binaries for:
- Linux AMD64
- Linux ARM64  
- macOS AMD64
- Windows AMD64"

git push origin "$PRERELEASE_VERSION"

# Check if GoReleaser is installed
if ! command -v goreleaser &> /dev/null; then
    echo -e "${RED}❌ GoReleaser is not installed. Please install it first:${NC}"
    echo "  # Using Homebrew"
    echo "  brew install goreleaser/tap/goreleaser"
    echo ""
    echo "  # Using Go"
    echo "  go install github.com/goreleaser/goreleaser@latest"
    echo ""
    echo "  # Using apt (Ubuntu/Debian)"
    echo "  echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | sudo tee /etc/apt/sources.list.d/goreleaser.list"
    echo "  sudo apt update && sudo apt install goreleaser"
    exit 1
fi

# Check if GitHub token is set
if [ -z "$GITHUB_TOKEN" ]; then
    echo -e "${RED}❌ GITHUB_TOKEN environment variable is not set.${NC}"
    echo "Please set your GitHub token:"
    echo "  export GITHUB_TOKEN=your_token_here"
    echo ""
    echo "You can create a token at: https://github.com/settings/tokens"
    echo "Required scopes: repo, write:packages"
    exit 1
fi

# Validate GoReleaser configuration
echo -e "${YELLOW}🔍 Validating GoReleaser configuration...${NC}"
if ! goreleaser check; then
    echo -e "${RED}❌ GoReleaser configuration is invalid${NC}"
    exit 1
fi

# Run GoReleaser with prerelease flag
echo -e "${YELLOW}🔨 Building and publishing prerelease...${NC}"
echo "This will create binaries for:"
echo "  - Linux AMD64"
echo "  - Linux ARM64"
echo "  - macOS AMD64 (Darwin)"
echo "  - Windows AMD64"

if goreleaser release --prerelease --clean --timeout 20m; then
    echo -e "${GREEN}✅ Prerelease published successfully!${NC}"
    echo -e "${BLUE}🔗 Check your release at: https://github.com/simulot/immich-go/releases/tag/${PRERELEASE_VERSION}${NC}"
    echo ""
    echo -e "${YELLOW}📦 Available downloads:${NC}"
    echo "  - immich-go_${PRERELEASE_VERSION#v}_Linux_x86_64.tar.gz"
    echo "  - immich-go_${PRERELEASE_VERSION#v}_Linux_arm64.tar.gz"
    echo "  - immich-go_${PRERELEASE_VERSION#v}_Darwin_x86_64.tar.gz"
    echo "  - immich-go_${PRERELEASE_VERSION#v}_Windows_x86_64.zip"
    echo ""
    echo -e "${BLUE}💡 Installation example:${NC}"
    echo "  curl -L https://github.com/simulot/immich-go/releases/download/${PRERELEASE_VERSION}/immich-go_${PRERELEASE_VERSION#v}_Linux_x86_64.tar.gz | tar -xz"
else
    echo -e "${RED}❌ GoReleaser failed${NC}"
    exit 1
fi

# Update release notes for next version
echo -e "${YELLOW}📝 Consider updating release notes in docs/releases.md${NC}"
echo "Current alpha version: ${PRERELEASE_VERSION}"