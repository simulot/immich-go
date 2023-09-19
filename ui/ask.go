package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

func ConfirmYesNo(ctx context.Context, prompt string, defaultAnswer string) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	reader := bufio.NewReader(os.Stdin)
	defaultAnswer = strings.ToLower(defaultAnswer)
	other := "n"
	if defaultAnswer == "n" {
		other = "y"
	}

	runeChan := make(chan (rune))

	go func() {
		for {
			r, _, _ := reader.ReadRune()
			select {
			case runeChan <- r:
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		fmt.Printf("%s [%s]/%s: ", prompt, defaultAnswer, other)
		select {
		case r := <-runeChan:
			userInput := strings.ToLower(string(r))
			switch userInput {
			case "":
				return defaultAnswer, nil
			case "y", "n":
				return userInput, nil
			}
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}
