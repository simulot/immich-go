# Immich-Go Documentation

Welcome to the complete documentation for **Immich-Go**, an open-source tool for uploading and managing photo collections with your self-hosted Immich server.

## 📖 Documentation Overview

This documentation is organized into several sections to help you get started quickly and master advanced features:

### 🚀 Getting Started
- [**Installation**](installation.md) - Complete installation guide for all platforms
- [**Examples**](examples.md) - Common use cases and practical examples
- [**Configuration**](configuration.md) - Environment variables and configuration options

### 📝 Command Reference
- [**Commands Overview**](commands/README.md) - Complete command structure and global options
- [**Upload Commands**](commands/upload.md) - Detailed upload command documentation
- [**Archive Commands**](commands/archive.md) - Export and archival operations
- [**Stack Commands**](commands/stack.md) - Photo organization and stacking

### 📋 Best Practices & Advanced Topics
- [**Best Practices**](best-practices.md) - Performance tips and optimization strategies
- [**Technical Details**](technical.md) - File processing, metadata handling, and internals
- [**Environment Setup**](environment.md) - Advanced environment configuration

### 🔧 Specialized Topics
- [**Concurrency**](concurrency/) - Multi-threading and performance optimization
- [**Miscellaneous**](misc/) - Additional guides and troubleshooting

## 🎯 Quick Navigation by Use Case

### New Users
1. Start with [Installation](installation.md) to set up Immich-Go
2. Review [Examples](examples.md) for your specific use case
3. Check [Best Practices](best-practices.md) for optimal performance

### Google Photos Migration
- [Google Photos Takeout Guide](misc/google-takeout.md)
- [Migration Best Practices](best-practices.md#google-photos-migration)
- [Upload from Google Photos](commands/upload.md#from-google-photos)

### Advanced Users
- [Technical Details](technical.md) for deep dive into functionality
- [Configuration](configuration.md) for advanced customization
- [Concurrency](concurrency/) for performance optimization

## 🛠 Common Commands Quick Reference

```bash
# Upload from local folder
immich-go upload from-folder --server=SERVER --api-key=KEY /path/to/photos

# Upload Google Photos takeout
immich-go upload from-google-photos --server=SERVER --api-key=KEY /path/to/takeout.zip

# Archive from Immich server
immich-go archive from-immich --server=SERVER --api-key=KEY --write-to-folder=/archive

# Stack similar photos
immich-go stack --server=SERVER --api-key=KEY
```

## 📚 Documentation Structure

```
docs/
├── README.md                    # This file - documentation hub
├── installation.md              # Installation guide
├── configuration.md             # Configuration options
├── environment.md              # Environment setup
├── examples.md                 # Practical examples
├── best-practices.md           # Performance and reliability tips
├── technical.md                # Technical details and internals
├── commands/                   # Command reference
│   ├── README.md              # Command overview
│   ├── upload.md              # Upload commands
│   ├── archive.md             # Archive commands
│   └── stack.md               # Stack commands
├── concurrency/               # Performance optimization
│   ├── README.md             # Concurrency overview
│   └── multi-threading.md    # Threading details
└── misc/                      # Additional guides
    ├── README.md             # Miscellaneous topics index
    ├── google-takeout.md     # Google Photos migration
    ├── motivation.md         # Project background
    └── troubleshooting.md    # Common issues and solutions
```

## 🆘 Getting Help

- **Documentation Issues**: Something unclear? [Open an issue](https://github.com/simulot/immich-go/issues)
- **Bug Reports**: Found a problem? [Report it](https://github.com/simulot/immich-go/issues)
- **Feature Requests**: Have an idea? [Share it](https://github.com/simulot/immich-go/discussions)
- **Debug Information**: Need to send logs? See [how to send debug data](misc/how-to-send-debug-data.md)

## 🤝 Contributing to Documentation

Documentation improvements are always welcome! See our [contributing guidelines](../CONTRIBUTING.md) for details on:
- Fixing typos or errors
- Adding examples
- Improving clarity
- Adding new sections

## 📄 License

This documentation is part of the Immich-Go project and is licensed under the same terms as specified in the [LICENSE](../LICENSE) file.

---

**Ready to get started?** Begin with the [Installation Guide](installation.md) or jump to [Examples](examples.md) for your specific use case.