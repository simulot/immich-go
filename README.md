# Immich-Go: Upload Your Photos to Your Immich Server

**Immich-Go** is an open-source tool designed to streamline uploading large photo collections to your self-hosted Immich server.

> ⚠️ This is an early version, not yet extensively tested<br>
> ⚠️ Keep a backup copy of your files for safety<br>

## 🌟 Key Features

- **Simple Installation**: No NodeJS or Docker required
- **Multiple Sources**: Upload from Google Photos Takeouts, iCloud, local folders, ZIP archives, and other Immich servers
- **Configuration Files**: Store `immich-go`'s settings and preferences in YAML, JSON, or TOML files
- **Large Collections**: Successfully handles 100,000+ photos
- **Smart Management**: Duplicate detection, burst photo stacking, RAW+JPEG handling
- **Cross-Platform**: Available for Windows, macOS, Linux, and FreeBSD

## 🚀 Quick Start

### 1. Install Immich-Go
Download the pre-built binary for your system from the [GitHub releases page](https://github.com/simulot/immich-go/releases).

### 2. Basic Usage
```bash
# Upload photos from a local folder
immich-go upload from-folder --server=http://your-ip:2283 --api-key=your-api-key /path/to/your/photos

# Upload Google Photos takeout
immich-go upload from-google-photos --server=http://your-ip:2283 --api-key=your-api-key /path/to/takeout-*.zip

# Archive photos from Immich server
immich-go archive from-immich --server=http://your-ip:2283 --api-key=your-api-key --write-to-folder=/path/to/archive
```

### 3. Configuration Files (Optional)
Store your server details and common options in a configuration file instead of using command-line flags:

```bash
# Generate a sample configuration file
immich-go config generate immich-config.yaml

# Edit the file with your server details, then use simplified commands
immich-go upload from-folder /path/to/your/photos
```

Supports YAML, JSON, and TOML formats. Configuration can also be set via environment variables with `IMMICHGO_` prefix.
Read more in the [Configuration documentation](docs/configuration.md).

### 4. Requirements
- A running Immich server with API access
- API key with appropriate permissions ([see full list](docs/installation.md#api-permissions))

## 📚 Documentation

| Topic | Description |
|-------|-------------|
| [Installation](docs/installation.md) | Detailed installation instructions for all platforms |
| [Commands](docs/commands/) | Complete command reference and options |
| [Configuration](docs/configuration.md) | Configuration options and environment variables |
| [Examples](docs/examples.md) | Common use cases and practical examples |
| [Best Practices](docs/best-practices.md) | Tips for optimal performance and reliability |
| [Technical Details](docs/technical.md) | File processing, metadata handling, and advanced features |

## 🎯 Popular Use Cases

- **Google Photos Migration**: [Complete guide](docs/best-practices.md#google-photos-migration)
- **iCloud Import**: [Step-by-step instructions](docs/examples.md#icloud-import)
- **Server Migration**: [Transfer between Immich instances](docs/examples.md#server-migration)
- **Bulk Organization**: [Stacking and tagging strategies](docs/best-practices.md#organization-strategies)

## 💡 Support the Project

- [GitHub Sponsor](https://github.com/sponsors/simulot)
- [PayPal Donation](https://www.paypal.com/donate/?hosted_button_id=VGU2SQE88T2T4)


## 🤝 Contributing

Contributions are welcome! Please see our [contributing guidelines](CONTRIBUTING.md) for details.

## 📄 License

This project is licensed under the terms specified in the [LICENSE](LICENSE) file.

---

**Need help?** Check our [documentation](docs/) or open an issue on GitHub.