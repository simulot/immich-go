package xmpsidecar

import (
	"fmt"
	"math"
)

/*
GPSCoordinate
A Text value in the form “DDD,MM,SSk” or “DDD,MM.mmk”, where:
	DDD is a number of degrees
	MM is a number of minutes
	SS is a number of seconds
	mm is a fraction of minutes
	k is a single character N, S, E, or W indicating a direction (north, south, east, west)

Leading zeros are not necessary for the for DDD, MM, and SS values. The DDD,MM.mmk form should be used
when any of the native EXIF component rational values has a denominator other than 1. There can be any
number of fractional digits.


*/

// GPSFloatToString converts a float GPS coordinate to a string in the format "48,55.68405768N"
func GPSFloatToString(coordinate float64, isLatitude bool) string {
	neg := coordinate < 0
	if coordinate < 0 {
		coordinate = -coordinate
	}

	degrees := int(math.Floor(coordinate))
	minutes := float64(coordinate-float64(degrees)) * 60
	direction := "N"
	if !isLatitude {
		direction = "E"
	}
	if neg {
		direction = "S"
		if !isLatitude {
			direction = "W"
		}
	}

	return fmt.Sprintf("%d,%08.5f%s", degrees, minutes, direction)
}

// GPTStringToFloat converts a string GPS coordinate in the format "48,55.68405768N" to a float
func GPTStringToFloat(coordinate string) (float64, error) {
	var degrees int
	var minutes float64
	var direction string

	if len(coordinate) > 0 {
		direction = string(coordinate[len(coordinate)-1])
		coordinate = coordinate[:len(coordinate)-1]
	}
	_, err := fmt.Sscanf(coordinate, "%d,%f", &degrees, &minutes)
	if err != nil {
		return 0, err
	}

	decimal := float64(degrees) + float64(minutes)/60

	if direction == "S" || direction == "W" {
		decimal = -decimal
	}

	return decimal, nil
}
