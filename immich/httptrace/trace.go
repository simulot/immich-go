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
	if req.Body != nil {
		if isJSON(req.Header.Get("Content-Type")) {
			rt.req.isJson = true
			rt.req.body = newDumpReader(req.Body, 0)
		} else {
			rt.req.body = newDumpReader(req.Body, maxBodyDumpSize)
		}
		return rt.req.body
	}
	return req.Body
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
		fmt.Fprint(rt.ht.out, rt.req.String())
		fmt.Fprintf(rt.ht.out, "+---- response  ---------------------------------------------------\n")
		if rt.resp.resp != nil {
			fmt.Fprint(rt.ht.out, rt.resp.String())
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

func (rt *roundTripTrace) Response(resp *http.Response, err error) io.ReadCloser {
	rt.resp.timestamp = time.Now()
	rt.resp.duration = rt.resp.timestamp.Sub(rt.req.timestamp).Round(time.Millisecond)
	rt.resp.err = err
	if err != nil {
		return nil
	}
	rt.resp.resp = resp
	if resp.Body != nil {
		if isJSON(resp.Header.Get("Content-Type")) {
			rt.resp.isJson = true
			rt.resp.body = newDumpReader(resp.Body, 0)
		} else {
			rt.resp.body = newDumpReader(resp.Body, maxBodyDumpSize)
		}
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
		sb.WriteString(writeBody(rt.isJson, rt.body))
		rt.body.Done()
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
	fmt.Fprintf(&sb, "Status: %s: %d\n", rt.resp.Status, rt.resp.StatusCode)
	fmt.Fprintf(&sb, "Timestamp: %s\n", rt.timestamp.Format(timeFormat))
	fmt.Fprintf(&sb, "Duration: %s\n", rt.duration)
	sb.WriteString(writeHeaders(rt.resp.Header))
	if rt.body != nil {
		sb.WriteString(writeBody(rt.isJson, rt.body))
		rt.body.Done()
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
