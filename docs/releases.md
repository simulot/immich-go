# Release notes 

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
