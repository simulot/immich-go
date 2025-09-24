package httptrace

import (
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"sync"
	"time"
)

const timeFormat = "2006-01-02T15:04:05.999Z07:00"

type HttpTracer struct {
	lock  sync.Mutex
	out   io.Writer
	count int
}

func NewHttpTracer(out io.Writer) *HttpTracer {
	return &HttpTracer{
		out:  out,
		lock: sync.Mutex{},
	}
}

func (ht *HttpTracer) Write(req *requestTrace, resp *responseTrace, rtErr error) {
	go func() {
		if req.Body != nil {
			req.Body.closed.Wait()
		}
		if resp != nil && resp.Body != nil {
			resp.Body.closed.Wait()
		}
		ht.lock.Lock()
		defer ht.lock.Unlock()
		ht.count++
		fmt.Fprintf(ht.out, "/---- request  #%d -----\n", ht.count)
		io.WriteString(ht.out, req.String())
		fmt.Fprintf(ht.out, "+---- response #%d -----\n", ht.count)
		if resp != nil {
			io.WriteString(ht.out, resp.String())
		} else if rtErr != nil {
			fmt.Fprintf(ht.out, "Error: %s\n", rtErr.Error())
		}
		fmt.Fprintf(ht.out, "\\---- response #%d -----\n", ht.count)
	}()
}

type requestTrace struct {
	Method    string
	URL       string
	Headers   map[string][]string
	Body      *dumpReader
	Timestamp time.Time
}

func (rt *requestTrace) String() string {
	sb := strings.Builder{}
	fmt.Fprintln(&sb, "Request:")
	fmt.Fprintf(&sb, "Method: %s\n", rt.Method)
	fmt.Fprintf(&sb, "URL: %s\n", rt.URL)
	sb.WriteString(writeHeaders(rt.Headers))
	fmt.Fprintf(&sb, "Timestamp: %s\n", rt.Timestamp.Format(timeFormat))
	if rt.Body != nil {
		sb.WriteString(writeBody(rt.Body))
	}
	return sb.String()
}

type responseTrace struct {
	Status     string
	StatusCode int
	Headers    map[string][]string
	Body       *dumpReader
	Timestamp  time.Time
	Duration   time.Duration
}

func (rt *responseTrace) String() string {
	sb := strings.Builder{}
	fmt.Fprintln(&sb, "Response:")
	fmt.Fprintf(&sb, "Status: %s (%d)\n", rt.Status, rt.StatusCode)
	sb.WriteString(writeHeaders(rt.Headers))
	fmt.Fprintf(&sb, "Timestamp: %s\n", rt.Timestamp.Format(timeFormat))
	fmt.Fprintf(&sb, "Duration: %s\n", rt.Duration)
	if rt.Body != nil {
		sb.WriteString(writeBody(rt.Body))
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

func writeBody(body *dumpReader) string {
	body.lock.Lock()
	defer body.lock.Unlock()
	sb := strings.Builder{}
	fmt.Fprintf(&sb, "Body:\n")
	for l := range strings.Lines(hex.Dump(body.buffer.Bytes())) {
		fmt.Fprintf(&sb, "  %s\n", l)
	}
	if body.truncated {
		fmt.Fprintf(&sb, "  ... truncated\n")
	}
	return sb.String()
}

func isJSON(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/json"
}
