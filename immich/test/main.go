package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

func getPhotoDate(filename string) (time.Time, error) {
	// Open the JPEG file
	file, err := os.Open(filename)
	if err != nil {
		return time.Time{}, err
	}
	defer file.Close()

	// Decode the EXIF data
	exifData, err := exif.Decode(file)

	if err != nil {
		if exif.IsCriticalError(err) {
			return time.Time{}, err
		}
	}
	// Get the date and time information
	dateTime, err := exifData.Get(exif.DateTime)
	if err != nil {
		return time.Time{}, err
	}

	// Parse the date and time
	dateTimeString, err := dateTime.StringVal()
	if err != nil {
		return time.Time{}, err
	}
	parsedTime, err := time.Parse("2006:01:02 15:04:05", dateTimeString)
	if err != nil {
		return time.Time{}, err
	}

	return parsedTime, nil
}

func main() {
	filename := "/home/jfcassan/takeout/jfc/2019.06.23/IMG_20190623_153911.jpg"

	parsedTime, err := getPhotoDate(filename)
	if err != nil {
		log.Fatal(err)
	}

	// Print the date and time
	fmt.Println("Date and Time:", parsedTime.Format("January 2, 2006 15:04:05"))
}
