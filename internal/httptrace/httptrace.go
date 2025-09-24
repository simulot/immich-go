/*
httptrace implement an http.RoundTripper that dumps the requests and their responses on a io.Writer
*/

package httptrace

import (
	"net/http"
	"time"
)

type TraceRoundTripper struct {
	rt  http.RoundTripper
	out *HttpTracer
}

func NewDumpRoundTripper(out *HttpTracer, rt http.RoundTripper) *TraceRoundTripper {
	return &TraceRoundTripper{out: out, rt: rt}
}

func (drt *TraceRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	reqTrace := requestTrace{
		Method:    req.Method,
		URL:       req.URL.String(),
		Headers:   req.Header,
		Timestamp: start,
	}
	if req.Body != nil {
		if isJSON(req.Header.Get("Content-Type")) {
			reqTrace.Body = newDumpReader(req.Body, 0)
		} else {
			reqTrace.Body = newDumpReader(req.Body, maxBodyDumpSize)
		}
		req.Body = reqTrace.Body
	}

	resp, err := drt.rt.RoundTrip(req)
	if err != nil {
		drt.out.Write(&reqTrace, nil, err)
		return nil, err
	}

	respTrace := responseTrace{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Timestamp:  time.Now(),
		Duration:   time.Since(start).Round(time.Millisecond),
	}
	if resp.Body != nil {
		if isJSON(resp.Header.Get("Content-Type")) {
			respTrace.Body = newDumpReader(resp.Body, 0)
		} else {
			respTrace.Body = newDumpReader(resp.Body, maxBodyDumpSize)
		}
		resp.Body = respTrace.Body
	}

	drt.out.Write(&reqTrace, &respTrace, nil)
	return resp, nil
}
