# Environment Variables

The following environment variables can be used to configure `immich-go`.

## Global

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| `IMMICH_GO_CONCURRENT_TASKS` | `--concurrent-tasks` | `12` | Number of concurrent tasks (1-20) |
| `IMMICH_GO_DRY_RUN` | `--dry-run` | `false` | dry run |
| `IMMICH_GO_LOG_FILE` | `--log-file` |  | Write log messages into the file |
| `IMMICH_GO_LOG_LEVEL` | `--log-level` | `INFO` | Log level (DEBUG|INFO|WARN|ERROR), default INFO |
| `IMMICH_GO_LOG_TYPE` | `--log-type` | `text` | Log formatted  as text of JSON file |
| `IMMICH_GO_ON_ERRORS` | `--on-errors` | `stop` | What to do when an error occurs (stop, continue, accept N errors at max) |
| `IMMICH_GO_SAVE_CONFIG` | `--save-config` | `false` | Save the configuration to immich-go.yaml |

## archive from-folder

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_ALBUM_PATH_JOINER` | `--album-path-joiner` | ` / ` | Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ') |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_BAN_FILE` | `--ban-file` | `'@eaDir/', '@__thumb/', 'SYNOFILE_THUMB_*.*', 'Lightroom Catalog/', 'thumbnails/', '.DS_Store/', '/._*', '.photostructure/', 'Recently Deleted/'` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_DATE_FROM_NAME` | `--date-from-name` | `true` | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_DATE_RANGE` | `--date-range` | `unset` | Only import photos taken within the specified date range |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_EXCLUDE_EXTENSIONS` | `--exclude-extensions` |  | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_FOLDER_AS_ALBUM` | `--folder-as-album` | `none` | Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_FOLDER_AS_TAGS` | `--folder-as-tags` | `false` | Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_IGNORE_SIDECAR_FILES` | `--ignore-sidecar-files` | `false` | Don't upload sidecar with the photo. |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_INCLUDE_EXTENSIONS` | `--include-extensions` |  | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_INCLUDE_TYPE` | `--include-type` |  | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_INTO_ALBUM` | `--into-album` |  | Specify an album to import all files into |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_RECURSIVE` | `--recursive` | `true` | Explore the folder and all its sub-folders |

## archive from-google-photos

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_BAN_FILE` | `--ban-file` | `'@eaDir/', '@__thumb/', 'SYNOFILE_THUMB_*.*', 'Lightroom Catalog/', 'thumbnails/', '.DS_Store/', '/._*', '.photostructure/', 'Recently Deleted/'` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_DATE_RANGE` | `--date-range` | `unset` | Only import photos taken within the specified date range |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_EXCLUDE_EXTENSIONS` | `--exclude-extensions` |  | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_FROM_ALBUM_NAME` | `--from-album-name` |  | Only import photos from the specified Google Photos album |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_ARCHIVED` | `--include-archived` | `true` | Import archived Google Photos |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_EXTENSIONS` | `--include-extensions` |  | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_PARTNER` | `--include-partner` | `true` | Import photos from your partner's Google Photos account |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_TRASHED` | `--include-trashed` | `false` | Import photos that are marked as trashed in Google Photos |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_TYPE` | `--include-type` |  | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_UNMATCHED` | `--include-unmatched` | `false` | Import photos that do not have a matching JSON file in the takeout |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_UNTITLED_ALBUMS` | `--include-untitled-albums` | `false` | Include photos from albums without a title in the import process |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_PARTNER_SHARED_ALBUM` | `--partner-shared-album` |  | Add partner's photo to the specified album name |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_PEOPLE_TAG` | `--people-tag` | `true` | Tag uploaded photos with tags "people/name" found in the JSON file |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_SYNC_ALBUMS` | `--sync-albums` | `true` | Automatically create albums in Immich that match the albums in your Google Photos takeout |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_TAKEOUT_TAG` | `--takeout-tag` | `true` | Tag uploaded photos with a tag "{takeout}/takeout-YYYYMMDDTHHMMSSZ" |

## archive from-icloud

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| `IMMICH_GO_ARCHIVE_FROM_ICLOUD_ALBUM_PATH_JOINER` | `--album-path-joiner` | ` / ` | Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ') |
| `IMMICH_GO_ARCHIVE_FROM_ICLOUD_BAN_FILE` | `--ban-file` | `'@eaDir/', '@__thumb/', 'SYNOFILE_THUMB_*.*', 'Lightroom Catalog/', 'thumbnails/', '.DS_Store/', '/._*', '.photostructure/', 'Recently Deleted/'` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_ARCHIVE_FROM_ICLOUD_DATE_FROM_NAME` | `--date-from-name` | `true` | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov) |
| `IMMICH_GO_ARCHIVE_FROM_ICLOUD_DATE_RANGE` | `--date-range` | `unset` | Only import photos taken within the specified date range |
| `IMMICH_GO_ARCHIVE_FROM_ICLOUD_EXCLUDE_EXTENSIONS` | `--exclude-extensions` |  | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_ARCHIVE_FROM_ICLOUD_FOLDER_AS_ALBUM` | `--folder-as-album` | `none` | Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name |
| `IMMICH_GO_ARCHIVE_FROM_ICLOUD_FOLDER_AS_TAGS` | `--folder-as-tags` | `false` | Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024) |
| `IMMICH_GO_ARCHIVE_FROM_ICLOUD_IGNORE_SIDECAR_FILES` | `--ignore-sidecar-files` | `false` | Don't upload sidecar with the photo. |
| `IMMICH_GO_ARCHIVE_FROM_ICLOUD_INCLUDE_EXTENSIONS` | `--include-extensions` |  | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_ICLOUD_INCLUDE_TYPE` | `--include-type` |  | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_ICLOUD_INTO_ALBUM` | `--into-album` |  | Specify an album to import all files into |
| `IMMICH_GO_ARCHIVE_FROM_ICLOUD_MEMORIES` | `--memories` | `false` | Import icloud memories as albums |
| `IMMICH_GO_ARCHIVE_FROM_ICLOUD_RECURSIVE` | `--recursive` | `true` | Explore the folder and all its sub-folders |

## archive from-immich

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_ADMIN_API_KEY` | `--from-admin-api-key` |  | Admin's API Key for managing server's jobs |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_ALBUMS` | `--from-albums` | `[]` | Get assets only from those albums, can be used multiple times |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_API_KEY` | `--from-api-key` |  | API Key |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_API_TRACE` | `--from-api-trace` | `false` | Enable trace of api calls |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_ARCHIVED` | `--from-archived` | `false` | Get only archived assets |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_CITY` | `--from-city` |  | Get only assets from this city |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_CLIENT_TIMEOUT` | `--from-client-timeout` | `20m0s` | Set server calls timeout |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_COUNTRY` | `--from-country` |  | Get only assets from this country |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_DATE_RANGE` | `--from-date-range` | `unset` | Only import photos taken within the specified date range |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_DEVICE_UUID` | `--from-device-uuid` | `gl65` | Set a device UUID |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_DRY_RUN` | `--from-dry-run` | `false` | Simulate all actions |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_EXCLUDE_EXTENSIONS` | `--from-exclude-extensions` |  | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_FAVORITE` | `--from-favorite` | `false` | Get only favorite assets |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_INCLUDE_EXTENSIONS` | `--from-include-extensions` |  | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_INCLUDE_TYPE` | `--from-include-type` |  | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_MAKE` | `--from-make` |  | Get only assets with this make |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_MINIMAL_RATING` | `--from-minimal-rating` | `0` | Get only assets with a rating greater or equal to this value |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_MODEL` | `--from-model` |  | Get only assets with this model |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_NO_ALBUM` | `--from-no-album` | `false` | Get only assets that are not in any album |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_PARTNERS` | `--from-partners` | `false` | Get partner's assets as well |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_PAUSE_IMMICH_JOBS` | `--from-pause-immich-jobs` | `true` | Pause Immich background jobs during upload operations |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_PEOPLE` | `--from-people` | `[]` | Get assets only with those people, can be used multiple times |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_SERVER` | `--from-server` |  | Immich server address (example http://your-ip:2283 or https://your-domain) |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_SKIP_VERIFY_SSL` | `--from-skip-verify-ssl` | `false` | Skip SSL verification |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_STATE` | `--from-state` |  | Get only assets from this state |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_TAGS` | `--from-tags` | `[]` | Get assets only with those tags, can be used multiple times |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_TIME_ZONE` | `--from-time-zone` |  | Override the system time zone |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_TRASH` | `--from-trash` | `false` | Get only trashed assets |

## archive from-picasa

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| `IMMICH_GO_ARCHIVE_FROM_PICASA_ALBUM_PATH_JOINER` | `--album-path-joiner` | ` / ` | Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ') |
| `IMMICH_GO_ARCHIVE_FROM_PICASA_ALBUM_PICASA` | `--album-picasa` | `true` | Use Picasa album name found in .picasa.ini file |
| `IMMICH_GO_ARCHIVE_FROM_PICASA_BAN_FILE` | `--ban-file` | `'@eaDir/', '@__thumb/', 'SYNOFILE_THUMB_*.*', 'Lightroom Catalog/', 'thumbnails/', '.DS_Store/', '/._*', '.photostructure/', 'Recently Deleted/'` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_ARCHIVE_FROM_PICASA_DATE_FROM_NAME` | `--date-from-name` | `true` | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov) |
| `IMMICH_GO_ARCHIVE_FROM_PICASA_DATE_RANGE` | `--date-range` | `unset` | Only import photos taken within the specified date range |
| `IMMICH_GO_ARCHIVE_FROM_PICASA_EXCLUDE_EXTENSIONS` | `--exclude-extensions` |  | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_ARCHIVE_FROM_PICASA_FOLDER_AS_ALBUM` | `--folder-as-album` | `none` | Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name |
| `IMMICH_GO_ARCHIVE_FROM_PICASA_FOLDER_AS_TAGS` | `--folder-as-tags` | `false` | Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024) |
| `IMMICH_GO_ARCHIVE_FROM_PICASA_IGNORE_SIDECAR_FILES` | `--ignore-sidecar-files` | `false` | Don't upload sidecar with the photo. |
| `IMMICH_GO_ARCHIVE_FROM_PICASA_INCLUDE_EXTENSIONS` | `--include-extensions` |  | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_PICASA_INCLUDE_TYPE` | `--include-type` |  | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_PICASA_INTO_ALBUM` | `--into-album` |  | Specify an album to import all files into |
| `IMMICH_GO_ARCHIVE_FROM_PICASA_RECURSIVE` | `--recursive` | `true` | Explore the folder and all its sub-folders |

## stack

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| `IMMICH_GO_STACK_ADMIN_API_KEY` | `--admin-api-key` |  | Admin's API Key for managing server's jobs |
| `IMMICH_GO_STACK_API_KEY` | `--api-key` |  | API Key |
| `IMMICH_GO_STACK_API_TRACE` | `--api-trace` | `false` | Enable trace of api calls |
| `IMMICH_GO_STACK_CLIENT_TIMEOUT` | `--client-timeout` | `20m0s` | Set server calls timeout |
| `IMMICH_GO_STACK_DATE_RANGE` | `--date-range` | `unset` | photos must be taken in the date range |
| `IMMICH_GO_STACK_DEVICE_UUID` | `--device-uuid` | `gl65` | Set a device UUID |
| `IMMICH_GO_STACK_DRY_RUN` | `--dry-run` | `false` | Simulate all actions |
| `IMMICH_GO_STACK_MANAGE_BURST` | `--manage-burst` | `NoStack` | Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG |
| `IMMICH_GO_STACK_MANAGE_EPSON_FASTFOTO` | `--manage-epson-fastfoto` | `false` | Manage Epson FastFoto file (default: false) |
| `IMMICH_GO_STACK_MANAGE_HEIC_JPEG` | `--manage-heic-jpeg` | `NoStack` | Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG |
| `IMMICH_GO_STACK_MANAGE_RAW_JPEG` | `--manage-raw-jpeg` | `NoStack` | Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG |
| `IMMICH_GO_STACK_PAUSE_IMMICH_JOBS` | `--pause-immich-jobs` | `true` | Pause Immich background jobs during upload operations |
| `IMMICH_GO_STACK_SERVER` | `--server` |  | Immich server address (example http://your-ip:2283 or https://your-domain) |
| `IMMICH_GO_STACK_SKIP_VERIFY_SSL` | `--skip-verify-ssl` | `false` | Skip SSL verification |
| `IMMICH_GO_STACK_TIME_ZONE` | `--time-zone` |  | Override the system time zone |

## upload

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| `IMMICH_GO_UPLOAD_ADMIN_API_KEY` | `--admin-api-key` |  | Admin's API Key for managing server's jobs |
| `IMMICH_GO_UPLOAD_API_KEY` | `--api-key` |  | API Key |
| `IMMICH_GO_UPLOAD_API_TRACE` | `--api-trace` | `false` | Enable trace of api calls |
| `IMMICH_GO_UPLOAD_CLIENT_TIMEOUT` | `--client-timeout` | `20m0s` | Set server calls timeout |
| `IMMICH_GO_UPLOAD_DEVICE_UUID` | `--device-uuid` | `gl65` | Set a device UUID |
| `IMMICH_GO_UPLOAD_DRY_RUN` | `--dry-run` | `false` | Simulate all actions |
| `IMMICH_GO_UPLOAD_MANAGE_BURST` | `--manage-burst` | `NoStack` | Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG |
| `IMMICH_GO_UPLOAD_MANAGE_EPSON_FASTFOTO` | `--manage-epson-fastfoto` | `false` | Manage Epson FastFoto file (default: false) |
| `IMMICH_GO_UPLOAD_MANAGE_HEIC_JPEG` | `--manage-heic-jpeg` | `NoStack` | Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG |
| `IMMICH_GO_UPLOAD_MANAGE_RAW_JPEG` | `--manage-raw-jpeg` | `NoStack` | Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG |
| `IMMICH_GO_UPLOAD_NO_UI` | `--no-ui` | `false` | Disable the user interface |
| `IMMICH_GO_UPLOAD_OVERWRITE` | `--overwrite` | `false` | Always overwrite files on the server with local versions |
| `IMMICH_GO_UPLOAD_PAUSE_IMMICH_JOBS` | `--pause-immich-jobs` | `true` | Pause Immich background jobs during upload operations |
| `IMMICH_GO_UPLOAD_SERVER` | `--server` |  | Immich server address (example http://your-ip:2283 or https://your-domain) |
| `IMMICH_GO_UPLOAD_SESSION_TAG` | `--session-tag` | `false` | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS" |
| `IMMICH_GO_UPLOAD_SKIP_VERIFY_SSL` | `--skip-verify-ssl` | `false` | Skip SSL verification |
| `IMMICH_GO_UPLOAD_TAG` | `--tag` | `[]` | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1') |
| `IMMICH_GO_UPLOAD_TIME_ZONE` | `--time-zone` |  | Override the system time zone |

## upload from-folder

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| `IMMICH_GO_UPLOAD_FROM_FOLDER_ALBUM_PATH_JOINER` | `--album-path-joiner` | ` / ` | Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ') |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_BAN_FILE` | `--ban-file` | `'@eaDir/', '@__thumb/', 'SYNOFILE_THUMB_*.*', 'Lightroom Catalog/', 'thumbnails/', '.DS_Store/', '/._*', '.photostructure/', 'Recently Deleted/'` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_DATE_FROM_NAME` | `--date-from-name` | `true` | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_DATE_RANGE` | `--date-range` | `unset` | Only import photos taken within the specified date range |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_EXCLUDE_EXTENSIONS` | `--exclude-extensions` |  | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_FOLDER_AS_ALBUM` | `--folder-as-album` | `none` | Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_FOLDER_AS_TAGS` | `--folder-as-tags` | `false` | Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_IGNORE_SIDECAR_FILES` | `--ignore-sidecar-files` | `false` | Don't upload sidecar with the photo. |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_INCLUDE_EXTENSIONS` | `--include-extensions` |  | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_INCLUDE_TYPE` | `--include-type` |  | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_INTO_ALBUM` | `--into-album` |  | Specify an album to import all files into |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_RECURSIVE` | `--recursive` | `true` | Explore the folder and all its sub-folders |

## upload from-google-photos

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_BAN_FILE` | `--ban-file` | `'@eaDir/', '@__thumb/', 'SYNOFILE_THUMB_*.*', 'Lightroom Catalog/', 'thumbnails/', '.DS_Store/', '/._*', '.photostructure/', 'Recently Deleted/'` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_DATE_RANGE` | `--date-range` | `unset` | Only import photos taken within the specified date range |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_EXCLUDE_EXTENSIONS` | `--exclude-extensions` |  | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_FROM_ALBUM_NAME` | `--from-album-name` |  | Only import photos from the specified Google Photos album |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_ARCHIVED` | `--include-archived` | `true` | Import archived Google Photos |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_EXTENSIONS` | `--include-extensions` |  | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_PARTNER` | `--include-partner` | `true` | Import photos from your partner's Google Photos account |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_TRASHED` | `--include-trashed` | `false` | Import photos that are marked as trashed in Google Photos |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_TYPE` | `--include-type` |  | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_UNMATCHED` | `--include-unmatched` | `false` | Import photos that do not have a matching JSON file in the takeout |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_UNTITLED_ALBUMS` | `--include-untitled-albums` | `false` | Include photos from albums without a title in the import process |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_PARTNER_SHARED_ALBUM` | `--partner-shared-album` |  | Add partner's photo to the specified album name |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_PEOPLE_TAG` | `--people-tag` | `true` | Tag uploaded photos with tags "people/name" found in the JSON file |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_SYNC_ALBUMS` | `--sync-albums` | `true` | Automatically create albums in Immich that match the albums in your Google Photos takeout |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_TAKEOUT_TAG` | `--takeout-tag` | `true` | Tag uploaded photos with a tag "{takeout}/takeout-YYYYMMDDTHHMMSSZ" |

## upload from-icloud

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_ALBUM_PATH_JOINER` | `--album-path-joiner` | ` / ` | Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ') |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_BAN_FILE` | `--ban-file` | `'@eaDir/', '@__thumb/', 'SYNOFILE_THUMB_*.*', 'Lightroom Catalog/', 'thumbnails/', '.DS_Store/', '/._*', '.photostructure/', 'Recently Deleted/'` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_DATE_FROM_NAME` | `--date-from-name` | `true` | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_DATE_RANGE` | `--date-range` | `unset` | Only import photos taken within the specified date range |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_EXCLUDE_EXTENSIONS` | `--exclude-extensions` |  | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_FOLDER_AS_ALBUM` | `--folder-as-album` | `none` | Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_FOLDER_AS_TAGS` | `--folder-as-tags` | `false` | Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_IGNORE_SIDECAR_FILES` | `--ignore-sidecar-files` | `false` | Don't upload sidecar with the photo. |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_INCLUDE_EXTENSIONS` | `--include-extensions` |  | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_INCLUDE_TYPE` | `--include-type` |  | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_INTO_ALBUM` | `--into-album` |  | Specify an album to import all files into |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_MEMORIES` | `--memories` | `false` | Import icloud memories as albums |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_RECURSIVE` | `--recursive` | `true` | Explore the folder and all its sub-folders |

## upload from-immich

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_ADMIN_API_KEY` | `--from-admin-api-key` |  | Admin's API Key for managing server's jobs |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_ALBUMS` | `--from-albums` | `[]` | Get assets only from those albums, can be used multiple times |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_API_KEY` | `--from-api-key` |  | API Key |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_API_TRACE` | `--from-api-trace` | `false` | Enable trace of api calls |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_ARCHIVED` | `--from-archived` | `false` | Get only archived assets |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_CITY` | `--from-city` |  | Get only assets from this city |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_CLIENT_TIMEOUT` | `--from-client-timeout` | `20m0s` | Set server calls timeout |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_COUNTRY` | `--from-country` |  | Get only assets from this country |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_DATE_RANGE` | `--from-date-range` | `unset` | Only import photos taken within the specified date range |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_DEVICE_UUID` | `--from-device-uuid` | `gl65` | Set a device UUID |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_DRY_RUN` | `--from-dry-run` | `false` | Simulate all actions |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_EXCLUDE_EXTENSIONS` | `--from-exclude-extensions` |  | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_FAVORITE` | `--from-favorite` | `false` | Get only favorite assets |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_INCLUDE_EXTENSIONS` | `--from-include-extensions` |  | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_INCLUDE_TYPE` | `--from-include-type` |  | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_MAKE` | `--from-make` |  | Get only assets with this make |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_MINIMAL_RATING` | `--from-minimal-rating` | `0` | Get only assets with a rating greater or equal to this value |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_MODEL` | `--from-model` |  | Get only assets with this model |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_NO_ALBUM` | `--from-no-album` | `false` | Get only assets that are not in any album |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_PARTNERS` | `--from-partners` | `false` | Get partner's assets as well |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_PAUSE_IMMICH_JOBS` | `--from-pause-immich-jobs` | `true` | Pause Immich background jobs during upload operations |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_PEOPLE` | `--from-people` | `[]` | Get assets only with those people, can be used multiple times |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_SERVER` | `--from-server` |  | Immich server address (example http://your-ip:2283 or https://your-domain) |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_SKIP_VERIFY_SSL` | `--from-skip-verify-ssl` | `false` | Skip SSL verification |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_STATE` | `--from-state` |  | Get only assets from this state |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_TAGS` | `--from-tags` | `[]` | Get assets only with those tags, can be used multiple times |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_TIME_ZONE` | `--from-time-zone` |  | Override the system time zone |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_TRASH` | `--from-trash` | `false` | Get only trashed assets |

## upload from-picasa

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| `IMMICH_GO_UPLOAD_FROM_PICASA_ALBUM_PATH_JOINER` | `--album-path-joiner` | ` / ` | Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ') |
| `IMMICH_GO_UPLOAD_FROM_PICASA_ALBUM_PICASA` | `--album-picasa` | `true` | Use Picasa album name found in .picasa.ini file |
| `IMMICH_GO_UPLOAD_FROM_PICASA_BAN_FILE` | `--ban-file` | `'@eaDir/', '@__thumb/', 'SYNOFILE_THUMB_*.*', 'Lightroom Catalog/', 'thumbnails/', '.DS_Store/', '/._*', '.photostructure/', 'Recently Deleted/'` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_UPLOAD_FROM_PICASA_DATE_FROM_NAME` | `--date-from-name` | `true` | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_DATE_RANGE` | `--date-range` | `unset` | Only import photos taken within the specified date range |
| `IMMICH_GO_UPLOAD_FROM_PICASA_EXCLUDE_EXTENSIONS` | `--exclude-extensions` |  | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_FOLDER_AS_ALBUM` | `--folder-as-album` | `none` | Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name |
| `IMMICH_GO_UPLOAD_FROM_PICASA_FOLDER_AS_TAGS` | `--folder-as-tags` | `false` | Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_IGNORE_SIDECAR_FILES` | `--ignore-sidecar-files` | `false` | Don't upload sidecar with the photo. |
| `IMMICH_GO_UPLOAD_FROM_PICASA_INCLUDE_EXTENSIONS` | `--include-extensions` |  | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_INCLUDE_TYPE` | `--include-type` |  | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_INTO_ALBUM` | `--into-album` |  | Specify an album to import all files into |
| `IMMICH_GO_UPLOAD_FROM_PICASA_RECURSIVE` | `--recursive` | `true` | Explore the folder and all its sub-folders |

