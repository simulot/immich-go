package httptrace

import (
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const timeFormat = "2006-01-02T15:04:05.999Z07:00"

type Tracer struct {
	lock       sync.Mutex
	out        io.Writer
	instanceId atomic.Int64
}

func NewTracer(out io.Writer) *Tracer {
	return &Tracer{
		out:  out,
		lock: sync.Mutex{},
	}
}

type roundTripTrace struct {
	ht       *Tracer       // reference to the HTTP Tracer
	instance int           // Client ID
	reqId    int           // Roundtrip id
	req      requestTrace  // the request
	resp     responseTrace // the response
}

func (ht *Tracer) newRoundTripTrace(instance, reqId int) *roundTripTrace {
	return &roundTripTrace{
		ht:       ht,
		instance: instance,
		reqId:    reqId,
	}
}

func (rt *roundTripTrace) Write() {
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
		io.WriteString(rt.ht.out, rt.req.String())
		fmt.Fprintf(rt.ht.out, "+---- response  -----\n")
		if rt.resp.resp != nil {
			io.WriteString(rt.ht.out, rt.resp.String())
		} else if rt.resp.err != nil {
			fmt.Fprintf(rt.ht.out, "Error: %s\n", rt.resp.err.Error())
		}
		fmt.Fprintf(rt.ht.out, "\\---- client #%d request  #%d ---------------------------------------------------\n", rt.instance, rt.reqId)
		fmt.Fprint(rt.ht.out, "\n")
	}()
}

type requestTrace struct {
	req       *http.Request
	isJson    bool
	body      *dumpReader
	timestamp time.Time
}

func (rt requestTrace) String() string {
	sb := strings.Builder{}
	fmt.Fprintf(&sb, "Method: %s\n", rt.req.Method)
	fmt.Fprintf(&sb, "URL: %s\n", rt.req.URL)
	sb.WriteString(writeHeaders(rt.req.Header))
	fmt.Fprintf(&sb, "Timestamp: %s\n", rt.timestamp.Format(timeFormat))
	if rt.body != nil {
		sb.WriteString(writeBody(rt.isJson, rt.body))
	}
	return sb.String()
}

type responseTrace struct {
	err       error
	resp      *http.Response
	isJson    bool
	body      *dumpReader
	timestamp time.Time
	duration  time.Duration
}

func (rt *responseTrace) String() string {
	sb := strings.Builder{}
	fmt.Fprintln(&sb, "Response:")
	fmt.Fprintf(&sb, "Status: %s: %d\n", rt.resp.Status, rt.resp.StatusCode)
	sb.WriteString(writeHeaders(rt.resp.Header))
	fmt.Fprintf(&sb, "Timestamp: %s\n", rt.timestamp.Format(timeFormat))
	fmt.Fprintf(&sb, "Duration: %s\n", rt.duration)
	if rt.body != nil {
		sb.WriteString(writeBody(rt.isJson, rt.body))
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

func writeBody(isJSON bool, body *dumpReader) string {
	body.lock.Lock()
	defer body.lock.Unlock()
	sb := strings.Builder{}
	fmt.Fprintf(&sb, "Body:\n")
	if isJSON {
		sb.Write(body.buffer.Bytes())
		sb.WriteRune('\n')
		return sb.String()
	}
	for l := range strings.Lines(hex.Dump(body.buffer.Bytes())) {
		sb.WriteString("  ")
		sb.WriteString(l)
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
