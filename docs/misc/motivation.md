# Motivation

The Immich project fulfills all my requirements for managing my photos:

- Self-hosted
- Open source
- Abundant functionalities
- User experience closely resembling Google Photos
- Machine learning capabilities
- Well-documented API
- Includes an import tool
- Continuously enhanced
- ...

Now, I need to migrate my photos to the new system in bulk. Most of my photos are stored in a NAS directory, while photos taken with my smartphone are in the Google Photos application often more compressed.

To completely transition away from the Google Photos service, I must set up an Immich server, import my NAS-stored photos, and merge them with my Google Photos collection.
However, there are instances where the same pictures exist in both systems, sometimes with varying quality. Of course, I want to keep only the best copy of the photo.

The  `immich-cli` installation isn't trivial on a client machine, and doesn't handle Google Photos Takeout archive oddities.

The immich-cli tool does a great for importing a tone of files at full speed. However, I want more. So I write this utility for my own purpose. Maybe, it could help some one else.

## Limitations of the `immich-CLI`:

While the provided tool is very fast and do the job, certain limitations persist:

### Advanced Expertise Required

The CLI tool is available within the Immich server container, eliminating the need to install `Node.js` on your PC. Editing the `docker-compose.yml` file is necessary to access the host's files and retrieve your photos. Uploading photos from a different PC than the Immich server requires advanced skills.

### Limitations with Google Takeout Data

The Google Photos Takeout service delivers your collection as massive zip files containing your photos and their JSON files.

After unzipping the archive, you can use the CLI tool to upload its contents. However, certain limitations exist:
- Photos are organized in folders by year and albums.
- Photos are duplicated across year folders and albums.
- Some folders aren't albums
- Photos might be compressed in the takeout archive, affecting the CLI's duplicate detection when comparing them to previously imported photos with finest details.
- File and album names are mangled and correct names are found in JSON files

## Why the GO language?

The main reason is that I am more profecient in GO compared to Typescript.
Additionally, deploying a Node.js program on user machines presents challenges.
