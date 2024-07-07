package immich

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
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
		if ce.message.Error != "" {
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

func (ic *ImmichClient) newServerCall(ctx context.Context, api string) *serverCall {
	sc := &serverCall{
		endPoint: api,
		ic:       ic,
		ctx:      ctx,
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

var callSequence atomic.Int64

const callSequenceID = "api-call-sequence"

func (sc *serverCall) request(method string, url string, opts ...serverRequestOption) *http.Request {
	if sc.ic.apiTraceWriter != nil {
		seq := callSequence.Add(1)
		sc.ctx = context.WithValue(sc.ctx, callSequenceID, seq)
	}
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

func getRequest(url string, opts ...serverRequestOption) requestFunction {
	return func(sc *serverCall) *http.Request {
		if sc.err != nil {
			return nil
		}
		return sc.request(http.MethodGet, sc.ic.endPoint+url, opts...)
	}
}

func postRequest(url string, cType string, opts ...serverRequestOption) requestFunction {
	return func(sc *serverCall) *http.Request {
		if sc.err != nil {
			return nil
		}
		return sc.request(http.MethodPost, sc.ic.endPoint+url, append(opts, setContentType(cType))...)
	}
}

func deleteRequest(url string, opts ...serverRequestOption) requestFunction {
	return func(sc *serverCall) *http.Request {
		if sc.err != nil {
			return nil
		}
		return sc.request(http.MethodDelete, sc.ic.endPoint+url, opts...)
	}
}

func putRequest(url string, opts ...serverRequestOption) requestFunction {
	return func(sc *serverCall) *http.Request {
		if sc.err != nil {
			return nil
		}
		return sc.request(http.MethodPut, sc.ic.endPoint+url, opts...)
	}
}

func (sc *serverCall) do(fnRequest requestFunction, opts ...serverResponseOption) error {
	var (
		resp *http.Response
		err  error
	)

	req := fnRequest(sc)
	if sc.err != nil || req == nil {
		return sc.Err(req, nil, nil)
	}

	if sc.ic.apiTraceWriter != nil /* && req.Header.Get("Content-Type") == "application/json"*/ {
		_ = sc.joinError(setTraceRequest()(sc, req))
	}

	resp, err = sc.ic.client.Do(req)
	// any non nil error must be returned
	if err != nil {
		_ = sc.joinError(err)
		return sc.Err(req, nil, nil)
	}

	// Any StatusCode above 300 denotes a problem
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
		return sc.Err(req, resp, &msg)
	}

	// We have a success
	for _, opt := range opts {
		_ = sc.joinError(opt(sc, resp))
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
		if sc.ic.apiTraceWriter != nil {
			enc.SetIndent("", " ")
		}
		err := enc.Encode(object)
		if err != nil {
			return err
		}
		req.Body = io.NopCloser(b)
		req.Header.Set("Content-Type", "application/json")
		return err
	}
}

func setContentType(cType string) serverRequestOption {
	return func(sc *serverCall, req *http.Request) error {
		req.Header.Set("Content-Type", cType)
		return nil
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
				err := json.NewDecoder(resp.Body).Decode(object)
				if sc.ic.apiTraceWriter != nil {
					seq := sc.ctx.Value(callSequenceID)
					fmt.Fprintln(sc.ic.apiTraceWriter, time.Now().Format(time.RFC3339), "RESPONSE", seq, sc.endPoint, resp.Request.Method, resp.Request.URL.String())
					fmt.Fprintln(sc.ic.apiTraceWriter, "-- response body --")
					dec := json.NewEncoder(newLimitWriter(sc.ic.apiTraceWriter, 100))
					dec.SetIndent("", " ")
					_ = dec.Encode(object)
					fmt.Fprint(sc.ic.apiTraceWriter, "-- response body end --\n\n")
				}
				return err
			}
		}
		return errors.New("can't decode nil response")
	}
}

/*
func responseAccumulateJSON[T any](acc *[]T) serverResponseOption {
	return func(sc *serverCall, resp *http.Response) error {
		eof := true
		if resp != nil {
			if resp.Body != nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusNoContent {
					return nil
				}
				arr := []T{}
				err := json.NewDecoder(resp.Body).Decode(&arr)
				if err != nil {
					return err
				}
				if len(arr) > 0 && sc.p != nil {
					eof = false
				}
				(*acc) = append((*acc), arr...)
				if eof {
					sc.p.setEOF()
				}
				return nil
			}
		}
		return errors.New("can't decode nil response")
	}
}
*/
/*
func responseJSONWithFilter[T any](filter func(*T)) serverResponseOption {
	return func(sc *serverCall, resp *http.Response) error {
		eof := true
		if resp != nil {
			if resp.Body != nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusNoContent {
					return nil
				}
				dec := json.NewDecoder(resp.Body)
				// read open bracket "["
				_, err := dec.Token()
				if err != nil {
					return nil
				}

				// while the array contains values
				for dec.More() {
					var o T
					// decode an array value (Message)
					err = dec.Decode(&o)
					if err != nil {
						return err
					}
					if sc.p != nil {
						eof = false
					}
					filter(&o)
				}
				// read closing bracket "]"
				_, err = dec.Token()
				if eof {
					sc.p.setEOF()
				}
				return err
			}
		}
		return errors.New("can't decode nil response")
	}
}
*/
