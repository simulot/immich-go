package immich

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// serverCall permit to decorate request and responses in one line
type serverCall struct {
	method      string
	ic          *ImmichClient
	req         *http.Request
	resp        *http.Response
	serverError serverError
	err         error
}

type serverError struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Error      string `json:"error"`
}

func (ic *ImmichClient) newServerCall(method string) *serverCall {
	return &serverCall{
		method: method,
		ic:     ic,
	}
}

func (sc *serverCall) Err() error {
	if sc.err == nil {
		return nil
	}

	var method, url, message string
	if sc.req != nil {
		method = sc.req.Method
		url = sc.req.URL.String()
		message = sc.serverError.Message
	}

	return fmt.Errorf("Error during the call of %q:\n%s %s\n%s\n%w", sc.method, method, url, message, sc.err)
}

func (sc *serverCall) joinError(err error) error {
	sc.err = errors.Join(sc.err, err)
	return err
}

func (sc *serverCall) getRequest(url string) *serverCall {
	req, err := http.NewRequest(http.MethodGet, sc.ic.endPoint+url, nil)
	sc.err = errors.Join(err)
	sc.req = req
	return sc
}

func (sc *serverCall) addKey() error {
	sc.req.Header.Add("x-api-key", sc.ic.key)
	return sc.err
}

func (sc *serverCall) encodeJSONRequest(object any) *serverCall {
	if sc.err != nil {
		return sc
	}
	sc.req.Header.Add("Content-Type", "application/json")
	buf := bytes.NewBuffer(nil)
	sc.joinError(json.NewEncoder(buf).Encode(object))
	sc.req.Body = io.NopCloser(buf)
	return sc
}

func (sc *serverCall) setAccept(t string) *serverCall {
	sc.req.Header.Add("Accept", t)
	return sc
}

func (sc *serverCall) callServer() *serverCall {
	if sc.err != nil {
		return sc
	}
	var serverError serverError
	sc.addKey()
	n := sc.ic.Retries
	for n > 0 {
		resp, err := sc.ic.client.Do(sc.req)

		if err != nil {
			sc.err = errors.Join(sc.err)
			sc.resp = resp
			return sc
		}
		if resp.StatusCode < 400 {
			sc.resp = resp
			return sc
		}

		if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
			json.NewDecoder(resp.Body).Decode(&serverError)
			resp.Body.Close()
			resp.Body = nil
		}
		if resp.StatusCode < 500 {
			sc.resp = resp
			sc.joinError(errors.New(sc.resp.Status))
			sc.serverError = serverError
			return sc
		}
		if resp.StatusCode >= 500 {
			sc.resp = resp
			n--
			time.Sleep(sc.ic.RetriesDelay)
		}
	}
	sc.serverError = serverError
	sc.joinError(fmt.Errorf("server error: %s", sc.resp.Status))
	return sc
}

func (sc *serverCall) decodeJSONResponse(object any) *serverCall {
	if sc.resp != nil && sc.resp.Body != nil {
		sc.joinError(json.NewDecoder(sc.resp.Body).Decode(object))
	}
	return sc
}

func (sc *serverCall) postFormRequest(url string, form MultipartWriter) *serverCall {
	if sc.err != nil {
		return sc
	}

	// build request body
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	err := sc.joinError(form.WriteMultiPart(w))
	if err != nil {
		return sc
	}
	err = sc.joinError(w.Close())
	if err != nil {
		return sc
	}

	req, err := http.NewRequest(http.MethodPost, sc.ic.endPoint+url, nil)
	if sc.joinError(err) != nil {
		return sc
	}
	sc.req = req
	sc.req.Header.Add("Content-Type", w.FormDataContentType())
	sc.req.Header.Add("Accept", "application/json")
	sc.req.Body = io.NopCloser(body)
	return sc
}
