//go:build e2e

package client

// How the stacks of an archive behave on a real server in the cases a fake cannot show: API keys
// without a stack permission, hidden and trashed members, and an archive restored in part.

import (
	"bytes"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/simulot/immich-go/app/root"
	"github.com/simulot/immich-go/internal/assets"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
)

// stackCasesData holds 10 images, mountains_01.jpg to mountains_10.jpg.
const (
	stackCasesData   = "DATA/fromFolder/recursive/a/b/c"
	stackCasesAssets = 10
)

// A key without stack.read: the archive cannot list the stacks with GET /stacks. It still records
// the stacks GET /assets/{id} reports (those with a member other than the primary on the
// timeline), and logs one warning with the number of assets whose stack it could not check, with
// no search for each. Restoring the archive with such a key: one warning that the stacks were not
// rebuilt, and the upload succeeds.
func TestStacksKeyWithoutStackRead(t *testing.T) {
	acc := newStackCasesAccount(t)
	reported := acc.stack(t, "mountains_03.jpg", "mountains_01.jpg", "mountains_02.jpg")
	acc.stack(t, "mountains_04.jpg", "mountains_05.jpg")
	// With its only other member archived, GET /assets/{id} reports the stack to neither asset.
	updateAsset(t, acc.tk, acc.ids["mountains_05.jpg"], map[string]any{"visibility": "archive"})

	proxy := newIDSearchCounter(t)
	dir, archiveLog := t.TempDir(), filepath.Join(t.TempDir(), "archive.log")
	stackCasesArchive(t, proxy.URL, withoutPermission(t, acc.u, "stack.read"), acc.adm, dir, archiveLog)
	w := checkOneWarning(t, archiveLog, "the API key lacks stack.read: the assets whose stack could not be checked are archived with no stack")
	if n, ok := w["assets"].(float64); !ok || n != 2 {
		t.Errorf("warning %v: want assets=2 (mountains_04 and mountains_05)", w)
	}
	if n := proxy.idSearches.Load(); n != 0 {
		t.Errorf("%d searches for one asset, want none", n)
	}
	archived := archivedPaths(t, dir, acc.checksums)
	checkSidecarStacks(t, archived, map[string]*assets.Stack{
		"mountains_03.jpg": {ID: reported, Primary: true, AssetCount: 3},
		"mountains_01.jpg": {ID: reported, AssetCount: 3},
		"mountains_02.jpg": {ID: reported, AssetCount: 3},
		"mountains_04.jpg": nil,
		"mountains_05.jpg": nil,
	})

	u2 := e2eNewUser(t)
	uploadLog := filepath.Join(t.TempDir(), "upload.log")
	uploadFromFolder(t, withoutPermission(t, u2, "stack.read"), acc.adm, "--log-file="+uploadLog, "--log-type=JSON", dir)
	w = checkOneWarning(t, uploadLog, "stacks from the source not rebuilt: the API key lacks the stack.read permission")
	if n, ok := w["stacks"].(float64); !ok || n != 1 {
		t.Errorf("warning %v: want stacks=1", w)
	}
	checkRestoredWithoutStacks(t, u2)
}

// A key with stack.read but without stack.create: the server refuses the first POST /stacks, one
// warning says the stacks were not rebuilt, no stack is made, and the upload succeeds.
func TestStacksKeyWithoutStackCreate(t *testing.T) {
	acc := newStackCasesAccount(t)
	acc.stack(t, "mountains_01.jpg", "mountains_02.jpg")
	acc.stack(t, "mountains_03.jpg", "mountains_04.jpg")
	dir := t.TempDir()
	stackCasesArchive(t, ImmichURL, acc.u, acc.adm, dir, filepath.Join(t.TempDir(), "archive.log"))

	u2 := e2eNewUser(t)
	uploadLog := filepath.Join(t.TempDir(), "upload.log")
	uploadFromFolder(t, withoutPermission(t, u2, "stack.create"), acc.adm, "--log-file="+uploadLog, "--log-type=JSON", dir)
	w := checkOneWarning(t, uploadLog, "stacks from the source not rebuilt: the API key lacks the stack.create permission")
	if n, ok := w["stacks"].(float64); !ok || n != 2 {
		t.Errorf("warning %v: want stacks=2", w)
	}
	checkRestoredWithoutStacks(t, u2)
}

// A stack whose only member other than its primary is hidden: GET /assets/{id} reports the stack
// to neither asset, and GET /stacks lists only the primary. Immich searches the hidden assets too
// when no visibility is given, and the search for the hidden asset alone with withStacked=false
// does not find it: the archive knows it is in a stack. It is archived with no stack, and a warning
// names it.
func TestStacksHiddenMember(t *testing.T) {
	acc := newStackCasesAccount(t)
	sid := acc.stack(t, "mountains_01.jpg", "mountains_02.jpg")
	hidden := acc.ids["mountains_02.jpg"]
	updateAsset(t, acc.tk, hidden, map[string]any{"visibility": "hidden"})

	dir, archiveLog := t.TempDir(), filepath.Join(t.TempDir(), "archive.log")
	stackCasesArchive(t, ImmichURL, acc.u, acc.adm, dir, archiveLog)
	w := checkOneWarning(t, archiveLog, "asset in a stack that Immich does not report (it is hidden): no stack in its sidecar")
	if w["id"] != hidden {
		t.Errorf("warning %v: want it for %s, mountains_02.jpg", w, hidden)
	}
	archived := archivedPaths(t, dir, acc.checksums)
	checkSidecarStacks(t, archived, map[string]*assets.Stack{
		"mountains_01.jpg": {ID: sid, Primary: true, AssetCount: 1}, // GET /stacks lists no other member
		"mountains_02.jpg": nil,
	})
}

// A trashed stack member. The archive: in a stack whose only other member is its primary, it is
// reported by neither GET /assets/{id} nor GET /stacks, and the search for it alone with
// withStacked=false (trashed included) does not find it: it is archived with no stack, and a
// warning names it. The upload run again over a restored stack whose members are put in the trash
// one by one: GET /stacks leaves a trashed member out, but GET /assets/{id} reports its stack while
// a member other than the primary is on the timeline, and GET /stacks names a trashed primary, so
// the runs make nothing. Once no member is reported, the stack is made again, and Immich merges it:
// still one stack.
func TestStacksTrashedMember(t *testing.T) {
	acc := newStackCasesAccount(t)
	acc.stack(t, "mountains_01.jpg", "mountains_02.jpg", "mountains_03.jpg")
	sid := acc.stack(t, "mountains_04.jpg", "mountains_05.jpg")
	trashed := acc.ids["mountains_05.jpg"]
	trashAssets(t, acc.tk, trashed)

	dir, archiveLog := t.TempDir(), filepath.Join(t.TempDir(), "archive.log")
	stackCasesArchive(t, ImmichURL, acc.u, acc.adm, dir, archiveLog)
	w := checkOneWarning(t, archiveLog, "trashed stack member that Immich does not report (GET /stacks leaves trashed members out): no stack in its sidecar")
	if w["id"] != trashed {
		t.Errorf("warning %v: want it for %s, mountains_05.jpg", w, trashed)
	}
	archived := archivedPaths(t, dir, acc.checksums)
	checkSidecarStacks(t, archived, map[string]*assets.Stack{
		"mountains_04.jpg": {ID: sid, Primary: true, AssetCount: 1}, // GET /stacks lists no other member
		"mountains_05.jpg": nil,
	})

	u2 := e2eNewUser(t)
	uploadFromFolder(t, u2, acc.adm, dir)
	tk2 := login(t, u2)
	ids2 := assetIDs(t, tk2)
	restored := listServerStacks(t, tk2)
	if len(restored) != 1 || restored[0].PrimaryAssetID != ids2["mountains_01.jpg"] {
		t.Fatalf("restored stacks %+v, want one, of mountains_01.jpg (%s)", restored, ids2["mountains_01.jpg"])
	}
	rerun := func(step string, made, present int) {
		t.Helper()
		logFile := filepath.Join(t.TempDir(), "upload.log")
		uploadFromFolder(t, u2, acc.adm, "--log-file="+logFile, "--log-type=JSON", dir)
		var summary map[string]any
		for _, r := range stackCasesLog(t, logFile) {
			if r["msg"] == "stacks from the source" {
				summary = r
			}
		}
		if summary["made"] != float64(made) || summary["present"] != float64(present) {
			t.Errorf("%s: %v, want made=%d present=%d", step, summary, made, present)
		}
	}

	trashAssets(t, tk2, ids2["mountains_03.jpg"])
	rerun("a member in the trash", 0, 1)
	trashAssets(t, tk2, ids2["mountains_01.jpg"])
	rerun("the primary in the trash too", 0, 1)
	if got := listServerStacks(t, tk2); len(got) != 1 || got[0].ID != restored[0].ID {
		t.Fatalf("stacks %+v after the runs that made nothing, want only %s", got, restored[0].ID)
	}

	trashAssets(t, tk2, ids2["mountains_02.jpg"])
	rerun("no member reported", 1, 0)
	got := listServerStacks(t, tk2)
	if len(got) != 1 || got[0].ID == restored[0].ID || got[0].PrimaryAssetID != ids2["mountains_01.jpg"] {
		t.Fatalf("stacks %+v, want one new stack of mountains_01.jpg (%s) in place of %s", got, ids2["mountains_01.jpg"], restored[0].ID)
	}
}

// Restoring one month folder of an archive: a stack whose members are archived in two folders has
// one member there, so it is not made, and a warning says so. A stack whose members are all in the
// folder is made.
func TestStacksPartialRestore(t *testing.T) {
	acc := newStackCasesAccount(t)
	jan, feb := time.Date(2020, 1, 15, 12, 0, 0, 0, time.UTC), time.Date(2020, 2, 15, 12, 0, 0, 0, time.UTC)
	dates := map[string]time.Time{
		"mountains_01.jpg": jan, "mountains_02.jpg": jan, // a stack in January
		"mountains_03.jpg": jan, "mountains_04.jpg": feb, // a stack across the two months
	}
	for name, d := range dates {
		updateAsset(t, acc.tk, acc.ids[name], map[string]any{"dateTimeOriginal": d.Format(time.RFC3339)})
	}
	acc.stack(t, "mountains_01.jpg", "mountains_02.jpg")
	across := acc.stack(t, "mountains_03.jpg", "mountains_04.jpg")
	waitForCaptureDates(t, acc.tk, dates)
	dir := t.TempDir()
	stackCasesArchive(t, ImmichURL, acc.u, acc.adm, dir, filepath.Join(t.TempDir(), "archive.log"))

	u2 := e2eNewUser(t)
	uploadLog := filepath.Join(t.TempDir(), "upload.log")
	uploadFromFolder(t, u2, acc.adm, "--log-file="+uploadLog, "--log-type=JSON", filepath.Join(dir, jan.Format("2006/2006-01")))
	w := checkOneWarning(t, uploadLog, "stack not made: only one of its members is here")
	if w["source stack"] != across || w["members on the source"] != float64(2) {
		t.Errorf("warning %v: want it for the source stack %s, of 2 members", w, across)
	}
	want := map[string][]string{acc.checksums["mountains_01.jpg"]: {acc.checksums["mountains_01.jpg"], acc.checksums["mountains_02.jpg"]}}
	slices.Sort(want[acc.checksums["mountains_01.jpg"]])
	if got, _ := userStacks(t, u2); !maps.EqualFunc(got, want, slices.Equal[[]string]) {
		t.Errorf("restored stacks %v, want only the January one %v", got, want)
	}
}

// stackCasesAccount is a new user with the assets of stackCasesData.
type stackCasesAccount struct {
	adm, u    user
	tk        e2eutils.Token
	ids       map[string]string // the assets' IDs by file name
	checksums map[string]string // the assets' checksums by file name
}

func newStackCasesAccount(t *testing.T) *stackCasesAccount {
	t.Helper()
	adm, u := e2eAdmin(t), e2eNewUser(t)
	uploadFromFolder(t, u, adm, stackCasesData)
	checksums := serverChecksums(t, u, stackCasesAssets)
	tk := login(t, u)
	return &stackCasesAccount{adm: adm, u: u, tk: tk, ids: assetIDs(t, tk), checksums: checksums}
}

// stack stacks the assets given by name, the first one its primary, and returns the stack's ID.
func (acc *stackCasesAccount) stack(t *testing.T, names ...string) string {
	t.Helper()
	ids := make([]string, 0, len(names))
	for _, n := range names {
		ids = append(ids, acc.ids[n])
	}
	return makeStack(t, acc.tk, ids)
}

// withoutPermission returns u with a new API key that has the "minimal" permissions but the one
// given.
func withoutPermission(t *testing.T, u user, permission string) user {
	t.Helper()
	perms := slices.DeleteFunc(slices.Clone(permissions["minimal"]), func(p string) bool { return p == permission })
	if len(perms) == len(permissions["minimal"]) {
		t.Fatalf("%s is not a permission of the minimal key", permission)
	}
	key, err := e2eutils.CreateApiKey(login(t, u), "minimal without "+permission, perms)
	if err != nil {
		t.Fatalf("can't create the API key: %v", err)
	}
	u.APIKey = key
	return u
}

// stackCasesArchive runs `archive from-immich` of the account of u's API key on server (the e2e
// server, or a proxy to it) into dir, with its log in JSON in logFile.
func stackCasesArchive(t *testing.T, server string, u, adm user, dir, logFile string) {
	t.Helper()
	ctx := t.Context()
	c, a := root.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"archive",
		"--write-to-folder=" + dir,
		"from-immich",
		"--from-server=" + server,
		"--from-api-key=" + u.APIKey,
		"--from-admin-api-key=" + adm.APIKey,
		"--log-level=debug",
		"--log-file=" + logFile,
		"--log-type=JSON",
	})
	if err := c.ExecuteContext(ctx); err != nil {
		if a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}
		t.Fatalf("archive: %v", err)
	}
}

// stackCasesLog returns the records of the JSON log file p.
func stackCasesLog(t *testing.T, p string) []map[string]any {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var records []map[string]any
	for line := range strings.Lines(string(b)) {
		var r map[string]any
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("%s: %v: %q", p, err, line)
		}
		records = append(records, r)
	}
	return records
}

// checkOneWarning checks that the log file p holds one warning, with the message msg, and no error.
// It returns the warning's record.
func checkOneWarning(t *testing.T, p, msg string) map[string]any {
	t.Helper()
	var found []map[string]any
	for _, r := range stackCasesLog(t, p) {
		if r["level"] == "WARN" || r["level"] == "ERROR" {
			found = append(found, r)
		}
	}
	if len(found) != 1 || found[0]["level"] != "WARN" || found[0]["msg"] != msg {
		t.Fatalf("%s: warnings and errors %v, want one warning %q", p, found, msg)
	}
	return found[0]
}

// checkSidecarStacks checks the stack the .JSON sidecar of each archived file records, by name
// (nil: none).
func checkSidecarStacks(t *testing.T, archived map[string]string, want map[string]*assets.Stack) {
	t.Helper()
	for name, w := range want {
		if archived[name] == "" {
			t.Errorf("%s is not archived", name)
			continue
		}
		b, err := os.ReadFile(archived[name] + ".JSON")
		if err != nil {
			t.Fatal(err)
		}
		var md assets.Metadata
		if err := json.Unmarshal(b, &md); err != nil {
			t.Fatalf("%s.JSON: %v", archived[name], err)
		}
		if (md.Stack == nil) != (w == nil) || (w != nil && *md.Stack != *w) {
			t.Errorf("%s: stack %+v in its sidecar, want %+v", name, md.Stack, w)
		}
	}
}

// checkRestoredWithoutStacks checks that the user has the stackCasesAssets assets, and no stack.
func checkRestoredWithoutStacks(t *testing.T, u user) {
	t.Helper()
	tk := login(t, u)
	if n := len(searchAssets(t, tk, map[string]any{})); n != stackCasesAssets {
		t.Errorf("%d assets restored, want %d", n, stackCasesAssets)
	}
	if got := listServerStacks(t, tk); len(got) != 0 {
		t.Errorf("stacks %+v made, want none", got)
	}
}

// serverStack is a stack as GET /stacks lists it.
type serverStack struct {
	ID             string `json:"id"`
	PrimaryAssetID string `json:"primaryAssetId"`
	Assets         []struct {
		ID string `json:"id"`
	} `json:"assets"`
}

// listServerStacks returns the user's stacks (GET /stacks).
func listServerStacks(t *testing.T, tk e2eutils.Token) []serverStack {
	t.Helper()
	var stacks []serverStack
	callAPI(t, tk, http.MethodGet, "/stacks", nil, &stacks)
	return stacks
}

// trashAssets puts the assets in the trash (DELETE /assets, answered with no content).
func trashAssets(t *testing.T, tk e2eutils.Token, ids ...string) {
	t.Helper()
	b, err := json.Marshal(map[string]any{"ids": ids})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, strings.TrimSuffix(ImmichURL, "/")+"/api/assets", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+string(tk))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /assets: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		msg, _ := io.ReadAll(resp.Body)
		t.Fatalf("DELETE /assets %v: %s %s", ids, resp.Status, msg)
	}
}

// idSearchCounter is a proxy to the e2e server that counts the searches for one asset
// (POST /search/metadata with an id).
type idSearchCounter struct {
	*httptest.Server
	idSearches atomic.Int32
}

func newIDSearchCounter(t *testing.T) *idSearchCounter {
	t.Helper()
	target, err := url.Parse(ImmichURL)
	if err != nil {
		t.Fatal(err)
	}
	rp := httputil.NewSingleHostReverseProxy(target)
	c := &idSearchCounter{}
	c.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/search/metadata" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			var q map[string]any
			if json.Unmarshal(body, &q) == nil && q["id"] != nil {
				c.idSearches.Add(1)
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
		}
		rp.ServeHTTP(w, r)
	}))
	t.Cleanup(c.Close)
	return c
}
