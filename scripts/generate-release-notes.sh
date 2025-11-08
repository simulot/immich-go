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
# Usage: ./scripts/generate-release-notes.sh
# Output: release-notes-prompt.txt (to be used with AI)
#         release-notes-draft.md (if API is available)

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

print_info "Generating release notes prompt..."

# Save the prompt file (this is the primary output)
cp "$TEMP_PROMPT" "$PROMPT_FILE"
print_info "✓ Prompt saved to $PROMPT_FILE"

# Try to use GitHub API if available (experimental)
# Note: GitHub's Copilot Chat API may not be available via gh api
print_info "Attempting automatic generation via GitHub API..."

# Read the prompt
PROMPT_CONTENT=$(cat "$TEMP_PROMPT")

if command -v jq &> /dev/null; then
    # Create a JSON payload for the API
    JSON_PAYLOAD=$(jq -n \
        --arg prompt "$PROMPT_CONTENT" \
        '{
            model: "gpt-4",
            messages: [
                {
                    role: "system",
                    content: "You are an expert technical writer specializing in software release notes."
                },
                {
                    role: "user",
                    content: $prompt
                }
            ],
            max_tokens: 4000,
            temperature: 0.7
        }')

    # Make API call using gh api (this may not work as the endpoint is not public)
    RESPONSE=$(gh api \
        -X POST \
        -H "Accept: application/vnd.github+json" \
        /copilot/chat/completions \
        --input - <<< "$JSON_PAYLOAD" 2>&1) && {
        
        # Extract the generated content
        GENERATED_NOTES=$(echo "$RESPONSE" | jq -r '.choices[0].message.content // empty')
        
        if [ -n "$GENERATED_NOTES" ]; then
            echo "$GENERATED_NOTES" > "$OUTPUT_FILE"
            print_info "✓ Release notes automatically generated: $OUTPUT_FILE"
        fi
    } || {
        print_info "API generation not available (expected)"
    }
fi

# Clean up temp file
rm "$TEMP_PROMPT"

# Provide instructions
echo ""
print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
print_info "NEXT STEPS: Generate release notes using AI"
print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
print_info "The prompt has been saved to: $PROMPT_FILE"
echo ""
print_info "Option 1 (Recommended): Use GitHub Copilot Chat in VS Code"
print_info "  1. Open the prompt file: code $PROMPT_FILE"
print_info "  2. Select all content (Ctrl+A)"
print_info "  3. Open Copilot Chat (Ctrl+Shift+I)"
print_info "  4. Paste and send the prompt"
print_info "  5. Save the response to $OUTPUT_FILE"
echo ""
print_info "Option 2: Use ChatGPT or Claude"
print_info "  1. Copy the content from: $PROMPT_FILE"
print_info "  2. Paste it into ChatGPT/Claude"
print_info "  3. Save the response to $OUTPUT_FILE"
echo ""
print_info "Option 3: Use with @workspace in Copilot Chat"
print_info "  In VS Code Copilot Chat, type:"
print_info "  @workspace Generate release notes from the commits in $PROMPT_FILE"
echo ""
print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

print_info "Release notes prompt generated successfully!"
print_info "Commits processed: $COMMIT_COUNT"
print_info "Range: $LAST_STABLE_TAG...$CURRENT_REF"
