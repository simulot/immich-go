# Release notes 

## You can now support my work on `Immich-go`:

- [Github Sponsor page](https://github.com/sponsors/simulot)
- [paypal donor page](https://www.paypal.com/donate/?hosted_button_id=VGU2SQE88T2T4)

## Release v0.23.0-alpha6 🏗️ Work in progress 🏗️ 

### New features

**Folder import tags**
Its now possible to assign tags to photos and videos:
```sh
--folder-as-tags                     Use the folder structure as tags, (ex: the file "holidays/summer 2024/file.jpg" get the tag holidays/summer 2024)
--session-tag                        Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS"
--tag strings                        Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1')
```

The session tag is useful to identify all photos imported at the same time. It's easy to remove them from the tag screen

**Google photos import tags**

```sh
--takeout-tag                        Tag uploaded photos with the takeout file name: "{takeout}/takeout-YYYYMMDDTHHMMSSZ"
--session-tag                        Tag uploaded photos with a tag "{immich-go}/YYYY-MM-DD HH-MM-SS"
--tag strings                        Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1')
```


#### Breaking change since v0.23.0-alpha5
A metadata file is created withe same name as the main file, but with the extension `.json`. The XMP file is left untouched.


### Fixes
* [#533](https://github.com/simulot/immich-go/issues/533) RAW file metadata
The efforts for determining the capture date from the file name are useless.
Now the file date is provided to Immich as if the file was dropped on the immich's page.
The `--capture-date-method` is now set to `NONE` by default.
* [[#534](https://github.com/simulot/immich-go/issues/534)] Errors on windows


## Release 0.23.0-alpha5 🏗️ Work in progress 🏗️ 

### New features

##### The command `archive --from-immich` archives the user content from an Immich into a folder structure

```sh
Archive photos from Immich

Usage:
  immich-go archive from-immich [from-flags] [flags]

Flags:
      --from-album strings             Get assets only from those albums, can be used multiple times
      --from-api-key string            API Key
      --from-api-trace                 Enable trace of api calls
      --from-client-timeout duration   Set server calls timeout (default 5m0s)
      --from-date-range date-range     Get assets only within this date range (fromat: YYYY[-MM[-DD[,YYYY-MM-DD]]]) (default unset)
      --from-server string             Immich server address (example http://your-ip:2283 or https://your-domain)
      --from-skip-verify-ssl           Skip SSL verification
  -h, --help                           help for from-immich

Global Flags:
  -l, --log-file string          Write log messages into the file
      --log-level string         Log level (DEBUG|INFO|WARN|ERROR), default INFO (default "INFO")
      --log-type string          Log formatted  as text of JSON file (default "text")
  -w, --write-to-folder string   Path where to write the archive
```
Comming soon:
--minimal-rating 
--from-favorite
--from-trashed
--from-archived

##### The command `upload --from-immich` uploads the user's content from another Immich
This command accepts the same flags as the `archive --from-immich` command.
It preserves albums and tags from the source Immich.



## Release 0.23.0-alpha4 🏗️ Work in progress 🏗️ 

### New features

#### New command `archive`
This command aims is to store photos and videos into a plain folder structure. The folder structure is YYYY/YYYY-MM/files, as following:

```sh
tree .
.
├── 2011
│   └── 2011-04
│       ├── 20110430.CR2
│       ├── 20110430.CR2.xmp
│       ├── 20110430.jpg
│       ├── 20110430.jpg.xmp
│       ├── IMG_2477.CR2
│       ├── IMG_2477.CR2.xmp
│       ├── IMG_2478.CR2
│       ├── IMG_2478.CR2.xmp
│       ├── IMG_2479.CR2
│       └── IMG_2479.CR2.xmp
└── 2023
    ├── 2023-06
    │   ├── PXL_20230607_063000139.jpg
    │   └── PXL_20230607_063000139.jpg.xmp
    └── 2023-10
        ├── PXL_20231006_063029647.jpg
        ├── PXL_20231006_063029647.jpg.xmp
        ├── PXL_20231006_063851485.jpg
        └── PXL_20231006_063851485.jpg.xmp
```

XMP files present in the source folder are copied in the destination folder.
Google Photos takeout JSON files are translated into customized XMP files and copied in the destination folder.
Those XMP files use a custom schema to store the Google Photos metadata:
```xml
<?xpacket begin='?' id='W5M0MpCehiHzreSzNTczkc9d'?>
<x:xmpmeta xmlns:x="adobe:ns:meta/" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:exif="http://ns.adobe.com/exif/1.0/" xmlns:xmp="http://ns.adobe.com/xap/1.0/" xmlns:tiff="http://ns.adobe.com/tiff/1.0/" xmlns:digikam="http://www.digikam.org/ns/1.0/" xmlns:immichgo="http://ns.immich-go.com/immich-go/1.0/" x:xmptk="immich-go version:dev, commit:none, date:unknown">
  <rdf:RDF>
    <rdf:Description>
      <immichgo:ImmichGoProperties>
        <immichgo:title>This is a title</immichgo:title>
        <immichgo:DateTimeOriginal>2023-10-10T01:11:00.000-04:00</immichgo:DateTimeOriginal>
        <immichgo:trashed>False</immichgo:trashed>
        <immichgo:archived>False</immichgo:archived>
        <immichgo:fromPartner>False</immichgo:fromPartner>
        <immichgo:favorite>True</immichgo:favorite>
        <immichgo:rating>3</immichgo:rating>
        <immichgo:albums>
          <rdf:Bag>
            <rdf:Li>
              <immichgo:album>
                <immichgo:title>Vacation 2024</immichgo:title>
                <immichgo:description>Vacation 2024 hawaii and more</immichgo:description>
                <immichgo:latitude>19,49.23661N</immichgo:latitude>
                <immichgo:longitude>155,28.39525W</immichgo:longitude>
              </immichgo:album>
            </rdf:Li>
          </rdf:Bag>
        </immichgo:albums>
      </immichgo:ImmichGoProperties>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end='w'?>
``` 



The general syntax is:
```sh
.\immich-go archive from-xxx [from-xxx flags...] --write-to-folder <destination> <source> 
```

##### The command `archive --from-google-photos` archives a Google Photos takeout into a folder structure

This command create a folder structure in `/path/to/destination` with the result of the takeout analysis.
The resulting folder structure can be re-imported into immich-go with the command `upload from-google-photo path/to/archived-folder`.

##### The command `archive --from-` archives a Google Photos takeout into a folder structure

Example:
```sh
.\immich-go archive from-google-photos  --include-partner  --write-to-folder /path/to/destination /path/to/takeout*.zip
```

Coming soon: 
- archiving an immich server into a folder.


#### Handling of scanned photos by Epson FastFoto
- `--manage-epson-fastfoto`  Manage Epson FastFoto file (default: false)
  <br>Group scanned photos in stacks 
  - Scan_0001.jpg Original photo
  - Scan_0001_a.jpg  Enhanced photo, the cover of the stack
  - Scan_0001_b.jpg  Back of the photo

## Release 0.23.0-alpha3 🏗️ Work in progress 🏗️ 

### New features

- `--manage-burst=BurstFlag`   Manage burst photos. Possible values are: 
  - `StackKeepRaw` Discard JPEG files, and stack the RAW files (default)
  - `StackKeepJPEG` Discard RAW files, and stack the JPEG files 
  - `Stack` Stack all photos, RAW and JPEG photos are imported in the same stack
      
- `--manage-heic-jpeg=HeicJpgFlag` Manage coupled HEIC and JPEG files. Possible values: 
  - `KeepHeic` Keep only the HEIC files (default)
  - `KeepJPG` Keep only the JPEG files
  - `StackCoverHeic` Stack both, the HEIC file is the cover
  - `StackCoverJPG` Stack both, the JPEG file is the cover

- `--manage-raw-jpeg=RawJPGFlag`   Manage coupled RAW and JPEG files. Possible values: 
  - `KeepRaw` Keep only the RAW files (default)
  - `KeepJPG` Keep only the JPEG files
  - `StackCoverRaw` Stack both, the RAW file is the cover
  - `StackCoverJPG` Stack both, the JPEG file is the cover


## Release 0.23.0-alpha2 🏗️ Work in progress 🏗️ 
This an early version of immich-go version v0.23.0-alpha2 
Yes, v0.23.0-alpha2, and not v1.0.0-alpha2. Let's stick to the semantic versioning.

- [x] better logging 
  - log level are effectives
  - adoption of the structured log package
  - the level DEBUG give file details and metadata
  - colored log on screen
- [x] clear separation between folder import and google import
- [x] adoption of the linux convention of double dashes flags
- [x] priority of EXIF data over file name for date capture
- code restructuration to enable further possibilities
  - [ ] Upload from Picasa
  - [ ] Exporting of google photos archive as a folder

### Big changes for the best

The toy project has grown up. The code has been refactored to be more modular and to allow further development. The code is now more readable and maintainable. This opens the door to new features and new import possibilities. The down side is that the code is not backward compatible. The command line options have changed.


### Upload from folder options
```
Upload photos from a folder

Usage:
  immich-go upload from-folder [flags] <path>...

Flags:
      --album-path-joiner string           Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ') (default " / ")
      --ban-file FileList                  Exclude a file based on a pattern (case-insensitive). Can be specified multiple times. (default '@eaDir/', '@__thumb/', 'SYNOFILE_THUMB_*.*', 'Lightroom Catalog/', 'thumbnails/', '.DS_Store/')
      --capture-date-method DateMethod     Specify the method to determine the capture date when not provided in a sidecar file. Options: NONE (do not attempt to determine), FILENAME (extract from filename), EXIF (extract from EXIF metadata), FILENAME-EXIF (try filename first, then EXIF), EXIF-FILENAME (try EXIF first, then filename) (default EXIF-FILENAME)
      --date-range date-range              Only import photos taken within the specified date range (default unset)
      --exclude-extensions ExtensionList   Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none)
      --exiftool-enabled                   Enable the use of the external 'exiftool' program (if installed and available in the system path) to extract EXIF metadata
      --exiftool-path string               Path to the ExifTool executable (default: search in system's PATH)
      --exiftool-timezone timezone         Timezone to use when parsing exif timestamps without timezone Options: LOCAL (use the system's local timezone), UTC (use UTC timezone), or a valid timezone name (e.g. America/New_York) (default Local)
      --filename-timezone timezone         Specify the timezone to use when detecting the date from the filename. Options: Local (use the system's local timezone), UTC (use UTC timezone), or a valid timezone name (e.g. America/New_York) (default Local)
      --folder-as-album folderMode         Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name (default NONE)
  -h, --help                               help for from-folder
      --ignore-sidecar-files               Don't upload sidecar with the photo.
      --include-extensions ExtensionList   Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all)
      --into-album string                  Specify an album to import all files into
      --recursive                          Explore the folder and all its sub-folders (default true)

Global Flags:
      --api string                Immich api endpoint (example http://container_ip:3301)
  -k, --api-key string            API Key
      --api-trace                 Enable trace of api calls
      --client-timeout duration   Set server calls timeout (default 5m0s)
      --device-uuid string        Set a device UUID (default "gl65")
      --dry-run                   Simulate all actions
  -l, --log-file string           Write log messages into the file
      --log-level string          Log level (DEBUG|INFO|WARN|ERROR), default INFO (default "INFO")
      --log-type string           Log formatted  as text of JSON file (default "text")
      --no-ui                     Disable the user interface
  -s, --server string             Immich server address (example http://your-ip:2283 or https://your-domain)
      --skip-verify-ssl           Skip SSL verification
      --time-zone string          Override the system time zone
```

### Upload from a google-photos 
```
Upload photos either from a zipped Google Photos takeout or decompressed archive

Usage:
  immich-go upload from-google-photos [flags] <takeout-*.zip> | <takeout-folder>

Flags:
      --ban-file FileList                  Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.
      --date-range date-range              Only import photos taken within the specified date range (default unset)
      --exclude-extensions ExtensionList   Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none)
      --from-album-name string             Only import photos from the specified Google Photos album
  -h, --help                               help for from-google-photos
  -a, --include-archived                   Import archived Google Photos (default true)
      --include-extensions ExtensionList   Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all)
  -p, --include-partner                    Import photos from your partner's Google Photos account (default true)
  -t, --include-trashed                    Import photos that are marked as trashed in Google Photos
  -u, --include-unmatched                  Import photos that do not have a matching JSON file in the takeout
      --include-untitled-albums            Include photos from albums without a title in the import process
      --partner-shared-album string        Add partner's photo to the specified album name
      --sync-albums                        Automatically create albums in Immich that match the albums in your Google Photos takeout (default true)

Global Flags:
      --api string                Immich api endpoint (example http://container_ip:3301)
  -k, --api-key string            API Key
      --api-trace                 Enable trace of api calls
      --client-timeout duration   Set server calls timeout (default 5m0s)
      --device-uuid string        Set a device UUID (default "gl65")
      --dry-run                   Simulate all actions
  -l, --log-file string           Write log messages into the file
      --log-level string          Log level (DEBUG|INFO|WARN|ERROR), default INFO (default "INFO")
      --log-type string           Log formatted  as text of JSON file (default "text")
      --no-ui                     Disable the user interface
  -s, --server string             Immich server address (example http://your-ip:2283 or https://your-domain)
      --skip-verify-ssl           Skip SSL verification
      --time-zone string          Override the system time zone
```


## Release 0.23.0-alpha1 🏗️ Work in progress 🏗️ 


## Release 0.22.1

### Fixes:
- [#509](https://github.com/simulot/immich-go/issues/509)      


## Release 0.22.0
Many thanks to @maybeanerd for their meticulous proofreading of the documentation files.

### New feature: Use the full image path as album name
Thanks to @giejay for their contribution
When the `-use-full-path-album-name` option is enabled, photos are added to a new album named after their full file path.
The path separator can be replaced using the `-album-name-path-separator=CHAR`


### New feature: google photos archived photos are imported as immich archive by default
Thanks to @Alex1607 for their contribution
Use the option `-auto-archive=FALSE` to disable this feature.

### What's Changed
* fix Takeout zip is unsupported file type #357 by @simulot in https://github.com/simulot/immich-go/pull/415
* docs: fix typos in readme by @maybeanerd in https://github.com/simulot/immich-go/pull/421
* Program errors out due to no ping API response despite API responding by @simulot in https://github.com/simulot/immich-go/pull/431
* remove "GetJobs" call from API traces by @simulot in https://github.com/simulot/immich-go/pull/442
* Add support for -use-full-path-album-name to be able to use the full path to the file as album name/title by @giejay in https://github.com/simulot/immich-go/pull/444
* Documentation-update by @simulot in https://github.com/simulot/immich-go/pull/446
* Add new AutoArchive option by @Alex1607 in https://github.com/simulot/immich-go/pull/450
* Update README.md, google-takeout.md, and motivation.md by @aaronjrodrigues in https://github.com/simulot/immich-go/pull/454

### New Contributors
* @maybeanerd made their first contribution in https://github.com/simulot/immich-go/pull/421
* @giejay made their first contribution in https://github.com/simulot/immich-go/pull/444
* @Alex1607 made their first contribution in https://github.com/simulot/immich-go/pull/450

**Full Changelog**: https://github.com/simulot/immich-go/compare/0.21.0...0.22.0

## Release 0.21.1

### Fixes:
- [#405](https://github.com/simulot/immich-go/issues/405) motion photo files with MP~2 extension marked unsupported and skipped
- Live photos not correctly counted

## Release 0.21.0

### Refactoring the Google Photos import another time
Lot of users have reported inconsistencies in upload counters. Each user case a different, and the takeout structure varies a bit. 
In order to debug those cases, I have developed a way to simulate the takeout import using only the the file list.  Read [how to send debug data](/docs/how-to-send-debug-data.md) without sharing photos.


### Option to force the upload of images despite the lack of JSON
Each image in a takeout is supposed to come with A JSON file giving the date of capture and the GPS coordinate. There a few reason for this:
1. The original file is copied, modified... and sometime there ins't a JSON for all versions
2. JSON aren't in the same ZIP file than the image, and only one part of the takeout is processed
3. The takeout misses a bunch of JSON

When asking another takeout isn't an option, it's possible to force the upload of photos with no JSON. Use the option `-upload-when-missing-JSON` 

### The stack function is disabled
The stack function need to be improved [#399](https://github.com/simulot/immich-go/issues/399), [#345](https://github.com/simulot/immich-go/issues/345), [#235](https://github.com/simulot/immich-go/issues/235)
Meanwhile, it is disabled by default. You can enable it using the option `-create-stacks=TRUE`.




### fixes:
- [#376](https://github.com/simulot/immich-go/issues/376) errors when uploading are disturbing the the % of the progression
- files with same path and name, but in different part of the takeout file set was forgotten in duplicate counters
- iPhone's Live photos recognition when the name is duplicated: ex IMG_2710(1).MP4 and IMG_2710(1).HEIC
- Missing a file when a directory contain several files with the same name, but of a different type. Ex: IMG_0170.HEIC,  IMG_0170.JPG
- Live videos attached to duplicated photos are now counted as duplicate as well, making the final report more relevant
- [#402](https://github.com/simulot/immich-go/issues/402) Wrong album assignment for images with the same name
- [#390](https://github.com/simulot/immich-go/issues/390) Question: report shows way less images uploaded than scanned
- [#376](https://github.com/simulot/immich-go/issues/376) errors when uploading are disturbing the the % of the progression
- [#401](https://github.com/simulot/immich-go/issues/401) Add an option to import images/movies even if there is no JSON file in the takeout


## Release 0.20.1

### changes
- add git action to build and release

### fixes:
- [#380](https://github.com/simulot/immich-go/issues/380) not all GP duplicates are detected correctly, counters are wrong

## Release 0.20

### feature exclude files based on a pattern

Use the `-exclude-files=PATTERN` to exclude certain files or directories from the upload. Repeat the option for each pattern do you need. The following directories are excluded automatically:
- @eaDir/
- @__thumb/
- SYNOFILE_THUMB_\*.\*
- Lightroom Catalog/
- thumbnails/
- .DS_Store/

Example, the following command excludes any files in directories called backup or draft and any file with name finishing with "copy)" as PXL_20231006_063121958 (another copy).jpg:
```sh
immich-go -sever=xxxxx -key=yyyyy upload -exclude-files=backup/ -exclude-files=draft/ -exclude=copy).*  /path/to/your/files
```

### fixes:
- [#365](https://github.com/simulot/immich-go/issues/365) missing associated metadata file isn't correct
- [#299](https://github.com/simulot/immich-go/issues/299) Real time GUI log only shows 4 lines
- [#370](https://github.com/simulot/immich-go/issues/370) ui: clearly mention when the upload in completed
- [#232](https://github.com/simulot/immich-go/issues/232) Exclude based on filename / glob
- [#357](https://github.com/simulot/immich-go/issues/357) clarify error message when a zip file is corrupted

## Release 0.19.1

### fix: UploadAsset
- [#359](https://github.com/simulot/immich-go/issues/359)Unexpected Discrepancy in 'Server has same quality' Metric After Re-uploading Images
- [#343](https://github.com/simulot/immich-go/issues/343)Getting stuck at 75% - server assets to delete

Fix the UploadAsset call causing some unexpected counts

## Release 0.19

### feat [#297](https://github.com/simulot/immich-go/issues/297) Derive Immich album description and location from Google photos JSON "enrichments"

Description, place and additional texts of Google Photos are now imported by `immich-go` in immich albums.
![screenshot](/docs/v0.19.Album%20description.png)

Immich-go add the album's place to photos not having GPS coordinates.

Thanks to @chrisscotland for his suggestion and the preparation of test samples.

### feat: provide a trace of all API calls.
Use the option `-api-trace` to log all immich calls in a file. The API key is redacted.

```log
2024-07-03T08:17:25+02:00 AssetUpload POST http://localhost:2283/api/assets
   Accept [application/json]
   Content-Type [multipart/form-data; boundary=1a9ca81d17452313f49073626c0ac04065fc7445efd3fadeffc5704663ed]
   X-Api-Key [xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx]
-- Binary body not dumped --
2024-07-03T08:17:26+02:00 201 Created
-- response body --
{
   "id": "1d839b04-fcf8-4bbb-bfbb-ab873159231b",
   "duplicate": false
}
-- response body end --
```

### fix: [#355](https://github.com/simulot/immich-go/issues/355) update albums of duplicated photos
Given the takeout 1 with
- photo1 in Album A
- photo2
- photo3

Given the takeout 2 with
- photo1 in Album A and Album B
- photo2 in Album B
- photo3
- photo4 in Album A

After importing the 2 takeouts:
- Album A: photo1, photo4
- Album B: photo1, photo2




### upload refactored
Photos are added the albums immediately after their upload to prevent a timeout at the end.




## Release 0.18.2

- fix [#347](https://github.com/simulot/immich-go/issues/347) Denied access to admin only route: /api/job

## Release 0.18.1

- fix [#336](https://github.com/simulot/immich-go/issues/336)  Processing stops with "context canceled" all the time 

## Release 0.18
![screen](/docs/render1719696528932.gif)

### feat: [#322](https://github.com/simulot/immich-go/issues/322) Add -version to get the immich version
The option `-version` return the version of the executable. 

### feat: [#289](https://github.com/simulot/immich-go/issues/289) Server's activity graph
The UI mode now show the current activity of the immich server. After 10 seconds of zero activity, the program stops

### feat: generate a CSV files with the fate of each file
Use the option `-debug-counters` to generate a CSV beside the log file

### feat: [#308](https://github.com/simulot/immich-go/issues/308) Immich-go gets photos date from filename or path
Immich-go tries to determine the date of capture with the file name, or the file path.

Ex:
| Path                                    | Photo's capture date |
| --------------------------------------- | -------------------- |
| photos/album/PXL_20220909_154515546.jpg | 2022-09-09 15:51:55  |
| photos/scanned/19991231.jpg             | 1999-12-31 00:00:00  |
| photos/20221109/IMG_1234.HEIC           | 2022-11-19 00:00:00  |
| photos/2022.11.09T20.30/IMG_1234.HEIC   | 2022-11-19 20:30:00  |
| photos/2022/11/09/IMG_1234.HEIC         | 2022-11-19 00:00:00  |

> Thanks to @erkexzcx for his contribution.


### fix: [#326, #303](https://github.com/simulot/immich-go/pull/326) Live Photo / Motion pictures 

Since a recent release of Immich, the live photos and motion picture were seen as a picture and a small movie.
The code has been refactored to be sure that the movie part is uploaded before the photo, and attached to the photo.

### fix: [#304](https://github.com/simulot/immich-go/issues/304)  Error when uploading images with a wild card without path .JPG 
Immich-go accepts "*.jpg" as parameter.


### fix: [#317](https://github.com/simulot/immich-go/issues/317) Explicit message when the call to /api/server-info/ping fails
The message is now explicit:
```
The ping API end point doesn't respond at this address: http://localhost:2283/api/server-info/ping
```


### fix: [#235,#240](https://github.com/simulot/immich-go/issues/235) Stack detection issue
> Thanks to @matteolomba for his contribution

### fix: Path of temporary files
Temporary files are created in the system's temporary folder.

### fix: [#311](https://github.com/simulot/immich-go/issues/311) Readme spelling

### fix: report unsupported files as unsupported  

### fix: report actual error instead of "context canceled"

### fix: stop all task on error in the no-ui mode

## Released 0.17.1

### Fix: UpdateAsset new API endpoint

### Fix: Typo in motivation.md file


## Released 0.17.0

### ⚠️ Immich has changed its API
This version of immich-go is compatible with immich v.1.106 and later. Use immich-go version 0.16.0 with older immich servers.

### feature: [[#284](https://github.com/simulot/immich-go/issues/284)] Be ready for the next immich version
See https://github.com/immich-app/immich/pull/9831 and https://github.com/immich-app/immich/pull/9667 for details

### fix: log upload errors with error level error in JSON logs

### fix: [[#287](https://github.com/simulot/immich-go/issues/287)] Prevent the Upload of photos that are trashed (Google Photos)
Trashed server's assets are excluded from the duplicate detection before uploading the same asset. 


## Release 0.16.0

### feature: [[#261](https://github.com/simulot/immich-go/issues/261)] Fallback to no-gui mode when the UI can't be created
When the terminal can't handle the UI mode, the program falls back to non gui mode automatically

### feature: The log can be written with the JSON format (JSONL)
Use the `-log-json` option to enable JSON logging (JSONL format). This allows using [./jq](https://jqlang.github.io/jq/)  to explore large logs.

### feature: [[#277](https://github.com/simulot/immich-go/issues/277)] Adjust client side timeouts 
The immich client timeout is set with the option `-client-timeout=duration`. 
 The duration is a decimal numbers with a unit suffix, such as "300ms", "1.5m" or "45m". Valid time units are "ms", "s", "m", "h".   

### fix: [[#270](https://github.com/simulot/immich-go/issues/270)]  `Missing associated metadata file` counter is not updated after the performance improvement
The counter `missing associated metadata` is broken since 0.15.0

### fix: [[#266](https://github.com/simulot/immich-go/issues/266)] Better handling of archive name with wildcards that matches with no file
When the file name pattern returns no files, a message is printed, and the program ends.

### fix: [[#273](https://github.com/simulot/immich-go/issues/273)] Missing upload files
Any error is counted as upload error, and reported in the log file.

### fix: Error handling during multitasking
Any error occurred during parallelized tasks cancels other as well.

### fix: Processed files count is displayed in no-ui mode
The processed files counter is updated whenever a file for the source is processed.

### fix: Unsupported files are now counted as unsupported files
There were previously counted as discarded files.

### fix: The name of the sidecar file is correctly written in the log

### fix: [[#272](https://github.com/simulot/immich-go/issues/272)] Wrong release downloaded for 0.15.0
Oops!

## Release 0.15.0

### fix [#255](https://github.com/simulot/immich-go/issues/255) Last percents of google puzzle solving are very slow when processing very large takeout archive
The google puzzle solving is now much faster for large takeout archives.

### fix [#215](https://github.com/simulot/immich-go/issues/215) Use XDG_CONFIG_HOME for storing config
The configuration file that contains the server and the key is now stored by default in following folder:
- Linux `$HOME/.config/immich-go/immich-go.json`
- Windows `%AppData%\immich-go\immich-go.json`
- Apple `$HOME/Library/Application Support/immich-go/immich-go.json` 

### Store the log files into sensible dir for user's system
The default log file is: 
- Linux `$HOME/.cache/immich-go/immich-go_YYYY-MM-DD_HH-MI-SS.log`
- Windows `%LocalAppData%\immich-go\immich-go_YYYY-MM-DD_HH-MI-SS.log`
- Apple `$HOME/Library/Caches/immich-go/immich-go_YYYY-MM-DD_HH-MI-SS.log`

### Feat: [[#249](https://github.com/simulot/immich-go/issues/249)] Fix Display the path of log file name
The log file name is printed when the program exits.


## Release 0.14.1

### fix [#246](https://github.com/simulot/immich-go/issues/246) Google Takeout 403 Forbidden on non admin user
Uses the endpoint /api/asset/statistics to get the number of user's assets.

## Release 0.14.0 "A better UI"

This release is focussed the improvement of the user experience. 

### A shiny user interface

```
. _ _  _ _ . _|_     _  _ 
|| | || | ||(_| | ─ (_|(_)
               v dev _)   

```

Working with big database and big takeout files take some time. Users are now informed about the progression of different tasks:

![image](/docs/render1716187129166.gif)

The screen presents number of processed photos, how they have been processes, the messages log, and at the bottom, the progression of the 3 mains tasks.


### A minimalist user interface

This shiny interface can be be disabled for quieter user interface (`-no-ui`).
The progression is visible. All details on operations are listed in the log file.


```
. _ _  _ _ . _|_  __  _  _ 
|| | || | ||(_| |    (_|(_)
      version dev     _)   

Server status: OK
Connected, user: demo@immich.app
Immich read 100%, Google Photos Analysis: 100%, Uploaded 100%  

Input analysis:
----------------------
scanned image file                      :   25420
scanned video file                      :    1447
scanned sidecar file                    :   26934
discarded file                          :     197
unsupported file                        :       0
file duplicated in the input            :    1706
associated metadata file                :   26867
missing associated metadata file        :       0

Uploading:
----------
uploaded                                :   25160
server error                            :       0
file not selected                       :       1
server's asset upgraded with the input  :       0
server has same photo                   :       0
server has a better asset               :       0
```

### Immich-go runs simultaneously the collect of immich-server's assets and the analysis of the Google takeout

The code has been refactored to run several task simultaneously to animate the progression screen. The program runs now the reading of immich asset and the the takeout analysis in parallel.

### Immich-go now always produces a log file 

The default name for the log file is `immich-go YYYY-MM-DD HH-MI-SS.log`, located in the current directory.

It's possible to give a path and a name to the log file with the option `-log-file=path/to/file.log`. 
If the file file exists already, the new messages will be added to its end.

The log level `OK` is removed.

### Immich-go is published under the AGPL-3.0 license

I chose the same license as the immich project license to release immich-go. 

### Next steps

- Issues closing
- A shiny user interface for the command `duplicate`


## Release 0.13.2

### Fix [[#211](https://github.com/simulot/immich-go/issues/211)]  immich-go appears to retain/cache an API key
Fix the logic for retaining the credential: 

When given, the credentials are saved into the configuration file.
When not given, the credential are read from the configuration file if possible.
 
## Release 0.13.0

### Improvement: [#189](https://github.com/simulot/immich-go/issues/189) Use a configuration file to store server's address and its API key  

The server URL and the API key are now stored into a configuration file (by default $HOME/.immich-go/immich-go.json).
If not provided in the CLI argument, those values are read from the configuration file.

The option `-use-configuration=path/to/config/file` let you specify the configuration file. 

### fix: [#193](https://github.com/simulot/immich-go/issues/193) Flags not being passed to subcommands #193


### Improvement: Better handling of wild cards in path 
`Immich-go` now accepts to handle path like `photos/Holydays*`. This, combined with the `-create-album-folder` will create 
an album per folder Holydays*.

It can handle patterns like : /photo/\*/raw/\*.dmg

### fix: Append Log #182
Log are now appended to the log file


## Release 0.12.0

### fix: #173 [Feature Request:] Set date from file system timestamp
When there is no date of take neither in the file name nor in EXIF data, the photo is uploaded with the file modification date.
This behavior can be changed with the option:
`-when-no-date FILE|NOW` 

## Release 0.11.0

### fix: stack command fails #169 
The immich version 1.95 changed the /asset api endpoint used to during the stack operation.
API call have been changed to match with new api endpoints

### fix: #140 Device UUID is not set
The option `-device-uuid VALUE` was not functional.

## Release 0.10.0

### fix: #135 feat: use the searchAssets API to workaround server's timeout

immich-go calls the endpoint `searchAssets` that provides a pagination system. 
This will avoid timeouts at the start of immich-go whit a busy server.


## Release 0.9.12

### fix: #131 panic syntax error in pattern

Some file names trigger a panic when checking the existence of XMP file



## Release 0.9.11

### fix: #130 Support RW2  
Add the support of Panasonic RW2 file format


## Release 0.9.10

### fix: #128 Parenthesis Name Only Error?
Some google photos users have lot of files named .jpg, (1).jpg, (x).jpg. Those names aren't makes immich-server crashing or timeout.
When such files are encountered, they are now uploaded with the name "No Name".

### fix: #125 XMP filenames don't always match what is expected 

For an named ABC.jpg, immich-go will check the presence of a XMP files in the following order

1. ABC.jpg.xmp
2. ABC.xmp
3. ABC.*.xmp

The latest allow the take a XMP file named ABC.RAW.xmp and associate it with the ABC.jpg
Note: when stacking is used, the xmp is visible only withe the cover file (immich-server behavior). 

### fix: flag -create-stacks not honoured

## Release 0.9.9

### fix: #123 Trying to use your go module
Removing utf-8 from tests file names

### fix: #129 BUG -exclude-types is not working
Command line options `-exclude-types` and `-select-types` are functional again.

## Release 0.9.8

### fix: XMP are rejected #120 
xmp files present aside assets are now correctly uploaded to immich.

## Release 0.9.7

### fix: XMP files are generated and uploaded to immich with importing strait folders #118 
In some conditions, and additional call is made to update the asset. This creates an XMP file on the server, with coordinate 0,0

### fix: Logs should not use colors code by default under windows OS #117
Most of windows terminals are still not able to understand ANSI colors sequences.


## Release 0.9.6

### feat: control archived Google photos
`-discard-archived` deactivate the import of archived photos.

### feat: better handling of boolean flags
Just mention the `-flag` to activate the functionality

Example:
`... upload -create-album-folder  ...` is now sufficient to activate the create album based on folder option
create-albums

Example: to deactivate a flag that is on by default:
`... upload -create-albums=FALSE ...` deactivate the album creation from google photos 


## Release 0.9.5

### fix: panic: runtime error: invalid memory address or nil pointer dereference at github.com/simulot/immich-go/cmdupload/upload.go:255

## Release 0.9.4

### fix: fixed incorrect bool value (#105)

## Release next

### fix: #108 less alarming message for unsupported file types

### feat: better import journal

The 1st section of the report gives information on the files found in the input.
The 2nd section explain what immich-go did with the files

```
Scan of the sources:
 53998 files in the input
--------------------------------------------------------
 25420 photos
  1447 videos
 26934 metadata files
 26867 files with metadata
   120 discarded files
     0 files having a type not supported
    77 discarded files because in folder failed videos
 53998 input total (difference 0)
--------------------------------------------------------
 11409 uploaded files on the server
    12 upgraded files on the server
   173 files already on the server
     1 discarded files because of options
  1529 discarded files because duplicated in the input
 13743 discarded files because server has a better image
     0 errors when uploading
 26867 handled total (difference 0)
```
### feat: #95 transfer GP description and favorite to immich


## Release 0.9.3

### feat: added ability to skip ssl verification (#103) 
The flag `-skip-verify-ssl=TRUE` permit a connection with an immich server using a self signed certificate.

Thank to sigmahour

## Release 0.9.2

### feat: added ability to skip ssl verification (#103) 
The flag `-skip-verify-ssl=TRUE` permit a connection with an immich server using a self signed certificate.

Thank to sigmahour

## Release 0.9.2

### fix Trim leading slashes from -server flag
This fixes the error invalid character '<' looking for beginning of value when the URL of the service ends with a `/`.

### fix .gitignore mistake

Thank to Erikas


## Release 0.9.1

### fix: stack: Samsung #99 
Now,Samsung bursts are detected

### fix:stack: Huawei Nexus 6P #100 
Now, Huawei bursts are detected


## Release 0.9.0

### feat:transfer google-photo favorite to immich
The favorite status in google photos is now replicated in immich.

### feat:  Add a flag to enable only stacking of RAW+JPG and NOT bursts #83 
It's now possible to control if stacks must be created for:
- couples raw + jpg
- burst of photos



### fixes: 
- jpg must be the cover of a raw+jpg stack
- stack: for Pixel 5 and Pixel 8 Pro naming schemes #94 
- Live photos files are stacked and not recognized as live photos #67 

## Release 0.8.9

### fix: A lot of images skipped from Google Photos Takeout #68

Improvement for the takeout import. 
- The log indicate with JSON is associated to an file.
- JSON and files are associated by applying successively rules from the most common to the strangest one
   
   Each JSON is checked. JSON is duplicated in albums folder.<br>
   Associated files with the JSON can be found in the JSON's folder, or in the Year photos.<br>

   Once associated and sent to the main program, files are tagged for not been associated with an other one JSON.<br>
   Association is done with the help of a set of matcher functions. Each one implement a rule<br>
   1 JSON can be associated with 1+ files that have a part of their name in common.<br>
   -   the file is named after the JSON name<br>
   -   the file name can be 1 UTF-16 char shorter (🤯) than the JSON name<br>
   -   the file name is longer than 46 UTF-16 chars (🤯) is truncated. But the truncation can creates duplicates,  then a number is added.
   -   if there are several files with same original name, the first instance kept as it is, the next have a a sequence number. File is renamed as IMG_1234(1).JPG and the JSON is renamed as IMG_1234.JPG(1).JSON
   -   of course those rules are likely to collide. They have to be applied from the most common to the least one.<br>
   -   sometimes the file isn't in the same folder than the json... It can be found in Year's photos folder<br>
   The duplicates files (same name, same length in bytes) found in the local source are discarded before been <br>presented to the immich server.
 Release 0.8.8<br>

### fix for #86: unknown time zone Argentina/Buenos_Aires

On some systems, the time zone name is not well recognized.

The new command line option set the time zone used by the program.
`-time-zone=time_zone_name`


## Release 0.8.7

### fix for #82: PM files causing server's bad request 
Android Live photos are delivered as one JPG and a MP files.
MP is the small movie.
The JPG files embeds this movie. 

The Immich server detects live photos without the help of the MP files. Just ignore them.

### Log Improvement

It's always difficult to understand how any file is handled. 

By setting the `-log-level=INFO`, immich-go now produce a comprehensive report on actions done with each files.
Example:
```
...
Server has photo         : Takeout/Google Photos/Photos from 2022/IMG_0135.HEIC: An asset with the same name:"IMG_0135", date:"2022-03-02 21:16:50" and size:3.5 MB exists on the server. No need to upload.
Server's asset is better : Takeout/Google Photos/Photos from 2022/PXL_20220916_140005541.jpg: An asset with the same name:"PXL_20220916_140005541" and date:"2022-09-16 16:00:05" but with bigger size:2.4 MB exists on the server. No need to upload.
Server has photo         : Takeout/Google Photos/Photos from 2022/PXL_20220807_181842919.jpg: An asset with the same name:"PXL_20220807_181842919", date:"2022-08-07 20:18:42" and size:881.1 KB exists on the server. No need to upload.
Server's asset is better : Takeout/Google Photos/Photos from 2022/PXL_20220907_081516648.jpg: An asset with the same name:"PXL_20220907_081516648" and date:"2022-09-07 10:15:16" but with bigger size:1.9 MB exists on the server. No need to upload.
Server's asset is better : Takeout/Google Photos/Photos from 2022/PXL_20220620_063743673.jpg: An asset with the same name:"PXL_20220620_063743673" and date:"2022-06-20 08:37:43" but with bigger size:1.8 MB exists on the server. No need to upload.
Local duplicate          : Takeout/Google Photos/Cat🐱/PXL_20220428_204810437.jpg: PXL_20220428_204810437.jpg
Uploaded                 : Takeout/Google Photos/Photos from 2022/PXL_20220428_204810437.jpg: PXL_20220428_204810437.jpg
Added to an album        : Takeout/Google Photos/Photos from 2022/PXL_20220428_204810437.jpg: Cat🐱
Server's asset is better : Takeout/Google Photos/Photos from 2022/PXL_20220526_181202808.jpg: An asset with the same name:"PXL_20220526_181202808" and date:"2022-05-26 20:12:02" but with bigger size:2.5 MB exists on the server. No need to upload.
... 
```

### Write log into a file

With the option `-log-file filename`, immich-go write all messages into the given file.


## Release 0.8.6

### fix for #68: A lot of images skipped from Google Photos Takeout

The Google takeout archive is full of traps. The difficulty is to associate all images with a JSON.
Now more files are now imported. There still few missing files, but they are now listed.

The program now reports how files are handled, or discarded.

```
Upload report:
 53998 scanned files
 53993 handled files
 26937 metadata files
   535 uploaded files on the server
    49 upgraded files on the server
  1540 duplicated files in the input
  8382 files already on the server
    77 discarded files because in folder failed videos
     1 discarded files because of options
 16470 discarded files because server has a better image
     1 files type not supported
     1 errors
     5 files without metadata file
7 files can't be handled
File: Takeout/Google Photos/Photos from 2019/1556189729458-8d2e2d13-bca5-467e-a242-9e4cb238e(1).jpg
        File unhandled, missing JSON
File: Takeout/Google Photos/Photos from 2022/original_1d4caa6f-16c6-4c3d-901b-9387de10e528_P(1).jpg
        File unhandled, missing JSON
File: Takeout/Google Photos/Photos from 2022/original_af12c386-e334-4c57-88be-fdfadea71f16_P(1).jpg
        File unhandled, missing JSON
File: Takeout/Google Photos/Photos from 2022/original_ec8d7b93-cbec-49c8-8707-38841db5e37d_P(1).jpg
        File unhandled, missing JSON
File: Takeout/Google Photos/Photos from 2023/original_d3671642-c937-49c0-917a-8ef9cbb449c5_P(1).jpg
        File unhandled, missing JSON
File: Takeout/Google Photos/user-generated-memory-titles.json
        Error , json: cannot unmarshal array into Go struct field GoogleMetaData.title of type string
File: Takeout/archive_browser.html
        File type not supported
Done.
```


The plenty of rules for associating image to JSON are somewhat contradictory. I have to rethink the system for applying  
rules from the most common to the strangest ones.

Still lot of work to deliver.


## Release 0.8.5

### fix for #78: mp4-files do not get imported

Thanks to @Zack and @jrasm91 to have nailed the problem.
 

## Release 0.8.4

###  fix for #67 : Live photos files are stacked and not recognized as live photos
Live photos are recognized when importing folders and google takeout archives

## Release 0.8.3

### New features include / exclude list of extensions
`-select-types .ext,.ext,.ext...` List of accepted extensions. <br>
`-exclude-types .ext,.ext,.ext...` List of excluded extensions. <br>

It's now possible to import only .jpg and .heic

Or it's possible to import everything but .heic

## Release 0.8.3

### Fix for #69: Panic: runtime error: slice bounds out of range
Rewriting searchPattern
Add tests

## Release 0.8.2

### Fix for #64: segfault when using *.jpg

The error is fixed.
I'll add later an option for selecting only an extension.


## Release 0.8.1

### workaround for #62
This prevents the error:
```
Bad Request
albumName should not be empty
```
Still working on the root cause

## Release 0.8.0

### New feature: create stacks when uploading images
The option `-create-stacks <bool>` drive the creation of stack of images for couples JPG/RAW or bursts of photos. The option is enabled by default.

Your asset must have the date of capture in the metadata.

Example:
```sh
./immich-go -server=http://mynas:2283 -key=zzV6k65KGLNB9mpGeri9n8Jk1VaNGHSCdoH1dY8jQ upload
 ~/mnt/sdcard/           

Server status: OK
Ask for server's assets...
....
Done, total 12 uploaded
Creating stacks
  Stacking 3H2A0018.CR3, 3H2A0018.JPG...
  Stacking 3H2A0019.CR3, 3H2A0019.JPG...
  Stacking 3H2A0020.CR3, 3H2A0020.JPG...
  Stacking 3H2A0021.CR3, 3H2A0021.JPG...
  Stacking 3H2A0022.CR3, 3H2A0022.JPG...
  Stacking 3H2A0023.CR3, 3H2A0023.JPG...
12 media scanned, 12 uploaded.
Done.
```


### New feature: command tool album delete \[regexp pattern\]

Delete albums that match with the regexp pattern. 

## Release 0.7.0

### Fix #52: Duplicate command fails with 504 timeout
A new option enable the possibility of calling directly the immich-server once it's port is published.

`-api URL` URL of the Immich api endpoint (http://container_ip:3301)<br>


### Fix #52: Duplicate command fails with 504 timeout
A new option enable the possibility of calling directly the immich-server once it's port is published.

`-api URL` URL of the Immich api endpoint (http://container_ip:3301)<br>


### Fix #50:  Duplicate detection fails when timezone of both images differs
Imported duplicated images with same name, but different timezone wasn't seen as duplicates.
The `-ignore-tz-errors=true` compares the time on date, and minute and ignores the hour of capture.



## Release 0.6.0

New options for Google Phots albums:
`-keep-untitled-albums <bool>` Untitled albums are imported into `immich` with the name of the folder as title (default: FALSE).<br>
`-use-album-folder-as-name <bool>` Use the folder's name instead of the album title (default: FALSE).<br>

### More integration tests
- For the command upload
- For Date based on file names


### Fix #53 import from folders always creates albums
Now albums are created only when requested.

### Fix #48: Import from google takeout duplicates albums with special characters
By default, Album are named after their title in JSON file, Special characters are allowed.
The option `-use-album-folder-as-name=TRUE` names albums after the folder name instead of their title

### Fix #42 [google photos] Lots of "No title" albums created with 1 file each  
By default untitled albums are not created.
Use the option `-keep-untitled-albums=TRUE` to keep them. 

### Fix #51 Import a single file doesn't work
It's now possible to import one file.

## Release 0.5.0

### Use the new stacking feature to group jpg and raw images, same for burst

Command `stack` added to stack images present in immich

### Fix #47: error when importing from a folder

PANIC when a file wasn't readable because of rights.

### Readme restructured

The Readme was too massive. It's now split in parts:
- [readme.md](../readme.md)
- [doc/google-takeout.md](./google-takeout.mdgoo)
- [doc/motivation.md](./motivation.md)

### Better getAssets handling
Filter out trashed asset when getting the list from the server

### Use-the-new-stacking-feature-to-group-jpg-and-raw-images,-same-for-burst-#45

## Release 0.4.0

### Fix #44: duplicate is not working?

At 1st run of the duplicate command, low quality images are moved to the trash and not deleted as before 1.82.
At next run, the trashed files are still seen as duplicate.
The fix consist in not considering trashed files during duplicate detection


### Fix #39: another problems with Takeout archives

I have reworked the Google takeout import to handle #39 cases. Following cases are now handled:
- normal FILE.jpg.json -> FILE.jpg
- less normal FILE.**jp**.json -> FILE.jpg
- long names truncated FIL.json -> FIL**E**.jpg
- long name with number and truncated VERY-LONG-NAM(150).json -> VERY-LONG-**NAME**(150).jpg
- duplicates names in same folder FILE.JPG(3).json -> **FILE(3)**.JPG
- edited images FILE.JSON -> FILE.JPG and **FILE-edited**.JPG

Also, there are cases where the image JSON's title is totally not related to the JSON name or the asset name.
Images are uploaded with the name found in the JSON's title field.

Thank to @bobokun for sharing details.


## Release 0.3.6
### Fix #40: Error 204 when deleting assets

## Release 0.3.5

### Fix #35: weird name cases in google photos takeout: truncated name or jp.json
Here are some weird cases found in takeout archives

example:
image title: 😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛😝😜🤪🤨🧐🤓😎🥸🤩🥳😏😒😞😔😟😕🙁☹️😣😖😫😩🥺😢😭😤😠😡🤬🤯😳🥵🥶.jpg
image file: 😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛.jpg
json file: 😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋.json

example:
image title: PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGINAL.jpg
image file: PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGINA.jpg
json file: PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGIN.json

example:
image title: 05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jpg
image file: 05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jpg
json file: 05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jp.json



### Fix #32: Albums contains album's images and all images having the same name
Once, I have a folder full of JSON files for an album, but it doesn't have any pictures. Instead, the pictures are in a folder organized by years. To fix this, I tried to match the JSON files with the pictures by their names.

The problem is that sometimes pictures have the same name in different years, so it's hard to be sure which picture goes with which JSON file. Because of this, created album contains image found in its folder, but also images having same name, taken in different years.

I decided to remove this feature. Now, if the image isn't found beside the JSON file, the JSON is ignored.


## Release 0.3.2, 0.3.3, 0.3.4
### Fix for #30 panic: time: missing Location in call to Time.In with release Windows_x86_64_0.3.1
Now handle correctly windows' timezone names even on windows.
Umm...

## Release 0.3.0 and 0.3.1
**Refactoring of Google Photo Takeout handling**

The takeout archive has flaws making the import task difficult and and error prone.
I have rewritten this part of the program to fix most of encountered error.

### google photos: can't find image of album #11 
Some image may miss from the album's folder. Those images files are located into the year folder. 
This fix looks for album images in the whole archive.

### photos with same name into the year folder #12 
Iphones and digital cameras produce images with the sequence number of 4 digits. This leads inevitably to have several images with the same number in the the year folder.

Google Photos disambiguates the files name by adding a counter at the end of the image file:
- IMG_3479.JPG
- IMG_3479(1).JPG
- IMG_3479(2).JPG

Surprisingly, matching JSON are named as 
- IMG_3479.JPG.json
- IMG_3479.JPG(1).json
- IMG_3479.JPG(2).json

This special case is now handled.

### Untitled albums are now handled correctly
Untitled albums now are named after the album's folder name.

This address partially the issue #19.

###  can't find the image with title "___", pattern: "___*.*": file does not exist: "___" #21 

The refactoring of the code don't use anymore a file pattern to find files in the archive. 
The image and the JSON file are quite identical, except for duplicate image (see #12) or when the file name is too long (how long is too long?).

Now, the program takes the image name, check if there is a JSON that matches, open it and use the title of the image to name the upload.

If the JSON isn't found, the image is uploaded with it's name in the archive, and with no date. Now all images are uploaded to immich, even when the JSON file is not found.

### MPG files not supported. #20 
Immich-go now accepts the same list of extension as the immich-server. This list is taken from the server source code.

### immich-go detects raw and jpg as duplicates #25 
The duplicate checker now uses the file name, its extension and the date of take to detect duplicates. 
So the system doesn't signal `IMG_3479.JPG` and `IMG_3479.CR2` as duplicate anymore.

### fix duplicate check before uploading #29
The date parsing now takes into account the time zone of the machine (ex: Europe/Paris). This handles correctly summer time and winter time. 
This isn't yet tested on Window or Mac machines.


## Release 0.2.3

- Improvement of duplicate command (issue#13)
  - `-yes` option to assume Yes to all questions
  - `-date` to limit the check to a a given date range
- Accept same type of files than the server (issue#15)
    - .3fr
    - .ari
    - .arw
    - .avif
    - .cap
    - .cin
    - .cr2
    - .cr3
    - .crw
    - .dcr
    - .dng
    - .erf
    - .fff
    - .gif
    - .heic
    - .heif
    - .iiq
    - .insp
    - .jpeg
    - .jpg
    - .jxl
    - .k25
    - .kdc
    - .mrw
    - .nef
    - .orf
    - .ori
    - .pef
    - .png
    - .raf
    - .raw
    - .rwl
    - .sr2
    - .srf
    - .srw
    - .tif
    - .tiff
    - .webp
    - .x3f
    - .3gp
    - .avi
    - .flv
    - .insv
    - .m2ts
    - .mkv
    - .mov
    - .mp4
    - .mpg
    - .mts
    - .webm
    - .wmv"
- new feature: add partner's assets to an album. Thanks to @mrwulf.
- fix: albums creation fails sometime

### Release 0.2.2
- improvement of date of capture when there isn't any exif data in the file
    1. test the file name for a date
    1. open the file and search for the date (.jpg, .mp4, .heic, .mov)
    1. if still not found, give the current date

> ⚠️ As of current version v1.77.0, immich fails to get the date of capture of some videos (IPhone), and place the video on the 01/01/1970.
> 
> You can use the -album to keep videos grouped in a same place despite the errors in date.

### Release 0.2.1
- Fix of -album option. uploaded images will be added into the album. Existing images will be added in the album if needed.

### Release 0.2.0
- When uploading from a directory, use the date inferred from the file name as file date.  Immich uses it as date of take. This is useful for images without Exif data.
- `duplicate` command check immich for several version of the same image, same file name, same date of capture
