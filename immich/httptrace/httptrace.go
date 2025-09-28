/*
httptrace implement an http.RoundTripper that dumps the requests and their responses on a io.Writer
*/

package httptrace

import (
	"net/http"
	"sync/atomic"
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
	req.Body = rt.Request(req)
	resp, err := drt.originRT.RoundTrip(req)
	resp.Body = rt.Response(resp, err)
	rt.Dump()
	return resp, nil
}
