# Immich API Wish List

This document outlines a wish list of features and improvements for the Immich API. The goal is to enhance the functionality, usability, and performance of the API to better serve developers and users.

## 1. Assets endpoint

1.1 updateAssets for changing the original filename

Immich-go take the original file name from JSON file attached to the asset, and not from the actual file name. 

When the takeout is a TGZ file, it's not possible to parse quickly the archive. The order of the files in the archive is unpredictable, and it happens that the asset file is encountered before the JSON file. The definitive "original file" name is known only after the JSON file is read. 

The workaround is to parse the TGZ file twice, first to read all JSON files and store the original file names in a map, and then to parse again the TGZ file to get the assets with the correct original file name. 


## 2. Search endpoint

2.1 searchAssets with multiple criteria

The current search functionality allows filtering assets based on a single criterion at a time.
Currently, immich-go creates a separate query for each criterion and combines the results.

Examples:

* Get all assets with a rating of 4 or higher
* Get timeline or archived assets 
* Get all timeline assets including trashed and archived
* Get all assets from a list of cities


