
# Better asset selection

The local file is analyzed to get following data:
- file size in bytes
- date of capture took from the takeout metadata, the exif data, or the file name with possible. The key is made of the file name + the size in the same way used by the immich server.

Digital cameras often generate file names with a sequence of 4 digits, leading to generate duplicated names. If the names matches, the capture date must be compared.

Tests are done in this order
1. the key is found in immich --> the name and the size match. We have the file, don't upload it.
1. the file name is found in immich and...
    1. dates match and immich file is smaller than the file --> Upload it, and discard the inferior file
    1. dates match and immich file is bigger than the file --> We have already a better version. Don't upload the file. 
1. Immich don't have it. --> Upload the file.
1. Update albums
