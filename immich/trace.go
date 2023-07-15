package immich

import (
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
)

type traceRoundTripper struct {
}

func (tr traceRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Println("-------------------------")
	fmt.Println(req.Method, req.URL.String())
	for k, h := range req.Header {
		fmt.Println(k, h)
	}
	if req.Body != nil {
		fmt.Println("Body:")
		req.Body = makeReqLogger(req.Body)
	}
	return http.DefaultTransport.RoundTrip(req)
}

type smartReadCloser struct {
	Tag string
	io.Reader
}

func makeSmartReadCloser(r io.Reader) io.ReadCloser {
	return &smartReadCloser{Reader: r}
}

func (src *smartReadCloser) Close() error {
	if c, ok := src.Reader.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

type reqLogger struct {
	io.ReadCloser
	dumper io.Writer
}

func makeReqLogger(r io.ReadCloser) io.ReadCloser {

	dumper := hex.Dumper(os.Stdout)
	r = makeSmartReadCloser(io.TeeReader(r, dumper))
	return &reqLogger{
		dumper:     dumper,
		ReadCloser: r,
	}
}

func (l *reqLogger) Read(b []byte) (int, error) {
	l.dumper.Write(b)
	return l.ReadCloser.Read(b)
}
