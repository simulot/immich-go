# Importing from Flickr

This page describes how `immich-go` handles a Flickr export.
Flickr's export format is simpler than Google Photos takeout but has its own quirks —
particularly around how image filenames embed photo IDs and how album membership is stored.

## What is a Flickr export?

Flickr's "Request your account data" feature (available at flickr.com/account) produces
a set of ZIP files. There are two distinct types:

- **Metadata archive** — has an opaque set-ID filename such as
  `72157724905358313_3ac1836c2225_part1.zip`. Inside you will find `albums.json`
  (a list of every album and its member photo IDs) and one `photo_<id>.json` per photo.
  The adapter identifies this archive by the presence of `albums.json` at the ZIP root.

- **Image archives** — named `data-download-*.zip`. These contain only image and video
  files; there is no JSON metadata inside them.

For large accounts Flickr may split the image download across several archives.
The metadata archive is always a single file.

## Image filename formats

Flickr image filenames follow one of two patterns depending on when the photo was uploaded:

| Format  | Pattern | Example |
| ------- | ------- | ------- |
| Current | `<slug>_<photoID>_o.<ext>` | `sunset-at-the-beach_54321_o.jpg` |
| Older   | `<photoID>_<hexhash>_o.<ext>` | `54321_ab1cd2ef3a_o.jpg` |

Both patterns are recognised automatically. The photo ID embedded in the filename is
the key used to link each image file to its JSON metadata.

## Album membership

Album membership is stored **exclusively** in `albums.json` at the root of the metadata
archive. The per-photo `photo_<id>.json` files always contain an empty `albums` field —
do not rely on them for album data. The adapter inverts `albums.json` into an internal
map of photo ID → album titles and attaches that data during asset assembly.

## Usage

```sh
immich-go upload from-flickr \
  --server http://your-immich-server:2283 \
  --api-key YOUR_KEY \
  [--sync-albums=true] \
  <metadata.zip> <images.zip>...
```

- `<metadata.zip>` is the archive that contains `albums.json` (the adapter identifies it
  automatically — you do not need to specify which ZIP is which, just pass them all).
- One or more `<images.zip>` arguments follow for the image archives.
- `--sync-albums` (default: `true`) creates albums in Immich that match the album
  structure in your Flickr export.

## Metadata preserved

| Flickr field | Immich field | Notes |
| ------------ | ------------ | ----- |
| `name` | Title / OriginalFileName | Falls back to the image filename when empty |
| `description` | Description | |
| `date_taken` | CaptureDate | Parsed as `"2006-01-02 15:04:05"` in local time |
| `date_upload` | CaptureDate (fallback) | Unix epoch; used only when `date_taken` is absent or unparseable |
| `tags[].tag` | Tags | All tags attached to the photo |
| `albums.json` entries | Albums | Reconstructed from album membership in `albums.json` |

## Metadata not preserved

The following data is present in a Flickr export but is **not** imported:

- **GPS / geo coordinates** — the `geo` field is present in per-photo JSON files but was
  empty in all tested real-world exports; it is not imported.
- **Comments**
- **Groups**
- **Favorites / faves**

## How the adapter works

These notes are intended for contributors who want to understand or modify the adapter.

**Archive classification** — At startup the adapter calls `fs.Stat` on each provided
archive to check whether `albums.json` is present. The archive that contains it is
classified as the metadata archive; all others are treated as image archives. The adapter
returns an error if zero archives or two or more archives contain `albums.json`.

**Photo ID extraction** — Two package-level compiled regexes extract the numeric photo ID
from each image filename. The primary pattern (`_(\d+)_o\.\w+$`) covers the current
`slug_ID_o.ext` format. The fallback pattern (`^(\d+)_[0-9a-f]+_o\.\w+$`) handles the
older `ID_hash_o.ext` format. The base filename is stripped of directory prefixes before
matching.

**Two-pass processing** — Pass one walks every image archive and builds a catalog keyed by
photo ID, then walks the metadata archive to parse all `photo_<id>.json` files and
`albums.json`. Pass two iterates the catalog and emits one `assets.Group` per photo,
attaching metadata and album assignments.

**Missing metadata** — A photo whose image file has no matching JSON entry is logged with
a `ProcessedMissingMetadata` event and still emitted (without title, description, or tags).
Malformed JSON files are also logged and skipped; they do not abort the import.

## Known limitations

- Exactly one metadata archive (one ZIP containing `albums.json`) is required. The adapter
  returns an error if none or more than one is found among the provided files.
- Geo/GPS data is not imported (empty in all tested real-world exports).
- Video support depends on Flickr's export format; the adapter passes video files through
  the standard media-type filter and imports them if the type is recognised.

## What if something goes wrong?

Please open an issue with details.
You can share files or logs via Discord DM `@simulot`.
