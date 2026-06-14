package immich

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type trackingReadCloser struct {
	reader io.Reader
	closed atomic.Bool
}

func (trc *trackingReadCloser) Read(p []byte) (int, error) {
	return trc.reader.Read(p)
}

func (trc *trackingReadCloser) Close() error {
	trc.closed.Store(true)
	return nil
}

type testServer struct {
	// endpoint       string
	responseStatus int
	responseBody   string
}

func TestCallRetriesTransientServerError(t *testing.T) {
	t.Parallel()

	var (
		mu       sync.Mutex
		attempts int
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		current := attempts
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if current < 3 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"Bad Gateway","statusCode":502,"message":"upstream busy"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	ic, err := NewImmichClient(server.URL, "1234")
	if err != nil {
		t.Fatalf("NewImmichClient() error = %v", err)
	}

	resp := map[string]string{}
	err = ic.newServerCall(context.Background(), "retry-test").do(getRequest("/assets", setAcceptJSON()), responseJSON(&resp))
	if err != nil {
		t.Fatalf("do() error = %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("response status = %q, want ok", resp["status"])
	}

	mu.Lock()
	defer mu.Unlock()
	if attempts != 3 {
		t.Fatalf("attempt count = %d, want 3", attempts)
	}
}

func TestShouldRetryCall(t *testing.T) {
	t.Parallel()

	err := callError{status: http.StatusBadGateway}
	if !shouldRetryCall(err, 1, 3, true) {
		t.Fatal("expected 502 to be retryable")
	}
	if shouldRetryCall(err, 3, 3, true) {
		t.Fatal("did not expect retry on last attempt")
	}
	if shouldRetryCall(context.Canceled, 1, 3, true) {
		t.Fatal("did not expect context cancellation to be retryable")
	}
	if !shouldRetryCall(errors.New("Put \"https://example.com/api/assets/1\": context deadline exceeded (Client.Timeout exceeded while awaiting headers)"), 1, 3, true) {
		t.Fatal("expected timeout transport error to be retryable")
	}
	if shouldRetryCall(err, 1, 3, false) {
		t.Fatal("did not expect retry when disabled")
	}
}

func TestRetryDelayUsesExponentialBackoffBounds(t *testing.T) {
	t.Parallel()

	ic := &ImmichClient{RetryBackoff: time.Second, RetryMaxDelay: 5 * time.Second}
	for i := 1; i <= 6; i++ {
		d := ic.retryDelay(i)
		min := time.Second
		for j := 1; j < i; j++ {
			if min < 5*time.Second {
				min *= 2
				if min > 5*time.Second {
					min = 5 * time.Second
				}
			}
		}
		if d < min {
			t.Fatalf("retryDelay(%d)=%s, want at least %s", i, d, min)
		}
		if d > min+min/2+time.Nanosecond {
			t.Fatalf("retryDelay(%d)=%s, want at most %s", i, d, min+min/2)
		}
	}
}

func TestCallRetryLogsAtInfoHook(t *testing.T) {
	t.Parallel()

	var logs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"Bad Gateway","statusCode":502,"message":"upstream busy"}`))
	}))
	defer server.Close()

	ic, err := NewImmichClient(server.URL, "1234", OptionRetryLogger(func(_ context.Context, msg string, args ...any) {
		logs = append(logs, msg)
	}), OptionRetryPolicy(3, time.Millisecond, time.Millisecond))
	if err != nil {
		t.Fatalf("NewImmichClient() error = %v", err)
	}

	err = ic.newServerCall(context.Background(), "retry-test").do(getRequest("/assets", setAcceptJSON()))
	if err == nil {
		t.Fatal("expected error")
	}
	want := []string{"retrying Immich request", "retrying Immich request"}
	if !reflect.DeepEqual(logs, want) {
		t.Fatalf("retry logs = %#v, want %#v", logs, want)
	}
}

func TestCallClosesResponseBodyWhenReturningNonRetryableError(t *testing.T) {
	t.Parallel()

	body := &trackingReadCloser{reader: io.NopCloser(strings.NewReader(`{"error":"bad request","statusCode":400,"message":"nope"}`))}

	ic, err := NewImmichClient("https://example.com", "1234", OptionRetryPolicy(1, time.Millisecond, time.Millisecond))
	if err != nil {
		t.Fatalf("NewImmichClient() error = %v", err)
	}
	ic.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       body,
			Request:    req,
		}, nil
	})

	err = ic.newServerCall(context.Background(), "retry-test").do(getRequest("/assets", setAcceptJSON()))
	if err == nil {
		t.Fatal("expected error")
	}
	if !body.closed.Load() {
		t.Fatal("expected response body to be closed")
	}
}

func (ts *testServer) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(ts.responseStatus)
	_, _ = resp.Write([]byte(ts.responseBody))
}

func TestCall(t *testing.T) {
	tt := []struct {
		name        string
		requestFn   requestFunction
		expectedErr bool
		server      testServer
	}{
		{
			name:        "happy path",
			requestFn:   getRequest("/assets", setAcceptJSON()),
			expectedErr: false,
			server: testServer{
				responseStatus: http.StatusOK,
				responseBody:   `{"status": "All correct"}`,
			},
		},
		{
			name:        "bad url",
			requestFn:   getRequest("/ass\nets", setAcceptJSON()),
			expectedErr: true,
			server: testServer{
				responseStatus: http.StatusOK,
				responseBody:   `{"status": "All correct"}`,
			},
		},
		{
			name:        "post / ok",
			requestFn:   postRequest("/albums", "application/json", setAcceptJSON(), setJSONBody(struct{ Name string }{Name: "test"})),
			expectedErr: false,
			server: testServer{
				responseStatus: http.StatusOK,
				responseBody:   `{"Name": "test"}`,
			},
		},
		{
			name:        "bad request / post",
			requestFn:   postRequest("/albums", "application/json", setAcceptJSON(), setJSONBody(struct{ Name string }{Name: "test"})),
			expectedErr: true,
			server: testServer{
				responseStatus: http.StatusBadRequest,
				responseBody:   `{"error": "Bad request", "statusCode": "400", "message": ["String1","String2"]}`,
			},
		},
	}

	for _, tst := range tt {
		t.Run(tst.name, func(t *testing.T) {
			server := httptest.NewServer(&tst.server)
			defer server.Close()
			ctx := context.Background()
			ic, err := NewImmichClient(server.URL, "1234")
			if err != nil {
				t.Fail()
				return
			}
			// ic.EnableAppTrace(true)
			r := map[string]string{}
			err = ic.newServerCall(ctx, tst.name).do(tst.requestFn, responseJSON(&r))
			if tst.expectedErr && err == nil {
				t.Errorf("expected error, but no error")
			}
			if !tst.expectedErr && err != nil {
				t.Errorf("no error expected, but error: %s", err.Error())
			}
			if err != nil {
				t.Logf("error received: %s", err.Error())
			}
			t.Logf("response received: %#v", r)
		})
	}
}
