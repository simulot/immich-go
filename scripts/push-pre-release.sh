#!/bin/bash

# Pre-release Push Script
# This script creates and pushes a pre-release tag based on the previous stable version +1 and a short commit hash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Pre-release Push Script${NC}"
echo "================================"

# Check if we're on the develop branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "develop" ]; then
    echo -e "${RED}❌ Not on develop branch. Current branch: $CURRENT_BRANCH${NC}"
    echo -e "${YELLOW}Please switch to develop branch before running this script.${NC}"
    exit 1
fi

# Find the latest tag
echo -e "${YELLOW}🔍 Finding latest tag...${NC}"
LATEST_TAG=$(git tag --list | sort -V | tail -1)

if [ -z "$LATEST_TAG" ]; then
    echo -e "${RED}❌ No tags found${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Latest tag: $LATEST_TAG${NC}"

# Parse version and determine new version
VERSION=${LATEST_TAG#v}  # Remove 'v' prefix
if [[ $LATEST_TAG == *-* ]]; then
    # Latest is a pre-release, increment patch
    BASE_VERSION=${VERSION%-*}  # Remove -commit part
    IFS='.' read -r MAJOR MINOR PATCH <<< "$BASE_VERSION"
    NEW_PATCH=$((PATCH + 1))
    NEW_VERSION="$MAJOR.$MINOR.$NEW_PATCH"
else
    # Latest is stable, increment minor and set patch to 1
    IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"
    NEW_MINOR=$((MINOR + 1))
    NEW_VERSION="$MAJOR.$NEW_MINOR.1"
fi

# Get short commit hash
SHORT_COMMIT=$(git rev-parse --short HEAD)
PRE_RELEASE_TAG="v${NEW_VERSION}-${SHORT_COMMIT}"

echo -e "${BLUE}📋 Pre-release details:${NC}"
echo "  Base version: $VERSION"
echo "  New version: $NEW_VERSION"
echo "  Short commit: $SHORT_COMMIT"
echo "  Pre-release tag: $PRE_RELEASE_TAG"
echo ""

# Check if tag already exists
if git tag --list | grep -q "^$PRE_RELEASE_TAG$"; then
    echo -e "${RED}❌ Tag $PRE_RELEASE_TAG already exists${NC}"
    exit 1
fi

# Confirm before creating and pushing
read -p "Create and push pre-release tag $PRE_RELEASE_TAG? (y/N): " confirm
if [[ ! $confirm =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}👋 Aborted${NC}"
    exit 0
fi

# Create tag
echo -e "${YELLOW}🏷️  Creating tag $PRE_RELEASE_TAG...${NC}"
git tag "$PRE_RELEASE_TAG"

# Push tag
echo -e "${YELLOW}📤 Pushing tag $PRE_RELEASE_TAG...${NC}"
git push origin "$PRE_RELEASE_TAG"

echo -e "${GREEN}✅ Pre-release $PRE_RELEASE_TAG pushed successfully!${NC}"

# Check if goreleaser is installed
if ! command -v goreleaser >/dev/null 2>&1; then
    echo -e "${RED}❌ goreleaser not found. Please install goreleaser to build and release binaries.${NC}"
    exit 1
fi

# Build and release binaries
echo -e "${YELLOW}🔨 Building and releasing binaries with goreleaser...${NC}"
goreleaser release --clean

echo -e "${GREEN}✅ Binaries built and attached to pre-release successfully!${NC}"