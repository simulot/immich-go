# Installation Guide

This guide covers all installation methods for Immich-Go across different platforms.

## Prerequisites

### For Pre-built Binaries
- No prerequisites needed - just download and run!

### For Building from Source
- Go 1.25 or higher
- Git

### System Requirements
- **Immich Server**: You need a running Immich server
- **API Key**: Generate from Account settings > API Keys > New API Key
- **Basic Command Line Knowledge**: Immich-Go is a command-line tool

## API Permissions
`
Create an` immich API key for each user account you plan to use with `Immich-Go` with the following permissions:
- `asset.read`
- `asset.statistics` 
- `asset.update`
- `asset.upload`
- `asset.replace`
- `asset.download`
- `album.create`
- `album.read`
- `albumAsset.create`
- `server.about`
- `stack.create`
- `tag.asset`
- `tag.create`
- `user.read`

Immich-Go needs to pause Immich jobs during upload operations. Create an admin-linked API key that includes the permissions listed above, plus the following additional permission:

- `job.create`
- `job.read`


## Installation Methods

### Option 1: Pre-built Binaries (Recommended)

#### Supported Platforms
- **Operating Systems**: Windows, macOS, Linux, FreeBSD
- **Architectures**: AMD64 (x86_64), ARM

#### Installation Steps

1. **Download**: Visit the [releases page](https://github.com/simulot/immich-go/releases/latest)

2. **Select your platform**:
   - Windows: `immich-go_Windows_x86_64.zip`
   - macOS: `immich-go_Darwin_x86_64.tar.gz`  
   - Linux: `immich-go_Linux_x86_64.tar.gz`
   - FreeBSD: `immich-go_Freebsd_x86_64.tar.gz`

3. **Extract the archive**:
   ```bash
   # Linux/macOS/FreeBSD
   tar -xzf immich-go*.tar.gz

   # Windows
   # Use Windows Explorer or your preferred zip tool
   ```

4. **Make it accessible (Optional)**:
   ```bash
   # Linux/macOS/FreeBSD - Add to system PATH
   sudo mv immich-go /usr/local/bin/

   # Windows - Move immich-go.exe to a directory in your PATH
   # Or add the current directory to your system PATH
   ```

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/simulot/immich-go.git

# Change to project directory
cd immich-go

# Build the binary
go build

# Install to GOPATH/bin (optional)
go install
```

### Option 3: Nix Package Manager

Immich-Go is available in nixpkgs:

```bash
# Try without installing
nix-shell -I "nixpkgs=https://github.com/NixOS/nixpkgs/archive/nixos-unstable-small.tar.gz" -p immich-go

# Or with flakes enabled
nix run "github:nixos/nixpkgs?ref=nixos-unstable-small#immich-go" -- --help
```

For NixOS users, add `immich-go` to your `configuration.nix`:
```nix
environment.systemPackages = with pkgs; [
  immich-go
];
```

### Special Case: Termux (Android)

Pre-built ARM64 binaries don't work in Termux. Build from source:

```bash
# Install dependencies
pkg install git golang

# Build following the standard source instructions above

# Add to PATH (if using go install)
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
source ~/.bashrc
```

## Verification

After installation, verify Immich-Go works correctly:

```bash
immich-go --version
```

This should display the version number.

## Running Immich-Go

### Basic Syntax
```bash
immich-go command sub-command [options] [path]
```

### Platform-Specific Notes

- **Linux, macOS, FreeBSD**: If in current directory, use `./immich-go`
- **Windows**: If in current directory, use `.\immich-go`

## Troubleshooting

### Permission Denied (Linux/macOS)
```bash
chmod +x immich-go
```

### Command Not Found
- Ensure the binary is in your PATH, or
- Use the full path to the binary, or  
- Run from the directory containing the binary

### SSL/TLS Issues
Use the `--skip-verify-ssl` flag if you have certificate issues (not recommended for production).

## Next Steps

- [Learn about commands](commands/) 
- [See configuration options](configuration.md)
- [Check out examples](examples.md)