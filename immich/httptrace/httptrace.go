/*
httptrace implement an http.RoundTripper that dumps the requests and their responses on a io.Writer
*/

package httptrace

import (
	"net/http"
	"sync/atomic"
	"time"
)

var _ http.RoundTripper = &TraceRoundTripper{}

type TraceRoundTripper struct {
	instance int
	originRT http.RoundTripper
	ht       *Tracer
	reqId    atomic.Int64
}

// DecorateRT decorates the given round tripper to implement logging
func (ht *Tracer) DecorateRT(rt http.RoundTripper) http.RoundTripper {
	return &TraceRoundTripper{
		ht:       ht,
		originRT: rt,
		instance: int(ht.instanceId.Add(1)),
	}
}

func (drt *TraceRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt := drt.ht.newRoundTripTrace(drt.instance, int(drt.reqId.Add(1)))
	rt.req.timestamp = time.Now()
	rt.req.req = req
	if req.Body != nil {
		if isJSON(req.Header.Get("Content-Type")) {
			rt.req.isJson = true
			rt.req.body = newDumpReader(req.Body, 0)
		} else {
			rt.req.body = newDumpReader(req.Body, maxBodyDumpSize)
		}
		req.Body = rt.req.body
	}

	resp, err := drt.originRT.RoundTrip(req)
	rt.resp.timestamp = time.Now()
	rt.resp.duration = rt.resp.timestamp.Sub(rt.req.timestamp).Round(time.Millisecond)
	rt.resp.err = err
	if err != nil {
		rt.Write()
		return nil, err
	}
	rt.resp.resp = resp
	if resp.Body != nil {
		if isJSON(resp.Header.Get("Content-Type")) {
			rt.resp.isJson = true
			rt.resp.body = newDumpReader(resp.Body, 0)
		} else {
			rt.resp.body = newDumpReader(resp.Body, maxBodyDumpSize)
		}
		resp.Body = rt.resp.body
	}

	rt.Write()
	return resp, nil
}
