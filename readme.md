# Immich-Go: Upload Your Photos to Your Immich Server

**Immich-Go** is an open-source tool designed to streamline uploading large photo collections to your self-hosted Immich server.

> ⚠️ This is an early version, not yet extensively tested<br>
> ⚠️ Keep a backup copy of your files for safety<br>

## Support the project `Immich-go`

- [GitHub Sponsor](https://github.com/sponsors/simulot)
- [PayPal Donation](https://www.paypal.com/donate/?hosted_button_id=VGU2SQE88T2T4)



## What Makes Immich-Go Special?

### Simple Installation:
  * Immich-Go doesn't require NodeJS or Docker for installation. This makes it easy to get started, even for those less familiar with technical environments.
  * Immich-Go can run on either your workstation or a NAS.

### Upload your existing photo collection to your Immich server:
  * **Upload from Google Photos Takeouts:** Immich-Go can process massive archives downloaded from Google Photos using Google Takeout. It efficiently processes these archives while preserving valuable metadata like GPS location, capture date, and album information.
  * **Upload from iCloud Takeouts:** Immich-Go can upload photos from an iCloud takeout archive, preserving the date of capture and album information.

### Handles Large Photo Collections:
  * **Upload Large Google Photos Takeouts:** Users have reported successfully uploading archives of over 100,000 photos. Read the [best practices](#google-photos-best-practices) below for more information.
  * **Upload Collections:** You can upload photos directly from your computer folders, folder trees, and compressed ZIP archives.
  * **Is Duplicate-aware:** Immich-Go identifies and discards duplicate photos, keeping only the highest-quality version on the server.
  * **Archive Your Immich Server:** Write the content of your Immich server to a folder tree, ready to be archived or migrated to another server.

### Has Many Options:
* Stack burst photos
* Manage coupled RAW and JPEG files, HEIC and JPEG files
* Use tags
* ... and much more

### Runs on Any Platform:
  * Immich-Go is available for Windows, MacOS, Linux, and FreeBSD. It can run on any platform where the Go language is ported.

## Requirements

* **Immich Server:** You need a running Immich server to use Immich-Go.
  * Prepare the server's URL (http://your-ip:2283 or https://your-domain.tld)
  * Generate an API key for each Immich user (Account settings > API Keys > New API Key).
* **Basic Knowledge of Command Line:** Immich-Go is a command-line tool, so you should be comfortable using a terminal.

## Upgrading from the Original `immich-go`, Version 0.22 and Earlier

This version is a complete rewrite of the original `immich-go` project. It is designed to be more efficient, reliable, and easier to use. It is also more flexible, with more options and features. As a consequence, the command line options have changed. Please refer to the documentation for the new options.

The visible changes are:
- **Adoption of the Linux Convention** for the command line options: use 2 dashes for long options.
- Complete restructuring of the CLI logic:
  - The `upload` command accepts 3 sub-commands: `from-google-photos`, `from-folder`, `from-immich`. This removes all ambiguity from the options.
  - The new `archive` command takes advantage of this sub-command logic. It is possible to archive from a Google Photos takeout, a folder tree, or an Immich server.

The upgrade process consists of installing the new version over the previous one. You can check the version of the installed `immich-go` by running `immich-go --version`.

# Installation

## Prerequisites

- For pre-built binaries: No prerequisites needed
- For building from source:
  - Go 1.23 or higher
  - Git

## Pre-built Binaries

The easiest way to install Immich-Go is to download the pre-built binary for your system from the [GitHub releases page](https://github.com/simulot/immich-go/releases).

### Supported Platforms:
- **Operating Systems**
  - MacOS
  - Windows
  - Linux
  - FreeBSD

- **Architectures**
  - AMD64 (x86_64)
  - ARM

### Installation Steps

1. Visit the [releases page](https://github.com/simulot/immich-go/releases/latest)
2. Download the archive for your operating system and architecture:
   - Windows: `immich-go_Windows_amd64.zip`
   - MacOS: `immich-go_Darwin_amd64.tar.gz`
   - Linux: `immich-go_Linux_amd64.tar.gz`
   - FreeBSD: `immich-go_Freebsd_amd64.tar.gz`
   - and more...

3. Extract the archive:
   ```bash
   # For Linux/MacOS/FreeBSD
   tar -xzf immich-go_*_amd64.tar.gz

   # For Windows
   # Use your preferred zip tool to extract the archive
   ```

4. (Optional) Move the binary to a directory in your PATH:
   ```bash
   # Linux/MacOS/FreeBSD
   sudo mv immich-go /usr/local/bin/

   # Windows
   # Move immich-go.exe to a directory in your PATH
   ```

## Building from Source
If pre-built binaries are not available, you can build Immich-Go from source.

### Prerequisites
- Go 1.23 or higher
- Git

### Build Steps
```bash
# Clone the repository
git clone https://github.com/simulot/immich-go.git

# Change to the project directory
cd immich-go

# Build the binary
go build

# (Optional) Install to GOPATH/bin
go install
```

### Building in Termux
The prebuilt `Linux_Arm64` binaries won't work in Termux for Android, so you have to build it yourself.
You can follow the same build steps as above, but install `git` and `golang` via `pkg`.

If you want to use `go install`, make sure to add `GOPATH/bin` to your `PATH`:
```bash
# Create or open .bashrc
nano  ~/.bashrc

# Add the following line in order to include GOPATH/bin in your $PATH
export PATH=$PATH:$(go env GOPATH)/bin

# Save and exit with Ctrl+X, then Y and Enter

# Restart your session or apply changes to your current session with:
source ~/.bashrc

```

## Installation with Nix

`immich-go` is packaged with [nix](https://nixos.org/) and distributed via [nixpkgs](https://search.nixos.org/packages?channel=unstable&type=packages&query=immich-go).
You can try `immich-go` without installing it with the following commands:

```bash
nix-shell -I "nixpkgs=https://github.com/NixOS/nixpkgs/archive/nixos-unstable-small.tar.gz" -p immich-go
# Or with flakes enabled
nix run "github:nixos/nixpkgs?ref=nixos-unstable-small#immich-go" -- --help
```

Or you can add `immich-go` to your `configuration.nix` in the `environment.systemPackages` section.

## Verifying the Installation

After installation, verify that immich-go is working correctly:

```bash
immich-go --version
```

This should display the version number of immich-go.
# Running Immich-Go

Immich-Go is a command-line tool. You need to run it from a terminal or command prompt.

## Commands and Sub-Commands Logic

The general syntax for running Immich-Go is:

```bash
immich-go command sub-command options path/to/files
```

Commands must be combined with sub-commands and options to perform the required action.
* immich-go
  * [upload](#the-upload-command)
    * from-folder
    * from-google-photos
    * from-icloud
    * from-picasa
    * from-immich
  * [archive](#the-archive-command)
    * from-folder
    * from-google-photos
    * from-icloud
    * from-picasa
    * from-immich
  * [stack](#the-stack-command)
  * version

Examples:
```bash
## Upload photos from a local folder to your Immich server
immich-go upload from-folder --server=http://your-ip:2283 --api-key=your-api-key /path/to/your/photos

## Archive photos from your Immich server to a local folder
immich-go archive from-immich --from-server=http://your-ip:2283 --from-api-key=your-api-key --write-to-folder=/path/to/archive

## Upload a Google Photos takeout to your Immich server
immich-go upload from-google-photos --server=http://your-ip:2283 --api-key=your-api-key /path/to/your/takeout-*.zip
```

> **Note:** Depending on your OS, you may need to invoke the program differently when Immich-Go is in the current directory:
> - Linux, macOS, FreeBSD: `./immich-go`
> - Windows: `.\immich-go`

### Global Options
The following options are shared by all commands:

| **Parameter**  | **Description**                                       |
| -------------- | ----------------------------------------------------- |
| -h, --help     | Help for Immich-Go                                    |
| -l, --log-file | Write log messages to a file                          |
| --log-level    | Log level (DEBUG\|INFO\|WARN\|ERROR) (default "INFO") |
| --log-type     | Log format (TEXT\|JSON) (default "TEXT")              |
| -v, --version  | Display current version of Immich-Go                  |

**The default path for the log files depend on your system:**

| **OS**  | **Path**                                                           |
| ------- | ------------------------------------------------------------------ |
| Linux   | `$HOME/.cache/immich-go/immich-go_YYYY-MM-DD_HH-MI-SS.log`         |
| Windows | `%LocalAppData%\immich-go\immich-go_YYYY-MM-DD_HH-MI-SS.log`       |
| MacOS   | `$HOME/Library/Caches/immich-go/immich-go_YYYY-MM-DD_HH-MI-SS.log` |

### Environment variables

| **Variable**     | **Description**                                                                                 |
| ---------------- | ----------------------------------------------------------------------------------------------- |
| IMMICHGO_TEMPDIR | Temporary directory used by Immich-go. Default: User's cache folder, or OS temporary directory. |


# The **upload** command:
The **upload** command loads photos and videos from the source designated by the sub-command to the Immich server.
**Upload** accepts three sub-commands:
  * [from-folder](#from-folder-sub-command) to upload photos from a local folder or a zipped archive
  * [from-google-photos](#from-google-photos-sub-command) to upload photos from a Google Photos takeout archive
  * [from-icloud](#from-icloud-sub-command) to create a folder archive from an iCloud archive TODO
  * [from-picasa](#from-picasa-sub-command)  to create a folder archive from a Picasa archive
  * [from-immich](#from-immich-sub-command) to upload photos from an Immich server to another Immich server

Examples:
```bash
immich-go upload from-folder --server=http://your-ip:2283 --api-key=your-api-key /path/to/your/photos
immich-go upload from-google-photos --server=http://your-ip:2283 --api-key=your-api-key /path/to/your/takeout-*.zip
```


The **upload** command need the following options to manage the connection with the Immich server:


| **Parameter**        | **Default value** | **Description**                                                                                                                           |
| -------------------- | :---------------: | ----------------------------------------------------------------------------------------------------------------------------------------- |
| -s, --server         |                   | Immich server address (e.g http://your-ip:2283 or https://your-domain) (**MANDATORY**)                                                    |
| -k, --api-key        |                   | API Key (**MANDATORY**)                                                                                                                   |
| --admin-api-key      |                   | The Immichs admin's API key, used to pause and resume the server's jobs during operations (**MANDATORY** when uploading for a non-admin ) |
| --no-ui              |      `FALSE`      | Disable the user interface                                                                                                                |
| --api-trace          |      `FALSE`      | Enable trace of api calls                                                                                                                 |
| --client-timeout     |       `20m`       | Set server calls timeout                                                                                                                  |
| --device-uuid string |   `$LOCALHOST`    | Set a device UUID                                                                                                                         |
| --dry-run            |                   | Simulate all server actions                                                                                                               |
| --skip-verify-ssl    |      `FALSE`      | Skip SSL verification                                                                                                                     |
| --time-zone          |                   | Override the system time zone (example: Europe/Paris)                                                                                     |
| --session-tag        |      `FALSE`      | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS"                                                                          |
| --tag strings        |                   | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1')        |
| --on-server-errors   |      `stop`       | Action to take on server errors, (stop,continue,\<n\> to stop after n errors)                                                             |
| --pause-immich-jobs  |      `TRUE`       | Pause Immich server jobs during the upload process                                                                                        |


## **--client-timeout**
Increase the **--client-timeout** when you have some timeout issues with the server, especialy when uploading large files.

## **--session-tag**
Thanks to the **--session-tag** option, it's easy to identify all photos uploaded during a session, and remove them if needed.
This tag is formatted as `{immich-go}/YYYY-MM-DD HH-MM-SS`. The tag can be deleted without removing the photos.

## **--overwrite**
The `--overwrite` flag ensures that files on the server are always replaced with their local versions during the upload process. If a file does not exist on the server, it will be uploaded as a new file. This option is useful for ensuring that the server always has the latest version of your files.

Example:
```bash
immich-go upload from-folder --server=http://your-ip:2283 --api-key=your-api-key --overwrite /path/to/your/photos
```

# The **archive** command:

The **archive** command writes the content taken from the source given by the sub-command to a folder tree.
The destination folder isn't wiped out before the operation, so it's possible to add new photos to an existing archive.

The command accepts three sub-commands:
  * [from-folder](#from-folder-sub-command) to create a folder archive from a local folder or a zipped archive
  * [from-google-photos](#from-google-photos-sub-command) to create a folder archive from a Google Photos takeout archive
  * [from-icloud](#from-icloud-sub-command) to create a folder archive from an iCloud archive TODO
  * [from-picasa](#from-picasa-sub-command)  to create a folder archive from a Picasa archive
  * [from-immich](#from-immich-sub-command) to create a folder archive from an Immich server

All photos and videos are sorted by date of capture, following this schema: `Folder/YYYY/YYYY-MM/photo.ext`.

Here is an example of what your folder structure might look like:

```
Folder/
├── 2022/
│   ├── 2022-01/
│   │   ├── photo01.jpg
│   │   └── photo01.jpg.JSON
│   ├── 2022-02/
│   │   ├── photo02.jpg
│   │   └── photo02.jpg.JSON
│   └── ...
├── 2023/
│   ├── 2023-03/
│   │   ├── photo03.jpg
│   │   └── photo03.jpg.JSON
│   ├── 2023-04/
│   │   ├── photo04.jpg
│   │   └── photo04.jpg.JSON
│   └── ...
├── 2024/
│   ├── 2024-05/
│   │   ├── photo05.jpg
│   │   └── photo05.jpg.JSON
│   ├── 2024-06/
│   │   ├── photo06.jpg
│   │   └── photo06.jpg.JSON
│   └── ...
```

This structure ensures that photos are neatly organized by year and month within the specified folder, making it easy to locate and manage them.
This folder tree is ready to be archived or migrated to another server.

The general syntax is:

```bash
immich-go archive from-sub-command --write-to-folder=folder options
```

# **from-folder** sub command:

The **from-folder** sub-command processes a folder tree to upload photos to the Immich server.

| **Parameter**           |           **Default value**           | **Description**                                                                                                                                                                        |
| ----------------------- | :-----------------------------------: | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| --ban-file              | [See banned files](#banned-file-list) | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.                                                                                                 |
| --date-from-name        |                `TRUE`                 | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov).                                               |
| --date-range            |                                       | Only import photos taken within the specified date range. [See date range possibilities](#date-range)                                                                                  |
| --exclude-extensions    |                                       | Comma-separated list of extension to exclude. (e.g. .gif,.PM)                                                                                                                          |
| --folder-as-album       |                `NONE`                 | Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name          |
| --folder-as-tags        |                `FALSE`                | Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024)                                                                   |
| --album-path-joiner     |                `" / "`                | Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ')                                                                                    |
| --album-picasa          |                `FALSE`                | Use the Picasa album name found in `.picasa.ini` files                                                                                                                                 |
| --ignore-sidecar-files  |                `FALSE`                | Don't upload sidecar with the photo.                                                                                                                                                   |
| --include-extensions    |                 `all`                 | Comma-separated list of extension to include. (e.g. .jpg,.heic)                                                                                                                        |
| --include-type          |                 `all`                 | Single file type to include. (`VIDEO` or `IMAGE`)                                                                                                                                      |
| --into-album            |                                       | Specify an album to import all files into                                                                                                                                              |
| --manage-burst          |                                       | Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG.  [See option's details](#burst-detection-and-management)                                            |
| --manage-epson-fastfoto |                `FALSE`                | Manage Epson FastFoto file                                                                                                                                                             |
| --manage-heic-jpeg      |                                       | Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG.     [See option's details](#management-of-coupled-heic-and-jpeg-files) |
| --manage-raw-jpeg       |                                       | Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG. [See options's details](#management-of-coupled-raw-and-jpeg-files)        |
| --recursive             |                `TRUE`                 | Explore the folder and all its sub-folders                                                                                                                                             |
| --session-tag           |                                       | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS"                                                                                                                       |
| --tag                   |                                       | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1')                                                     |


## Date of capture

The Immich server takes the date of capture from the metadata of the photo, or in the XMP sidecar file if present.
However, some photos may not have this information.  In this case, Immich-go can infer the date of capture from the filename.

The option `--date-from-name` instructs Immich-go to extract the date of capture from the filename if the date isn't available in the metadata.

Immich-go can extract the date of capture for the following formats: .heic, .heif, .jpg, .jpeg, .dng, .cr2, .mp4, .mov, .cr3..

> Note: `--date-from-name` slows down the process because immich-go needs to parse files to check if the capture date is present in the file.


# **From-google-photos** sub command:


The **from-google-photos** sub-command processes a Google Photos takeout archive to upload photos to the Immich server.

| **Parameter**             |           **Default value**           | **Description**                                                                                                                                                                    |
| ------------------------- | :-----------------------------------: | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| --ban-file FileList       | [See banned files](#banned-file-list) | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.                                                                                             |
| --date-range              |                                       | Only import photos taken within the specified date range [See date range possibilities](#date-range)                                                                               |
| --exclude-extensions      |                                       | Comma-separated list of extension to exclude. (e.g. .gif, .PM)                                                                                                                     |
| --from-album-name string  |                                       | Only import photos from the specified Google Photos album                                                                                                                          |
| -a, --include-archived    |                `TRUE`                 | Import archived Google Photos                                                                                                                                                      |
| --include-extensions      |                 `all`                 | Comma-separated list of extension to include. (e.g. .jpg, .heic)                                                                                                                   |
| --include-type            |                 `all`                 | Single file type to include. (`VIDEO` or `IMAGE`)                                                                                                                                  |
| -p, --include-partner     |                `TRUE`                 | Import photos from your partner's Google Photos account                                                                                                                            |
| -t, --include-trashed     |                `FALSE`                | Import photos that are marked as trashed in Google Photos                                                                                                                          |
| -u, --include-unmatched   |                `FALSE`                | Import photos that do not have a matching JSON file in the takeout                                                                                                                 |
| --include-untitled-albums |                `FALSE`                | Include photos from albums without a title in the import process                                                                                                                   |
| --manage-burst            |                                       | Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG. [See option's details](#burst-detection-and-management)                                         |
| --manage-epson-fastfoto   |                `FALSE`                | Manage Epson FastFoto file (default: false)                                                                                                                                        |
| --manage-heic-jpeg        |                                       | Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG. [See option's details](#management-of-coupled-heic-and-jpeg-files) |
| --manage-raw-jpeg         |                                       | Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG. [See options's details](#management-of-coupled-raw-and-jpeg-files)    |
| --partner-shared-album    |                                       | Add partner's photo to the specified album name                                                                                                                                    |
| --session-tag             |                `FALSE`                | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS"                                                                                                                   |
| --sync-albums             |                `TRUE`                 | Automatically create albums in Immich that match the albums in your Google Photos takeout                                                                                          |
| --tag strings             |                                       | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1')                                                 |
| --takeout-tag             |                `TRUE`                 | Tag uploaded photos with a tag "{takeout}/takeout-YYYYMMDDTHHMMSSZ"                                                                                                                |
| --people-tag              |                `TRUE`                 | Tag uploaded photos with tags \"people/name\" found in the JSON file                                                                                                               |

## Google Photos Best Practices:

* **Taking Out Your Photos:**
  * Choose the ZIP format when creating your takeout for easier import.
  * Select the largest file size available (50GB) to minimize the number of archive parts.
  * Download all parts to your computer.

* **Importing Your Photos:**
  * If your takeout is in ZIP format, you can import it directly without needing to unzip the files first.
  * It's important to import all parts of the takeout together, as some data might be spread across multiple files. Use `/path/to/your/files/takeout-*.zip` as the file name.
  * For **.tgz** files (compressed tar archives), you'll need to decompress all the files into a single folder before importing. Then use the command `immich-go upload from-google-photos /path/to/your/files`.  
  * You can remove any unwanted files or folders from your takeout before importing.
  * Restarting an interrupted import won't cause any problems and will resume where it left off.

* **What if many of my files are not imported?**
  * Verify if all takeout parts have been included in the processing. Have you used the `takeout-*.zip` file name pattern?
  * Sometimes, the takeout result is incomplete. Request another takeout, either for an entire year or in smaller increments.
  * Force the import of files despite missing JSON files using the option `--include-unmatched`.

## Takeout Tag

Immich-Go can tag all imported photos with a takeout tag. The tag is formatted as `{takeout}/takeout-YYYYMMDDTHHMMSSZ`. This tag can be used to identify all photos imported from a Google Photos takeout, making it easy to remove them if needed.


# **from-icloud** sub command:

The **from-icloud** sub-command processes an ICloud takeout.

| **Parameter**        |           **Default value**           | **Description**                                                                                                                                                                        |
| -------------------- | :-----------------------------------: | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| --memories           |                `FALSE`                | Import icloud memories as albums                                                                                                                                                       |
| --ban-file           | [See banned files](#banned-file-list) | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.                                                                                                 |
| --date-from-name     |                `TRUE`                 | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov).                                               |
| --date-range         |                                       | Only import photos taken within the specified date range. [See date range possibilities](#date-range)                                                                                  |
| --exclude-extensions |                                       | Comma-separated list of extension to exclude. (e.g. .gif,.PM)                                                                                                                          |
| --include-extensions |                 `all`                 | Comma-separated list of extension to include. (e.g. .jpg,.heic)                                                                                                                        |
| --include-type       |                 `all`                 | Single file type to include. (`VIDEO` or `IMAGE`)                                                                                                                                      |
| --into-album         |                                       | Specify an album to import all files into                                                                                                                                              |
| --manage-burst       |                                       | Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG.  [See option's details](#burst-detection-and-management)                                            |
| --manage-heic-jpeg   |                                       | Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG.     [See option's details](#management-of-coupled-heic-and-jpeg-files) |
| --manage-raw-jpeg    |                                       | Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG. [See options's details](#management-of-coupled-raw-and-jpeg-files)        |
| --session-tag        |                                       | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS"                                                                                                                       |
| --tag                |                                       | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1')                                                     |



# **from-picasa** sub command:

The **from-picasa** sub-command processes picasa file tree to upload photos to the Immich server.

| **Parameter**           |           **Default value**           | **Description**                                                                                                                                                                        |
| ----------------------- | :-----------------------------------: | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| --ban-file              | [See banned files](#banned-file-list) | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.                                                                                                 |
| --date-from-name        |                `TRUE`                 | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov).                                               |
| --date-range            |                                       | Only import photos taken within the specified date range. [See date range possibilities](#date-range)                                                                                  |
| --exclude-extensions    |                                       | Comma-separated list of extension to exclude. (e.g. .gif,.PM)                                                                                                                          |
| --folder-as-album       |                `NONE`                 | Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name          |
| --folder-as-tags        |                `FALSE`                | Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024)                                                                   |
| --album-path-joiner     |                `" / "`                | Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ')                                                                                    |
| --include-extensions    |                 `all`                 | Comma-separated list of extension to include. (e.g. .jpg,.heic)                                                                                                                        |
| --include-type          |                 `all`                 | Single file type to include. (`VIDEO` or `IMAGE`)                                                                                                                                      |
| --into-album            |                                       | Specify an album to import all files into                                                                                                                                              |
| --manage-burst          |                                       | Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG.  [See option's details](#burst-detection-and-management)                                            |
| --manage-epson-fastfoto |                `FALSE`                | Manage Epson FastFoto file                                                                                                                                                             |
| --manage-heic-jpeg      |                                       | Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG.     [See option's details](#management-of-coupled-heic-and-jpeg-files) |
| --manage-raw-jpeg       |                                       | Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG. [See options's details](#management-of-coupled-raw-and-jpeg-files)        |
| --recursive             |                `TRUE`                 | Explore the folder and all its sub-folders                                                                                                                                             |
| --session-tag           |                                       | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS"                                                                                                                       |
| --tag                   |                                       | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1')                                                     |


# Options details

## Burst Detection and Management

The system detects burst photos in the following cases:

| Case                | Description                                                                                                                                                          |
| ------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Series of photos    | When the time difference between two photos is less than 900 ms                                                                                                      |
| Huawei smartphones  | Based on file names: <br>- IMG_20231014_183246_BURST001_COVER.jpg<br>- IMG_20231014_183246_BURST002.jpg<br>- IMG_20231014_183246_BURST003.jpg                        |
| Nexus smartphones   | Based on file names:<br>- 00001IMG_00001_BURST20171111030039.jpg<br>-...<br>-00014IMG_00014_BURST20171111030039.jpg<br>-00015IMG_00015_BURST20171111030039_COVER.jpg |
| Pixel smartphones   | Based on file names:<br>- PXL_20230330_184138390.MOTION-01.COVER.jpg<br>- PXL_20230330_184138390.MOTION-02.ORIGINAL.jpg                                              |
| Samsung smartphones | Based on file names:<br>- 20231207_101605_001.jpg<br>- 20231207_101605_002.jpg<br>- 20231207_101605_xxx.jpg                                                          |
| Sony Xperia         | Based on file names:<br>- DSC_0001_BURST20230709220904977.JPG<br>- ...<br>- DSC_0035_BURST20230709220904977_COVER.JPG                                                |
| Nothing Phones      | Based on file names:<br>- 00001IMG_00001_BURST1723801037429_COVER.jpg<br>- 00002IMG_00002_BURST1723801037429.jpg<br>- ...<br>                                        |

The option `--manage-burst` instructs Immich-Go on how to manage burst photos. The following options are available:

| Option          | Description                                                                                                                                      |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| `NoStack`       | Do not stack burst photos.                                                                                                                       |
| `Stack`         | Stack all burst photos together. When the cover photo can't be identified with the file name, the first photo of the burst is used as the cover. |
| `StackKeepRaw`  | Stack all burst photos together. Keep only the RAW photos.                                                                                       |
| `StackKeepJPEG` | Stack all burst photos together. Keep only the JPEG photos.                                                                                      |

## Management of Coupled HEIC and JPEG Files

The option `--manage-heic-jpeg` instructs Immich-Go on how to manage coupled HEIC and JPEG files. The following options are available:

| Option           | Description                                                         |
| ---------------- | ------------------------------------------------------------------- |
| `NoStack`        | Do not stack HEIC and JPEG files.                                   |
| `KeepHeic`       | Keep only the HEIC file.                                            |
| `KeepJPG`        | Keep only the JPEG file.                                            |
| `StackCoverHeic` | Stack the HEIC and JPEG files together. The HEIC file is the cover. |
| `StackCoverJPG`  | Stack the HEIC and JPEG files together. The JPEG file is the cover. |

## Management of Coupled RAW and JPEG Files

The option `--manage-raw-jpeg` instructs Immich-Go on how to manage coupled RAW and JPEG files. The following options are available:

| Option          | Description                                                        |
| --------------- | ------------------------------------------------------------------ |
| `NoStack`       | Do not stack RAW and JPEG files.                                   |
| `KeepRaw`       | Keep only the RAW file.                                            |
| `KeepJPG`       | Keep only the JPEG file.                                           |
| `StackCoverRaw` | Stack the RAW and JPEG files together. The RAW file is the cover.  |
| `StackCoverJPG` | Stack the RAW and JPEG files together. The JPEG file is the cover. |

## Management of Epson FastFoto Scanned Photos

This device outputs three files for each scanned photo: the original scan, a "corrected" scan, and the backside of the photo if it has writing on it. The structure looks like this:
- specified-image-name.jpg (Original)
- specified-image-name_a.jpg (Corrected)
- specified-image-name_b.jpg (Back of Photo)

The option `--manage-epson-fastfoto=TRUE` instructs Immich-Go to stack related photos, with the corrected scan as the cover.


# **from-immich** sub-command:

The sub-command **from-immich** processes an Immich server to upload photos to another Immich server.

| **Parameter**                  | **Default value** | **Description**                                                                      |
| ------------------------------ | :---------------: | ------------------------------------------------------------------------------------ |
| --exclude-extensions           |                   | Comma-separated list of extension to exclude. (e.g. .gif,.PM)                        |
| --from-server                  |                   | Immich server address (e.g http://your-ip:2283 or https://your-domain)               |
| --from-api-key string          |                   | Immich API Key                                                                       |
| --from-album                   |                   | Get assets only from those albums, can be used multiple times                        |
| --from-api-trace               |      `FALSE`      | Enable trace of api calls                                                            |
| --from-client-timeout duration |      `5m0s`       | Set server calls timeout                                                             |
| --from-date-range              |                   | Get assets only within this date range.  [See date range possibilities](#date-range) |
| --from-skip-verify-ssl         |      `FALSE`      | Skip SSL verification                                                                |
| --include-extensions           |       `all`       | Comma-separated list of extension to include. (e.g. .jpg, .heic)                     |
| --include-type                 |       `all`       | Single file type to include. (`VIDEO` or `IMAGE`)                                    |


# The **stack** command:
The stack command open the immich server, for the user associated with the the API-KEY, and stacks related photos together.

The command accepts the following options:

| **Parameter**           | **Default value** | **Description**                                                                                                                                                                     |
| ----------------------- | :---------------: | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| -s, --server            |                   | Immich server address (e.g http://your-ip:2283 or https://your-domain) (**MANDATORY**)                                                                                              |
| -k, --api-key           |                   | API Key (**MANDATORY**)                                                                                                                                                             |
| --api-trace             |      `FALSE`      | Enable trace of api calls                                                                                                                                                           |
| --client-timeout        |      `5m0s`       | Set server calls timeout                                                                                                                                                            |
| --dry-run               |                   | Simulate all server actions...                                                                                                                                                      |
| --skip-verify-ssl       |      `FALSE`      | Skip SSL verification                                                                                                                                                               |
| --time-zone             |                   | Override the system time zone (example: Europe/Paris)                                                                                                                               |
| --manage-burst          |                   | Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG. [See option's details](#burst-detection-and-management)                                          |
| --manage-epson-fastfoto |      `FALSE`      | Manage Epson FastFoto file                                                                                                                                                          |
| --manage-heic-jpeg      |                   | Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG   [See option's details](#management-of-coupled-heic-and-jpeg-files) |
| --manage-raw-jpeg       |                   | Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG. [See options's details](#management-of-coupled-raw-and-jpeg-files)     |


# Additional information and best practices

## **XMP** files process

**XMP** files found in source folder are passed to Immich server without any modification. Immich uses them to collect photo's date of capture, tags, description and GPS location.

## Google photos **JSON** files process

Google photos **JSON** files found in source folders are opened by Immich-go to get the album belonging, the date of capture, the GPS location, the favorite status, the partner status, the archive status and the trashed status. This information is used to trigger Immich features.

## Folder archive **JSON** files process

Those files are generated by the **archive** command. Their are used to restore Immich features like album, date of capture, GPS location, rating, tags and archive status.

```json
{
  "fileName": "example.jpg",
  "latitude": 37.7749,
  "longitude": -122.4194,
  "dateTaken": "2023-10-01T12:34:56Z",
  "description": "A beautiful view of the Golden Gate Bridge.",
  "albums": [
    {
      "title": "San Francisco Trip",
      "description": "Photos from my trip to San Francisco",
    }
  ],
  "tags": [
    {
      "value": "USA/California/San Francisco"
    },

  ],
  "rating": 5,
  "trashed": false,
  "archived": false,
  "favorited": true,
  "fromPartner": false
}
```

## Session tags
Immich-go can tag all imported photos with a session tag. The tag is formatted as `{immich-go}/YYYY-MM-DD HH-MM-SS`. This tag can be used to identify all photos imported during a session. This it easy to remove them if needed.


## Banned file list
The following files are excluded automatically:
- `@eaDir/`
- `@__thumb/`
- `SYNOFILE_THUMB_*.*`
- `Lightroom Catalog/`
- `thumbnails/`
- `.DS_Store/`
- `._*.*`
- `.photostructure/`

## Date range

The `--date-range` option allows you to process photos taken within a specific date range. The following date range formats are supported:

| **Parameter**                        | **Description**                                          |
| ------------------------------------ | -------------------------------------------------------- |
| `--date-range=YYYY-MM-DD`            | Import photos taken on a specific day.                   |
| `--date-range=YYYY-MM`               | Import photos taken during a specific month of the year. |
| `--date-range=YYYY`                  | Import photos taken during a particular year.            |
| `--date-range=YYYY-MM-DD,YYYY-MM-DD` | Import photos taken between a specific date range        |



# Examples

#### Importing a Google Takeout with Stacking JPEG and RAW

To import a Google Photos takeout and stack JPEG and RAW files together, with the RAW file as the cover, use the following command:

```bash
immich-go upload from-google-photos --server=http://your-ip:2283 --api-key=your-api-key --manage-raw-jpeg=StackCoverRaw /path/to/your/takeout-*.zip
```

#### Uploading Photos from a Local Folder

To upload photos from a local folder to your Immich server, use the following command:

```bash
immich-go upload from-folder --server=http://your-ip:2283 --api-key=your-api-key /path/to/your/photos
```

#### Archiving Photos from Immich Server

To archive photos from your Immich server to a local folder, use the following command:

```bash
immich-go archive from-immich --server=http://your-ip:2283 --api-key=your-api-key --write-to-folder=/path/to/archive
```

#### Transferring Photos Between Immich Servers

To transfer photos from one Immich server to another, use the following command:

```bash
immich-go upload from-immich --from-server=http://source-ip:2283 --from-api-key=source-api-key --server=http://destination-ip:2283 --api-key=destination-api-key
```

#### Importing Photos with Specific Date Range

To import photos taken within a specific date range from a local folder, use the following command:

```bash
immich-go upload from-folder --server=http://your-ip:2283 --api-key=your-api-key --date-range=2022-01-01,2022-12-31 /path/to/your/photos
```


# Acknowledgments

Kudos to the Immich team for their stunning project! 🤩

This program uses the following 3rd party libraries:
- [https://github.com/rivo/tview](https://github.com/rivo/tview) Terminal User Interface

A big thank you to the project contributors:
- [rodneyosodo](https://github.com/rodneyosodo) GitHub CI, Go linter, and advice
- [sigmahour](https://github.com/sigmahour) SSL management
- [mrwulf](https://github.com/mrwulf) Partner sharing album
- [erkexzcx](https://github.com/erkexzcx) Date determination based on file path and file name
- [benjamonnguyen](https://github.com/benjamonnguyen) Tag API calls
