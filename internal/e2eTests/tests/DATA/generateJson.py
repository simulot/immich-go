import os
import json
from datetime import datetime
import sys
def generate_json_files(folder_path):
    # Get the list of files in the folder
    files = os.listdir(folder_path)

    for file_name in files:
        # Create the JSON file name
        json_file_name = f"{file_name}.supplemental-metadata.json"
        json_file_path = os.path.join(folder_path, json_file_name)

        # Get the creation time of the file
        creation_time = os.path.getctime(os.path.join(folder_path, file_name))
        creation_time_formatted = datetime.utcfromtimestamp(creation_time).strftime('%a, %d %b %Y %H:%M:%S UTC')

        # Create the JSON data
        json_data = {
            "title": file_name,
            "description": "",
            "creationTime": {
                "timestamp": str(int(creation_time)),
                "formatted": creation_time_formatted
            },
            "photoTakenTime": {
                "timestamp": str(int(creation_time)),
                "formatted": creation_time_formatted
            },
            "url": "https://photos.google.com/photo/AF1QipMjB5UVZ4V257bEr9MUafTu-bZEyk7WKmQs3TV_",
            "googlePhotosOrigin": {
                "mobileUpload": {
                    "deviceFolder": {
                        "localFolderName": ""
                    },
                    "deviceType": "ANDROID_PHONE"
                }
            }
        }

        # Write the JSON data to the file
        with open(json_file_path, 'w') as json_file:
            json.dump(json_data, json_file, indent=4)

# Example usage
if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python generateJson.py <folder_path>")
        sys.exit(1)

    folder_path = sys.argv[1]
    generate_json_files(folder_path)
