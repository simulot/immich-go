# Contributing to immich-go

Hello, and thank you for your interest in contributing to `immich-go`! Your help is vital for the health and growth of this project. To ensure a smooth and effective collaboration, please follow the guidelines outlined in this document.

By following these rules, you help us maintain a clean, stable, and well-organized codebase.

## Getting Started

To get started with your contribution, please follow the standard GitHub workflow:

1.  **Fork the Repository:** Start by forking the `simulot/immich-go` repository to your own GitHub account. This creates a personal copy of the project where you can work freely.
2.  **Clone your Fork:** Clone your fork to your local machine to begin development. You can do this using the Git command line:
    ```sh
    git clone https://github.com/<your-username>/immich-go.git
    cd immich-go
    ```
3.  **Add the Original Repository as a Remote:** To stay up to date with the main project, add the original repository as an "upstream" remote.
    ```sh
    git remote add upstream https://github.com/simulot/immich-go.git
    ```
    You can now use `git pull upstream develop` to fetch the latest changes from the main development branch.

## Development Setup

Before you can start contributing to `immich-go`, you need to set up your development environment. A `go.mod` file is present and defines the Go version to use.

### Prerequisites

- **Go Installation**: `immich-go` requires the version of Go specified in the `go.mod` file.
- **golangci-lint**: Optional, but recommended for running the linter locally.

### Building and Testing

Once you have Go installed and your fork cloned:

1. **Navigate to the project directory:**
   ```sh
   cd immich-go
   ```
2. **Install dependencies:**
   ```sh
   go mod download
   ```
3. **Build the project:**
   ```sh
   go build -o immich-go main.go
   ```
4. **Run tests:**
   ```sh
   go test ./...
   ```
5. **Run the application:**
   ```sh
   ./immich-go --help
   ```

You can run the linter locally before submitting your PR:
```sh
golangci-lint run
```

## CI/CD Workflows

Our repository uses a multi-tier CI workflow system that provides fast feedback to all contributors while securely managing E2E tests that require secrets.

### 1. Fast Feedback Workflow (`.github/workflows/pr-fast-feedback.yml`)

**Runs automatically on all pull requests** - No secrets required.

Provides quick validation in 3-5 minutes with parallel jobs:
- **Linting:** `golangci-lint` for code quality checks
- **Unit Tests (Linux):** Comprehensive tests with race detection (CGO_ENABLED=0)
- **Security Scanning:** `govulncheck` for vulnerabilities + dependency review
- **Build Check:** Cross-platform compilation validation (Linux, Windows, macOS, ARM)

**Path Filtering:** Only runs when code files change. Documentation-only PRs skip this workflow for efficiency.

**Security:** This workflow runs on all PRs, including from external contributors, without exposing any secrets.

### 2. E2E Tests Workflow (`.github/workflows/e2e-tests.yml`)

**Requires maintainer approval for external contributors** - Uses Tailscale secrets.

Runs comprehensive end-to-end tests with real Immich server (12-15 minutes):
- **E2E Server:** Ubuntu runner deploying Immich in Docker, accessible via Tailscale network
- **Linux Client:** Creates admin user and runs E2E tests against the server
- **Windows Client:** Runs unit tests, builds binary, verifies build, then runs E2E tests

**Cost Optimization:** Windows testing is consolidated into the E2E workflow only (not in fast feedback) to reduce runner costs.

#### Who Can Run E2E Tests?

- **Trusted Contributors** (Repository members/collaborators): E2E tests run **automatically** when code changes
- **External Contributors**: E2E tests require **maintainer approval**

#### How to Run E2E Tests on an External PR

When you submit a pull request from a fork:
1. ✅ Fast feedback checks run immediately (lint, test, security, build)
2. 🤖 A comment will explain that E2E tests require maintainer approval
3. ⌛ A maintainer will review your code for safety and correctness
4. ✅ To approve, the maintainer can either:
   - Post a comment: `/run-e2e`
   - Approve the PR (triggers E2E automatically)
5. 🚀 The E2E test workflow will then start

**Why?** This approval process prevents malicious code in a PR from accessing sensitive credentials (Tailscale network, Immich server).

### 3. Documentation Quality Checks (`.github/workflows/doc-quality-checks.yml`)

**Manual trigger only** - Disabled by default, can be enabled later.

Validates documentation quality with auto-fix capability:
- **Markdown Linting:** Style and formatting checks
- **Link Validation:** Detects broken links
- **Spell Checking:** Technical dictionary with project terms
- **Grammar:** Microsoft Writing Style Guide

To run manually: Go to Actions → Documentation Quality Checks → Run workflow

### 4. Workflow Security Gate (`.github/workflows/check-workflow-changes.yml`)

**Runs automatically on PRs modifying workflow files** - No secrets required.

Detects changes to `.github/workflows/**` or `CODEOWNERS` and:
- Posts a security review checklist for maintainers
- Automatically requests review from code owners
- Helps prevent malicious workflow modifications

**CODEOWNERS Protection:** Workflow files require approval from designated code owners before merge.

## Our Git Branching Model

Our repository uses a simple branching model:

  * **`main`:** This branch always contains the code for the latest official release. It is considered stable.
  * **`develop`:** This is the primary development branch. All new features and bug fixes should be integrated here.

## Your Contribution Workflow

1.  **Sync with `develop`:** Ensure your local `develop` branch is up to date with the latest changes from the main repository.
    ```sh
    git checkout develop
    git pull upstream develop
    ```
2.  **Create a New Branch:** Create a new branch for your work from the `develop` branch. Use a short, descriptive name (e.g., `fix-login-bug`, `add-album-feature`).
    ```sh
    git checkout -b my-new-feature
    ```
3.  **Develop and Commit:** Make your changes, test them, and commit your work. Use clear and descriptive commit messages.
4.  **Push your Branch:** Push your new branch to your personal fork on GitHub.
    ```sh
    git push origin my-new-feature
    ```
5.  **Create a Pull Request:** Go to your fork on GitHub and open a new Pull Request.
    *   The **base branch must be `develop`**.
    *   Your Pull Request will automatically be checked by our CI system.

## Pull Request Guidelines

To make the review process as efficient as possible, please follow these guidelines:

  * **Descriptive Title and Body:** Provide a clear and concise title for your PR. In the description, explain the purpose of your changes, the problem they solve, and any relevant context.
  * **Pass CI Checks:** All Pull Requests must have a passing status from our automated checks before they can be merged.
  * **Target the `develop` Branch:** All pull requests should target the `develop` branch. Hotfixes to `main` are handled by maintainers as an exception.

Thank you for your contribution!

