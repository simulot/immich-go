#!/bin/bash
# add_text_to_images.sh

# Check if the correct number of arguments is provided
if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <folder_path> <text>"
    exit 1
fi

FOLDER_PATH=$1
TEXT=$2

# Check if the folder exists
if [ ! -d "$FOLDER_PATH" ]; then
    echo "Folder $FOLDER_PATH does not exist."
    exit 1
fi

# Create a temporary text image
TEXT_IMAGE="text_image.png"
convert -background none -fill white -pointsize 24 -gravity SouthEast label:"$TEXT" "$TEXT_IMAGE"

# Loop through each image in the folder
for IMAGE in "$FOLDER_PATH"/*; do
    if [ -f "$IMAGE" ]; then
        # Add text to the lower right part of the image
        composite -gravity SouthEast "$TEXT_IMAGE" "$IMAGE" "$IMAGE"
        echo "Processed $IMAGE"
    fi
done

# Remove the temporary text image
rm "$TEXT_IMAGE"

echo "All images processed."
