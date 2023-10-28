package immich

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type TooManyInternalError struct {
	error
}

func (e TooManyInternalError) Is(target error) bool {
	_, ok := target.(*TooManyInternalError)
	return ok
}

// serverCall permit to decorate request and responses in one line
type serverCall struct {
	endPoint string
	ic       *ImmichClient
	err      error
	ctx      context.Context
}

type serverCallOption func(sc *serverCall) error

// callError represents errors returned by the server
type callError struct {
	endPoint string
	method   string
	url      string
	status   int
	err      error
	message  *ServerMessage
}

type ServerMessage struct {
	Error      string   `json:"error"`
	StatusCode string   `json:"statusCode"`
	Message    []string `json:"message"`
}

func (u callError) Is(target error) bool {
	_, ok := target.(*callError)
	return ok
}

func (ce callError) Error() string {
	b := strings.Builder{}
	b.WriteString(ce.endPoint)
	b.WriteString(", ")
	b.WriteString(ce.method)
	b.WriteString(", ")
	b.WriteString(ce.url)
	if ce.status > 0 {
		b.WriteString(", ")
		b.WriteString(fmt.Sprintf("%d %s", ce.status, http.StatusText(ce.status)))
	}
	b.WriteRune('\n')
	if ce.err != nil && !errors.Is(ce.err, &callError{}) {
		b.WriteString(ce.err.Error())
		b.WriteRune('\n')
	}
	if ce.message != nil {
		if len(ce.message.Error) > 0 {
			b.WriteString(ce.message.Error)
			b.WriteRune('\n')
		}
		if len(ce.message.Message) > 0 {
			for _, m := range ce.message.Message {
				b.WriteString(m)
				b.WriteRune('\n')
			}
		}
	}
	return b.String()
}

func (ic *ImmichClient) newServerCall(ctx context.Context, api string, opts ...serverCallOption) *serverCall {
	sc := &serverCall{
		endPoint: api,
		ic:       ic,
		ctx:      ctx,
	}
	if sc.err == nil {
		for _, opt := range opts {
			sc.joinError(opt(sc))
		}
	}
	return sc
}

func (sc *serverCall) Err(req *http.Request, resp *http.Response, msg *ServerMessage) error {
	ce := callError{
		endPoint: sc.endPoint,
		err:      sc.err,
	}
	if req != nil {
		ce.method = req.Method
		ce.url = req.URL.String()
	}
	if resp != nil {
		ce.status = resp.StatusCode
	}
	ce.message = msg
	return ce
}
func (sc *serverCall) joinError(err error) error {
	sc.err = errors.Join(sc.err, err)
	return err
}

type requestFunction func(sc *serverCall) *http.Request

func (sc *serverCall) request(method string, url string, opts ...serverRequestOption) *http.Request {

	req, err := http.NewRequestWithContext(sc.ctx, method, url, nil)
	if sc.joinError(err) != nil {
		return nil
	}

	opts = append(opts, setAPIKey())
	for _, opt := range opts {
		if sc.joinError(opt(sc, req)) != nil {
			return nil
		}
	}
	return req
}

func get(url string, opts ...serverRequestOption) requestFunction {
	return func(sc *serverCall) *http.Request {
		if sc.err != nil {
			return nil
		}
		return sc.request(http.MethodGet, sc.ic.endPoint+url, opts...)
	}
}
func post(url string, ctype string, opts ...serverRequestOption) requestFunction {
	return func(sc *serverCall) *http.Request {
		if sc.err != nil {
			return nil
		}
		return sc.request(http.MethodPost, sc.ic.endPoint+url, append(opts, setContentType(ctype))...)
	}
}

func delete(url string, opts ...serverRequestOption) requestFunction {
	return func(sc *serverCall) *http.Request {
		if sc.err != nil {
			return nil
		}
		return sc.request(http.MethodDelete, sc.ic.endPoint+url, opts...)
	}
}

func put(url string, opts ...serverRequestOption) requestFunction {
	return func(sc *serverCall) *http.Request {
		if sc.err != nil {
			return nil
		}
		return sc.request(http.MethodPut, sc.ic.endPoint+url, opts...)
	}
}

func (sc *serverCall) do(fnRequest requestFunction, opts ...serverResponseOption) error {
	if sc.err != nil || fnRequest == nil {
		return sc.Err(nil, nil, nil)
	}

	req := fnRequest(sc)
	if sc.err != nil || req == nil {
		return sc.Err(req, nil, nil)
	}

	var (
		resp *http.Response
		err  error
	)

	resp, err = sc.ic.client.Do(req)

	// any non nil error must be returned
	if err != nil {
		sc.joinError(err)
		return sc.Err(req, nil, nil)
	}

	// Any StatusCode above 300 denote a problem
	if resp.StatusCode >= 300 {
		msg := ServerMessage{}
		if resp.Body != nil {
			if json.NewDecoder(resp.Body).Decode(&msg) == nil {
				return sc.Err(req, resp, &msg)
			}
		}
		if resp.Body != nil {
			resp.Body.Close()
		}
		// StatusCode below 500 are
		return sc.Err(req, resp, &msg)
	}

	// We have a success
	for _, opt := range opts {
		sc.joinError(opt(sc, resp))
	}
	if sc.err != nil {
		return sc.Err(req, resp, nil)
	}
	return nil
}

type serverRequestOption func(sc *serverCall, req *http.Request) error

func setBody(body io.ReadCloser) serverRequestOption {
	return func(sc *serverCall, req *http.Request) error {
		req.Body = body
		return nil
	}
}

func setHeader(key, value string) serverRequestOption {
	return func(sc *serverCall, req *http.Request) error {
		req.Header.Set(key, value)
		return nil
	}
}
func setAcceptJSON() serverRequestOption {
	return func(sc *serverCall, req *http.Request) error {
		req.Header.Add("Accept", "application/json")
		return nil
	}
}

func setAPIKey() serverRequestOption {
	return func(sc *serverCall, req *http.Request) error {
		req.Header.Set("x-api-key", sc.ic.key)
		return nil
	}
}

func setJSONBody(object any) serverRequestOption {
	return func(sc *serverCall, req *http.Request) error {
		b := bytes.NewBuffer(nil)
		enc := json.NewEncoder(b)
		if sc.ic.ApiTrace {
			enc.SetIndent("", " ")
		}
		if sc.joinError(enc.Encode(object)) == nil {
			req.Body = io.NopCloser(b)
		}
		req.Header.Set("Content-Type", "application/json")
		return sc.err
	}
}

func setContentType(ctype string) serverRequestOption {
	return func(sc *serverCall, req *http.Request) error {
		req.Header.Set("Content-Type", ctype)
		return sc.err
	}
}

func setUrlValues(values url.Values) serverRequestOption {
	return func(sc *serverCall, req *http.Request) error {
		if values != nil {
			req.URL.RawPath = values.Encode()
		}
		return sc.err
	}
}

type serverResponseOption func(sc *serverCall, resp *http.Response) error

func responseJSON[T any](object *T) serverResponseOption {
	return func(sc *serverCall, resp *http.Response) error {
		if resp != nil {
			if resp.Body != nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusNoContent {
					return nil
				}

				if sc.joinError(json.NewDecoder(resp.Body).Decode(object)) != nil {
					return sc.err
				}
				return nil
			}
		}
		return errors.New("can't decode nil response")
	}
}
