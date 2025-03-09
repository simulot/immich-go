#!/bin/bash

# Check if the correct number of arguments is provided
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <timestamp>"
    exit 1
fi

# Get the timestamp from the argument
timestamp="$1"

# Find all files in subfolders (excluding the current directory) and set their modified time
find . -mindepth 2 -type f -exec touch -d "$timestamp" {} \;

echo "All files in subfolders have been set to the modified time: $timestamp"
