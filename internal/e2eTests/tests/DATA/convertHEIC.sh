#!/bin/bash
# file.sh

# Define the input and output folders
SOURCE_DIR="high"
DEST_DIR="high_heic"


# Create the output folder if it doesn't exist
mkdir -p "$DEST_DIR"

# Iterate through all files in the input folder
for IMAGE in "$SOURCE_DIR"/*; do
    BASENAME=$(basename "$IMAGE")
    EXTENSION="${BASENAME##*.}"
    NAME="${BASENAME%.*}"
    convert "$IMAGE" "$DEST_DIR/$NAME".HEIC
done

