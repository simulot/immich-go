package immich

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testServer struct {
	// endpoint       string
	responseStatus int
	responseBody   string
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

func TestCallRetry_TransientError(t *testing.T) {
	attempts := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"Service Unavailable","statusCode":503,"message":"overloaded"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	ic, err := NewImmichClient(server.URL, "key")
	if err != nil {
		t.Fatal(err)
	}
	ic.RetryEnabled = true

	r := map[string]string{}
	err = ic.newServerCall(context.Background(), "test").
		do(getRequest("/test", setAcceptJSON()), responseJSON(&r))
	if err != nil {
		t.Errorf("expected success after retries, got error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestCallRetry_NonRetryableError(t *testing.T) {
	attempts := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"Forbidden","statusCode":403,"message":"no access"}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	ic, err := NewImmichClient(server.URL, "key")
	if err != nil {
		t.Fatal(err)
	}
	ic.RetryEnabled = true

	r := map[string]string{}
	err = ic.newServerCall(context.Background(), "test").
		do(getRequest("/test", setAcceptJSON()), responseJSON(&r))
	if err == nil {
		t.Error("expected error for 403, got nil")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt (no retry for 403), got %d", attempts)
	}
}

func TestCallRetry_Disabled(t *testing.T) {
	attempts := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"Service Unavailable","statusCode":503,"message":"overloaded"}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	ic, err := NewImmichClient(server.URL, "key")
	if err != nil {
		t.Fatal(err)
	}
	// RetryEnabled defaults to false

	r := map[string]string{}
	err = ic.newServerCall(context.Background(), "test").
		do(getRequest("/test", setAcceptJSON()), responseJSON(&r))
	if err == nil {
		t.Error("expected error, got nil")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt (retry disabled), got %d", attempts)
	}
}
