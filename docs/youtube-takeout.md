# The YouTube Takeout case
Most of the flaws in the YouTube Takeout data are probably a result of optimizing for "normal" users dealing with the data, rather than machines.  Thankfully, it's largely possible to accurately and automatically import data from a YouTube Takeout into another program, except for the shortcomings noted below.

# Shortcomings

## Playlists with similar names

YouTube's algorithm serializes playlists to files uses the first ~47 characters of the playlist name as the stem of the filename.  If two playlists have the same 47 characters in common, only one of them is serialized to the playlist file, and the other is lost, and there is no way of knowing which one is which.

## Videos with similar names and/or file extensions
YouTube's algorithm for serializing videos to files uses the first ~43 characters of the video name as the stem of the filename.  After those 43 characters, a one-up counter is appended to the name, e.g., `(1)`.  It is not clear if the file the provides the video metadata nd the filename counters are synched, i.e., is the first video named "Example" written to `Example.mp4`, the second video named "Example" written to `Example(1).mp4`, etc.

Additionally, the file that provides video metadata does not provide any information about the video's file extension, but YouTube Takeout videos can have multiple file extensions.  If the Takeout includes both `video.mp4` and `video.wmv` and `video.avi` it is highly likely that the video file and metadata will not be paired correctly.

## Missing videos
YouTube Takeout sometimes fails to include individual video files for unknown reasons.  The metadata appears to still be present but the video itself is mssing.  In this case `immich-go` will report an error about the missing video file.

## Playlist descriptions
Both the YouTube channel and and playlists are turned into Immich albums.  Both YouTube channels and playlists can include descriptions, but `immich-go` does not currently support adding these descriptions to albums, so they are lost.

Video descriptions are preserved.

# What if you have problems with a takeout archive?
Please open an issue with details. Tag the issue with `@thecabinet`.
