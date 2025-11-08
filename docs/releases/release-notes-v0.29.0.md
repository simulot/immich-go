# Release Notes - v0.29.0

This release brings powerful new filtering capabilities for the `from-immich` command and configuration file support, making it easier to manage complex migration scenarios.

## ✨ New Features

* **Advanced asset filtering for `from-immich`** - Filter your Immich library with precision using:
  * `--from-people` - Filter by people/faces in photos
  * `--from-tag` - Filter by tags
  * `--from-album`, `--from-no-album` - Filter by albums, including assets not in any album
  * `--from-city`, `--from-country`, `--from-state` - Geographic filtering
  * `--from-make`, `--from-model` - Camera make and model filtering with validation
  * `--from-archived`, `--from-trashed`, `--from-favorite` - Filter by asset state
  * `--from-minimal-rating` - Filter by minimum rating
  * `--from-partners` - Exclude partner's shared photos
  * `--from-date-range` - Filter by a date range (dates, date, year) 
  * `--from-include-extensions`, `--from-exclude-extensions` - Include/exclude file extensions (e.g., include `.jpg` but exclude `.bmp`)
  * `--from-include-type` - Select only videos or only photos

  Example:
  ```bash
  immich-go from-immich [server parameters] --from-people="John" --from-date=2024 --from-favorite
  ```

* **Configuration file support** - Create YAML/TOML configuration files to manage settings across projects:
  ```yaml
  server: https://immich.example.com
  api-key: your-api-key
  concurrency: 10
  ```
  Configuration values can come from files, environment variables, or command-line flags, with clear logging of which source is used.
  Check the documentation for supported formats and usage at [/docs/configuration.md](/docs/configuration.md).

## 🚀 Improvements

* **Renamed `--server-errors` to `--on-errors`** - More consistent flag naming across commands for error handling behavior

* **Better upload logging** - Upload events now show asset details instead of just filenames, making it easier to track progress during large migrations

* **Enhanced configuration visibility** - The application now logs:
  * Which configuration file is being used (if any)
  * Where each setting comes from (file, environment, or command-line)
  * All active flags and parameters at startup for easier debugging

* **Streamlined CLI structure**:
  * Concurrency settings (e.g., `--concurrent-upload`) now available from the root command
  * Clearer separation between `upload`, `archive`, and `from-*` commands
  * Simplified `from-folder` command usage

* **Date range validation** - Automatically prevents invalid date ranges where "before" is after "after"

* **Comprehensive documentation** - New guides for installation, usage examples, migration scenarios, and performance tuning

## 💥 Breaking Changes

* **Asset copy API change** - If you use the API directly: `ReplaceAsset` method is deprecated in favor of `CopyAsset`, which uses the `assets/copy` endpoint

* **API key permissions update** - API keys must now include `asset.copy` and `asset.delete` permissions in addition to previously required permissions. Please update your API keys accordingly.

## 🔧 Internal Changes

* **Complete E2E testing infrastructure** - Comprehensive automated testing with Linux and Windows clients, ensuring reliability across platforms

* **Improved CI/CD** - Smarter test execution (skips E2E tests for documentation-only changes), updated GitHub Actions, unified workflow

* **Code quality** - Updated to Immich API 2.0.0, better code organization, enhanced debugging capabilities

* **Developer tools** - VS Code tasks for common operations, automated release workflow, development branch now named `develop`
