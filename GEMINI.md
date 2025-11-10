# GEMINI.md

read instructions in .github/copilot-instructions.md for more details on development conventions and release note generation.


## Project Overview

This project, `immich-go`, is a command-line tool written in Go. Its primary purpose is to provide an efficient way to upload large photo and video collections to a self-hosted [Immich](https://immich.app/) server.

It is designed as a cross-platform application (Windows, macOS, Linux, FreeBSD) with no external dependencies like Node.js or Docker, making it easy to install and use.

Key features include:
-   Uploading from multiple sources: local folders, Google Photos Takeout archives, iCloud, and even other Immich servers.
-   Smart management of assets, including duplicate detection, photo stacking (bursts, RAW+JPEG), and metadata handling.
-   A rich set of commands for interacting with Immich, such as `upload`, `archive`, and `stack`.

The project uses the Cobra library for its CLI structure and `goreleaser` to manage its release process.

## Building and Running

### Prerequisites

-   Go (version 1.25 or higher, as specified in `go.mod`)

### Building the Application

To build the application from the source, run the following command from the project root:

```sh
go build -o immich-go main.go
```

For a production-ready, statically linked binary similar to the CI build, use:

```sh
CGO_ENABLED=0 go build -ldflags="-s -w -extldflags=-static" -o immich-go main.go
```

### Running the Application

Once built, the application can be run directly from the command line:

```sh
./immich-go --help
```

The main commands are `upload`, `archive`, and `stack`. Here is a basic usage example:

```sh
./immich-go upload from-folder --server http://your-immich-server:2283 --api-key YOUR_API_KEY /path/to/photos
```

## Development

### Testing

The project has both unit and end-to-end (E2E) tests.

**Unit Tests:**

Run all unit tests with race condition detection and coverage:

```sh
go test -race -v -count=1 -coverprofile=coverage.out ./...
```

**End-to-End (E2E) Tests:**

E2E tests require a running Immich server and are run against a specific client test suite.

```sh
# Navigate to the E2E client directory
cd internal/e2e/client

# Run the E2E tests
go test -v -tags=e2e -timeout=30m ./...
```

### Linting

The project uses `golangci-lint` for code quality. To run the linter, you can use the `golangci-lint` command:

```sh
golangci-lint run
```

This is also integrated into the CI pipeline.

