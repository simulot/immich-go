#!/bin/bash

# Script to generate release notes prompt for AI assistants
# This script retrieves commits since the last stable release and creates a prompt
# that can be used with GitHub Copilot Chat, ChatGPT, Claude, or other AI assistants
#
# Note: The GitHub Copilot CLI extension (gh-copilot) has been deprecated.
# This script now generates a prompt file that you can use with:
#   - GitHub Copilot Chat in VS Code (recommended)
#   - ChatGPT or Claude web interfaces
#   - Any other AI assistant
#
# Usage: ./scripts/generate-release-notes.sh [version]
# Arguments:
#   version - Optional. Version number for the release (e.g., v0.30.0)
#             If not provided, will try to detect from branch name or use 'NEXT'
# Output: release-notes-prompt.txt (to be used with AI)
#         release-notes-draft.md (if API is available)
#         docs/releases/release-notes-VERSION.md (where to save final notes)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored messages
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check if gh is installed
if ! command -v gh &> /dev/null; then
    print_error "GitHub CLI (gh) is not installed. Please install it first."
    exit 1
fi

# Check if user is authenticated
if ! gh auth status &> /dev/null; then
    print_error "Not authenticated with GitHub CLI."
    print_info "Run: gh auth login"
    exit 1
fi

# Get the last stable release tag (semantic versioning pattern: vX.Y.Z without -dev, -rc, etc.)
print_info "Finding last stable release..."
LAST_STABLE_TAG=$(git tag -l "v*.*.*" | grep -v -E '\-(dev|rc|alpha|beta)' | sort -V | tail -n 1)

if [ -z "$LAST_STABLE_TAG" ]; then
    print_error "No stable release tag found."
    exit 1
fi

print_info "Last stable release: $LAST_STABLE_TAG"

# Determine next version (you can modify this logic as needed)
# For now, we'll use the current date or ask for input
NEXT_VERSION="${1:-}"
if [ -z "$NEXT_VERSION" ]; then
    # Try to determine next version from branch name or use a placeholder
    if [[ "$CURRENT_REF" =~ ^release/v([0-9]+\.[0-9]+\.[0-9]+) ]]; then
        NEXT_VERSION="${BASH_REMATCH[1]}"
    else
        # Use placeholder - user can provide version as argument
        NEXT_VERSION="NEXT"
    fi
fi

print_info "Next version: $NEXT_VERSION"

# Get current branch/commit
CURRENT_REF=$(git rev-parse --abbrev-ref HEAD)
print_info "Generating release notes from $LAST_STABLE_TAG to $CURRENT_REF"

# Get commit messages since last stable release
print_info "Retrieving commits..."
COMMITS=$(git log ${LAST_STABLE_TAG}..HEAD --pretty=format:"- %s" --no-merges)

if [ -z "$COMMITS" ]; then
    print_warning "No commits found since $LAST_STABLE_TAG"
    exit 0
fi

# Count commits
COMMIT_COUNT=$(echo "$COMMITS" | wc -l)
print_info "Found $COMMIT_COUNT commits to process"

# Create temporary file with the prompt
TEMP_PROMPT=$(mktemp)

cat > "$TEMP_PROMPT" << 'EOF'
You are an expert technical writer specializing in software release notes.
Your task is to generate clear, concise, and user-friendly release notes from a list of git commits.

**Instructions:**

1.  **Analyze the commits:** Review the commit messages provided below.
2.  **Categorize changes:** Group the commits into the following categories. If a category has no items, omit it.
    - **✨ New Features**
    - **🚀 Improvements**
    - **🐛 Bug Fixes**
    - **💥 Breaking Changes**
    - **🔧 Internal Changes** (for things like refactoring, CI/CD, tests, or dependency updates)
3.  **Rewrite commit messages:** Convert the raw commit messages into human-readable notes.
    - Remove prefixes like 'feat:', 'fix:', 'chore:', 'refactor:', 'doc:', 'docs:', 'test:', 'e2e:'.
    - Summarize the change and its impact from a user's perspective.
    - Combine multiple related commits into a single point if it makes sense.
    - Skip commits that are purely internal or not relevant to users.
4.  **Format the output:** Present the notes in a clean Markdown format. Start with a brief, friendly introductory paragraph.

**Commits since LAST_TAG:**

COMMIT_LIST

---
**Generate the release notes now based on the provided commits.**
EOF

# Replace placeholders
sed -i "s/LAST_TAG/$LAST_STABLE_TAG/g" "$TEMP_PROMPT"
sed -i "/COMMIT_LIST/r /dev/stdin" "$TEMP_PROMPT" <<< "$COMMITS"
sed -i "/COMMIT_LIST/d" "$TEMP_PROMPT"

# Output file
OUTPUT_FILE="release-notes-draft.md"
PROMPT_FILE="release-notes-prompt.txt"
RELEASES_DIR="docs/releases"
VERSION_FILE="${RELEASES_DIR}/release-notes-${NEXT_VERSION}.md"

# Create releases directory if it doesn't exist
mkdir -p "$RELEASES_DIR"

print_info "Generating release notes prompt..."

# Save the prompt file (this is the primary output)
cp "$TEMP_PROMPT" "$PROMPT_FILE"
print_info "✓ Prompt saved to $PROMPT_FILE"

# Clean up temp file
rm "$TEMP_PROMPT"

# Provide instructions
echo ""
print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
print_info "NEXT STEP: Process the prompt with GitHub Copilot"
print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
print_info "Prompt file created: $PROMPT_FILE"
print_info "Target output file: $VERSION_FILE"
echo ""
print_info "In GitHub Copilot Chat, type: generate the release note for the version $NEXT_VERSION"
print_info "and it will generate the release notes following the project guidelines."
echo ""
print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
if [ "$NEXT_VERSION" = "NEXT" ]; then
    print_warning "Version is set to 'NEXT'. Run with version number:"
    print_info "  ./scripts/generate-release-notes.sh v0.30.0"
fi

print_info "Release notes prompt generated successfully!"
print_info "Commits processed: $COMMIT_COUNT"
print_info "Range: $LAST_STABLE_TAG...$CURRENT_REF"
