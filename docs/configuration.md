# Configuration File

The configuration file can be a `TOML`, `YAML` or `JSON` file. By default, `immich-go` looks for a file named `immich-go.toml` in the current directory.

## Configuration file structure

<details>
<summary>TOML</summary>

```toml
concurrent-tasks = 12
dry-run = false
log-file = ''
log-level = 'INFO'
log-type = 'text'
on-errors = 'stop'
save-config = false

[archive]
write-to-folder = ''

[archive.from-folder]
album-path-joiner = ' / '
date-from-name = true
date-range = '2024-01-15,2024-03-31'
exclude-extensions = []
folder-as-album = 'none'
folder-as-tags = false
ignore-sidecar-files = false
include-extensions = []
include-type = ''
into-album = ''
recursive = true

[archive.from-folder.ban-file]

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
sync-albums = true
takeout-tag = true

[archive.from-google-photos.ban-file]

[archive.from-icloud]
album-path-joiner = ' / '
date-from-name = true
date-range = '2024-01-15,2024-03-31'
exclude-extensions = []
folder-as-album = 'none'
folder-as-tags = false
ignore-sidecar-files = false
include-extensions = []
include-type = ''
into-album = ''
memories = false
recursive = true

[archive.from-icloud.ban-file]

[archive.from-immich]
from-admin-api-key = ''
from-api-key = 'OLD-API-KEY'
from-api-trace = false
from-archived = false
from-city = ''
from-client-timeout = '20m'
from-country = ''
from-date-range = '2024-01-15,2024-03-31'
from-device-uuid = 'gl65'
from-dry-run = false
from-exclude-extensions = []
from-favorite = false
from-include-extensions = []
from-include-type = ''
from-make = ''
from-minimal-rating = 0
from-model = ''
from-no-album = false
from-partners = false
from-pause-immich-jobs = true
from-server = 'https://old.immich.app'
from-skip-verify-ssl = false
from-state = ''
from-time-zone = ''
from-trash = false

[archive.from-immich.from-albums]

[archive.from-immich.from-people]

[archive.from-immich.from-tags]

[archive.from-picasa]
album-path-joiner = ' / '
album-picasa = true
date-from-name = true
date-range = '2024-01-15,2024-03-31'
exclude-extensions = []
folder-as-album = 'none'
folder-as-tags = false
ignore-sidecar-files = false
include-extensions = []
include-type = ''
into-album = ''
recursive = true

[archive.from-picasa.ban-file]

[stack]
admin-api-key = ''
api-key = 'YOUR-API-KEY'
api-trace = false
client-timeout = '20m'
date-range = '2024-01-15,2024-03-31'
device-uuid = 'HOSTNAME'
dry-run = false
manage-burst = 'NoStack'
manage-epson-fastfoto = false
manage-heic-jpeg = 'NoStack'
manage-raw-jpeg = 'NoStack'
pause-immich-jobs = true
server = 'https://immich.app'
skip-verify-ssl = false
time-zone = ''

[upload]
admin-api-key = ''
api-key = 'YOUR-API-KEY'
api-trace = false
client-timeout = '20m'
device-uuid = 'HOSTNAME'
dry-run = false
manage-burst = 'NoStack'
manage-epson-fastfoto = false
manage-heic-jpeg = 'NoStack'
manage-raw-jpeg = 'NoStack'
no-ui = false
overwrite = false
pause-immich-jobs = true
server = 'https://immich.app'
session-tag = false
skip-verify-ssl = false
time-zone = ''

[upload.from-folder]
album-path-joiner = ' / '
date-from-name = true
date-range = '2024-01-15,2024-03-31'
exclude-extensions = []
folder-as-album = 'none'
folder-as-tags = false
ignore-sidecar-files = false
include-extensions = []
include-type = ''
into-album = ''
recursive = true

[upload.from-folder.ban-file]

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
partner-shared-album = ''
people-tag = true
sync-albums = true
takeout-tag = true

[upload.from-google-photos.ban-file]

[upload.from-icloud]
album-path-joiner = ' / '
date-from-name = true
date-range = '2024-01-15,2024-03-31'
exclude-extensions = []
folder-as-album = 'none'
folder-as-tags = false
ignore-sidecar-files = false
include-extensions = []
include-type = ''
into-album = ''
memories = false
recursive = true

[upload.from-icloud.ban-file]

[upload.from-immich]
from-admin-api-key = ''
from-api-key = 'OLD-API-KEY'
from-api-trace = false
from-archived = false
from-city = ''
from-client-timeout = '20m'
from-country = ''
from-date-range = '2024-01-15,2024-03-31'
from-device-uuid = 'gl65'
from-dry-run = false
from-exclude-extensions = []
from-favorite = false
from-include-extensions = []
from-include-type = ''
from-make = ''
from-minimal-rating = 0
from-model = ''
from-no-album = false
from-partners = false
from-pause-immich-jobs = true
from-server = 'https://old.immich.app'
from-skip-verify-ssl = false
from-state = ''
from-time-zone = ''
from-trash = false

[upload.from-immich.from-albums]

[upload.from-immich.from-people]

[upload.from-immich.from-tags]

[upload.from-picasa]
album-path-joiner = ' / '
album-picasa = true
date-from-name = true
date-range = '2024-01-15,2024-03-31'
exclude-extensions = []
folder-as-album = 'none'
folder-as-tags = false
ignore-sidecar-files = false
include-extensions = []
include-type = ''
into-album = ''
recursive = true

[upload.from-picasa.ban-file]

[upload.tag]
```

</details>

<details>
<summary>YAML</summary>

```yaml
archive:
  from-folder:
    album-path-joiner: ' / '
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
    recursive: true
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
    sync-albums: true
    takeout-tag: true
  from-icloud:
    album-path-joiner: ' / '
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
    memories: false
    recursive: true
  from-immich:
    from-admin-api-key: ""
    from-albums: {}
    from-api-key: OLD-API-KEY
    from-api-trace: false
    from-archived: false
    from-city: ""
    from-client-timeout: 20m
    from-country: ""
    from-date-range: 2024-01-15,2024-03-31
    from-device-uuid: gl65
    from-dry-run: false
    from-exclude-extensions: []
    from-favorite: false
    from-include-extensions: []
    from-include-type: ""
    from-make: ""
    from-minimal-rating: 0
    from-model: ""
    from-no-album: false
    from-partners: false
    from-pause-immich-jobs: true
    from-people: {}
    from-server: https://old.immich.app
    from-skip-verify-ssl: false
    from-state: ""
    from-tags: {}
    from-time-zone: ""
    from-trash: false
  from-picasa:
    album-path-joiner: ' / '
    album-picasa: true
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
    recursive: true
  write-to-folder: ""
concurrent-tasks: 12
dry-run: false
log-file: ""
log-level: INFO
log-type: text
on-errors: stop
save-config: false
stack:
  admin-api-key: ""
  api-key: YOUR-API-KEY
  api-trace: false
  client-timeout: 20m
  date-range: 2024-01-15,2024-03-31
  device-uuid: HOSTNAME
  dry-run: false
  manage-burst: NoStack
  manage-epson-fastfoto: false
  manage-heic-jpeg: NoStack
  manage-raw-jpeg: NoStack
  pause-immich-jobs: true
  server: https://immich.app
  skip-verify-ssl: false
  time-zone: ""
upload:
  admin-api-key: ""
  api-key: YOUR-API-KEY
  api-trace: false
  client-timeout: 20m
  device-uuid: HOSTNAME
  dry-run: false
  from-folder:
    album-path-joiner: ' / '
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
    recursive: true
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
    sync-albums: true
    takeout-tag: true
  from-icloud:
    album-path-joiner: ' / '
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
    memories: false
    recursive: true
  from-immich:
    from-admin-api-key: ""
    from-albums: {}
    from-api-key: OLD-API-KEY
    from-api-trace: false
    from-archived: false
    from-city: ""
    from-client-timeout: 20m
    from-country: ""
    from-date-range: 2024-01-15,2024-03-31
    from-device-uuid: gl65
    from-dry-run: false
    from-exclude-extensions: []
    from-favorite: false
    from-include-extensions: []
    from-include-type: ""
    from-make: ""
    from-minimal-rating: 0
    from-model: ""
    from-no-album: false
    from-partners: false
    from-pause-immich-jobs: true
    from-people: {}
    from-server: https://old.immich.app
    from-skip-verify-ssl: false
    from-state: ""
    from-tags: {}
    from-time-zone: ""
    from-trash: false
  from-picasa:
    album-path-joiner: ' / '
    album-picasa: true
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
    recursive: true
  manage-burst: NoStack
  manage-epson-fastfoto: false
  manage-heic-jpeg: NoStack
  manage-raw-jpeg: NoStack
  no-ui: false
  overwrite: false
  pause-immich-jobs: true
  server: https://immich.app
  session-tag: false
  skip-verify-ssl: false
  tag: {}
  time-zone: ""
```

</details>

<details>
<summary>JSON</summary>

```json
{
  "archive": {
    "from-folder": {
      "album-path-joiner": " / ",
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
      "recursive": true
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
      "sync-albums": true,
      "takeout-tag": true
    },
    "from-icloud": {
      "album-path-joiner": " / ",
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
      "memories": false,
      "recursive": true
    },
    "from-immich": {
      "from-admin-api-key": "",
      "from-albums": {},
      "from-api-key": "OLD-API-KEY",
      "from-api-trace": false,
      "from-archived": false,
      "from-city": "",
      "from-client-timeout": "20m",
      "from-country": "",
      "from-date-range": "2024-01-15,2024-03-31",
      "from-device-uuid": "gl65",
      "from-dry-run": false,
      "from-exclude-extensions": null,
      "from-favorite": false,
      "from-include-extensions": null,
      "from-include-type": "",
      "from-make": "",
      "from-minimal-rating": 0,
      "from-model": "",
      "from-no-album": false,
      "from-partners": false,
      "from-pause-immich-jobs": true,
      "from-people": {},
      "from-server": "https://old.immich.app",
      "from-skip-verify-ssl": false,
      "from-state": "",
      "from-tags": {},
      "from-time-zone": "",
      "from-trash": false
    },
    "from-picasa": {
      "album-path-joiner": " / ",
      "album-picasa": true,
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
      "recursive": true
    },
    "write-to-folder": ""
  },
  "concurrent-tasks": 12,
  "dry-run": false,
  "log-file": "",
  "log-level": "INFO",
  "log-type": "text",
  "on-errors": "stop",
  "save-config": false,
  "stack": {
    "admin-api-key": "",
    "api-key": "YOUR-API-KEY",
    "api-trace": false,
    "client-timeout": "20m",
    "date-range": "2024-01-15,2024-03-31",
    "device-uuid": "HOSTNAME",
    "dry-run": false,
    "manage-burst": "NoStack",
    "manage-epson-fastfoto": false,
    "manage-heic-jpeg": "NoStack",
    "manage-raw-jpeg": "NoStack",
    "pause-immich-jobs": true,
    "server": "https://immich.app",
    "skip-verify-ssl": false,
    "time-zone": ""
  },
  "upload": {
    "admin-api-key": "",
    "api-key": "YOUR-API-KEY",
    "api-trace": false,
    "client-timeout": "20m",
    "device-uuid": "HOSTNAME",
    "dry-run": false,
    "from-folder": {
      "album-path-joiner": " / ",
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
      "recursive": true
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
      "sync-albums": true,
      "takeout-tag": true
    },
    "from-icloud": {
      "album-path-joiner": " / ",
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
      "memories": false,
      "recursive": true
    },
    "from-immich": {
      "from-admin-api-key": "",
      "from-albums": {},
      "from-api-key": "OLD-API-KEY",
      "from-api-trace": false,
      "from-archived": false,
      "from-city": "",
      "from-client-timeout": "20m",
      "from-country": "",
      "from-date-range": "2024-01-15,2024-03-31",
      "from-device-uuid": "gl65",
      "from-dry-run": false,
      "from-exclude-extensions": null,
      "from-favorite": false,
      "from-include-extensions": null,
      "from-include-type": "",
      "from-make": "",
      "from-minimal-rating": 0,
      "from-model": "",
      "from-no-album": false,
      "from-partners": false,
      "from-pause-immich-jobs": true,
      "from-people": {},
      "from-server": "https://old.immich.app",
      "from-skip-verify-ssl": false,
      "from-state": "",
      "from-tags": {},
      "from-time-zone": "",
      "from-trash": false
    },
    "from-picasa": {
      "album-path-joiner": " / ",
      "album-picasa": true,
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
      "recursive": true
    },
    "manage-burst": "NoStack",
    "manage-epson-fastfoto": false,
    "manage-heic-jpeg": "NoStack",
    "manage-raw-jpeg": "NoStack",
    "no-ui": false,
    "overwrite": false,
    "pause-immich-jobs": true,
    "server": "https://immich.app",
    "session-tag": false,
    "skip-verify-ssl": false,
    "tag": {},
    "time-zone": ""
  }
}
```

</details>
