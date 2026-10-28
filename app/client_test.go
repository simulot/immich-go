package app

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const testAssetID = "aaaaaaaa-0000-4000-8000-000000000001"

// fakeImmich is an Immich server that answers the calls of Client.Open, and hands the other calls
// to other. It records when each of those other calls arrived.
func fakeImmich(t *testing.T, other http.HandlerFunc) (*httptest.Server, func() []time.Time) {
	t.Helper()
	var mu sync.Mutex
	var arrivals []time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/server/ping":
			_, _ = io.WriteString(w, `{"res":"pong"}`)
		case "/api/users/me":
			_, _ = io.WriteString(w, `{"id":"1","email":"test@example.com"}`)
		case "/api/server/media-types":
			_, _ = io.WriteString(w, `{"image":[".jpg"],"video":[".mp4"]}`)
		case "/api/server/about":
			_, _ = io.WriteString(w, `{"version":"v2.0.0"}`)
		default:
			mu.Lock()
			arrivals = append(arrivals, time.Now())
			mu.Unlock()
			other(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, func() []time.Time {
		mu.Lock()
		defer mu.Unlock()
		return append([]time.Time(nil), arrivals...)
	}
}

// clientWithFlags is a Client whose flags are parsed from args, for the server url.
func clientWithFlags(ctx context.Context, t *testing.T, url string, args ...string) (*Client, *Application) {
	t.Helper()
	var c Client
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	c.RegisterFlags(flags, "")
	if err := flags.Parse(append([]string{"--server", url, "--api-key", "test-key"}, args...)); err != nil {
		t.Fatalf("flags %v: %v", args, err)
	}
	a := New(ctx, &cobra.Command{})
	a.SetLog(&Log{Logger: slog.New(slog.DiscardHandler)})
	return &c, a
}

func busy(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "busy", http.StatusServiceUnavailable)
}

// --retries sets the tries in all of a failed read, and --retry-wait the first wait, doubled before
// each next one. Below 1 try, or a first wait that is negative or above 1 h, is an error.
func TestRetriesFlag(t *testing.T) {
	ctx := context.Background()
	t.Run("--retries 1 is one try", func(t *testing.T) {
		srv, arrivals := fakeImmich(t, busy)
		c, a := clientWithFlags(ctx, t, srv.URL, "--retries", "1")
		if err := c.Open(ctx, a); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Immich.GetAssetInfo(ctx, testAssetID); err == nil {
			t.Fatal("GetAssetInfo: no error")
		}
		if n := len(arrivals()); n != 1 {
			t.Errorf("%d tries, want 1", n)
		}
	})

	t.Run("--retries 3 --retry-wait 10ms", func(t *testing.T) {
		srv, arrivals := fakeImmich(t, busy)
		c, a := clientWithFlags(ctx, t, srv.URL, "--retries", "3", "--retry-wait", "10ms")
		if err := c.Open(ctx, a); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Immich.GetAssetInfo(ctx, testAssetID); err == nil {
			t.Fatal("GetAssetInfo: no error")
		}
		at := arrivals()
		if len(at) != 3 {
			t.Fatalf("%d tries, want 3", len(at))
		}
		// Each wait is scaled by a random 0.75 to 1.25; the slack above is for a slow machine.
		for i, nominal := range []time.Duration{10 * time.Millisecond, 20 * time.Millisecond} {
			if gap := at[i+1].Sub(at[i]); gap < nominal*3/4 || gap > nominal*5/4+200*time.Millisecond {
				t.Errorf("wait %d: %s between the tries, want about %s", i+1, gap, nominal)
			}
		}
	})

	t.Run("rejected", func(t *testing.T) {
		srv, arrivals := fakeImmich(t, busy)
		for _, args := range [][]string{
			{"--retries", "0"},
			{"--retries", "-1"},
			{"--retry-wait", "-1s"},
			{"--retry-wait", "61m"},
		} {
			c, a := clientWithFlags(ctx, t, srv.URL, args...)
			if err := c.Open(ctx, a); err == nil || !strings.Contains(err.Error(), args[0]) {
				t.Errorf("%s: %v; want an error about %s", strings.Join(args, " "), err, args[0])
			}
		}
		if n := len(arrivals()); n != 0 {
			t.Errorf("%d calls; want none", n)
		}
	})
}

// The help of --client-timeout (and --from-client-timeout) says that it bounds each try, as it does.
func TestClientTimeoutHelpSaysEachTry(t *testing.T) {
	for _, prefix := range []string{"", "from-"} {
		var c Client
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		c.RegisterFlags(flags, prefix)
		f := flags.Lookup(prefix + "client-timeout")
		if f == nil {
			t.Fatalf("no --%sclient-timeout flag", prefix)
		}
		if !strings.Contains(f.Usage, "each try") {
			t.Errorf("--%sclient-timeout help %q; want it to say each try", prefix, f.Usage)
		}
	}
}

// --client-timeout bounds each try, not the whole call: against a server that stalls 200 ms
// before it answers, each try ends after about 50 ms, and the call is tried again as --retries says.
func TestClientTimeoutBoundsEachTry(t *testing.T) {
	ctx := context.Background()
	srv, arrivals := fakeImmich(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(200 * time.Millisecond):
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id": "`+testAssetID+`"}`)
	})
	c, a := clientWithFlags(ctx, t, srv.URL, "--client-timeout", "50ms", "--retries", "3", "--retry-wait", "1ms")
	if err := c.Open(ctx, a); err != nil {
		t.Fatal(err)
	}
	_, err := c.Immich.GetAssetInfo(ctx, testAssetID)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "timeout") {
		t.Fatalf("GetAssetInfo: %v; want a timeout", err)
	}
	at := arrivals()
	if len(at) != 3 {
		t.Fatalf("%d tries, want 3", len(at))
	}
	// Between two tries: the try, cut at 50 ms, and a wait of a few ms. A try that waited for the
	// answer would take 200 ms.
	for i := 1; i < len(at); i++ {
		if gap := at[i].Sub(at[i-1]); gap < 45*time.Millisecond || gap >= 190*time.Millisecond {
			t.Errorf("try %d started %s after try %d; want about 50 ms", i+1, gap, i)
		}
	}
}
