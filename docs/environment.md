# Environment Variables

The following environment variables can be used to configure `immich-go`.

## Global

| Variable | Flag | Description |
|----------|------|-------------|
| `IMMICH_GO_DRY_RUN` | `--dry-run` | dry run |
| `IMMICH_GO_LOG_FILE` | `--log-file` | Write log messages into the file |
| `IMMICH_GO_LOG_LEVEL` | `--log-level` | Log level (DEBUG|INFO|WARN|ERROR), default INFO |
| `IMMICH_GO_LOG_TYPE` | `--log-type` | Log formatted  as text of JSON file |
| `IMMICH_GO_ON_ERRORS` | `--on-errors` | Action to take on errors, (stop|continue| <n> errors) |
| `IMMICH_GO_SAVE_CONFIG` | `--save-config` | Save the configuration to immich-go.yaml |

## archive from-folder

| Variable | Flag | Description |
|----------|------|-------------|
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_ALBUM_PATH_JOINER` | `--album-path-joiner` | Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ') |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_ALBUM_PICASA` | `--album-picasa` | Use Picasa album name found in .picasa.ini file (default: false) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_BAN_FILE` | `--ban-file` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_DATE_FROM_NAME` | `--date-from-name` | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_DATE_RANGE` | `--date-range` | Only import photos taken within the specified date range |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_EXCLUDE_EXTENSIONS` | `--exclude-extensions` | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_FOLDER_AS_ALBUM` | `--folder-as-album` | Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_FOLDER_AS_TAGS` | `--folder-as-tags` | Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_IGNORE_SIDECAR_FILES` | `--ignore-sidecar-files` | Don't upload sidecar with the photo. |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_INCLUDE_EXTENSIONS` | `--include-extensions` | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_INCLUDE_TYPE` | `--include-type` | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_INTO_ALBUM` | `--into-album` | Specify an album to import all files into |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_RECURSIVE` | `--recursive` | Explore the folder and all its sub-folders |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_SESSION_TAG` | `--session-tag` | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS" |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_TAG` | `--tag` | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1') |

## archive from-google-photos

| Variable | Flag | Description |
|----------|------|-------------|
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_BAN_FILE` | `--ban-file` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_DATE_RANGE` | `--date-range` | Only import photos taken within the specified date range |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_EXCLUDE_EXTENSIONS` | `--exclude-extensions` | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_FROM_ALBUM_NAME` | `--from-album-name` | Only import photos from the specified Google Photos album |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_ARCHIVED` | `--include-archived` | Import archived Google Photos |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_EXTENSIONS` | `--include-extensions` | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_PARTNER` | `--include-partner` | Import photos from your partner's Google Photos account |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_TRASHED` | `--include-trashed` | Import photos that are marked as trashed in Google Photos |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_TYPE` | `--include-type` | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_UNMATCHED` | `--include-unmatched` | Import photos that do not have a matching JSON file in the takeout |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_UNTITLED_ALBUMS` | `--include-untitled-albums` | Include photos from albums without a title in the import process |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_PARTNER_SHARED_ALBUM` | `--partner-shared-album` | Add partner's photo to the specified album name |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_PEOPLE_TAG` | `--people-tag` | Tag uploaded photos with tags "people/name" found in the JSON file |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_SESSION_TAG` | `--session-tag` | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS" |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_SYNC_ALBUMS` | `--sync-albums` | Automatically create albums in Immich that match the albums in your Google Photos takeout |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_TAG` | `--tag` | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1') |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_TAKEOUT_TAG` | `--takeout-tag` | Tag uploaded photos with a tag "{takeout}/takeout-YYYYMMDDTHHMMSSZ" |

## archive from-immich

| Variable | Flag | Description |
|----------|------|-------------|
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_ALBUMS` | `--from-albums` | Get assets only from those albums, can be used multiple times |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_API_KEY` | `--from-api-key` | API Key |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_API_TRACE` | `--from-api-trace` | Enable trace of api calls |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_ARCHIVED` | `--from-archived` | Get only archived assets |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_CITY` | `--from-city` | Get only assets from this city |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_CLIENT_TIMEOUT` | `--from-client-timeout` | Set server calls timeout |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_COUNTRY` | `--from-country` | Get only assets from this country |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_FAVORITE` | `--from-favorite` | Get only favorite assets |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_MAKE` | `--from-make` | Get only assets with this make |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_MINIMAL_RATING` | `--from-minimal-rating` | Get only assets with a rating greater or equal to this value |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_MODEL` | `--from-model` | Get only assets with this model |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_NO_ALBUM` | `--from-no-album` | Get only assets that are not in any album |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_PARTNERS` | `--from-partners` | Get partner's assets as well |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_PEOPLE` | `--from-people` | Get assets only with those people, can be used multiple times |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_SERVER` | `--from-server` | Immich server address (example http://your-ip:2283 or https://your-domain) |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_SKIP_VERIFY_SSL` | `--from-skip-verify-ssl` | Skip SSL verification |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_STATE` | `--from-state` | Get only assets from this state |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_TAGS` | `--from-tags` | Get assets only with those tags, can be used multiple times |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_TRASH` | `--from-trash` | Get only trashed assets |

## stack

| Variable | Flag | Description |
|----------|------|-------------|
| `IMMICH_GO_STACK_DATE_RANGE` | `--date-range` | photos must be taken in the date range |
| `IMMICH_GO_STACK_MANAGE_BURST` | `--manage-burst` | Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG |
| `IMMICH_GO_STACK_MANAGE_EPSON_FASTFOTO` | `--manage-epson-fastfoto` | Manage Epson FastFoto file (default: false) |
| `IMMICH_GO_STACK_MANAGE_HEIC_JPEG` | `--manage-heic-jpeg` | Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG |
| `IMMICH_GO_STACK_MANAGE_RAW_JPEG` | `--manage-raw-jpeg` | Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG |

## upload from-folder

| Variable | Flag | Description |
|----------|------|-------------|
| `IMMICH_GO_UPLOAD_FROM_FOLDER_ALBUM_PATH_JOINER` | `--album-path-joiner` | Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ') |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_ALBUM_PICASA` | `--album-picasa` | Use Picasa album name found in .picasa.ini file (default: false) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_BAN_FILE` | `--ban-file` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_DATE_FROM_NAME` | `--date-from-name` | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_DATE_RANGE` | `--date-range` | Only import photos taken within the specified date range |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_EXCLUDE_EXTENSIONS` | `--exclude-extensions` | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_FOLDER_AS_ALBUM` | `--folder-as-album` | Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_FOLDER_AS_TAGS` | `--folder-as-tags` | Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_IGNORE_SIDECAR_FILES` | `--ignore-sidecar-files` | Don't upload sidecar with the photo. |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_INCLUDE_EXTENSIONS` | `--include-extensions` | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_INCLUDE_TYPE` | `--include-type` | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_INTO_ALBUM` | `--into-album` | Specify an album to import all files into |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_MANAGE_BURST` | `--manage-burst` | Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_MANAGE_EPSON_FASTFOTO` | `--manage-epson-fastfoto` | Manage Epson FastFoto file (default: false) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_MANAGE_HEIC_JPEG` | `--manage-heic-jpeg` | Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_MANAGE_RAW_JPEG` | `--manage-raw-jpeg` | Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_RECURSIVE` | `--recursive` | Explore the folder and all its sub-folders |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_SESSION_TAG` | `--session-tag` | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS" |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_TAG` | `--tag` | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1') |

## upload from-google-photos

| Variable | Flag | Description |
|----------|------|-------------|
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_BAN_FILE` | `--ban-file` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_DATE_RANGE` | `--date-range` | Only import photos taken within the specified date range |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_EXCLUDE_EXTENSIONS` | `--exclude-extensions` | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_FROM_ALBUM_NAME` | `--from-album-name` | Only import photos from the specified Google Photos album |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_ARCHIVED` | `--include-archived` | Import archived Google Photos |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_EXTENSIONS` | `--include-extensions` | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_PARTNER` | `--include-partner` | Import photos from your partner's Google Photos account |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_TRASHED` | `--include-trashed` | Import photos that are marked as trashed in Google Photos |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_TYPE` | `--include-type` | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_UNMATCHED` | `--include-unmatched` | Import photos that do not have a matching JSON file in the takeout |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_UNTITLED_ALBUMS` | `--include-untitled-albums` | Include photos from albums without a title in the import process |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_MANAGE_BURST` | `--manage-burst` | Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_MANAGE_EPSON_FASTFOTO` | `--manage-epson-fastfoto` | Manage Epson FastFoto file (default: false) |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_MANAGE_HEIC_JPEG` | `--manage-heic-jpeg` | Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_MANAGE_RAW_JPEG` | `--manage-raw-jpeg` | Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_PARTNER_SHARED_ALBUM` | `--partner-shared-album` | Add partner's photo to the specified album name |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_PEOPLE_TAG` | `--people-tag` | Tag uploaded photos with tags "people/name" found in the JSON file |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_SESSION_TAG` | `--session-tag` | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS" |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_SYNC_ALBUMS` | `--sync-albums` | Automatically create albums in Immich that match the albums in your Google Photos takeout |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_TAG` | `--tag` | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1') |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_TAKEOUT_TAG` | `--takeout-tag` | Tag uploaded photos with a tag "{takeout}/takeout-YYYYMMDDTHHMMSSZ" |

## upload from-icloud

| Variable | Flag | Description |
|----------|------|-------------|
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_ALBUM_PATH_JOINER` | `--album-path-joiner` | Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ') |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_ALBUM_PICASA` | `--album-picasa` | Use Picasa album name found in .picasa.ini file (default: false) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_BAN_FILE` | `--ban-file` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_DATE_FROM_NAME` | `--date-from-name` | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_DATE_RANGE` | `--date-range` | Only import photos taken within the specified date range |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_EXCLUDE_EXTENSIONS` | `--exclude-extensions` | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_FOLDER_AS_ALBUM` | `--folder-as-album` | Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_FOLDER_AS_TAGS` | `--folder-as-tags` | Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_IGNORE_SIDECAR_FILES` | `--ignore-sidecar-files` | Don't upload sidecar with the photo. |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_INCLUDE_EXTENSIONS` | `--include-extensions` | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_INCLUDE_TYPE` | `--include-type` | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_INTO_ALBUM` | `--into-album` | Specify an album to import all files into |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_MANAGE_BURST` | `--manage-burst` | Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_MANAGE_EPSON_FASTFOTO` | `--manage-epson-fastfoto` | Manage Epson FastFoto file (default: false) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_MANAGE_HEIC_JPEG` | `--manage-heic-jpeg` | Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_MANAGE_RAW_JPEG` | `--manage-raw-jpeg` | Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_MEMORIES` | `--memories` | Import icloud memories as albums (default: false) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_RECURSIVE` | `--recursive` | Explore the folder and all its sub-folders |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_SESSION_TAG` | `--session-tag` | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS" |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_TAG` | `--tag` | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1') |

## upload from-immich

| Variable | Flag | Description |
|----------|------|-------------|
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_ALBUMS` | `--from-albums` | Get assets only from those albums, can be used multiple times |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_API_KEY` | `--from-api-key` | API Key |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_API_TRACE` | `--from-api-trace` | Enable trace of api calls |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_ARCHIVED` | `--from-archived` | Get only archived assets |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_CITY` | `--from-city` | Get only assets from this city |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_CLIENT_TIMEOUT` | `--from-client-timeout` | Set server calls timeout |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_COUNTRY` | `--from-country` | Get only assets from this country |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_DATE_RANGE` | `--from-date-range` | Only import photos taken within the specified date range |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_EXCLUDE_EXTENSIONS` | `--from-exclude-extensions` | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_FAVORITE` | `--from-favorite` | Get only favorite assets |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_INCLUDE_EXTENSIONS` | `--from-include-extensions` | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_INCLUDE_TYPE` | `--from-include-type` | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_MAKE` | `--from-make` | Get only assets with this make |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_MINIMAL_RATING` | `--from-minimal-rating` | Get only assets with a rating greater or equal to this value |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_MODEL` | `--from-model` | Get only assets with this model |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_NO_ALBUM` | `--from-no-album` | Get only assets that are not in any album |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_PARTNERS` | `--from-partners` | Get partner's assets as well |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_PEOPLE` | `--from-people` | Get assets only with those people, can be used multiple times |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_SERVER` | `--from-server` | Immich server address (example http://your-ip:2283 or https://your-domain) |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_SKIP_VERIFY_SSL` | `--from-skip-verify-ssl` | Skip SSL verification |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_STATE` | `--from-state` | Get only assets from this state |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_TAGS` | `--from-tags` | Get assets only with those tags, can be used multiple times |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_TRASH` | `--from-trash` | Get only trashed assets |

## upload from-picasa

| Variable | Flag | Description |
|----------|------|-------------|
| `IMMICH_GO_UPLOAD_FROM_PICASA_ALBUM_PATH_JOINER` | `--album-path-joiner` | Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ') |
| `IMMICH_GO_UPLOAD_FROM_PICASA_ALBUM_PICASA` | `--album-picasa` | Use Picasa album name found in .picasa.ini file (default: false) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_BAN_FILE` | `--ban-file` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_UPLOAD_FROM_PICASA_DATE_FROM_NAME` | `--date-from-name` | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_DATE_RANGE` | `--date-range` | Only import photos taken within the specified date range |
| `IMMICH_GO_UPLOAD_FROM_PICASA_EXCLUDE_EXTENSIONS` | `--exclude-extensions` | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_FOLDER_AS_ALBUM` | `--folder-as-album` | Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name |
| `IMMICH_GO_UPLOAD_FROM_PICASA_FOLDER_AS_TAGS` | `--folder-as-tags` | Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_IGNORE_SIDECAR_FILES` | `--ignore-sidecar-files` | Don't upload sidecar with the photo. |
| `IMMICH_GO_UPLOAD_FROM_PICASA_INCLUDE_EXTENSIONS` | `--include-extensions` | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_INCLUDE_TYPE` | `--include-type` | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_INTO_ALBUM` | `--into-album` | Specify an album to import all files into |
| `IMMICH_GO_UPLOAD_FROM_PICASA_MANAGE_BURST` | `--manage-burst` | Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG |
| `IMMICH_GO_UPLOAD_FROM_PICASA_MANAGE_EPSON_FASTFOTO` | `--manage-epson-fastfoto` | Manage Epson FastFoto file (default: false) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_MANAGE_HEIC_JPEG` | `--manage-heic-jpeg` | Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG |
| `IMMICH_GO_UPLOAD_FROM_PICASA_MANAGE_RAW_JPEG` | `--manage-raw-jpeg` | Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG |
| `IMMICH_GO_UPLOAD_FROM_PICASA_RECURSIVE` | `--recursive` | Explore the folder and all its sub-folders |
| `IMMICH_GO_UPLOAD_FROM_PICASA_SESSION_TAG` | `--session-tag` | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS" |
| `IMMICH_GO_UPLOAD_FROM_PICASA_TAG` | `--tag` | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1') |

