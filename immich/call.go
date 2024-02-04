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
	"strconv"
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
	p        *paginator
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

func (ce callError) Is(target error) bool {
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

// paginator controls the paged API calls
type paginator struct {
	pageNumber    int    // current page
	pageParameter string // page parameter name on the URL
	EOF           bool   // true when the last page was empty
}

func (p paginator) setPage(v url.Values) {
	v.Set(p.pageParameter, strconv.Itoa(p.pageNumber))
}

func (p *paginator) nextPage() {
	p.pageNumber++
}

func setPaginator() serverCallOption {
	return func(sc *serverCall) error {
		p := paginator{
			pageParameter: "page",
			pageNumber:    1,
		}
		sc.p = &p
		return nil
	}
}

type requestFunction func(sc *serverCall) *http.Request

func (sc *serverCall) request(method string, url string, opts ...serverRequestOption) *http.Request {
	req, err := http.NewRequestWithContext(sc.ctx, method, url, http.NoBody)
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

func deleteItem(url string, opts ...serverRequestOption) requestFunction {
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

	if sc.p == nil {
		return sc._callDo(fnRequest, opts...)
	}

	for !sc.p.EOF {
		err := sc._callDo(fnRequest, opts...)
		if err != nil {
			return err
		}
		sc.p.nextPage()
	}
	return nil
}

func (sc *serverCall) _callDo(fnRequest requestFunction, opts ...serverResponseOption) error {
	var (
		resp *http.Response
		err  error
	)

	req := fnRequest(sc)
	if sc.err != nil || req == nil {
		return sc.Err(req, nil, nil)
	}

	if sc.p != nil {
		v := req.URL.Query()
		sc.p.setPage(v)
		req.URL.RawQuery = v.Encode()
	}
	if sc.ic.APITrace /* && req.Header.Get("Content-Type") == "application/json"*/ {
		setTraceJSONRequest()(sc, req)
	}

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
		if sc.ic.APITrace {
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

func setURLValues(values url.Values) serverRequestOption {
	return func(sc *serverCall, req *http.Request) error {
		if values != nil {
			rValues := req.URL.Query()
			for k, v := range values {
				for _, s := range v {
					rValues.Set(k, s)
				}
			}
			req.URL.RawQuery = rValues.Encode()
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

func responseAccumulateJSON[T any](acc *[]T) serverResponseOption {
	return func(sc *serverCall, resp *http.Response) error {
		if sc.p != nil {
			sc.p.EOF = true
		}
		if resp != nil {
			if resp.Body != nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusNoContent {
					return nil
				}
				arr := []T{}
				if sc.joinError(json.NewDecoder(resp.Body).Decode(&arr)) != nil {
					return sc.err
				}
				if len(arr) > 0 && sc.p != nil {
					sc.p.EOF = false
				}
				(*acc) = append((*acc), arr...)
				return nil
			}
		}
		return errors.New("can't decode nil response")
	}
}

func responseJSONWithFilter[T any](filter func(*T)) serverResponseOption {
	return func(sc *serverCall, resp *http.Response) error {
		if sc.p != nil {
			sc.p.EOF = true
		}
		if resp != nil {
			if resp.Body != nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusNoContent {
					return nil
				}
				dec := json.NewDecoder(resp.Body)
				// read open bracket "["
				_, err := dec.Token()
				if sc.joinError(err) != nil {
					return sc.err
				}

				// while the array contains values
				for dec.More() {
					var o T
					// decode an array value (Message)
					err := dec.Decode(&o)
					if sc.joinError(err) != nil {
						return sc.err
					}
					if sc.p != nil {
						sc.p.EOF = false
					}
					filter(&o)
				}
				// read closing bracket "]"
				_, err = dec.Token()
				if sc.joinError(err) != nil {
					return sc.err
				}

				return nil
			}
		}
		return errors.New("can't decode nil response")
	}
}
