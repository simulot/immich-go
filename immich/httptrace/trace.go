package httptrace

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	timeFormat     = "2006-01-02T15:04:05.999Z07:00"
	binaryDumpSize = 1024 // 1KB for binary content
	jsonMediaType  = "application/json"
)

type Tracer struct {
	lock       sync.Mutex
	out        io.Writer
	instanceId atomic.Int64
	wg         sync.WaitGroup
}

func NewTracer(out io.Writer) *Tracer {
	return &Tracer{
		out:  out,
		lock: sync.Mutex{},
	}
}

func (ht *Tracer) Close() error {
	ht.wg.Wait()
	return nil
}

func (ht *Tracer) newRoundTripTrace(instance, reqId int) *roundTripTrace {
	return &roundTripTrace{
		ht:       ht,
		instance: instance,
		reqId:    reqId,
	}
}

type roundTripTrace struct {
	ht       *Tracer       // reference to the HTTP Tracer
	instance int           // Client ID
	reqId    int           // Roundtrip id
	req      requestTrace  // the request
	resp     responseTrace // the response
}

func (rt *roundTripTrace) Request(req *http.Request) io.ReadCloser {
	rt.req.timestamp = time.Now()
	rt.req.req = req
	rt.req.contentType = req.Header.Get("Content-Type")
	if req.Body != nil {
		limit := getDumpLimit(rt.req.contentType)
		rt.req.body = newDumpReader(req.Body, limit)
		return rt.req.body
	}
	return req.Body
}

func dump(rt *roundTripTrace) {
	go func() {
		if rt.req.body != nil {
			// Wait for the request body to be closed...
			rt.req.body.closed.Wait()
		}
		if rt.resp.resp != nil && rt.resp.body != nil {
			// Wait for the response body to be closed...
			rt.resp.body.closed.Wait()
		}
		rt.ht.lock.Lock()
		defer rt.ht.lock.Unlock()
		fmt.Fprintf(rt.ht.out, "/---- client #%d request  #%d ---------------------------------------------------\n", rt.instance, rt.reqId)
		fmt.Fprint(rt.ht.out, rt.req.String())
		fmt.Fprintf(rt.ht.out, "+---- response  ---------------------------------------------------\n")
		fmt.Fprint(rt.ht.out, rt.resp.String())
		fmt.Fprintf(rt.ht.out, "\\---- client #%d request  #%d ---------------------------------------------------\n", rt.instance, rt.reqId)
		fmt.Fprint(rt.ht.out, "\n")
	}()
}

type requestTrace struct {
	req         *http.Request
	contentType string
	body        *dumpReader
	timestamp   time.Time
}

func (rt *roundTripTrace) Response(resp *http.Response, err error) io.ReadCloser {
	rt.resp.timestamp = time.Now()
	rt.resp.duration = rt.resp.timestamp.Sub(rt.req.timestamp).Round(time.Millisecond)
	rt.resp.err = err
	if err != nil {
		return nil
	}
	rt.resp.resp = resp
	rt.resp.contentType = resp.Header.Get("Content-Type")
	if resp.Body != nil {
		limit := getDumpLimit(rt.resp.contentType)
		rt.resp.body = newDumpReader(resp.Body, limit)
		return rt.resp.body
	}
	return resp.Body
}

func (rt requestTrace) String() string {
	sb := strings.Builder{}
	fmt.Fprintf(&sb, "Method: %s, URL: %s\n", rt.req.Method, rt.req.URL)
	fmt.Fprintf(&sb, "Timestamp: %s\n", rt.timestamp.Format(timeFormat))
	sb.WriteString(writeHeaders(rt.req.Header))
	if rt.body != nil {
		sb.WriteString(writeBody(rt.contentType, rt.body))
		rt.body.Done()
	}
	return sb.String()
}

type responseTrace struct {
	err         error
	resp        *http.Response
	contentType string
	body        *dumpReader
	timestamp   time.Time
	duration    time.Duration
}

func (rt *responseTrace) String() string {
	sb := strings.Builder{}
	fmt.Fprintf(&sb, "Timestamp: %s\n", rt.timestamp.Format(timeFormat))
	fmt.Fprintf(&sb, "Duration: %s\n", rt.duration)
	if rt.err != nil {
		fmt.Fprintf(&sb, "Error: %s\n", rt.err.Error())
	} else {
		fmt.Fprintf(&sb, "Status: %s: %d\n", rt.resp.Status, rt.resp.StatusCode)
		sb.WriteString(writeHeaders(rt.resp.Header))
		if rt.body != nil {
			sb.WriteString(writeBody(rt.contentType, rt.body))
			rt.body.Done()
		}
	}
	return sb.String()
}

func writeHeaders(headers http.Header) string {
	sb := strings.Builder{}
	fmt.Fprintf(&sb, "Headers:\n")
	for k, v := range headers {
		val := strings.Join(v, ",")
		if k == "X-Api-Key" {
			if len(val) > 8 {
				val = val[:4] + "***" + val[len(val)-4:]
			} else {
				val = "****"
			}
		}
		fmt.Fprintf(&sb, "  %s: %s\n", k, val)
	}
	return sb.String()
}

// getDumpLimit returns the appropriate dump limit based on content type
func getDumpLimit(contentType string) int {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return binaryDumpSize // Default to binary size for unknown types
	}

	switch {
	case mediaType == jsonMediaType:
		return 0 // No limit for JSON
	case strings.HasPrefix(mediaType, "multipart/"):
		return 0 // No limit for multipart to parse all parts
	default:
		return binaryDumpSize // 1KB for binary content
	}
}

func writeBody(contentType string, body *dumpReader) string {
	sb := strings.Builder{}
	fmt.Fprintf(&sb, "Body:\n")

	data := body.Bytes()
	if len(data) == 0 {
		fmt.Fprintf(&sb, "  (empty)\n")
		return sb.String()
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		// Unknown content type, treat as binary
		return writeBinaryBody(&sb, data, body.Truncated())
	}

	switch {
	case mediaType == jsonMediaType:
		return writeJSONBody(&sb, data)
	case strings.HasPrefix(mediaType, "multipart/"):
		return writeMultipartBody(&sb, data, params["boundary"])
	default:
		return writeBinaryBody(&sb, data, body.Truncated())
	}
}

func writeJSONBody(sb *strings.Builder, data []byte) string {
	sb.WriteString("  [JSON]\n")
	// Add indentation to each line of JSON
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			sb.WriteString("  ")
			sb.WriteString(line)
			sb.WriteRune('\n')
		}
	}
	return sb.String()
}

func writeBinaryBody(sb *strings.Builder, data []byte, truncated bool) string {
	sb.WriteString("  [Binary/HexDump]\n")
	for l := range strings.Lines(hex.Dump(data)) {
		sb.WriteString("  ")
		sb.WriteString(l)
	}
	if truncated {
		fmt.Fprintf(sb, "  ... truncated (showing first %d bytes)\n", binaryDumpSize)
	}
	return sb.String()
}

func writeMultipartBody(sb *strings.Builder, data []byte, boundary string) string {
	sb.WriteString("  [Multipart]\n")

	if boundary == "" {
		sb.WriteString("  Error: No boundary found in multipart content\n")
		return sb.String()
	}

	reader := multipart.NewReader(bytes.NewReader(data), boundary)
	partNum := 1

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(sb, "  Error reading part %d: %s\n", partNum, err.Error())
			break
		}

		fmt.Fprintf(sb, "  --- Part %d ---\n", partNum)

		// Write part headers
		for key, values := range part.Header {
			fmt.Fprintf(sb, "  %s: %s\n", key, strings.Join(values, ", "))
		}

		// Read part content
		partData, err := io.ReadAll(part)
		part.Close()

		if err != nil {
			fmt.Fprintf(sb, "  Error reading part data: %s\n", err.Error())
			continue
		}

		// Determine if this is a text field or binary content
		contentType := part.Header.Get("Content-Type")
		disposition := part.Header.Get("Content-Disposition")

		// Check if it's a form field (simple text)
		if isSimpleFormField(disposition, contentType) {
			sb.WriteString("  Content:\n")
			sb.WriteString("    ")
			sb.Write(partData)
			sb.WriteRune('\n')
		} else {
			// Binary content - show first 1KB as hex dump
			sb.WriteString("  Content: [Binary - HexDump of first 1KB]\n")
			dumpSize := min(len(partData), binaryDumpSize)
			for l := range strings.Lines(hex.Dump(partData[:dumpSize])) {
				sb.WriteString("    ")
				sb.WriteString(l)
			}
			if len(partData) > binaryDumpSize {
				fmt.Fprintf(sb, "    ... (%d more bytes)\n", len(partData)-binaryDumpSize)
			}
		}

		partNum++
	}

	return sb.String()
}

func isSimpleFormField(disposition, contentType string) bool {
	// Parse Content-Disposition header to check if it's a simple form field
	_, params, err := mime.ParseMediaType(disposition)
	if err != nil {
		return false
	}

	// If there's no filename, it's likely a simple form field
	if _, hasFilename := params["filename"]; hasFilename {
		return false
	}

	// If content type is not specified or is text, treat as simple field
	if contentType == "" {
		return true
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	return strings.HasPrefix(mediaType, "text/") || mediaType == "application/x-www-form-urlencoded"
}
