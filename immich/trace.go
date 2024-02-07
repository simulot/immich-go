package immich

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

/*
  To inspect requests or response request, add setTraceJSONRequest or setTraceJSONResponse to the request options

*/

type smartBodyCloser struct {
	r    io.Reader
	body io.ReadCloser
}

func (sb *smartBodyCloser) Close() error {
	fmt.Println("\n--- BODY ---")
	return sb.body.Close()
}

func (sb *smartBodyCloser) Read(b []byte) (int, error) {
	return sb.r.Read(b)
}

func setTraceJSONRequest() serverRequestOption {
	return func(sc *serverCall, req *http.Request) error {
		fmt.Println("--------------------")
		fmt.Println(req.Method, req.URL.String())
		for h, v := range req.Header {
			fmt.Println(h, v)
		}
		if req.Body != nil {
			tr := io.TeeReader(req.Body, os.Stdout)
			req.Body = &smartBodyCloser{body: req.Body, r: tr}
		}
		return nil
	}
}
