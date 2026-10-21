package immich

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

const (
	testAssetID = "aaaaaaaa-0000-4000-8000-000000000001"
	callRetried = "Immich call failed, trying again"
	downRetried = "Immich download broke off, trying again"
)

// retryClient is a client of url that has reached the server (so failures are tried again), whose
// waits between tries are recorded instead of slept, with no jitter (each wait is the nominal
// one), and whose warnings go to the returned buffer.
func retryClient(t *testing.T, url string) (*ImmichClient, *syncBuffer, *[]time.Duration) {
	t.Helper()
	ic, err := NewImmichClient(url, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	logs := &syncBuffer{}
	ic.log = slog.New(slog.NewTextHandler(logs, nil))
	waits := &[]time.Duration{}
	var mu sync.Mutex
	ic.reachable.Store(true)
	ic.retry.jitter = func() float64 { return 0.5 }
	ic.retry.pause = func(ctx context.Context, d time.Duration) error {
		mu.Lock()
		*waits = append(*waits, d)
		mu.Unlock()
		return ctx.Err()
	}
	return ic, logs, waits
}

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) count(msg string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Count(s.b.String(), msg)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func assetJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"id": "`+testAssetID+`", "originalFileName": "IMG_0001.JPG"}`)
}

func secs(ss ...int) []time.Duration {
	out := make([]time.Duration, len(ss))
	for i, s := range ss {
		out[i] = time.Duration(s) * time.Second
	}
	return out
}

func sameWaits(got []time.Duration, want []time.Duration) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// The server refuses connections for a while, then answers again: the read gets through on its
// second try. The refusal comes from a dialer hook that fails as a refused connect does, so the
// test needs no port to be closed and taken again.
func TestRetryConnectionRefusedThenAnswers(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		assetJSON(w)
	}))
	defer srv.Close()
	ic, logs, waits := retryClient(t, srv.URL)
	var refuse atomic.Bool
	var dials atomic.Int32
	var dialer net.Dialer
	ic.transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dials.Add(1)
		if refuse.Load() {
			return nil, &net.OpError{Op: "dial", Net: network, Err: os.NewSyscallError("connect", syscall.ECONNREFUSED)}
		}
		return dialer.DialContext(ctx, network, addr)
	}
	if _, err := ic.GetAssetInfo(context.Background(), testAssetID); err != nil {
		t.Fatalf("GetAssetInfo, server up: %v", err)
	}
	refuse.Store(true)
	ic.transport.CloseIdleConnections()
	calls.Store(0)
	dials.Store(0)
	pause := ic.retry.pause
	ic.retry.pause = func(ctx context.Context, d time.Duration) error {
		refuse.Store(false)
		return pause(ctx, d)
	}
	a, err := ic.GetAssetInfo(context.Background(), testAssetID)
	if err != nil {
		t.Fatalf("GetAssetInfo: %v", err)
	}
	if a.ID != testAssetID || calls.Load() != 1 || dials.Load() != 2 {
		t.Errorf("asset %q after %d dials, %d calls reached the server; want %s after 2 and 1", a.ID, dials.Load(), calls.Load(), testAssetID)
	}
	if !sameWaits(*waits, secs(15)) || logs.count(callRetried) != 1 || !strings.Contains(logs.String(), "connection refused") {
		t.Errorf("waits %v, want [15s]; log:\n%s", *waits, logs)
	}
}

func TestRetry503Then200(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "busy", http.StatusServiceUnavailable)
			return
		}
		assetJSON(w)
	}))
	defer srv.Close()
	ic, logs, waits := retryClient(t, srv.URL)
	a, err := ic.GetAssetInfo(context.Background(), testAssetID)
	if err != nil || a.ID != testAssetID {
		t.Fatalf("GetAssetInfo: %v, %v", a, err)
	}
	if calls.Load() != 2 || !sameWaits(*waits, secs(15)) || logs.count(callRetried) != 1 || !strings.Contains(logs.String(), "503 Service Unavailable") {
		t.Errorf("%d calls, waits %v; want 2 and [15s]; log:\n%s", calls.Load(), *waits, logs)
	}
}

// 502, 503 and 504 are tried 5 times in all, with waits of 15 s, 30 s, 60 s and 120 s, then the
// last error is returned.
func TestRetryGatewayErrorsGiveUp(t *testing.T) {
	for _, status := range []int{http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "gateway", status)
			}))
			defer srv.Close()
			ic, logs, waits := retryClient(t, srv.URL)
			_, err := ic.GetAssetInfo(context.Background(), testAssetID)
			if err == nil || !strings.Contains(err.Error(), strconv.Itoa(status)) {
				t.Fatalf("GetAssetInfo: %v, want the %d", err, status)
			}
			if calls.Load() != 5 || !sameWaits(*waits, secs(15, 30, 60, 120)) || logs.count(callRetried) != 4 {
				t.Errorf("%d calls, waits %v; want 5 and [15s 30s 1m0s 2m0s]; log:\n%s", calls.Load(), *waits, logs)
			}
		})
	}
}

// Any other answer from the server is its answer: never tried again.
func TestRetryNotOtherStatus(t *testing.T) {
	for _, status := range []int{
		http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
		http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusNotImplemented,
	} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "no", status)
			}))
			defer srv.Close()
			ic, logs, waits := retryClient(t, srv.URL)
			if _, err := ic.GetAssetInfo(context.Background(), testAssetID); err == nil {
				t.Fatal("GetAssetInfo: no error")
			}
			if calls.Load() != 1 || len(*waits) != 0 || logs.count(callRetried) != 0 {
				t.Errorf("%d calls, waits %v; want 1 and none; log:\n%s", calls.Load(), *waits, logs)
			}
		})
	}
}

// A write is never tried again, whatever the failure: it may have reached the server. (A read
// first, so that the server has answered this client: a read would now be tried again.)
func TestRetryNeverAWrite(t *testing.T) {
	writes := []struct {
		name string
		call func(ic *ImmichClient) error
	}{
		{"POST /albums", func(ic *ImmichClient) error {
			_, err := ic.CreateAlbum(context.Background(), "Summer", "", []string{testAssetID})
			return err
		}},
		{"PUT /albums/{id}/assets", func(ic *ImmichClient) error {
			_, err := ic.AddAssetToAlbum(context.Background(), "bbbbbbbb-0000-4000-8000-000000000001", []string{testAssetID})
			return err
		}},
		{"POST /stacks", func(ic *ImmichClient) error {
			_, err := ic.CreateStack(context.Background(), []string{testAssetID, "aaaaaaaa-0000-4000-8000-000000000002"})
			return err
		}},
		{"DELETE /albums/{id}", func(ic *ImmichClient) error {
			return ic.DeleteAlbum(context.Background(), "bbbbbbbb-0000-4000-8000-000000000001")
		}},
		{"PUT /assets", func(ic *ImmichClient) error {
			return ic.UpdateAssets(context.Background(), []string{testAssetID}, false, false, 0, 0, false, "")
		}},
	}
	failures := map[string]http.HandlerFunc{
		"503": func(w http.ResponseWriter, r *http.Request) { http.Error(w, "busy", http.StatusServiceUnavailable) },
		"connection dropped": func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			panic(http.ErrAbortHandler)
		},
	}
	for _, wr := range writes {
		for fname, fail := range failures {
			t.Run(wr.name+", "+fname, func(t *testing.T) {
				var calls atomic.Int32
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodGet && r.URL.Path == "/api/assets/"+testAssetID {
						assetJSON(w)
						return
					}
					calls.Add(1)
					fail(w, r)
				}))
				defer srv.Close()
				ic, logs, waits := retryClient(t, srv.URL)
				if _, err := ic.GetAssetInfo(context.Background(), testAssetID); err != nil {
					t.Fatal(err)
				}
				if err := wr.call(ic); err == nil {
					t.Fatal("no error")
				}
				if calls.Load() != 1 || len(*waits) != 0 || logs.count(callRetried) != 0 {
					t.Errorf("%d calls, waits %v; want 1 and none; log:\n%s", calls.Load(), *waits, logs)
				}
			})
		}
	}
}

// POST /search/metadata only reads, so it is tried again.
func TestRetrySearch(t *testing.T) {
	const checksum = "2jmj7l5rSw0yVb/vlWAYkK/YBwk="
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/search/metadata" {
			t.Errorf("unexpected call %s %s", r.Method, r.URL)
		}
		if calls.Add(1) == 1 {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"assets": {"total": 1, "count": 1, "items": [{"id": "`+testAssetID+`", "checksum": "`+checksum+`"}], "nextPage": null}}`)
	}))
	defer srv.Close()
	ic, logs, _ := retryClient(t, srv.URL)
	list, err := ic.GetAssetsByHash(context.Background(), checksum)
	if err != nil || len(list) != 1 || list[0].ID != testAssetID {
		t.Fatalf("GetAssetsByHash: %v, %v; want the asset %s", list, err, testAssetID)
	}
	if calls.Load() != 2 || logs.count(callRetried) != 1 {
		t.Errorf("%d calls; want 2; log:\n%s", calls.Load(), logs)
	}
}

// The server closes the connection before it answers (EOF): tried again. (Each answer closes its
// connection, so the failing call is on a new one, which Go's transport does not try again itself.)
func TestRetryEOFBeforeResponse(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		if calls.Add(1) == 2 {
			panic(http.ErrAbortHandler)
		}
		assetJSON(w)
	}))
	defer srv.Close()
	ic, logs, _ := retryClient(t, srv.URL)
	for i := 0; i < 2; i++ {
		if _, err := ic.GetAssetInfo(context.Background(), testAssetID); err != nil {
			t.Fatalf("GetAssetInfo %d: %v", i+1, err)
		}
	}
	if calls.Load() != 3 || logs.count(callRetried) != 1 || !strings.Contains(logs.String(), "EOF") {
		t.Errorf("%d calls; want 3; log:\n%s", calls.Load(), logs)
	}
}

// A server that has not answered this client yet is more likely a wrong address: a failure then
// is not tried again, so it fails at once, as a refused connection or a gateway error. (A new
// client is in that state.)
func TestNoRetryBeforeTheServerAnswered(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	ic, logs, waits := retryClient(t, "http://"+addr)
	ic.reachable.Store(false)
	if _, err := ic.GetAssetInfo(context.Background(), testAssetID); err == nil {
		t.Fatal("GetAssetInfo: no error")
	}
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "busy", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	gw, gwLogs, gwWaits := retryClient(t, srv.URL)
	gw.reachable.Store(false)
	if _, err := gw.GetAssetInfo(context.Background(), testAssetID); err == nil {
		t.Fatal("GetAssetInfo: no error")
	}
	if len(*waits) != 0 || logs.count(callRetried) != 0 || len(*gwWaits) != 0 || gwLogs.count(callRetried) != 0 || calls.Load() != 1 {
		t.Errorf("waits %v and %v, %d calls; want none, none and 1; logs:\n%s%s", *waits, *gwWaits, calls.Load(), logs, gwLogs)
	}
	if fresh, err := NewImmichClient(srv.URL, "test-key"); err != nil || fresh.reachable.Load() {
		t.Errorf("a new client is reachable: %v, %v", fresh.reachable.Load(), err)
	}
}

// A call that gives up after all its tries turns the retries off: the next failure fails at once,
// so a long outage costs one round of tries, not one per call. The next answer turns them on.
func TestRetriesOffAfterAGiveUp(t *testing.T) {
	var calls, down atomic.Int32
	down.Store(1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if down.Load() == 1 {
			http.Error(w, "busy", http.StatusServiceUnavailable)
			return
		}
		assetJSON(w)
	}))
	defer srv.Close()
	ic, logs, waits := retryClient(t, srv.URL)
	ctx := context.Background()
	if _, err := ic.GetAssetInfo(ctx, testAssetID); err == nil || calls.Load() != 5 {
		t.Fatalf("first call: %v after %d tries; want an error after 5", err, calls.Load())
	}
	calls.Store(0)
	if _, err := ic.GetAssetInfo(ctx, testAssetID); err == nil || calls.Load() != 1 {
		t.Fatalf("after the give-up: %v after %d tries; want an error after 1", err, calls.Load())
	}
	down.Store(0)
	if _, err := ic.GetAssetInfo(ctx, testAssetID); err != nil {
		t.Fatalf("server back: %v", err)
	}
	down.Store(1)
	calls.Store(0)
	if _, err := ic.GetAssetInfo(ctx, testAssetID); err == nil || calls.Load() != 5 {
		t.Fatalf("after the answer: %v after %d tries; want an error after 5 again", err, calls.Load())
	}
	if len(*waits) != 8 || logs.count(callRetried) != 8 {
		t.Errorf("waits %v; want two rounds of 4; log:\n%s", *waits, logs)
	}

	// The same for a download: after a give-up, a transfer cut midway is not started again.
	content := bigContent(7)
	dsrv, dcalls := original(t, func(int) []byte { return content }, func(n int) int {
		if n == 1 {
			return len(content) / 2
		}
		return -1
	})
	defer dsrv.Close()
	dc, dlogs, _ := retryClient(t, dsrv.URL)
	rc, err := dc.DownloadAsset(ctx, testAssetID)
	if err != nil {
		t.Fatal(err)
	}
	dc.reachable.Store(false) // as when another call gave up meanwhile
	_, err = io.ReadAll(rc)
	rc.Close()
	if err == nil || dcalls.Load() != 1 || dlogs.count(downRetried) != 0 {
		t.Errorf("download after a give-up: %v, %d requests; want an error after 1; log:\n%s", err, dcalls.Load(), dlogs)
	}
}

// A body cut short while its JSON is read is a network failure too: the read is tried again.
func TestRetryJSONBodyCutShort(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", "400")
			_, _ = io.WriteString(w, `{"id": "`+testAssetID+`", "originalF`)
			w.(http.Flusher).Flush()
			panic(http.ErrAbortHandler)
		}
		assetJSON(w)
	}))
	defer srv.Close()
	ic, logs, _ := retryClient(t, srv.URL)
	a, err := ic.GetAssetInfo(context.Background(), testAssetID)
	if err != nil || a.OriginalFileName != "IMG_0001.JPG" {
		t.Fatalf("GetAssetInfo: %+v, %v", a, err)
	}
	if calls.Load() != 2 || logs.count(callRetried) != 1 {
		t.Errorf("%d calls; want 2; log:\n%s", calls.Load(), logs)
	}
}

// A cancel during the wait ends the call at once, with its error.
func TestRetryStopsWhenCanceled(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "busy", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	ic, _, _ := retryClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	ic.retry.pause = func(ctx context.Context, d time.Duration) error {
		cancel()
		return pause(ctx, d)
	}
	if _, err := ic.GetAssetInfo(ctx, testAssetID); err == nil {
		t.Fatal("GetAssetInfo: no error")
	}
	if calls.Load() != 1 {
		t.Errorf("%d calls after the cancel; want 1", calls.Load())
	}
}

// OptionRetries sets the tries and the first wait; 1 try is no retry.
func TestOptionRetries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "busy", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	for _, tc := range []struct{ attempts, calls int }{{1, 1}, {3, 3}} {
		calls.Store(0)
		ic, err := NewImmichClient(srv.URL, "test-key", OptionRetries(tc.attempts, time.Millisecond))
		if err != nil {
			t.Fatal(err)
		}
		ic.log = slog.New(slog.DiscardHandler)
		ic.reachable.Store(true)
		start := time.Now()
		if _, err := ic.GetAssetInfo(context.Background(), testAssetID); err == nil {
			t.Fatal("GetAssetInfo: no error")
		}
		if int(calls.Load()) != tc.calls || time.Since(start) > 5*time.Second {
			t.Errorf("OptionRetries(%d, 1ms): %d calls in %s; want %d", tc.attempts, calls.Load(), time.Since(start), tc.calls)
		}
		// Only a give-up after a try again turns the retries off: with 1 try there is none.
		if want := tc.attempts == 1; ic.reachable.Load() != want {
			t.Errorf("OptionRetries(%d, 1ms): reachable %v after the failure, want %v", tc.attempts, ic.reachable.Load(), want)
		}
	}
	if _, err := NewImmichClient(srv.URL, "test-key", OptionRetries(0, time.Second)); err == nil {
		t.Error("OptionRetries(0, 1s): no error")
	}
}

func TestRetryDelays(t *testing.T) {
	p := defaultRetryPolicy()
	for _, j := range []float64{0, 0.5, 0.999} {
		p.jitter = func() float64 { return j }
		for n, nominal := range secs(15, 30, 60, 120) {
			d := p.delay(n + 1)
			if d < nominal*3/4 || d > nominal*5/4 {
				t.Errorf("jitter %v: wait %d is %s, want %s +-25%%", j, n+1, d, nominal)
			}
		}
	}
	if p.attempts != 5 {
		t.Errorf("%d tries, want 5", p.attempts)
	}
}

func TestTransient(t *testing.T) {
	tlsAlert := &net.OpError{Op: "remote error", Err: errors.New("tls: bad certificate")}
	for _, tc := range []struct {
		err  error
		want bool
	}{
		{nil, false},
		{&url.Error{Op: "Get", URL: "https://x/api/assets", Err: &net.OpError{Op: "dial", Net: "tcp", Err: os.ErrDeadlineExceeded}}, true},
		{&net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}, true},
		{&net.OpError{Op: "dial", Net: "tcp", Err: &net.DNSError{Err: "no such host", Name: "x"}}, true},
		{&net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET}, true},
		{&net.OpError{Op: "write", Net: "tcp", Err: syscall.EPIPE}, true},
		{&url.Error{Op: "Get", URL: "https://x/api/assets", Err: io.EOF}, true},
		{io.ErrUnexpectedEOF, true},
		{fmt.Errorf("can't decode JSON response: %w", io.ErrUnexpectedEOF), true},
		{&url.Error{Op: "Get", URL: "https://x", Err: os.ErrDeadlineExceeded}, true},
		{&url.Error{Op: "Get", URL: "https://x", Err: tlsAlert}, false},
		{&url.Error{Op: "Get", URL: "https://x", Err: x509.UnknownAuthorityError{}}, false},
		{fmt.Errorf("can't decode JSON response: %w", errors.New("invalid character 'x'")), false},
		{context.Canceled, false},
		{errors.New("bad"), false},
	} {
		if got := transient(tc.err); got != tc.want {
			t.Errorf("transient(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}

	// A real refused connect, to a port no one listens on any more.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	if conn, err := net.DialTimeout("tcp", addr, 5*time.Second); err == nil {
		conn.Close()
		t.Logf("%s was taken again meanwhile; the real refused connect is not checked", addr)
	} else if !transient(err) {
		t.Errorf("transient(%v) = false for a refused connect", err)
	}
}

// original serves content at GET /assets/{id}/original; cut(n) says how much of the n-th request
// (from 1) it sends before it drops the connection, or -1 for all of it; body(n) what it sends.
func original(t *testing.T, content func(n int) []byte, cut func(n int) int) (*httptest.Server, *atomic.Int32) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/assets/"+testAssetID+"/original" {
			t.Errorf("unexpected call %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", http.StatusNotFound)
			return
		}
		n := int(calls.Add(1))
		b := content(n)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(len(b)))
		if c := cut(n); c >= 0 {
			_, _ = w.Write(b[:c])
			w.(http.Flusher).Flush()
			panic(http.ErrAbortHandler)
		}
		_, _ = w.Write(b)
	}))
	return srv, &calls
}

func bigContent(seed byte) []byte {
	b := make([]byte, 1<<20)
	for i := range b {
		b[i] = byte(i*7) ^ seed
	}
	return b
}

// A download that breaks off midway starts again from the first byte, and the reader gets the
// whole original.
func TestDownloadRestartsAfterCut(t *testing.T) {
	content := bigContent(1)
	srv, calls := original(t, func(int) []byte { return content }, func(n int) int {
		if n == 1 {
			return len(content) / 2
		}
		return -1
	})
	defer srv.Close()
	ic, logs, waits := retryClient(t, srv.URL)
	rc, err := ic.DownloadAsset(context.Background(), testAssetID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("read %d bytes, not the %d of the original", len(got), len(content))
	}
	if calls.Load() != 2 || !sameWaits(*waits, secs(15)) || logs.count(downRetried) != 1 {
		t.Errorf("%d requests, waits %v; want 2 and [15s]; log:\n%s", calls.Load(), *waits, logs)
	}
}

// Cut before its first byte, then refused at first on the next try: still the whole original.
func TestDownloadRestartsAfterCutAndGatewayError(t *testing.T) {
	content := bigContent(2)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			panic(http.ErrAbortHandler)
		case 2:
			http.Error(w, "busy", http.StatusServiceUnavailable)
		default:
			_, _ = w.Write(content)
		}
	}))
	defer srv.Close()
	ic, logs, waits := retryClient(t, srv.URL)
	rc, err := ic.DownloadAsset(context.Background(), testAssetID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(rc)
	rc.Close()
	if err != nil || !bytes.Equal(got, content) {
		t.Fatalf("read %d bytes, %v; want the %d of the original", len(got), err, len(content))
	}
	if calls.Load() != 3 || !sameWaits(*waits, secs(15, 30)) || logs.count(downRetried) != 2 {
		t.Errorf("%d requests, waits %v; want 3 and [15s 30s]; log:\n%s", calls.Load(), *waits, logs)
	}
}

// When the next try sends other bytes than those already read, the download fails: the reader
// never gets a mix of two files.
func TestDownloadRestartRefusesAnotherOriginal(t *testing.T) {
	a, b := bigContent(3), bigContent(4)
	srv, _ := original(t, func(n int) []byte {
		if n == 1 {
			return a
		}
		return b
	}, func(n int) int {
		if n == 1 {
			return len(a) / 2
		}
		return -1
	})
	defer srv.Close()
	ic, _, _ := retryClient(t, srv.URL)
	rc, err := ic.DownloadAsset(context.Background(), testAssetID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(rc)
	rc.Close()
	if err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("read %d bytes, %v; want an error saying the original changed", len(got), err)
	}
	if len(got) > len(a)/2 || !bytes.Equal(got, a[:len(got)]) {
		t.Fatalf("the reader got bytes of the other file")
	}
}

// Shorter on the next try than what was already read: the download fails.
func TestDownloadRestartRefusesShorterOriginal(t *testing.T) {
	a := bigContent(5)
	srv, _ := original(t, func(n int) []byte {
		if n == 1 {
			return a
		}
		return a[:len(a)/4]
	}, func(n int) int {
		if n == 1 {
			return len(a) / 2
		}
		return -1
	})
	defer srv.Close()
	ic, _, _ := retryClient(t, srv.URL)
	rc, err := ic.DownloadAsset(context.Background(), testAssetID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(rc)
	rc.Close()
	if err == nil || !strings.Contains(err.Error(), "shorter") {
		t.Fatalf("%v; want an error saying the original is shorter", err)
	}
}

// Cut every time: 5 transfers, then the error.
func TestDownloadGivesUp(t *testing.T) {
	content := bigContent(6)
	srv, calls := original(t, func(int) []byte { return content }, func(int) int { return len(content) / 2 })
	defer srv.Close()
	ic, logs, waits := retryClient(t, srv.URL)
	rc, err := ic.DownloadAsset(context.Background(), testAssetID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(rc)
	rc.Close()
	if err == nil {
		t.Fatal("read: no error")
	}
	if calls.Load() != 5 || !sameWaits(*waits, secs(15, 30, 60, 120)) || logs.count(downRetried) != 4 {
		t.Errorf("%d requests, waits %v; want 5 and [15s 30s 1m0s 2m0s]; log:\n%s", calls.Load(), *waits, logs)
	}
}
