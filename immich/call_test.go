package immich

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

func TestOptionRetries(t *testing.T) {
	// Test that OptionRetries sets the retry configuration correctly
	ic, err := NewImmichClient("http://test.example.com", "test-key", OptionRetries(5, 2*time.Second))
	if err != nil {
		t.Fatalf("Failed to create client with retries: %v", err)
	}

	if ic.Retries != 5 {
		t.Errorf("Expected Retries to be 5, got %d", ic.Retries)
	}

	if ic.RetriesDelay != 2*time.Second {
		t.Errorf("Expected RetriesDelay to be 2s, got %v", ic.RetriesDelay)
	}
}

func TestOptionRetriesMinimumValue(t *testing.T) {
	// Test that OptionRetries enforces minimum value of 1
	ic, err := NewImmichClient("http://test.example.com", "test-key", OptionRetries(0, time.Second))
	if err != nil {
		t.Fatalf("Failed to create client with zero retries: %v", err)
	}

	if ic.Retries != 1 {
		t.Errorf("Expected Retries to be 1 (minimum), got %d", ic.Retries)
	}
}
