package immich

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// callError represents errors returned by the server
type callError struct {
	endPoint string
	method   string
	url      string
	status   int
	err      error
	message  serverError
}

type serverError interface {
	lines() []string
}

type serverErrorV2 struct {
	Error         string `json:"error"`
	StatusCode    int    `json:"statusCode"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlationId"`
}

func (e serverErrorV2) lines() []string {
	return []string{e.Message}
}

type serverErrorV3 struct {
	Message       string          `json:"message"`
	Errors        json.RawMessage `json:"errors"`
	CorrelationID string          `json:"-"` // from the X-Correlation-ID header
}

func (e serverErrorV3) lines() []string {
	var out []string
	if e.Message != "" {
		out = append(out, e.Message)
	}
	if len(e.Errors) > 0 {
		out = append(out, string(e.Errors))
	}
	if e.CorrelationID != "" {
		out = append(out, "correlationId: "+e.CorrelationID)
	}
	return out
}

func (ce callError) Is(target error) bool {
	_, ok := target.(*callError)
	return ok
}

func (ce callError) Error() string {
	head := fmt.Sprintf("%s, %s, %s", ce.endPoint, ce.method, ce.url)
	if ce.status > 0 {
		head += fmt.Sprintf(", %d %s", ce.status, http.StatusText(ce.status))
	}
	lines := []string{head}
	if ce.err != nil && !errors.Is(ce.err, &callError{}) {
		lines = append(lines, ce.err.Error())
	}
	if ce.message != nil {
		lines = append(lines, ce.message.lines()...)
	}
	return strings.Join(lines, "\n") + "\n"
}
