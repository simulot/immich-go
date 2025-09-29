# Environment Variables

The following environment variables can be used to configure `immich-go`.

| Variable | Flag | Description |
|----------|------|-------------|
| `IMMICH_GO_ARCHIVE_DRY_RUN` | `--archive-dry-run` | dry run |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_ALBUM_PATH_JOINER` | `--archive-from-folder-album-path-joiner` | Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ') |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_ALBUM_PICASA` | `--archive-from-folder-album-picasa` | Use Picasa album name found in .picasa.ini file (default: false) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_BAN_FILE` | `--archive-from-folder-ban-file` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_DATE_FROM_NAME` | `--archive-from-folder-date-from-name` | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_DATE_RANGE` | `--archive-from-folder-date-range` | Only import photos taken within the specified date range |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_EXCLUDE_EXTENSIONS` | `--archive-from-folder-exclude-extensions` | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_FOLDER_AS_ALBUM` | `--archive-from-folder-folder-as-album` | Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_FOLDER_AS_TAGS` | `--archive-from-folder-folder-as-tags` | Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_IGNORE_SIDECAR_FILES` | `--archive-from-folder-ignore-sidecar-files` | Don't upload sidecar with the photo. |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_INCLUDE_EXTENSIONS` | `--archive-from-folder-include-extensions` | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_INCLUDE_TYPE` | `--archive-from-folder-include-type` | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_INTO_ALBUM` | `--archive-from-folder-into-album` | Specify an album to import all files into |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_RECURSIVE` | `--archive-from-folder-recursive` | Explore the folder and all its sub-folders |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_SESSION_TAG` | `--archive-from-folder-session-tag` | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS" |
| `IMMICH_GO_ARCHIVE_FROM_FOLDER_TAG` | `--archive-from-folder-tag` | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1') |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_BAN_FILE` | `--archive-from-google-photos-ban-file` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_DATE_RANGE` | `--archive-from-google-photos-date-range` | Only import photos taken within the specified date range |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_EXCLUDE_EXTENSIONS` | `--archive-from-google-photos-exclude-extensions` | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_FROM_ALBUM_NAME` | `--archive-from-google-photos-from-album-name` | Only import photos from the specified Google Photos album |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_ARCHIVED` | `--archive-from-google-photos-include-archived` | Import archived Google Photos |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_EXTENSIONS` | `--archive-from-google-photos-include-extensions` | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_PARTNER` | `--archive-from-google-photos-include-partner` | Import photos from your partner's Google Photos account |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_TRASHED` | `--archive-from-google-photos-include-trashed` | Import photos that are marked as trashed in Google Photos |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_TYPE` | `--archive-from-google-photos-include-type` | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_UNMATCHED` | `--archive-from-google-photos-include-unmatched` | Import photos that do not have a matching JSON file in the takeout |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_INCLUDE_UNTITLED_ALBUMS` | `--archive-from-google-photos-include-untitled-albums` | Include photos from albums without a title in the import process |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_PARTNER_SHARED_ALBUM` | `--archive-from-google-photos-partner-shared-album` | Add partner's photo to the specified album name |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_PEOPLE_TAG` | `--archive-from-google-photos-people-tag` | Tag uploaded photos with tags "people/name" found in the JSON file |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_SESSION_TAG` | `--archive-from-google-photos-session-tag` | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS" |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_SYNC_ALBUMS` | `--archive-from-google-photos-sync-albums` | Automatically create albums in Immich that match the albums in your Google Photos takeout |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_TAG` | `--archive-from-google-photos-tag` | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1') |
| `IMMICH_GO_ARCHIVE_FROM_GOOGLE_PHOTOS_TAKEOUT_TAG` | `--archive-from-google-photos-takeout-tag` | Tag uploaded photos with a tag "{takeout}/takeout-YYYYMMDDTHHMMSSZ" |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_DATE_RANGE` | `--archive-from-immich-date-range` | Only import photos taken within the specified date range |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_EXCLUDE_EXTENSIONS` | `--archive-from-immich-exclude-extensions` | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_API_KEY` | `--archive-from-immich-from-api-key` | API Key |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_API_TRACE` | `--archive-from-immich-from-api-trace` | Enable trace of api calls |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_ARCHIVED` | `--archive-from-immich-from-archived` | Get only archived assets |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_CLIENT_TIMEOUT` | `--archive-from-immich-from-client-timeout` | Set server calls timeout |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_DATE_RANGE` | `--archive-from-immich-from-date-range` | Get assets only within this date range (fromat: YYYY[-MM[-DD[,YYYY-MM-DD]]]) |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_FAVORITE` | `--archive-from-immich-from-favorite` | Get only favorite assets |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_MINIMAL_RATING` | `--archive-from-immich-from-minimal-rating` | Get only assets with a rating greater or equal to this value |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_SERVER` | `--archive-from-immich-from-server` | Immich server address (example http://your-ip:2283 or https://your-domain) |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_SKIP_VERIFY_SSL` | `--archive-from-immich-from-skip-verify-ssl` | Skip SSL verification |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_FROM_TRASH` | `--archive-from-immich-from-trash` | Get only trashed assets |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_INCLUDE_EXTENSIONS` | `--archive-from-immich-include-extensions` | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_ARCHIVE_FROM_IMMICH_INCLUDE_TYPE` | `--archive-from-immich-include-type` | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_STACK_DATE_RANGE` | `--stack-date-range` | photos must be taken in the date range |
| `IMMICH_GO_STACK_MANAGE_BURST` | `--stack-manage-burst` | Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG |
| `IMMICH_GO_STACK_MANAGE_EPSON_FASTFOTO` | `--stack-manage-epson-fastfoto` | Manage Epson FastFoto file (default: false) |
| `IMMICH_GO_STACK_MANAGE_HEIC_JPEG` | `--stack-manage-heic-jpeg` | Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG |
| `IMMICH_GO_STACK_MANAGE_RAW_JPEG` | `--stack-manage-raw-jpeg` | Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG |
| `IMMICH_GO_UPLOAD_CONCURRENT_UPLOADS` | `--upload-concurrent-uploads` | Number of concurrent upload workers (1-20) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_ALBUM_PATH_JOINER` | `--upload-from-folder-album-path-joiner` | Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ') |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_ALBUM_PICASA` | `--upload-from-folder-album-picasa` | Use Picasa album name found in .picasa.ini file (default: false) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_BAN_FILE` | `--upload-from-folder-ban-file` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_DATE_FROM_NAME` | `--upload-from-folder-date-from-name` | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_DATE_RANGE` | `--upload-from-folder-date-range` | Only import photos taken within the specified date range |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_EXCLUDE_EXTENSIONS` | `--upload-from-folder-exclude-extensions` | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_FOLDER_AS_ALBUM` | `--upload-from-folder-folder-as-album` | Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_FOLDER_AS_TAGS` | `--upload-from-folder-folder-as-tags` | Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_IGNORE_SIDECAR_FILES` | `--upload-from-folder-ignore-sidecar-files` | Don't upload sidecar with the photo. |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_INCLUDE_EXTENSIONS` | `--upload-from-folder-include-extensions` | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_INCLUDE_TYPE` | `--upload-from-folder-include-type` | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_INTO_ALBUM` | `--upload-from-folder-into-album` | Specify an album to import all files into |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_MANAGE_BURST` | `--upload-from-folder-manage-burst` | Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_MANAGE_EPSON_FASTFOTO` | `--upload-from-folder-manage-epson-fastfoto` | Manage Epson FastFoto file (default: false) |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_MANAGE_HEIC_JPEG` | `--upload-from-folder-manage-heic-jpeg` | Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_MANAGE_RAW_JPEG` | `--upload-from-folder-manage-raw-jpeg` | Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_RECURSIVE` | `--upload-from-folder-recursive` | Explore the folder and all its sub-folders |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_SESSION_TAG` | `--upload-from-folder-session-tag` | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS" |
| `IMMICH_GO_UPLOAD_FROM_FOLDER_TAG` | `--upload-from-folder-tag` | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1') |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_BAN_FILE` | `--upload-from-google-photos-ban-file` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_FROM_ALBUM_NAME` | `--upload-from-google-photos-from-album-name` | Only import photos from the specified Google Photos album |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_ARCHIVED` | `--upload-from-google-photos-include-archived` | Import archived Google Photos |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_PARTNER` | `--upload-from-google-photos-include-partner` | Import photos from your partner's Google Photos account |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_TRASHED` | `--upload-from-google-photos-include-trashed` | Import photos that are marked as trashed in Google Photos |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_UNMATCHED` | `--upload-from-google-photos-include-unmatched` | Import photos that do not have a matching JSON file in the takeout |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_INCLUDE_UNTITLED_ALBUMS` | `--upload-from-google-photos-include-untitled-albums` | Include photos from albums without a title in the import process |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_PARTNER_SHARED_ALBUM` | `--upload-from-google-photos-partner-shared-album` | Add partner's photo to the specified album name |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_PEOPLE_TAG` | `--upload-from-google-photos-people-tag` | Tag uploaded photos with tags "people/name" found in the JSON file |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_SESSION_TAG` | `--upload-from-google-photos-session-tag` | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS" |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_SYNC_ALBUMS` | `--upload-from-google-photos-sync-albums` | Automatically create albums in Immich that match the albums in your Google Photos takeout |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_TAG` | `--upload-from-google-photos-tag` | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1') |
| `IMMICH_GO_UPLOAD_FROM_GOOGLE_PHOTOS_TAKEOUT_TAG` | `--upload-from-google-photos-takeout-tag` | Tag uploaded photos with a tag "{takeout}/takeout-YYYYMMDDTHHMMSSZ" |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_ALBUM_PATH_JOINER` | `--upload-from-icloud-album-path-joiner` | Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ') |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_ALBUM_PICASA` | `--upload-from-icloud-album-picasa` | Use Picasa album name found in .picasa.ini file (default: false) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_BAN_FILE` | `--upload-from-icloud-ban-file` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_DATE_FROM_NAME` | `--upload-from-icloud-date-from-name` | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_DATE_RANGE` | `--upload-from-icloud-date-range` | Only import photos taken within the specified date range |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_EXCLUDE_EXTENSIONS` | `--upload-from-icloud-exclude-extensions` | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_FOLDER_AS_ALBUM` | `--upload-from-icloud-folder-as-album` | Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_FOLDER_AS_TAGS` | `--upload-from-icloud-folder-as-tags` | Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_IGNORE_SIDECAR_FILES` | `--upload-from-icloud-ignore-sidecar-files` | Don't upload sidecar with the photo. |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_INCLUDE_EXTENSIONS` | `--upload-from-icloud-include-extensions` | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_INCLUDE_TYPE` | `--upload-from-icloud-include-type` | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_INTO_ALBUM` | `--upload-from-icloud-into-album` | Specify an album to import all files into |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_MANAGE_BURST` | `--upload-from-icloud-manage-burst` | Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_MANAGE_EPSON_FASTFOTO` | `--upload-from-icloud-manage-epson-fastfoto` | Manage Epson FastFoto file (default: false) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_MANAGE_HEIC_JPEG` | `--upload-from-icloud-manage-heic-jpeg` | Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_MANAGE_RAW_JPEG` | `--upload-from-icloud-manage-raw-jpeg` | Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_MEMORIES` | `--upload-from-icloud-memories` | Import icloud memories as albums (default: false) |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_RECURSIVE` | `--upload-from-icloud-recursive` | Explore the folder and all its sub-folders |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_SESSION_TAG` | `--upload-from-icloud-session-tag` | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS" |
| `IMMICH_GO_UPLOAD_FROM_ICLOUD_TAG` | `--upload-from-icloud-tag` | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1') |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_DATE_RANGE` | `--upload-from-immich-date-range` | Only import photos taken within the specified date range |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_EXCLUDE_EXTENSIONS` | `--upload-from-immich-exclude-extensions` | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_API_KEY` | `--upload-from-immich-from-api-key` | API Key |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_API_TRACE` | `--upload-from-immich-from-api-trace` | Enable trace of api calls |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_ARCHIVED` | `--upload-from-immich-from-archived` | Get only archived assets |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_CLIENT_TIMEOUT` | `--upload-from-immich-from-client-timeout` | Set server calls timeout |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_DATE_RANGE` | `--upload-from-immich-from-date-range` | Get assets only within this date range (fromat: YYYY[-MM[-DD[,YYYY-MM-DD]]]) |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_FAVORITE` | `--upload-from-immich-from-favorite` | Get only favorite assets |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_MINIMAL_RATING` | `--upload-from-immich-from-minimal-rating` | Get only assets with a rating greater or equal to this value |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_SERVER` | `--upload-from-immich-from-server` | Immich server address (example http://your-ip:2283 or https://your-domain) |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_SKIP_VERIFY_SSL` | `--upload-from-immich-from-skip-verify-ssl` | Skip SSL verification |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_FROM_TRASH` | `--upload-from-immich-from-trash` | Get only trashed assets |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_INCLUDE_EXTENSIONS` | `--upload-from-immich-include-extensions` | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_IMMICH_INCLUDE_TYPE` | `--upload-from-immich-include-type` | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_ALBUM_PATH_JOINER` | `--upload-from-picasa-album-path-joiner` | Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ') |
| `IMMICH_GO_UPLOAD_FROM_PICASA_ALBUM_PICASA` | `--upload-from-picasa-album-picasa` | Use Picasa album name found in .picasa.ini file (default: false) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_BAN_FILE` | `--upload-from-picasa-ban-file` | Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. |
| `IMMICH_GO_UPLOAD_FROM_PICASA_DATE_FROM_NAME` | `--upload-from-picasa-date-from-name` | Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_DATE_RANGE` | `--upload-from-picasa-date-range` | Only import photos taken within the specified date range |
| `IMMICH_GO_UPLOAD_FROM_PICASA_EXCLUDE_EXTENSIONS` | `--upload-from-picasa-exclude-extensions` | Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_FOLDER_AS_ALBUM` | `--upload-from-picasa-folder-as-album` | Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name |
| `IMMICH_GO_UPLOAD_FROM_PICASA_FOLDER_AS_TAGS` | `--upload-from-picasa-folder-as-tags` | Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_IGNORE_SIDECAR_FILES` | `--upload-from-picasa-ignore-sidecar-files` | Don't upload sidecar with the photo. |
| `IMMICH_GO_UPLOAD_FROM_PICASA_INCLUDE_EXTENSIONS` | `--upload-from-picasa-include-extensions` | Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_INCLUDE_TYPE` | `--upload-from-picasa-include-type` | Single file type to include. (VIDEO or IMAGE) (default: all) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_INTO_ALBUM` | `--upload-from-picasa-into-album` | Specify an album to import all files into |
| `IMMICH_GO_UPLOAD_FROM_PICASA_MANAGE_BURST` | `--upload-from-picasa-manage-burst` | Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG |
| `IMMICH_GO_UPLOAD_FROM_PICASA_MANAGE_EPSON_FASTFOTO` | `--upload-from-picasa-manage-epson-fastfoto` | Manage Epson FastFoto file (default: false) |
| `IMMICH_GO_UPLOAD_FROM_PICASA_MANAGE_HEIC_JPEG` | `--upload-from-picasa-manage-heic-jpeg` | Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG |
| `IMMICH_GO_UPLOAD_FROM_PICASA_MANAGE_RAW_JPEG` | `--upload-from-picasa-manage-raw-jpeg` | Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG |
| `IMMICH_GO_UPLOAD_FROM_PICASA_RECURSIVE` | `--upload-from-picasa-recursive` | Explore the folder and all its sub-folders |
| `IMMICH_GO_UPLOAD_FROM_PICASA_SESSION_TAG` | `--upload-from-picasa-session-tag` | Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS" |
| `IMMICH_GO_UPLOAD_FROM_PICASA_TAG` | `--upload-from-picasa-tag` | Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1') |
| `IMMICH_GO_UPLOAD_NO_UI` | `--upload-no-ui` | Disable the user interface |
| `IMMICH_GO_UPLOAD_OVERWRITE` | `--upload-overwrite` | Always overwrite files on the server with local versions |
