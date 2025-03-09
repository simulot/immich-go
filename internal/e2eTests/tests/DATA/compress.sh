#!/bin/bash

# Define the source and destination directories
SOURCE_DIR="high_jpg"
DEST_DIR="low"

# Create the destination directory if it doesn't exist
mkdir -p "$DEST_DIR"

# Loop through all image files in the source directory
for IMAGE in "$SOURCE_DIR"/*; do
    # Get the base name and extension of the image file
    BASENAME=$(basename "$IMAGE")
    EXTENSION="${BASENAME##*.}"
    NAME="${BASENAME%.*}"

    # Recompress the image with very bad quality and save it to the destination directory
    convert "$IMAGE" -quality 10 "$DEST_DIR/$NAME".jpg
done

echo "Recompression complete. Recompressed images are saved in the 'low' subfolder."
