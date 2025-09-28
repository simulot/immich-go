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

Before you can start contributing to `immich-go`, you need to set up your development environment.

### Prerequisites

#### Go Installation

`immich-go` requires **Go 1.25 or higher**. Here's how to install it:

**Option 1: Official Go Installer (Recommended)**
1. Visit the [official Go downloads page](https://golang.org/dl/)
2. Download the installer for your operating system
3. Follow the installation instructions for your platform

**Option 2: Package Manager Installation**

- **macOS (using Homebrew):**
  ```sh
  brew install go
  ```

- **Linux (Ubuntu/Debian):**
  ```sh
  # Remove any existing Go installation
  sudo rm -rf /usr/local/go
  
  # Download and install Go 1.25+ (check for latest version)
  wget https://golang.org/dl/go1.25.linux-amd64.tar.gz
  sudo tar -C /usr/local -xzf go1.25.linux-amd64.tar.gz
  
  # Add Go to your PATH (add this to your ~/.bashrc or ~/.zshrc)
  export PATH=$PATH:/usr/local/go/bin
  ```

- **Windows:**
  Use the official installer from golang.org or use a package manager like Chocolatey:
  ```powershell
  choco install golang
  ```

#### Verify Installation

After installation, verify that Go is properly installed:

```sh
go version
```

You should see output similar to: `go version go1.25.x linux/amd64`

#### Set Up Your Go Workspace

Make sure your `GOPATH` and `GOBIN` are properly configured:

```sh
# Check your Go environment
go env GOPATH
go env GOBIN

# If GOBIN is empty, set it (add to your shell profile)
export GOBIN=$GOPATH/bin
```

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

### Development Tools (Optional but Recommended)

For a better development experience, consider installing these tools:

- **golangci-lint** (used in our CI pipeline): 
Check the latest installation instructions at [golangci-lint](https://golangci-lint.run/docs/welcome/install/#local-installation)

```sh
# binary will be $(go env GOPATH)/bin/golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.5.0
golangci-lint --version  
```

- **gofmt** and **goimports** (for code formatting):
  ```sh
  go install golang.org/x/tools/cmd/goimports@latest
  ```

You can run the linter locally before submitting your PR:
```sh
golangci-lint run
```

## Our Git Branching Model

Our repository uses a structured branching model to manage development and releases effectively.

  * **`main`:** This branch always contains the code for the latest official release. It should be considered stable and ready for production at all times. All new code is merged into `main` only from `hotfix` or `develop` branches.
  * **`develop`:** This is our primary development branch. All new features and regular bug fixes are integrated here. It represents the state of the project for the upcoming release.
  * **`hotfix/*`:** Short-lived branches used for urgent bug fixes that must be applied directly to the latest release on `main`. These are always created from `main`.
  * **`feature/*`** and **`bugfix/*`:** Short-lived branches for developing new features or fixing non-urgent bugs. These are always created from `develop`.

## Your Contribution Workflow

Your workflow depends on the nature of your contribution:

### 1. For a New Feature or a Regular Bug Fix

For all non-urgent changes, your work should be based on the `develop` branch.

1.  **Sync with `develop`:** Ensure your local `develop` branch is up to date with the latest changes from the main repository.
    ```sh
    git checkout develop
    git pull upstream develop
    ```
2.  **Create a New Branch:** Create a new branch for your work using a descriptive name that follows our convention:
      * For features: `feature/your-feature-name`
      * For bug fixes: `bugfix/your-bug-description`
    ```sh
    git checkout -b feature/my-new-feature
    ```
3.  **Develop and Commit:** Make your changes, test them, and commit your work. Use clear and descriptive commit messages.
4.  **Push your Branch:** Push your new branch to your personal fork on GitHub.
    ```sh
    git push origin feature/my-new-feature
    ```
5.  **Create a Pull Request:** Go to your fork on GitHub and open a new Pull Request. The **base branch must be `develop`**. Your Pull Request will automatically be checked by our Continuous Integration (CI) system to ensure it meets our quality standards.

### 2. For an Urgent Hotfix

A hotfix is a critical bug that needs to be fixed in the current production version. This process is handled with extra care.

1.  **Sync with `main`:** Ensure your local `main` branch is up to date.
    ```sh
    git checkout main
    git pull upstream main
    ```
2.  **Create a New Branch:** Create a hotfix branch from `main` using a descriptive name:
      * For a hotfix: `hotfix/critical-bug`
    ```sh
    git checkout -b hotfix/critical-bug
    ```
3.  **Develop, Commit, and Push:** Make your changes, commit them, and push your hotfix branch to your fork.
4.  **Create a Pull Request:** Open a new Pull Request on GitHub. The **base branch must be `main`**. Our CI/CD pipeline will automatically run to validate the fix.

## Pull Request Guidelines

To make the review process as efficient as possible, please follow these guidelines when creating a Pull Request:

  * **Descriptive Title and Body:** Provide a clear and concise title for your PR. In the description, explain the purpose of your changes, the problem they solve, and any relevant context.
  * **Pass CI/CD Checks:** All Pull Requests must have a passing status from our automated checks before they can be merged. These checks include building the project and running tests.
  * **Target the Right Branch:** Double-check that you are opening the PR to the correct target branch (`develop` for new features/bugfixes, `main` for hotfixes). Our automated system will block incorrect merges.
  * **Code Style:** Please follow the existing code style.

## Creating Pre-releases

Pre-releases allow us to distribute development versions of the software for testing before official releases. These are created from the `develop` branch and are marked as pre-releases on GitHub.

### Automated Pre-release Workflow

We have a GitHub Actions workflow that can create pre-releases on demand:

1. **Navigate to Actions:** Go to the [Actions tab](../../actions) in the GitHub repository
2. **Select Workflow:** Choose "Create Pre-release" from the workflow list
3. **Run Workflow:** Click "Run workflow" and provide:
   - **Version:** Follow semantic versioning with a pre-release identifier (e.g., `v1.0.0-beta.1`, `v1.0.0-rc.1`)
   - **Draft:** Choose whether to create as a draft release (optional, defaults to false)
4. **Automatic Process:** The workflow will:
   - Validate the version format
   - Check that the version tag doesn't already exist
   - Run tests and linting
   - Create and push the version tag
   - Build and publish the pre-release using GoReleaser
   - Generate a changelog and update the release description

### Manual Pre-release Script

For maintainers who prefer a local approach, we provide a script at `scripts/create-prerelease.sh`:

```bash
# Basic usage
./scripts/create-prerelease.sh v1.0.0-beta.1

# Create as draft
./scripts/create-prerelease.sh v1.0.0-beta.1 --draft

# Run local checks only (no release creation)
./scripts/create-prerelease.sh v1.0.0-beta.1 --local-only
```

**Requirements for manual script:**
- GitHub CLI (`gh`) must be installed and authenticated
- `golangci-lint` should be installed for linting (optional but recommended)
- Repository must be on the `develop` branch

### Pre-release Version Naming

Follow these conventions for pre-release versions:
- **Alpha releases:** `v1.0.0-alpha.1`, `v1.0.0-alpha.2`, etc.
- **Beta releases:** `v1.0.0-beta.1`, `v1.0.0-beta.2`, etc.
- **Release candidates:** `v1.0.0-rc.1`, `v1.0.0-rc.2`, etc.

Pre-releases are automatically marked with a warning that they may contain bugs or incomplete features.

Thank you for your contribution!

