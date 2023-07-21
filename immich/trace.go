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
		tr := io.TeeReader(req.Body, os.Stdout)
		req.Body = &smartBodyCloser{body: req.Body, r: tr}
		return nil
	}
}
func setTraceJSONResponse() serverResponseOption {
	return func(sc *serverCall, resp *http.Response) error {
		fmt.Println("--------------------")
		fmt.Println(resp.Request.Method, resp.Request.URL.String())
		for h, v := range resp.Header {
			fmt.Println(h, v)
		}
		tr := io.TeeReader(resp.Body, os.Stdout)
		resp.Body = &smartBodyCloser{body: resp.Body, r: tr}
		return nil
	}
}
