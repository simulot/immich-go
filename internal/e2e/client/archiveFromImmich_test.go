//go:build e2e

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/app/root"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"github.com/simulot/immich-go/internal/fileevent"
)

// TestArchiveSecondRunDownloadsNothing archives an account twice into the same folder. The second
// run finds every asset already archived: it downloads nothing and writes nothing.
func TestArchiveSecondRunDownloadsNothing(t *testing.T) {
	const assetCount = 40
	adm, u := e2eAdmin(t), e2eNewUser(t)
	uploadFromFolder(t, u, adm, "DATA/fromFolder/recursive")
	checksums := serverChecksums(t, u, assetCount)
	dir := t.TempDir()
	proxy := newDownloadProxy(t, 0, nil)

	if _, err := archiveFromImmich(t.Context(), proxy.URL, u, adm, dir); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if got := proxy.downloads.Load(); got != assetCount {
		t.Fatalf("first run: %d downloads, want %d", got, assetCount)
	}

	// Back-date every file, so that a file the second run writes again gets a new mtime whatever
	// the resolution of the file system's clock.
	past := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
	before := listFiles(t, dir)
	for p := range before {
		if err := os.Chtimes(p, past, past); err != nil {
			t.Fatal(err)
		}
	}

	proxy.downloads.Store(0)
	a, err := archiveFromImmich(t.Context(), proxy.URL, u, adm, dir)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if got := proxy.downloads.Load(); got != 0 {
		t.Errorf("second run: %d downloads, want 0", got)
	}
	e2eutils.CheckResults(t, map[fileevent.Code]int64{
		fileevent.ProcessedFileArchived: assetCount,
		fileevent.ErrorFileAccess:       0,
	}, false, a.FileProcessor())

	after := listFiles(t, dir)
	for p, mtime := range after {
		switch _, existed := before[p]; {
		case !existed:
			t.Errorf("%s: written by the second run", p)
		case !mtime.Equal(past):
			t.Errorf("%s: written again by the second run", p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			t.Errorf("%s: gone after the second run", p)
		}
	}
	if archived := checkArchivedFiles(t, dir, checksums); len(archived) != assetCount {
		t.Errorf("%d assets archived, want %d", len(archived), assetCount)
	}
}

// TestArchiveInterruptedRunResumes cuts a download short halfway and cancels the run, then leaves
// one archived file as a run killed while writing it would. The next run downloads only what is
// missing, and every asset ends up archived once, under its own name.
func TestArchiveInterruptedRunResumes(t *testing.T) {
	const assetCount, cutAt = 40, 6
	adm, u := e2eAdmin(t), e2eNewUser(t)
	uploadFromFolder(t, u, adm, "DATA/fromFolder/recursive")
	checksums := serverChecksums(t, u, assetCount)
	dir := t.TempDir()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cutting := newDownloadProxy(t, cutAt, cancel)
	if _, err := archiveFromImmich(ctx, cutting.URL, u, adm, dir); !errors.Is(err, context.Canceled) {
		t.Fatalf("interrupted run returned %v, want %v", err, context.Canceled)
	}
	if got := cutting.downloads.Load(); got != cutAt {
		t.Fatalf("interrupted run: %d downloads, want %d, the last one cut short", got, cutAt)
	}
	// The run writes one asset at a time: the files downloaded in full are archived, the one cut
	// short is not.
	archived := checkArchivedFiles(t, dir, checksums)
	if len(archived) != cutAt-1 {
		t.Fatalf("interrupted run: %d assets archived, want %d", len(archived), cutAt-1)
	}

	// A run killed while it writes a file leaves the bytes it got in .<name>.partial, and neither
	// the file nor its .JSON sidecar.
	killed := archived[slices.Sorted(maps.Keys(archived))[0]]
	b, err := os.ReadFile(killed)
	if err != nil {
		t.Fatal(err)
	}
	partial := filepath.Join(filepath.Dir(killed), "."+filepath.Base(killed)+".partial")
	if err := os.WriteFile(partial, b[:len(b)/2], 0o644); err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(os.Remove(killed), os.Remove(killed+".JSON")); err != nil {
		t.Fatal(err)
	}

	resuming := newDownloadProxy(t, 0, nil)
	a, err := archiveFromImmich(t.Context(), resuming.URL, u, adm, dir)
	if err != nil {
		t.Fatalf("resumed run: %v", err)
	}
	// The asset cut short, the ones the interrupted run did not reach, and the killed one.
	if got, want := resuming.downloads.Load(), int32(assetCount-(cutAt-1)+1); got != want {
		t.Errorf("resumed run: %d downloads, want %d", got, want)
	}
	e2eutils.CheckResults(t, map[fileevent.Code]int64{
		fileevent.ProcessedFileArchived: assetCount,
		fileevent.ErrorFileAccess:       0,
	}, false, a.FileProcessor())
	archived = checkArchivedFiles(t, dir, checksums)
	for name := range checksums {
		if _, ok := archived[name]; !ok {
			t.Errorf("%s: not archived", name)
		}
	}
}

// TestArchiveRestoreAlbums archives an account whose assets are in two albums, some of them in
// both, and restores the archive into a new account with `upload from-folder`. The albums come
// back with the same assets.
func TestArchiveRestoreAlbums(t *testing.T) {
	const assetCount = 30
	adm, u1 := e2eAdmin(t), e2eNewUser(t)
	// The bridges in no album, the horses in Album One, the mountains in both albums.
	uploadFromFolder(t, u1, adm, "DATA/fromFolder/recursive/a")
	checksums := serverChecksums(t, u1, assetCount)
	uploadFromFolder(t, u1, adm, "--into-album=Album One", "DATA/fromFolder/recursive/a/b")
	uploadFromFolder(t, u1, adm, "--into-album=Album Two", "DATA/fromFolder/recursive/a/b/c")
	want := albumMembers(t, u1)
	if len(want) != 2 || len(want["Album One"]) != 20 || len(want["Album Two"]) != 10 {
		t.Fatalf("albums before the archive: %v, want Album One with 20 assets and Album Two with 10", want)
	}

	dir := t.TempDir()
	if _, err := archiveFromImmich(t.Context(), ImmichURL, u1, adm, dir); err != nil {
		t.Fatalf("archive: %v", err)
	}

	u2 := e2eNewUser(t)
	uploadFromFolder(t, u2, adm, dir)
	if got := serverChecksums(t, u2, assetCount); !maps.Equal(got, checksums) {
		t.Errorf("restored assets by name: %v, want %v", got, checksums)
	}
	if got := albumMembers(t, u2); !maps.EqualFunc(got, want, slices.Equal[[]string]) {
		t.Errorf("restored albums: %v, want %v", got, want)
	}
}

// originalDownload matches the download of an asset's original file.
var originalDownload = regexp.MustCompile(`^/api/assets/[^/]+/original$`)

// downloadProxy is a reverse proxy to the e2e server that counts the downloads of original files.
type downloadProxy struct {
	*httptest.Server
	downloads atomic.Int32
}

// newDownloadProxy starts a downloadProxy. With cutAt > 0, the cutAt-th download sends half of the
// file, then calls onCut and holds the connection until the client drops it.
func newDownloadProxy(t *testing.T, cutAt int32, onCut func()) *downloadProxy {
	t.Helper()
	target, err := url.Parse(ImmichURL)
	if err != nil {
		t.Fatal(err)
	}
	rp := &httputil.ReverseProxy{
		Rewrite:  func(r *httputil.ProxyRequest) { r.SetURL(target) },
		ErrorLog: log.New(io.Discard, "", 0), // the client drops the download cut short
	}
	p := &downloadProxy{}
	p.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isDownload := r.Method == http.MethodGet && originalDownload.MatchString(r.URL.Path)
		if !isDownload || p.downloads.Add(1) != cutAt {
			rp.ServeHTTP(w, r)
			return
		}
		full := httptest.NewRecorder()
		rp.ServeHTTP(full, r)
		maps.Copy(w.Header(), full.Header())
		w.WriteHeader(full.Code)
		_, _ = w.Write(full.Body.Bytes()[:full.Body.Len()/2])
		_ = http.NewResponseController(w).Flush()
		onCut()
		select {
		case <-r.Context().Done():
		case <-time.After(time.Minute):
		}
	}))
	t.Cleanup(p.Close)
	return p
}

// archiveFromImmich runs `archive from-immich` of the user's account on server into dir.
func archiveFromImmich(ctx context.Context, server string, u, adm user, dir string) (*app.Application, error) {
	c, a := root.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"archive",
		"--write-to-folder=" + dir,
		"from-immich",
		"--from-server=" + server,
		"--from-api-key=" + u.APIKey,
		"--from-admin-api-key=" + adm.APIKey,
		"--log-level=debug",
	})
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
	return a, err
}

// uploadFromFolder runs `upload from-folder` into the user's account, with args after the
// connection flags.
func uploadFromFolder(t *testing.T, u, adm user, args ...string) {
	t.Helper()
	ctx := t.Context()
	c, a := root.RootImmichGoCommand(ctx)
	c.SetArgs(append([]string{
		"upload", "from-folder",
		"--server=" + ImmichURL,
		"--api-key=" + u.APIKey,
		"--admin-api-key=" + adm.APIKey,
		"--no-ui",
		"--log-level=debug",
	}, args...))
	if err := c.ExecuteContext(ctx); err != nil {
		if a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}
		t.Fatalf("upload from-folder %v: %v", args, err)
	}
}

func e2eAdmin(t *testing.T) user {
	t.Helper()
	adm, err := getUser("admin@immich.app")
	if err != nil {
		t.Fatalf("can't get admin user: %v", err)
	}
	return adm
}

func e2eNewUser(t *testing.T) user {
	t.Helper()
	u, err := createUser("minimal")
	if err != nil {
		t.Fatalf("can't create user: %v", err)
	}
	return u
}

// checkArchivedFiles walks the archive in dir and returns the path of each archived asset by its
// name. It fails the test on a file that is no asset's (a ~N copy), an asset archived twice,
// content whose SHA-1 is not the asset's checksum, a .JSON sidecar without that checksum, and a
// .partial file left behind.
func checkArchivedFiles(t *testing.T, dir string, checksums map[string]string) map[string]string {
	t.Helper()
	archived := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := d.Name()
		switch {
		case strings.HasSuffix(name, ".partial"):
			t.Errorf("%s: a partial file is left", p)
		case strings.HasSuffix(name, ".JSON"):
			// checked with its file
		case checksums[name] == "":
			t.Errorf("%s: no asset has this name", p)
		case archived[name] != "":
			t.Errorf("%s: %s is already archived as %s", p, name, archived[name])
		default:
			archived[name] = p
			if sum, err := computeSHA1(p); err != nil || sum != checksums[name] {
				t.Errorf("%s: SHA-1 %q (%v), want the asset's checksum %s", p, sum, err, checksums[name])
			}
			var sidecar struct {
				Checksum string `json:"checksum"`
			}
			b, err := os.ReadFile(p + ".JSON")
			if err == nil {
				err = json.Unmarshal(b, &sidecar)
			}
			if err != nil || sidecar.Checksum != checksums[name] {
				t.Errorf("%s.JSON: checksum %q (%v), want %s", p, sidecar.Checksum, err, checksums[name])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return archived
}

// listFiles returns the mtime of each file under dir, by path.
func listFiles(t *testing.T, dir string) map[string]time.Time {
	t.Helper()
	files := map[string]time.Time{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		files[p] = fi.ModTime()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// serverAsset is an asset as /search/metadata and GET /stacks return it.
type serverAsset struct {
	ID               string `json:"id"`
	OriginalFileName string `json:"originalFileName"`
	Checksum         string `json:"checksum"`
	ExifInfo         *struct {
		DateTimeOriginal *string `json:"dateTimeOriginal"`
		FileSizeInByte   int64   `json:"fileSizeInByte"`
	} `json:"exifInfo"`
}

// serverChecksums waits until the server has read the metadata of the user's n assets, which sets
// the folder each one is archived in, and returns their checksums by file name.
func serverChecksums(t *testing.T, u user, n int) map[string]string {
	t.Helper()
	tk := login(t, u)
	deadline := time.Now().Add(2 * time.Minute)
	for {
		items := searchAssets(t, tk, map[string]any{"withExif": true})
		if len(items) == n && !slices.ContainsFunc(items, func(a serverAsset) bool {
			return a.ExifInfo == nil || a.ExifInfo.DateTimeOriginal == nil || a.ExifInfo.FileSizeInByte == 0
		}) {
			checksums := make(map[string]string, n)
			for _, a := range items {
				if _, ok := checksums[a.OriginalFileName]; ok {
					t.Fatalf("two assets are named %s", a.OriginalFileName)
				}
				checksums[a.OriginalFileName] = a.Checksum
			}
			return checksums
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: %d assets with their metadata read after 2 minutes, want %d", u.Email, len(items), n)
		}
		time.Sleep(time.Second)
	}
}

// albumMembers returns the user's albums by name, each with the sorted checksums of its assets.
func albumMembers(t *testing.T, u user) map[string][]string {
	t.Helper()
	tk := login(t, u)
	var albums []struct {
		ID        string `json:"id"`
		AlbumName string `json:"albumName"`
	}
	callAPI(t, tk, http.MethodGet, "/albums", nil, &albums)
	members := map[string][]string{}
	for _, al := range albums {
		if _, ok := members[al.AlbumName]; ok {
			t.Fatalf("two albums are named %s", al.AlbumName)
		}
		checksums := []string{}
		for _, a := range searchAssets(t, tk, map[string]any{"albumIds": []string{al.ID}}) {
			checksums = append(checksums, a.Checksum)
		}
		slices.Sort(checksums)
		members[al.AlbumName] = checksums
	}
	return members
}

// searchAssets returns the assets that match the /search/metadata query q.
func searchAssets(t *testing.T, tk e2eutils.Token, q map[string]any) []serverAsset {
	t.Helper()
	q["size"] = 1000
	var resp struct {
		Assets struct {
			Items    []serverAsset `json:"items"`
			NextPage *string       `json:"nextPage"`
		} `json:"assets"`
	}
	callAPI(t, tk, http.MethodPost, "/search/metadata", q, &resp)
	if resp.Assets.NextPage != nil {
		t.Fatalf("more than %d assets match %v", q["size"], q)
	}
	return resp.Assets.Items
}

func login(t *testing.T, u user) e2eutils.Token {
	t.Helper()
	tk, err := e2eutils.UserLogin(u.Email, u.Password)
	if err != nil {
		t.Fatalf("login %s: %v", u.Email, err)
	}
	return tk
}

// callAPI sends a request to the e2e server's API with the token, body as JSON when not nil, and
// decodes the JSON answer into out.
func callAPI(t *testing.T, tk e2eutils.Token, method, endpoint string, body, out any) {
	t.Helper()
	var rd io.Reader = http.NoBody
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(t.Context(), method, strings.TrimSuffix(ImmichURL, "/")+"/api"+endpoint, rd)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+string(tk))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		msg, _ := io.ReadAll(resp.Body)
		t.Fatalf("%s %s: %s %s", method, endpoint, resp.Status, msg)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("%s %s: %v", method, endpoint, err)
	}
}
