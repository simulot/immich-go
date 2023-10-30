package immich

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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
func setTraceJSONResponse() serverResponseOption {
	return func(sc *serverCall, resp *http.Response) error {
		fmt.Println("--- API RESPONSE -- ")
		for h, v := range resp.Header {
			fmt.Println(h, strings.Join(v, ","))
		}
		fmt.Println("--- RESPONSE BODY ---")
		tr := io.TeeReader(resp.Body, os.Stdout)
		resp.Body = &smartBodyCloser{body: resp.Body, r: tr}
		return nil
	}
}

func traceRequest(req *http.Request) {
	isJSON := req.Header.Get("Content-Type") == "application/json"
	fmt.Println("--- API CALL ---")
	u := *req.URL
	u.Host = "***"
	fmt.Println(req.Method, u.String())
	for h, vs := range req.Header {
		if h == "X-Api-Key" {
			vs = []string{"***"}
		}
		fmt.Println(h, ":", strings.Join(vs, ","))
	}
	if isJSON {
		fmt.Println("--- JSON BODY ---")
		tr := io.TeeReader(req.Body, os.Stdout)
		req.Body = &smartBodyCloser{body: req.Body, r: tr}
	}

}
