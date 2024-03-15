# Release notes 

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
