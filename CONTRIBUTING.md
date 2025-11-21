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

Our repository uses a two-tier CI workflow system that provides fast feedback to all contributors while securely managing E2E tests that require secrets.

### 1. Fast Feedback Workflow (`.github/workflows/pr-checks.yml`)

**Runs automatically on all pull requests** - No secrets required.

Provides quick feedback on every pull request and push:
- **Linting:** `golangci-lint` for code quality.
- **Unit Tests:** Comprehensive tests with race detection and coverage.
- **Build Check:** Validates successful compilation.

**Security:** This workflow runs on all PRs, including from external contributors, without exposing any secrets.

### 2. E2E Tests Workflow (`.github/workflows/run-e2e.yml`)

**Requires maintainer approval for external contributors** - Uses Tailscale secrets.

Runs comprehensive end-to-end tests:
- **E2E Server:** Ubuntu runner with Immich in Docker (via Tailscale).
- **Linux & Windows Client Tests:** E2E tests on both Linux and Windows runners.

#### Who Can Run E2E Tests?

- **Trusted Contributors** (Repository collaborators): E2E tests run automatically.
- **External Contributors**: E2E tests are **skipped** by default. A maintainer must approve them.

#### How to Run E2E Tests on an External PR

When you submit a pull request from a fork:
1. ✅ Fast feedback checks run immediately (lint, test, build).
2. 🤖 A bot will comment on your PR explaining that E2E tests require maintainer approval.
3. ⌛ A maintainer will review your code for safety and correctness.
4. ✅ To approve, the maintainer will post a comment with the command `/run-e2e`.
5. 🚀 The E2E test workflow will then start automatically.

**Why?** This approval process prevents malicious code in a PR from accessing sensitive credentials.

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

