//go:build darwin

package tzone

import (
	"os/exec"
)

func getTimezoneName() (string, error) {
	cmd := exec.Command("systemsetup", "-gettimezone")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}
