package immich

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/simulot/immich-go/helpers/gen"
)

/*
  To inspect requests or response request, add setTraceJSONRequest or setTraceJSONResponse to the request options

*/

type limitWriter struct {
	W     io.Writer
	Err   error
	Lines int
}

func newLimitWriter(w io.Writer, lines int) *limitWriter {
	return &limitWriter{W: w, Lines: lines, Err: nil}
}

func (lw *limitWriter) Write(b []byte) (int, error) {
	if lw.Lines < 0 {
		return 0, lw.Err
	}
	total := 0
	for len(b) > 0 && lw.Lines >= 0 && lw.Err == nil {
		p := bytes.Index(b, []byte{'\n'})
		var n int
		if p > 0 {
			n, lw.Err = lw.W.Write(b[:p+1])
			b = b[p+1:]
			lw.Lines--
		} else {
			n, lw.Err = lw.W.Write(b)
		}
		total += n
	}
	if lw.Lines < 0 {
		_, _ = lw.W.Write([]byte(".... truncated ....\n"))
	}
	return total, lw.Err
}

func (lw *limitWriter) Close() error {
	if closer, ok := lw.W.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

type smartBodyCloser struct {
	r    io.Reader
	body io.ReadCloser
	w    io.Writer
}

func (sb *smartBodyCloser) Close() error {
	fmt.Fprint(sb.w, "-- request body end --\n\n")
	return sb.body.Close()
}

func (sb *smartBodyCloser) Read(b []byte) (int, error) {
	return sb.r.Read(b)
}

func setTraceRequest() serverRequestOption {
	return func(sc *serverCall, req *http.Request) error {
		seq := sc.ctx.Value(ctxCallSequenceID)
		fmt.Fprintln(sc.ic.apiTraceWriter, time.Now().Format(time.RFC3339), "QUERY", seq, sc.endPoint, req.Method, req.URL.String())
		for h, v := range req.Header {
			if h == "X-Api-Key" {
				fmt.Fprintln(sc.ic.apiTraceWriter, "  ", h, "redacted")
			} else {
				fmt.Fprintln(sc.ic.apiTraceWriter, "  ", h, v)
			}
		}
		if v := sc.ctx.Value(ctxCallValues); v != nil {
			if values, ok := v.(map[string]string); ok {
				keys := gen.MapKeys(values)
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Fprintln(sc.ic.apiTraceWriter, "  ", k+": ", values[k])
				}
			}
		}
		if req.Header.Get("Content-Type") == "application/json" {
			fmt.Fprintln(sc.ic.apiTraceWriter, "-- request JSON Body --")
			if req.Body != nil {
				// tr := io.TeeReader(req.Body, newLimitWriter(sc.ic.apiTraceWriter, 100))
				tr := io.TeeReader(req.Body, sc.ic.apiTraceWriter)
				req.Body = &smartBodyCloser{body: req.Body, r: tr, w: sc.ic.apiTraceWriter}
			}
		} else {
			if req.Body != nil {
				fmt.Fprintln(sc.ic.apiTraceWriter, "-- Empty body or binary body not dumped --")
			}
		}
		return nil
	}
}
