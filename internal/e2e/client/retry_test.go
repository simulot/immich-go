//go:build e2e

package client

import (
	"context"
	"io/fs"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/app/root"
	"github.com/simulot/immich-go/internal/assettracker"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"github.com/simulot/immich-go/internal/fileevent"
)

const (
	retryData       = "DATA/fromFolder/recursive/a/b/c" // 10 images, each with its own name
	retryAssetCount = 10
	retryCutAt      = 2               // the n-th download of an original breaks off halfway
	retryOutage     = 2 * time.Second // then every GET gets 503, as from a proxy while Immich restarts
)

// TestArchiveDownloadCutMidwayIsRetried breaks off the download of an original halfway, then
// answers 503 to every GET for 2 s. The download starts again once the server is back, and every
// asset is archived whole.
func TestArchiveDownloadCutMidwayIsRetried(t *testing.T) {
	adm, u, assets := retrySetup(t)
	proxy := newRetryCutProxy(t, retryCutAt, retryOutage)
	dir := t.TempDir()

	// With the default 5 tries, the waits from 200 ms (0.2, 0.4, 0.8 and 1.6 s, each scaled by
	// 0.75 to 1.25) add up to 2.25 s at least: more than the outage.
	a, err := retryArchive(t.Context(), proxy.URL, u, adm, dir, "--from-retry-wait=200ms")
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	cut := proxy.cut()
	if cut == "" {
		t.Fatal("no download was cut")
	}
	if got := proxy.downloads(cut); got < 3 {
		t.Errorf("asset %s: %d downloads, want the one cut, one or more answered 503, and the one that ends it", cut, got)
	}
	e2eutils.CheckResults(t, map[fileevent.Code]int64{
		fileevent.ProcessedFileArchived: retryAssetCount,
		fileevent.ErrorFileAccess:       0,
	}, false, a.FileProcessor())
	if c := a.FileProcessor().GetAssetCounters(); c.Errors != 0 {
		t.Errorf("%d assets in error, want 0", c.Errors)
	}
	retryCheckArchive(t, dir, assets)
}

// TestArchiveNoRetryWhenDisabled is TestArchiveDownloadCutMidwayIsRetried with
// --from-retries=1 (no retry): the download cut short is not tried again, and its asset is an
// error.
func TestArchiveNoRetryWhenDisabled(t *testing.T) {
	adm, u, _ := retrySetup(t)
	proxy := newRetryCutProxy(t, retryCutAt, retryOutage)

	// The other reads during the outage fail at once too, and --on-errors=stop ends the run: only
	// the asset cut short is checked.
	a, _ := retryArchive(t.Context(), proxy.URL, u, adm, t.TempDir(), "--from-retries=1")
	cut := proxy.cut()
	if cut == "" {
		t.Fatal("no download was cut")
	}
	if got := proxy.downloads(cut); got != 1 {
		t.Errorf("asset %s: %d downloads, want 1", cut, got)
	}
	state := "not seen"
	for _, r := range a.FileProcessor().Tracker().GetAllAssets() {
		if r.File.Name() == cut {
			state = r.State.String()
		}
	}
	if want := assettracker.StateError.String(); state != want {
		t.Errorf("asset %s: %s, want %s", cut, state, want)
	}
}

// retryOriginal matches the download of an asset's original, and captures the asset's ID.
var retryOriginal = regexp.MustCompile(`^/api/assets/([^/]+)/original$`)

// retryCutProxy is a reverse proxy to the e2e server. It breaks off one download of an original
// halfway, then answers 503 to every GET for a while.
type retryCutProxy struct {
	*httptest.Server
	mu           sync.Mutex
	total        int            // GET /original so far, the 503s included
	byAsset      map[string]int // the same by asset ID
	cutID        string         // the asset whose download was cut
	unavailUntil time.Time      // the end of the 503s
}

// newRetryCutProxy starts a retryCutProxy. The cutAt-th download of an original sends the length
// of the whole file and half of it, then the connection drops. From then on, every GET gets 503
// for outage.
func newRetryCutProxy(t *testing.T, cutAt int, outage time.Duration) *retryCutProxy {
	t.Helper()
	target, err := url.Parse(ImmichURL)
	if err != nil {
		t.Fatal(err)
	}
	rp := &httputil.ReverseProxy{Rewrite: func(r *httputil.ProxyRequest) { r.SetURL(target) }}
	p := &retryCutProxy{byAsset: map[string]int{}}
	p.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := retryOriginal.FindStringSubmatch(r.URL.Path)
		isDownload := r.Method == http.MethodGet && m != nil
		p.mu.Lock()
		unavailable := r.Method == http.MethodGet && time.Now().Before(p.unavailUntil)
		cut := false
		if isDownload {
			p.total++
			p.byAsset[m[1]]++
			cut = p.total == cutAt
		}
		p.mu.Unlock()
		switch {
		case unavailable:
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		case !cut:
			rp.ServeHTTP(w, r)
			return
		}
		full := httptest.NewRecorder()
		rp.ServeHTTP(full, r)
		if full.Code != http.StatusOK {
			t.Errorf("download to cut: %d %s", full.Code, full.Body)
		}
		body := full.Body.Bytes()
		maps.Copy(w.Header(), full.Header())
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(full.Code)
		_, _ = w.Write(body[:len(body)/2])
		_ = http.NewResponseController(w).Flush()
		p.mu.Lock()
		p.cutID, p.unavailUntil = m[1], time.Now().Add(outage)
		p.mu.Unlock()
		panic(http.ErrAbortHandler) // drops the connection: the client gets a body cut short
	}))
	t.Cleanup(p.Close)
	return p
}

// cut returns the ID of the asset whose download was cut, or "" when none was.
func (p *retryCutProxy) cut() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cutID
}

// downloads returns how many times the asset's original was asked for, the 503s included.
func (p *retryCutProxy) downloads(id string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.byAsset[id]
}

// retrySetup uploads retryData into a new user's account and returns the admin, the user, and
// the user's assets.
func retrySetup(t *testing.T) (user, user, []*e2eutils.Asset) {
	t.Helper()
	adm, err := getUser("admin@immich.app")
	if err != nil {
		t.Fatalf("can't get admin user: %v", err)
	}
	u, err := createUser("minimal")
	if err != nil {
		t.Fatalf("can't create user: %v", err)
	}

	ctx := t.Context()
	c, a := root.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + ImmichURL,
		"--api-key=" + u.APIKey,
		"--admin-api-key=" + adm.APIKey,
		"--no-ui",
		"--log-level=debug",
		retryData,
	})
	if err := c.ExecuteContext(ctx); err != nil {
		if a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}
		t.Fatalf("upload from-folder: %v", err)
	}

	assets, err := e2eutils.GetAllAssetList(u.Email, u.Password)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != retryAssetCount {
		t.Fatalf("%d assets uploaded, want %d", len(assets), retryAssetCount)
	}
	return adm, u, assets
}

// retryArchive runs `archive from-immich` of the user's account on server into dir, with args
// after the connection flags.
func retryArchive(ctx context.Context, server string, u, adm user, dir string, args ...string) (*app.Application, error) {
	c, a := root.RootImmichGoCommand(ctx)
	c.SetArgs(append([]string{
		"archive",
		"--write-to-folder=" + dir,
		"from-immich",
		"--from-server=" + server,
		"--from-api-key=" + u.APIKey,
		"--from-admin-api-key=" + adm.APIKey,
		"--log-level=debug",
	}, args...))
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
	return a, err
}

// retryCheckArchive checks that dir holds each asset once, under its name, with the content whose
// SHA-1 is the asset's checksum.
func retryCheckArchive(t *testing.T, dir string, assets []*e2eutils.Asset) {
	t.Helper()
	checksums := map[string]string{}
	for _, a := range assets {
		if _, ok := checksums[a.OriginalFileName]; ok {
			t.Fatalf("two assets are named %s", a.OriginalFileName)
		}
		checksums[a.OriginalFileName] = a.Checksum
	}
	archived := map[string]bool{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasSuffix(p, ".JSON") {
			return err
		}
		name := d.Name()
		sum, err := computeSHA1(p)
		switch {
		case err != nil:
			return err
		case checksums[name] == "":
			t.Errorf("%s: no asset has this name", p)
		case archived[name]:
			t.Errorf("%s: %s is archived twice", p, name)
		case sum != checksums[name]:
			t.Errorf("%s: SHA-1 %s, want the asset's checksum %s", p, sum, checksums[name])
		}
		archived[name] = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for name := range checksums {
		if !archived[name] {
			t.Errorf("%s: not archived", name)
		}
	}
}
