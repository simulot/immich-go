/*
This package handles google photos takeout files.

It determines the original file name, the date of capture and gps data using the json companion file of any photos.
It identifies the albums.

Please, remember that google takeout doesn't provide all information for rebuilding albums, and give back the original name of the photos.

- image and related json may be into another part of the archive.
- album's images are generally present into the album folder, but not always.
- edited images may match with the original image's JSON, but not always.
- all images taken at the same year are placed into the same directory
  - images with the same names have a sequence like IMG_3479(2).JPG,
  - the matching JSON is IMG_3479.JPG(2).json
  - the image's title is IMG_3479.JPG
*/
package gp
