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

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
)

const (
	EndPointGetJobs                = "GetJobs"
	EndPointSendJobCommand         = "SendJobCommand"
	EndPointCreateJob              = "CreateJob"
	EndPointGetAllAlbums           = "GetAllAlbums"
	EndPointGetAlbumInfo           = "GetAlbumInfo"
	EndPointAddAsstToAlbum         = "AddAssetToAlbum"
	EndPointCreateAlbum            = "CreateAlbum"
	EndPointGetAssetAlbums         = "GetAssetAlbums"
	EndPointDeleteAlbum            = "DeleteAlbum"
	EndPointPingServer             = "PingServer"
	EndPointValidateConnection     = "ValidateConnection"
	EndPointGetServerStatistics    = "GetServerStatistics"
	EndPointGetAssetStatistics     = "GetAssetStatistics"
	EndPointGetSupportedMediaTypes = "GetSupportedMediaTypes"
	EndPointGetAllAssets           = "GetAllAssets"
	EndPointUpsertTags             = "UpsertTags"
	EndPointTagAssets              = "TagAssets"
	EndPointBulkTagAssets          = "BulkTagAssets"
	EndPointGetAllTags             = "GetAllTags"
	EndPointAssetUpload            = "AssetUpload"
	EndPointAssetReplace           = "AssetReplace"
	EndPointGetAboutInfo           = "GetAboutInfo"
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
	message  *ServerErrorMessage
}

type ServerErrorMessage struct {
	Error         string `json:"error"`
	StatusCode    int    `json:"statusCode"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlationId"`
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
		b.WriteString(ce.message.Message)
		b.WriteRune('\n')
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

func (sc *serverCall) Err(req *http.Request, resp *http.Response, msg *ServerErrorMessage) error {
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

type callSequenceID string

const ctxCallSequenceID callSequenceID = "api-call-sequence"

func (sc *serverCall) request(
	method string,
	url string,
	opts ...serverRequestOption,
) *http.Request {
	if sc.ic.apiTraceWriter != nil && sc.endPoint != EndPointGetJobs {
		seq := callSequence.Add(1)
		sc.ctx = context.WithValue(sc.ctx, ctxCallSequenceID, seq)
	}
	req, err := http.NewRequestWithContext(sc.ctx, method, url, http.NoBody)
	if sc.joinError(err) != nil {
		return nil
	}
	opts = append(opts, setAPIKey())
	for _, opt := range opts {
		if opt != nil {
			if sc.joinError(opt(sc, req)) != nil {
				return nil
			}
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
		return sc.request(
			http.MethodPost,
			sc.ic.endPoint+url,
			append(opts, setContentType(cType))...)
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

	if sc.ic.apiTraceWriter != nil && sc.endPoint != EndPointGetJobs {
		_ = sc.joinError(setTraceRequest()(sc, req))
	}

	resp, err = sc.ic.client.Do(req)
	// any non nil error must be returned
	if err != nil {
		err = sc.joinError(err)
		if sc.ic.apiTraceWriter != nil && sc.endPoint != EndPointGetJobs {
			seq := sc.ctx.Value(ctxCallSequenceID)
			fmt.Fprintln(
				sc.ic.apiTraceWriter,
				time.Now().Format(time.RFC3339),
				"RESPONSE",
				seq,
				sc.endPoint,
			)
			fmt.Fprintln(sc.ic.apiTraceWriter, "  Error:", err.Error())
		}
		return sc.Err(req, nil, nil)
	}

	// Any StatusCode above 300 denotes a problem
	if resp.StatusCode >= 300 {
		msg := ServerErrorMessage{}
		if resp.Body != nil {
			defer resp.Body.Close()
			b := bytes.NewBuffer(nil)
			_, _ = io.Copy(b, resp.Body)
			if json.NewDecoder(b).Decode(&msg) == nil {
				if sc.ic.apiTraceWriter != nil && sc.endPoint != EndPointGetJobs {
					seq := sc.ctx.Value(ctxCallSequenceID)
					fmt.Fprintln(
						sc.ic.apiTraceWriter,
						time.Now().Format(time.RFC3339),
						"RESPONSE",
						seq,
						sc.endPoint,
						resp.Request.Method,
						resp.Request.URL.String(),
					)
					fmt.Fprintln(sc.ic.apiTraceWriter, "  Status:", resp.Status)
					fmt.Fprintln(sc.ic.apiTraceWriter, "-- response body --")
					dec := json.NewEncoder(newLimitWriter(sc.ic.apiTraceWriter, 100))
					dec.SetIndent("", " ")
					fmt.Fprint(sc.ic.apiTraceWriter, "-- response body end --\n\n")
				}
				return sc.Err(req, resp, &msg)
			} else {
				if sc.ic.apiTraceWriter != nil && sc.endPoint != EndPointGetJobs {
					seq := sc.ctx.Value(ctxCallSequenceID)
					fmt.Fprintln(
						sc.ic.apiTraceWriter,
						time.Now().Format(time.RFC3339),
						"RESPONSE",
						seq,
						sc.endPoint,
						resp.Request.Method,
						resp.Request.URL.String(),
					)
					fmt.Fprintln(sc.ic.apiTraceWriter, "  Status:", resp.Status)
					fmt.Fprintln(sc.ic.apiTraceWriter, "-- response body --")
					fmt.Fprintln(sc.ic.apiTraceWriter, b.String())
					fmt.Fprint(sc.ic.apiTraceWriter, "-- response body end --\n\n")
				}
			}
		}
		return sc.Err(req, resp, &msg)
	}

	// We have a success
	for _, opt := range opts {
		if opt != nil {
			_ = sc.joinError(opt(sc, resp))
		}
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

func setImmichChecksum(a *assets.Asset) serverRequestOption {
	if a.Checksum == "" {
		return nil
	}
	return func(sc *serverCall, req *http.Request) error {
		req.Header.Set("x-immich-checksum", a.Checksum)
		return nil
	}
}

func setAcceptJSON() serverRequestOption {
	return func(sc *serverCall, req *http.Request) error {
		req.Header.Add("Accept", "application/json")
		return nil
	}
}

func setOctetStream() serverRequestOption {
	return func(sc *serverCall, req *http.Request) error {
		req.Header.Add("Accept", "application/octet-stream")
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

				if sc.ic.apiTraceWriter != nil && sc.endPoint != EndPointGetJobs {
					sc.ic.apiTraceLock.Lock()
					defer sc.ic.apiTraceLock.Unlock()
					resp.Body = hijackBody(resp.Body, sc.ic.apiTraceWriter)
					seq := sc.ctx.Value(ctxCallSequenceID)
					fmt.Fprintln(
						sc.ic.apiTraceWriter,
						time.Now().Format(time.RFC3339),
						"RESPONSE",
						seq,
						sc.endPoint,
						resp.Request.Method,
						resp.Request.URL.String(),
					)
					fmt.Fprintln(sc.ic.apiTraceWriter, "  Header:")
					for k, v := range resp.Header {
						fmt.Fprintln(sc.ic.apiTraceWriter, "    ", k, ":", strings.Join(v, "; "))
					}
					fmt.Fprintln(sc.ic.apiTraceWriter, "  Status:", resp.Status)
					fmt.Fprintln(sc.ic.apiTraceWriter, "-- response body start --")
					defer fmt.Fprint(sc.ic.apiTraceWriter, "\n-- response body end --\n\n")
				}

				err := json.NewDecoder(resp.Body).Decode(object)
				if err != nil {
					err = fmt.Errorf("can't decode JSON response: %w", err)
				}
				return err
			}
		}
		return errors.New("can't decode nil response")
	}
}

func responseCopy(buffer *bytes.Buffer) serverResponseOption {
	return func(sc *serverCall, resp *http.Response) error {
		if resp != nil {
			if resp.Body != nil {
				newBody := fshelper.TeeReadCloser(resp.Body, buffer)
				resp.Body = newBody
				return nil
			}
		}
		return nil
	}
}

func responseOctetStream(rc *io.ReadCloser) serverResponseOption {
	return func(sc *serverCall, resp *http.Response) error {
		if resp != nil {
			if resp.Body != nil {
				*rc = resp.Body
				return nil
			}
		}
		return nil
	}
}
