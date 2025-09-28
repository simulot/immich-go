#!/bin/bash

# Pre-release script for immich-go
# This script helps create pre-releases from the develop branch

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to validate version format
validate_version() {
    local version=$1
    if [[ ! $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
        print_error "Invalid version format: $version"
        print_info "Version must follow semantic versioning format (e.g., v1.0.0-beta.1)"
        exit 1
    fi
}

# Function to check if tag exists
check_tag_exists() {
    local version=$1
    if git rev-parse "$version" >/dev/null 2>&1; then
        print_error "Tag '$version' already exists"
        exit 1
    fi
}

# Function to check if we're on the right branch
check_branch() {
    local current_branch=$(git branch --show-current)
    if [[ "$current_branch" != "develop" ]]; then
        print_warning "You are on branch '$current_branch', not 'develop'"
        read -p "Do you want to switch to develop branch? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_info "Switching to develop branch..."
            git checkout develop
            git pull origin develop
        else
            print_error "Pre-release must be created from the develop branch"
            exit 1
        fi
    fi
}

# Function to run tests
run_tests() {
    print_info "Running tests..."
    if go test ./...; then
        print_success "Tests passed"
    else
        print_error "Tests failed"
        exit 1
    fi
}

# Function to run linter
run_linter() {
    print_info "Running linter..."
    if command -v golangci-lint >/dev/null 2>&1; then
        if golangci-lint run; then
            print_success "Linter passed"
        else
            print_error "Linter failed"
            exit 1
        fi
    else
        print_warning "golangci-lint not found, skipping linter check"
    fi
}

# Function to generate changelog
generate_changelog() {
    local version=$1
    local last_tag=$(git describe --tags --abbrev=0 HEAD~1 2>/dev/null || echo "")
    
    print_info "Generating changelog..."
    
    if [ -n "$last_tag" ]; then
        print_info "Changes since $last_tag:"
        git log --pretty=format:"- %s (%h)" $last_tag..HEAD
    else
        print_info "No previous tag found, showing recent commits:"
        git log --pretty=format:"- %s (%h)" -10
    fi
    echo
}

# Function to create GitHub release using gh CLI
create_github_release() {
    local version=$1
    local draft_flag=$2
    
    if ! command -v gh >/dev/null 2>&1; then
        print_error "GitHub CLI (gh) is not installed"
        print_info "Please install it from https://cli.github.com/ or use the GitHub Actions workflow instead"
        exit 1
    fi
    
    print_info "Creating GitHub release..."
    
    local release_notes="## Pre-release $version

This is a pre-release build based on the develop branch.

⚠️ **Warning**: This is a pre-release version and may contain bugs or incomplete features. Use at your own risk.

### Changes since last release:
$(generate_changelog $version)

### Installation

Download the appropriate binary for your platform from the assets below.

### Feedback

Please report any issues or feedback in the GitHub Issues."
    
    local gh_flags="--prerelease --target develop"
    if [[ "$draft_flag" == "true" ]]; then
        gh_flags="$gh_flags --draft"
    fi
    
    echo "$release_notes" | gh release create "$version" $gh_flags --title "Pre-release $version" --notes-file -
    
    print_success "GitHub release created: $version"
}

# Main function
main() {
    print_info "immich-go Pre-release Script"
    echo
    
    # Check if we have the required arguments
    if [[ $# -lt 1 ]]; then
        print_error "Usage: $0 <version> [--draft] [--local-only]"
        print_info "Example: $0 v1.0.0-beta.1"
        print_info "Options:"
        print_info "  --draft      Create as draft release"
        print_info "  --local-only Only run local checks, don't create release"
        exit 1
    fi
    
    local version=$1
    local draft_flag="false"
    local local_only="false"
    
    # Parse additional arguments
    shift
    while [[ $# -gt 0 ]]; do
        case $1 in
            --draft)
                draft_flag="true"
                shift
                ;;
            --local-only)
                local_only="true"
                shift
                ;;
            *)
                print_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Validate inputs
    validate_version "$version"
    check_tag_exists "$version"
    check_branch
    
    # Update develop branch
    print_info "Updating develop branch..."
    git pull origin develop
    
    # Run checks
    run_tests
    run_linter
    
    # Show changelog
    generate_changelog "$version"
    
    if [[ "$local_only" == "true" ]]; then
        print_success "Local checks completed successfully!"
        print_info "To create the actual release, run without --local-only flag"
        exit 0
    fi
    
    # Confirm before proceeding
    echo
    print_warning "This will create a pre-release $version from the current develop branch"
    read -p "Do you want to continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Operation cancelled"
        exit 0
    fi
    
    # Create and push tag
    print_info "Creating and pushing tag $version..."
    git tag -a "$version" -m "Pre-release $version"
    git push origin "$version"
    
    # Create GitHub release
    create_github_release "$version" "$draft_flag"
    
    print_success "Pre-release $version created successfully!"
    print_info "You can view it at: https://github.com/$(gh repo view --json owner,name -q '.owner.login + "/" + .name')/releases/tag/$version"
}

# Run main function with all arguments
main "$@"