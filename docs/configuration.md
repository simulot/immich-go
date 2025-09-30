# Configuration File

The configuration file can be a `TOML`, `YAML` or `JSON` file. By default, `immich-go` looks for a file named `immich-go.toml` in the current directory.

## Configuration file structure

````
---
title: TOML
---
```toml
dry-run = false
log-file = ''
log-level = 'INFO'
log-type = 'text'
save-config = false

[archive]
dry-run = false
write-to-folder = ''

[archive.from-folder]
album-path-joiner = ' / '
album-picasa = false
date-from-name = true
exclude-extensions = []
folder-as-album = ''
folder-as-tags = false
ignore-sidecar-files = false
include-extensions = []
include-type = ''
into-album = ''
recursive = true
session-tag = false

[archive.from-folder.ban-file]

[archive.from-folder.date-range]
After = 0001-01-01T00:00:00Z
Before = 0001-01-01T00:00:00Z

[archive.from-folder.tag]

[archive.from-google-photos]
exclude-extensions = []
from-album-name = ''
include-archived = true
include-extensions = []
include-partner = true
include-trashed = false
include-type = ''
include-unmatched = false
include-untitled-albums = false
partner-shared-album = ''
people-tag = true
session-tag = false
sync-albums = true
takeout-tag = true

[archive.from-google-photos.ban-file]

[archive.from-google-photos.date-range]
After = 0001-01-01T00:00:00Z
Before = 0001-01-01T00:00:00Z

[archive.from-google-photos.tag]

[archive.from-immich]
from-api-key = ''
from-api-trace = false
from-archived = false
from-client-timeout = 1200000000000
from-favorite = false
from-minimal-rating = 0
from-server = ''
from-skip-verify-ssl = false
from-trash = false

[archive.from-immich.from-date-range]
After = 0001-01-01T00:00:00Z
Before = 0001-01-01T00:00:00Z

[stack]
admin-api-key = ''
api-key = ''
api-trace = false
client-timeout = 1200000000000
device-uuid = 'gl65'
dry-run = false
manage-burst = 0
manage-epson-fastfoto = false
manage-heic-jpeg = 0
manage-raw-jpeg = 0
on-server-errors = 0
pause-immich-jobs = true
server = ''
skip-verify-ssl = false
time-zone = ''

[stack.date-range]
After = 0001-01-01T00:00:00Z
Before = 0001-01-01T00:00:00Z

[upload]
admin-api-key = ''
api-key = ''
api-trace = false
client-timeout = 1200000000000
concurrent-uploads = 12
device-uuid = 'gl65'
dry-run = false
no-ui = false
on-server-errors = 0
overwrite = false
pause-immich-jobs = true
server = ''
skip-verify-ssl = false
time-zone = ''

[upload.from-folder]
album-path-joiner = ' / '
album-picasa = false
date-from-name = true
exclude-extensions = []
folder-as-album = 'none'
folder-as-tags = false
ignore-sidecar-files = false
include-extensions = []
include-type = ''
into-album = ''
manage-burst = 0
manage-epson-fastfoto = false
manage-heic-jpeg = 0
manage-raw-jpeg = 0
recursive = true
session-tag = false

[upload.from-folder.ban-file]

[upload.from-folder.date-range]
After = 0001-01-01T00:00:00Z
Before = 0001-01-01T00:00:00Z

[upload.from-folder.tag]

[upload.from-google-photos]
exclude-extensions = []
from-album-name = ''
include-archived = true
include-extensions = []
include-partner = true
include-trashed = false
include-type = ''
include-unmatched = false
include-untitled-albums = false
manage-burst = 0
manage-epson-fastfoto = false
manage-heic-jpeg = 0
manage-raw-jpeg = 0
partner-shared-album = ''
people-tag = true
session-tag = false
sync-albums = true
takeout-tag = true

[upload.from-google-photos.ban-file]

[upload.from-google-photos.date-range]
After = 0001-01-01T00:00:00Z
Before = 0001-01-01T00:00:00Z

[upload.from-google-photos.tag]

[upload.from-icloud]
album-path-joiner = ' / '
album-picasa = false
date-from-name = true
exclude-extensions = []
folder-as-album = 'NONE'
folder-as-tags = false
ignore-sidecar-files = false
include-extensions = []
include-type = ''
into-album = ''
manage-burst = 0
manage-epson-fastfoto = false
manage-heic-jpeg = 0
manage-raw-jpeg = 0
memories = false
recursive = true
session-tag = false

[upload.from-icloud.ban-file]

[upload.from-icloud.date-range]
After = 0001-01-01T00:00:00Z
Before = 0001-01-01T00:00:00Z

[upload.from-icloud.tag]

[upload.from-immich]
exclude-extensions = []
from-api-key = ''
from-api-trace = false
from-archived = false
from-client-timeout = 1200000000000
from-favorite = false
from-minimal-rating = 0
from-server = ''
from-skip-verify-ssl = false
from-trash = false
include-extensions = []
include-type = ''

[upload.from-immich.date-range]
After = 0001-01-01T00:00:00Z
Before = 0001-01-01T00:00:00Z

[upload.from-immich.from-date-range]
After = 0001-01-01T00:00:00Z
Before = 0001-01-01T00:00:00Z

[upload.from-picasa]
album-path-joiner = ' / '
album-picasa = false
date-from-name = true
exclude-extensions = []
folder-as-album = 'NONE'
folder-as-tags = false
ignore-sidecar-files = false
include-extensions = []
include-type = ''
into-album = ''
manage-burst = 0
manage-epson-fastfoto = false
manage-heic-jpeg = 0
manage-raw-jpeg = 0
recursive = true
session-tag = false

[upload.from-picasa.ban-file]

[upload.from-picasa.date-range]
After = 0001-01-01T00:00:00Z
Before = 0001-01-01T00:00:00Z

[upload.from-picasa.tag]

[version]
```
````
````
---
title: YAML
---
```yaml
archive:
  dry-run: false
  from-folder:
    album-path-joiner: ' / '
    album-picasa: false
    ban-file: {}
    date-from-name: true
    date-range:
      after: 0001-01-01T00:00:00Z
      before: 0001-01-01T00:00:00Z
    exclude-extensions: []
    folder-as-album: ""
    folder-as-tags: false
    ignore-sidecar-files: false
    include-extensions: []
    include-type: ""
    into-album: ""
    recursive: true
    session-tag: false
    tag: {}
  from-google-photos:
    ban-file: {}
    date-range:
      after: 0001-01-01T00:00:00Z
      before: 0001-01-01T00:00:00Z
    exclude-extensions: []
    from-album-name: ""
    include-archived: true
    include-extensions: []
    include-partner: true
    include-trashed: false
    include-type: ""
    include-unmatched: false
    include-untitled-albums: false
    partner-shared-album: ""
    people-tag: true
    session-tag: false
    sync-albums: true
    tag: {}
    takeout-tag: true
  from-immich:
    from-api-key: ""
    from-api-trace: false
    from-archived: false
    from-client-timeout: 1200000000000
    from-date-range:
      after: 0001-01-01T00:00:00Z
      before: 0001-01-01T00:00:00Z
    from-favorite: false
    from-minimal-rating: 0
    from-server: ""
    from-skip-verify-ssl: false
    from-trash: false
  write-to-folder: ""
dry-run: false
log-file: ""
log-level: INFO
log-type: text
save-config: false
stack:
  admin-api-key: ""
  api-key: ""
  api-trace: false
  client-timeout: 1200000000000
  date-range:
    after: 0001-01-01T00:00:00Z
    before: 0001-01-01T00:00:00Z
  device-uuid: gl65
  dry-run: false
  manage-burst: 0
  manage-epson-fastfoto: false
  manage-heic-jpeg: 0
  manage-raw-jpeg: 0
  on-server-errors: 0
  pause-immich-jobs: true
  server: ""
  skip-verify-ssl: false
  time-zone: ""
upload:
  admin-api-key: ""
  api-key: ""
  api-trace: false
  client-timeout: 1200000000000
  concurrent-uploads: 12
  device-uuid: gl65
  dry-run: false
  from-folder:
    album-path-joiner: ' / '
    album-picasa: false
    ban-file: {}
    date-from-name: true
    date-range:
      after: 0001-01-01T00:00:00Z
      before: 0001-01-01T00:00:00Z
    exclude-extensions: []
    folder-as-album: none
    folder-as-tags: false
    ignore-sidecar-files: false
    include-extensions: []
    include-type: ""
    into-album: ""
    manage-burst: 0
    manage-epson-fastfoto: false
    manage-heic-jpeg: 0
    manage-raw-jpeg: 0
    recursive: true
    session-tag: false
    tag: {}
  from-google-photos:
    ban-file: {}
    date-range:
      after: 0001-01-01T00:00:00Z
      before: 0001-01-01T00:00:00Z
    exclude-extensions: []
    from-album-name: ""
    include-archived: true
    include-extensions: []
    include-partner: true
    include-trashed: false
    include-type: ""
    include-unmatched: false
    include-untitled-albums: false
    manage-burst: 0
    manage-epson-fastfoto: false
    manage-heic-jpeg: 0
    manage-raw-jpeg: 0
    partner-shared-album: ""
    people-tag: true
    session-tag: false
    sync-albums: true
    tag: {}
    takeout-tag: true
  from-icloud:
    album-path-joiner: ' / '
    album-picasa: false
    ban-file: {}
    date-from-name: true
    date-range:
      after: 0001-01-01T00:00:00Z
      before: 0001-01-01T00:00:00Z
    exclude-extensions: []
    folder-as-album: NONE
    folder-as-tags: false
    ignore-sidecar-files: false
    include-extensions: []
    include-type: ""
    into-album: ""
    manage-burst: 0
    manage-epson-fastfoto: false
    manage-heic-jpeg: 0
    manage-raw-jpeg: 0
    memories: false
    recursive: true
    session-tag: false
    tag: {}
  from-immich:
    date-range:
      after: 0001-01-01T00:00:00Z
      before: 0001-01-01T00:00:00Z
    exclude-extensions: []
    from-api-key: ""
    from-api-trace: false
    from-archived: false
    from-client-timeout: 1200000000000
    from-date-range:
      after: 0001-01-01T00:00:00Z
      before: 0001-01-01T00:00:00Z
    from-favorite: false
    from-minimal-rating: 0
    from-server: ""
    from-skip-verify-ssl: false
    from-trash: false
    include-extensions: []
    include-type: ""
  from-picasa:
    album-path-joiner: ' / '
    album-picasa: false
    ban-file: {}
    date-from-name: true
    date-range:
      after: 0001-01-01T00:00:00Z
      before: 0001-01-01T00:00:00Z
    exclude-extensions: []
    folder-as-album: NONE
    folder-as-tags: false
    ignore-sidecar-files: false
    include-extensions: []
    include-type: ""
    into-album: ""
    manage-burst: 0
    manage-epson-fastfoto: false
    manage-heic-jpeg: 0
    manage-raw-jpeg: 0
    recursive: true
    session-tag: false
    tag: {}
  no-ui: false
  on-server-errors: 0
  overwrite: false
  pause-immich-jobs: true
  server: ""
  skip-verify-ssl: false
  time-zone: ""
version: {}
```
````
````
---
title: JSON
---
```json
{
  "archive": {
    "dry-run": false,
    "from-folder": {
      "album-path-joiner": " / ",
      "album-picasa": false,
      "ban-file": {},
      "date-from-name": true,
      "date-range": {
        "After": "0001-01-01T00:00:00Z",
        "Before": "0001-01-01T00:00:00Z"
      },
      "exclude-extensions": null,
      "folder-as-album": "",
      "folder-as-tags": false,
      "ignore-sidecar-files": false,
      "include-extensions": null,
      "include-type": "",
      "into-album": "",
      "recursive": true,
      "session-tag": false,
      "tag": {}
    },
    "from-google-photos": {
      "ban-file": {},
      "date-range": {
        "After": "0001-01-01T00:00:00Z",
        "Before": "0001-01-01T00:00:00Z"
      },
      "exclude-extensions": null,
      "from-album-name": "",
      "include-archived": true,
      "include-extensions": null,
      "include-partner": true,
      "include-trashed": false,
      "include-type": "",
      "include-unmatched": false,
      "include-untitled-albums": false,
      "partner-shared-album": "",
      "people-tag": true,
      "session-tag": false,
      "sync-albums": true,
      "tag": {},
      "takeout-tag": true
    },
    "from-immich": {
      "from-api-key": "",
      "from-api-trace": false,
      "from-archived": false,
      "from-client-timeout": 1200000000000,
      "from-date-range": {
        "After": "0001-01-01T00:00:00Z",
        "Before": "0001-01-01T00:00:00Z"
      },
      "from-favorite": false,
      "from-minimal-rating": 0,
      "from-server": "",
      "from-skip-verify-ssl": false,
      "from-trash": false
    },
    "write-to-folder": ""
  },
  "dry-run": false,
  "log-file": "",
  "log-level": "INFO",
  "log-type": "text",
  "save-config": false,
  "stack": {
    "admin-api-key": "",
    "api-key": "",
    "api-trace": false,
    "client-timeout": 1200000000000,
    "date-range": {
      "After": "0001-01-01T00:00:00Z",
      "Before": "0001-01-01T00:00:00Z"
    },
    "device-uuid": "gl65",
    "dry-run": false,
    "manage-burst": 0,
    "manage-epson-fastfoto": false,
    "manage-heic-jpeg": 0,
    "manage-raw-jpeg": 0,
    "on-server-errors": 0,
    "pause-immich-jobs": true,
    "server": "",
    "skip-verify-ssl": false,
    "time-zone": ""
  },
  "upload": {
    "admin-api-key": "",
    "api-key": "",
    "api-trace": false,
    "client-timeout": 1200000000000,
    "concurrent-uploads": 12,
    "device-uuid": "gl65",
    "dry-run": false,
    "from-folder": {
      "album-path-joiner": " / ",
      "album-picasa": false,
      "ban-file": {},
      "date-from-name": true,
      "date-range": {
        "After": "0001-01-01T00:00:00Z",
        "Before": "0001-01-01T00:00:00Z"
      },
      "exclude-extensions": null,
      "folder-as-album": "none",
      "folder-as-tags": false,
      "ignore-sidecar-files": false,
      "include-extensions": null,
      "include-type": "",
      "into-album": "",
      "manage-burst": 0,
      "manage-epson-fastfoto": false,
      "manage-heic-jpeg": 0,
      "manage-raw-jpeg": 0,
      "recursive": true,
      "session-tag": false,
      "tag": {}
    },
    "from-google-photos": {
      "ban-file": {},
      "date-range": {
        "After": "0001-01-01T00:00:00Z",
        "Before": "0001-01-01T00:00:00Z"
      },
      "exclude-extensions": null,
      "from-album-name": "",
      "include-archived": true,
      "include-extensions": null,
      "include-partner": true,
      "include-trashed": false,
      "include-type": "",
      "include-unmatched": false,
      "include-untitled-albums": false,
      "manage-burst": 0,
      "manage-epson-fastfoto": false,
      "manage-heic-jpeg": 0,
      "manage-raw-jpeg": 0,
      "partner-shared-album": "",
      "people-tag": true,
      "session-tag": false,
      "sync-albums": true,
      "tag": {},
      "takeout-tag": true
    },
    "from-icloud": {
      "album-path-joiner": " / ",
      "album-picasa": false,
      "ban-file": {},
      "date-from-name": true,
      "date-range": {
        "After": "0001-01-01T00:00:00Z",
        "Before": "0001-01-01T00:00:00Z"
      },
      "exclude-extensions": null,
      "folder-as-album": "NONE",
      "folder-as-tags": false,
      "ignore-sidecar-files": false,
      "include-extensions": null,
      "include-type": "",
      "into-album": "",
      "manage-burst": 0,
      "manage-epson-fastfoto": false,
      "manage-heic-jpeg": 0,
      "manage-raw-jpeg": 0,
      "memories": false,
      "recursive": true,
      "session-tag": false,
      "tag": {}
    },
    "from-immich": {
      "date-range": {
        "After": "0001-01-01T00:00:00Z",
        "Before": "0001-01-01T00:00:00Z"
      },
      "exclude-extensions": null,
      "from-api-key": "",
      "from-api-trace": false,
      "from-archived": false,
      "from-client-timeout": 1200000000000,
      "from-date-range": {
        "After": "0001-01-01T00:00:00Z",
        "Before": "0001-01-01T00:00:00Z"
      },
      "from-favorite": false,
      "from-minimal-rating": 0,
      "from-server": "",
      "from-skip-verify-ssl": false,
      "from-trash": false,
      "include-extensions": null,
      "include-type": ""
    },
    "from-picasa": {
      "album-path-joiner": " / ",
      "album-picasa": false,
      "ban-file": {},
      "date-from-name": true,
      "date-range": {
        "After": "0001-01-01T00:00:00Z",
        "Before": "0001-01-01T00:00:00Z"
      },
      "exclude-extensions": null,
      "folder-as-album": "NONE",
      "folder-as-tags": false,
      "ignore-sidecar-files": false,
      "include-extensions": null,
      "include-type": "",
      "into-album": "",
      "manage-burst": 0,
      "manage-epson-fastfoto": false,
      "manage-heic-jpeg": 0,
      "manage-raw-jpeg": 0,
      "recursive": true,
      "session-tag": false,
      "tag": {}
    },
    "no-ui": false,
    "on-server-errors": 0,
    "overwrite": false,
    "pause-immich-jobs": true,
    "server": "",
    "skip-verify-ssl": false,
    "time-zone": ""
  },
  "version": {}
}
```
````
