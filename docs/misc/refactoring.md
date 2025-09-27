# Refactoring

One year later, the necessity to refactor the code is obvious:
- spaghetti code
- poor adherence to single responsibility rule
- new requests from the users


## For better architecture

- adapters: adapters for reading a writing photo collection
- cmd: immich-go commands
- immich: immich client
- internal: better not look inside


## Refactoring of the command line 

### Use of linux traditional command-line options with a double-dash  `--option`
 


### Reorganization of the commands

immich-go [global flags] command sub-command [flags] arguments

ex:
immich-go --log-file=file import from-folder --server=xxxx --key=qqqqq --folder-as-album-name=PATH path/to/photos


## Better logging



