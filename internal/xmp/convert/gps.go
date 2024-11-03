package convert

import (
	"fmt"
	"math"
)

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

	_, err := fmt.Sscanf(coordinate, "%d,%f%s", &degrees, &minutes, &direction)
	if err != nil {
		return 0, err
	}

	decimal := float64(degrees) + float64(minutes)/60

	if direction == "S" || direction == "W" {
		decimal = -decimal
	}

	return decimal, nil
}
