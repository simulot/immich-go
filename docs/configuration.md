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
on-errors = 'stop'
save-config = false

[archive]
[archive.from-folder]
album-path-joiner = ' / '
album-picasa = false
date-from-name = true
date-range = '2024-01-15,2024-03-31'
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

[archive.from-folder.tag]

[archive.from-google-photos]
date-range = '2024-01-15,2024-03-31'
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

[archive.from-google-photos.tag]

[archive.from-immich]
from-api-key = 'OLD-API-KEY'
from-api-trace = false
from-archived = false
from-city = ''
from-client-timeout = '20m'
from-country = ''
from-favorite = false
from-make = ''
from-minimal-rating = 0
from-model = ''
from-no-album = false
from-partners = false
from-server = 'https://old.immich.app'
from-skip-verify-ssl = false
from-state = ''
from-trash = false

[archive.from-immich.from-albums]

[archive.from-immich.from-people]

[archive.from-immich.from-tags]

[stack]
api-key = 'YOUR-API-KEY'
client-timeout = '20m'
date-range = '2024-01-15,2024-03-31'
device-uuid = 'HOSTNAME'
manage-burst = 'NoStack'
manage-epson-fastfoto = false
manage-heic-jpeg = 'NoStack'
manage-raw-jpeg = 'NoStack'
server = 'https://immich.app'

[upload]
api-key = 'YOUR-API-KEY'
client-timeout = '20m'
device-uuid = 'HOSTNAME'
server = 'https://immich.app'

[upload.from-folder]
album-path-joiner = ' / '
album-picasa = false
date-from-name = true
date-range = '2024-01-15,2024-03-31'
exclude-extensions = []
folder-as-album = 'none'
folder-as-tags = false
ignore-sidecar-files = false
include-extensions = []
include-type = ''
into-album = ''
manage-burst = 'NoStack'
manage-epson-fastfoto = false
manage-heic-jpeg = 'NoStack'
manage-raw-jpeg = 'NoStack'
recursive = true
session-tag = false

[upload.from-folder.ban-file]

[upload.from-folder.tag]

[upload.from-google-photos]
date-range = '2024-01-15,2024-03-31'
exclude-extensions = []
from-album-name = ''
include-archived = true
include-extensions = []
include-partner = true
include-trashed = false
include-type = ''
include-unmatched = false
include-untitled-albums = false
manage-burst = 'NoStack'
manage-epson-fastfoto = false
manage-heic-jpeg = 'NoStack'
manage-raw-jpeg = 'NoStack'
partner-shared-album = ''
people-tag = true
session-tag = false
sync-albums = true
takeout-tag = true

[upload.from-google-photos.ban-file]

[upload.from-google-photos.tag]

[upload.from-icloud]
album-path-joiner = ' / '
album-picasa = false
date-from-name = true
date-range = '2024-01-15,2024-03-31'
exclude-extensions = []
folder-as-album = 'NONE'
folder-as-tags = false
ignore-sidecar-files = false
include-extensions = []
include-type = ''
into-album = ''
manage-burst = 'NoStack'
manage-epson-fastfoto = false
manage-heic-jpeg = 'NoStack'
manage-raw-jpeg = 'NoStack'
memories = false
recursive = true
session-tag = false

[upload.from-icloud.ban-file]

[upload.from-icloud.tag]

[upload.from-immich]
from-api-key = 'OLD-API-KEY'
from-api-trace = false
from-archived = false
from-city = ''
from-client-timeout = '20m'
from-country = ''
from-date-range = '2024-01-15,2024-03-31'
from-exclude-extensions = []
from-favorite = false
from-include-extensions = []
from-include-type = ''
from-make = ''
from-minimal-rating = 0
from-model = ''
from-no-album = false
from-partners = false
from-server = 'https://old.immich.app'
from-skip-verify-ssl = false
from-state = ''
from-trash = false

[upload.from-immich.from-albums]

[upload.from-immich.from-people]

[upload.from-immich.from-tags]

[upload.from-picasa]
album-path-joiner = ' / '
album-picasa = false
date-from-name = true
date-range = '2024-01-15,2024-03-31'
exclude-extensions = []
folder-as-album = 'NONE'
folder-as-tags = false
ignore-sidecar-files = false
include-extensions = []
include-type = ''
into-album = ''
manage-burst = 'NoStack'
manage-epson-fastfoto = false
manage-heic-jpeg = 'NoStack'
manage-raw-jpeg = 'NoStack'
recursive = true
session-tag = false

[upload.from-picasa.ban-file]

[upload.from-picasa.tag]
```
````
````
---
title: YAML
---
```yaml
archive:
  from-folder:
    album-path-joiner: ' / '
    album-picasa: false
    ban-file: {}
    date-from-name: true
    date-range: 2024-01-15,2024-03-31
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
    date-range: 2024-01-15,2024-03-31
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
    from-albums: {}
    from-api-key: OLD-API-KEY
    from-api-trace: false
    from-archived: false
    from-city: ""
    from-client-timeout: 20m
    from-country: ""
    from-favorite: false
    from-make: ""
    from-minimal-rating: 0
    from-model: ""
    from-no-album: false
    from-partners: false
    from-people: {}
    from-server: https://old.immich.app
    from-skip-verify-ssl: false
    from-state: ""
    from-tags: {}
    from-trash: false
dry-run: false
log-file: ""
log-level: INFO
log-type: text
on-errors: stop
save-config: false
stack:
  api-key: YOUR-API-KEY
  client-timeout: 20m
  date-range: 2024-01-15,2024-03-31
  device-uuid: HOSTNAME
  manage-burst: NoStack
  manage-epson-fastfoto: false
  manage-heic-jpeg: NoStack
  manage-raw-jpeg: NoStack
  server: https://immich.app
upload:
  api-key: YOUR-API-KEY
  client-timeout: 20m
  device-uuid: HOSTNAME
  from-folder:
    album-path-joiner: ' / '
    album-picasa: false
    ban-file: {}
    date-from-name: true
    date-range: 2024-01-15,2024-03-31
    exclude-extensions: []
    folder-as-album: none
    folder-as-tags: false
    ignore-sidecar-files: false
    include-extensions: []
    include-type: ""
    into-album: ""
    manage-burst: NoStack
    manage-epson-fastfoto: false
    manage-heic-jpeg: NoStack
    manage-raw-jpeg: NoStack
    recursive: true
    session-tag: false
    tag: {}
  from-google-photos:
    ban-file: {}
    date-range: 2024-01-15,2024-03-31
    exclude-extensions: []
    from-album-name: ""
    include-archived: true
    include-extensions: []
    include-partner: true
    include-trashed: false
    include-type: ""
    include-unmatched: false
    include-untitled-albums: false
    manage-burst: NoStack
    manage-epson-fastfoto: false
    manage-heic-jpeg: NoStack
    manage-raw-jpeg: NoStack
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
    date-range: 2024-01-15,2024-03-31
    exclude-extensions: []
    folder-as-album: NONE
    folder-as-tags: false
    ignore-sidecar-files: false
    include-extensions: []
    include-type: ""
    into-album: ""
    manage-burst: NoStack
    manage-epson-fastfoto: false
    manage-heic-jpeg: NoStack
    manage-raw-jpeg: NoStack
    memories: false
    recursive: true
    session-tag: false
    tag: {}
  from-immich:
    from-albums: {}
    from-api-key: OLD-API-KEY
    from-api-trace: false
    from-archived: false
    from-city: ""
    from-client-timeout: 20m
    from-country: ""
    from-date-range: 2024-01-15,2024-03-31
    from-exclude-extensions: []
    from-favorite: false
    from-include-extensions: []
    from-include-type: ""
    from-make: ""
    from-minimal-rating: 0
    from-model: ""
    from-no-album: false
    from-partners: false
    from-people: {}
    from-server: https://old.immich.app
    from-skip-verify-ssl: false
    from-state: ""
    from-tags: {}
    from-trash: false
  from-picasa:
    album-path-joiner: ' / '
    album-picasa: false
    ban-file: {}
    date-from-name: true
    date-range: 2024-01-15,2024-03-31
    exclude-extensions: []
    folder-as-album: NONE
    folder-as-tags: false
    ignore-sidecar-files: false
    include-extensions: []
    include-type: ""
    into-album: ""
    manage-burst: NoStack
    manage-epson-fastfoto: false
    manage-heic-jpeg: NoStack
    manage-raw-jpeg: NoStack
    recursive: true
    session-tag: false
    tag: {}
  server: https://immich.app
```
````
````
---
title: JSON
---
```json
{
  "archive": {
    "from-folder": {
      "album-path-joiner": " / ",
      "album-picasa": false,
      "ban-file": {},
      "date-from-name": true,
      "date-range": "2024-01-15,2024-03-31",
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
      "date-range": "2024-01-15,2024-03-31",
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
      "from-albums": {},
      "from-api-key": "OLD-API-KEY",
      "from-api-trace": false,
      "from-archived": false,
      "from-city": "",
      "from-client-timeout": "20m",
      "from-country": "",
      "from-favorite": false,
      "from-make": "",
      "from-minimal-rating": 0,
      "from-model": "",
      "from-no-album": false,
      "from-partners": false,
      "from-people": {},
      "from-server": "https://old.immich.app",
      "from-skip-verify-ssl": false,
      "from-state": "",
      "from-tags": {},
      "from-trash": false
    }
  },
  "dry-run": false,
  "log-file": "",
  "log-level": "INFO",
  "log-type": "text",
  "on-errors": "stop",
  "save-config": false,
  "stack": {
    "api-key": "YOUR-API-KEY",
    "client-timeout": "20m",
    "date-range": "2024-01-15,2024-03-31",
    "device-uuid": "HOSTNAME",
    "manage-burst": "NoStack",
    "manage-epson-fastfoto": false,
    "manage-heic-jpeg": "NoStack",
    "manage-raw-jpeg": "NoStack",
    "server": "https://immich.app"
  },
  "upload": {
    "api-key": "YOUR-API-KEY",
    "client-timeout": "20m",
    "device-uuid": "HOSTNAME",
    "from-folder": {
      "album-path-joiner": " / ",
      "album-picasa": false,
      "ban-file": {},
      "date-from-name": true,
      "date-range": "2024-01-15,2024-03-31",
      "exclude-extensions": null,
      "folder-as-album": "none",
      "folder-as-tags": false,
      "ignore-sidecar-files": false,
      "include-extensions": null,
      "include-type": "",
      "into-album": "",
      "manage-burst": "NoStack",
      "manage-epson-fastfoto": false,
      "manage-heic-jpeg": "NoStack",
      "manage-raw-jpeg": "NoStack",
      "recursive": true,
      "session-tag": false,
      "tag": {}
    },
    "from-google-photos": {
      "ban-file": {},
      "date-range": "2024-01-15,2024-03-31",
      "exclude-extensions": null,
      "from-album-name": "",
      "include-archived": true,
      "include-extensions": null,
      "include-partner": true,
      "include-trashed": false,
      "include-type": "",
      "include-unmatched": false,
      "include-untitled-albums": false,
      "manage-burst": "NoStack",
      "manage-epson-fastfoto": false,
      "manage-heic-jpeg": "NoStack",
      "manage-raw-jpeg": "NoStack",
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
      "date-range": "2024-01-15,2024-03-31",
      "exclude-extensions": null,
      "folder-as-album": "NONE",
      "folder-as-tags": false,
      "ignore-sidecar-files": false,
      "include-extensions": null,
      "include-type": "",
      "into-album": "",
      "manage-burst": "NoStack",
      "manage-epson-fastfoto": false,
      "manage-heic-jpeg": "NoStack",
      "manage-raw-jpeg": "NoStack",
      "memories": false,
      "recursive": true,
      "session-tag": false,
      "tag": {}
    },
    "from-immich": {
      "from-albums": {},
      "from-api-key": "OLD-API-KEY",
      "from-api-trace": false,
      "from-archived": false,
      "from-city": "",
      "from-client-timeout": "20m",
      "from-country": "",
      "from-date-range": "2024-01-15,2024-03-31",
      "from-exclude-extensions": null,
      "from-favorite": false,
      "from-include-extensions": null,
      "from-include-type": "",
      "from-make": "",
      "from-minimal-rating": 0,
      "from-model": "",
      "from-no-album": false,
      "from-partners": false,
      "from-people": {},
      "from-server": "https://old.immich.app",
      "from-skip-verify-ssl": false,
      "from-state": "",
      "from-tags": {},
      "from-trash": false
    },
    "from-picasa": {
      "album-path-joiner": " / ",
      "album-picasa": false,
      "ban-file": {},
      "date-from-name": true,
      "date-range": "2024-01-15,2024-03-31",
      "exclude-extensions": null,
      "folder-as-album": "NONE",
      "folder-as-tags": false,
      "ignore-sidecar-files": false,
      "include-extensions": null,
      "include-type": "",
      "into-album": "",
      "manage-burst": "NoStack",
      "manage-epson-fastfoto": false,
      "manage-heic-jpeg": "NoStack",
      "manage-raw-jpeg": "NoStack",
      "recursive": true,
      "session-tag": false,
      "tag": {}
    },
    "server": "https://immich.app"
  }
}
```
````
