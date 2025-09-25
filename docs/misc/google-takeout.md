# The Google Photos takeout case
This project aims to make the process of importing Google Photos takeouts as easy and accurate as possible. But keep in mind that 
Google takeout structure is complex and not documented. Some information may be missed or may even be wrong. 

## Folders in takeout
  - The Year folder contains all images taken that year
  - Albums are in separate folders named as the album
    - A json file contains the album title
    - The title can be empty
    - The JSON is named in the user's language : metadata.json métadonnées.json, metadatos.json, metadades.json ...
    - Contains all of an album's images, most of the time
    - Images are also in the year folders if you have them. 
  - The trash folder is names in the user's language Trash, Corbeille..
    - Hopefully, the JSON has a Trashed field.
  - The "Failed Videos" contains unreadable videos
  - Untitled albums are named in the user's language and a number: Untitled, Sin título, Sans Titre 
  - All untitled albums can be stored into the same folder. Therefore it's impossible to rebuild original albums.

## Images have a JSON companion file
  - the JSON contains some information on the image
    - The title has the original image name (as uploaded into GP server, that can be totally different of the image name in the archive)
    - the date of capture (epoch)
    - the GPS coordinates
  - Trashed flag
  - Partner flag
  - But not all images have a JSON companion

## The JSON file and the image name matches with some weird rules
The name length of the image can be shorter by 1 char compared to the name of the JSON.

### 2+ different images having the same name taken the same year are placed into the same folder with a number
  - IMG_3479.JPG
  - IMG_3479(1).JPG
  - IMG_3479(2).JPG

#### In that case, the JSONs are named:
  - IMG_3479.JPG.json
  - IMG_3479.JPG(1).json
  - IMG_3479.JPG(2).json

### Edited images may not have corresponding JSON.
two images
  - PXL_20220405_090123740.PORTRAIT.jpg
  - PXL_20220405_090123740.PORTRAIT-modifié.jpg

but one JSON
  - PXL_20220405_090123740.PORTRAIT.jpg.json

Note that "edited" name is localized.

## Images are duplicated with no apparent logic
Example from the  [#380](https://github.com/simulot/immich-go/issues/380)
```sh
~$ for f in *.zip; do echo "$f: "; unzip -l $f | grep 130917ad28385b5a; done
takeout-20240712T112341Z-001.zip:
   167551  2024-07-12 13:38   Takeout/Google Foto/1 anno fa/130917ad28385b5a-photo.jpg
      808  2024-07-12 13:38   Takeout/Google Foto/1 anno fa/130917ad28385b5a-photo.jpg.json
takeout-20240712T112341Z-002.zip:
takeout-20240712T112341Z-003.zip:
      808  2024-07-12 13:52   Takeout/Google Foto/Photos from 2022/130917ad28385b5a-photo.jpg.json
takeout-20240712T112341Z-004.zip:
   167551  2024-07-12 13:45   Takeout/Google Foto/Photos from 2022/130917ad28385b5a-photo.jpg
takeout-20240712T112341Z-005.zip:
takeout-20240712T112341Z-006.zip:
takeout-20240712T112341Z-007.zip:
takeout-20240712T112341Z-008.zip:
takeout-20240712T112341Z-009.zip:
      808  2024-07-12 14:33   Takeout/Google Foto/Amsterdam 2022/130917ad28385b5a-photo.jpg.json
   167551  2024-07-12 14:33   Takeout/Google Foto/Amsterdam 2022/130917ad28385b5a-photo.jpg
      808  2024-07-12 14:35   Takeout/Google Foto/1 anno fa/130917ad28385b5a-photo.jpg.json
takeout-20240712T112341Z-010.zip:
   167551  2024-07-12 14:35   Takeout/Google Foto/1 anno fa/130917ad28385b5a-photo.jpg
```


## Some key file names are spelled in the user language

| Language   | Google Photos folder | Album's metadata | Edited photo name |
| ---------- | -------------------- | ---------------- | ----------------- |
| US English | Google Photos        | metadata.json    |                   |
| French     | Google Photos        | métadonnées.json | \*-modifié.\*     |
| Italian    | Google Foto          | metadati.json    | \*-modificato.\*  |
| Slovak     | Fotky Google         | metadáta.json    |                   |
| German     | Google Fotos         | Metadaten.json   | \*-bearbeitet.\*  |
| Russian    | Google Фото          | метаданные.json  |                   |
| Bulgarian  | Google Снимки        | метаданни.json   |                   |



## The "-edited" suffix in different languages
TODO: verify the translations

Here is the list of translations for the "-edited" suffix in the requested languages presented in a table:

| Language   | Translation      | Confirmed |
| ---------- | ---------------- | --------- |
| Spanish    | -editado         | [ ]       |
| French     | -modifié         | [X]       |
| German     | -bearbeitet      | [X]       |
| Italian    | -modificato      | [X]       |
| Portuguese | -editado         | [ ]       |
| Russian    | -отредактировано | [ ]       |
| Chinese    | -编辑            | [ ]       |
| Japanese   | -編集済み        | [ ]       |
| Korean     | -편집됨          | [ ]       |
| Dutch      | -bewerkt         | [ ]       |
| Finnish    | -muokattu        | [ ]       |
| Ukrainian  | -відредаговано   | [ ]       |
| Polish     | -edytowane       | [ ]       |
| Danish     | -redigeret       | [ ]       |
| Slovak     | -upravené        | [ ]       |


# What if I have problems with a takeout archive?
Please open an issue with details. You can share your files using Discord DM `@simulot`.
I'll check if I can improve the program.
Sometimes a manual import is the best option.
