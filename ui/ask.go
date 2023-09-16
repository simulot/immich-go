package ui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

func ConfirmYesNo(prompt string, defaultAnswer string) string {
	reader := bufio.NewReader(os.Stdin)
	defaultAnswer = strings.ToLower(defaultAnswer)
	other := "n"
	if defaultAnswer == "n" {
		other = "y"
	}

	for {
		fmt.Printf("%s [%s]/%s: ", prompt, defaultAnswer, other)
		userInput, _ := reader.ReadString('\n')
		userInput = strings.ToLower(strings.TrimSpace(userInput))
		switch userInput {
		case "":
			return defaultAnswer
		case "y", "n":
			return userInput
		}
	}
}

type CancelableReader struct {
	ctx  context.Context
	data chan []byte
	err  error
	r    io.Reader
}

func (c *CancelableReader) begin() {
	buf := make([]byte, 1024)
	for {
		n, err := c.r.Read(buf)
		if n > 0 {
			tmp := make([]byte, n)
			copy(tmp, buf[:n])
			c.data <- tmp
		}
		if err != nil {
			c.err = err
			close(c.data)
			return
		}
	}
}

func (c *CancelableReader) Read(p []byte) (int, error) {
	select {
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	case d, ok := <-c.data:
		if !ok {
			return 0, c.err
		}
		copy(p, d)
		return len(d), nil
	}
}

func New(ctx context.Context, r io.Reader) *CancelableReader {
	c := &CancelableReader{
		r:    r,
		ctx:  ctx,
		data: make(chan []byte),
	}
	go c.begin()
	return c
}
