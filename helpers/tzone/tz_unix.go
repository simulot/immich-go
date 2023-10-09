//go:build unix && !darwin

package tzone

import "os"

func getTimezoneName() (string, error) {
	data, err := os.ReadFile("/etc/timezone")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
