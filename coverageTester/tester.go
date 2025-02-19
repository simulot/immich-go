package coverageTester

import (
	"bufio"
	"os"
)

func WriteUniqueLine(input string) error {
	filename := "/Users/samvelhovhannisyan/Documents/dev/dd2480/immich-go/coverageBranch-OpenClient.txt"
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if scanner.Text() == input {
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	file, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(input + "\n")
	return err
}
