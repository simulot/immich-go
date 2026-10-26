package upload

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
)

// fakeStacks is a server whose stacks are fixed: GET /stacks lists them as given, and POST /stacks
// only records the call. How Immich applies a new stack, and what GET /stacks and GET /assets/{id}
// show afterwards, is tested on a real server (the e2e tests). It gives on cue the failures a
// server does not: GET /assets/{id} failing for some assets, and GET /stacks or POST /stacks failing.
type fakeStacks struct {
	stacks    []immich.Stack  // what GET /stacks lists
	infoFails map[string]bool // assets whose GET /assets/{id} fails
	listErr   error           // what GET /stacks returns instead of the stacks
	createErr error           // what POST /stacks returns instead of making the stack
	creates   [][]string      // the POST /stacks calls, in order
}

// GetAssetInfo reports an asset in no stack: GET /stacks lists every member of the stacks here.
func (f *fakeStacks) GetAssetInfo(_ context.Context, id string) (*immich.Asset, error) {
	if f.infoFails[id] {
		return &immich.Asset{}, fmt.Errorf("GetAssetInfo %s: failed on purpose", id)
	}
	return &immich.Asset{ID: id}, nil
}

func (f *fakeStacks) GetAllStacks(context.Context) ([]immich.Stack, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.stacks, nil
}

func (f *fakeStacks) CreateStack(_ context.Context, ids []string) (string, error) {
	f.creates = append(f.creates, append([]string(nil), ids...))
	if f.createErr != nil {
		return "", f.createErr
	}
	return fmt.Sprintf("stack-%d", len(f.creates)), nil
}

// stack is a stack as GET /stacks lists it: its primary, and the members it lists (Immich leaves
// out the trashed and hidden ones, the primary included).
func stack(id, primary string, listed ...string) immich.Stack {
	s := immich.Stack{ID: id, PrimaryAssetID: primary}
	for _, a := range listed {
		s.Assets = append(s.Assets, struct {
			ID string `json:"id"`
		}{a})
	}
	return s
}

func member(file, id string, st *assets.Stack) (*assets.Asset, string) {
	a := &assets.Asset{File: fshelper.FSName(nil, file)}
	a.FromApplication = &assets.Metadata{Stack: st}
	return a, id
}

func collect(pairs ...func() (*assets.Asset, string)) *sourceStacks {
	s := &sourceStacks{}
	for _, p := range pairs {
		s.add(p())
	}
	return s
}

func m(file, id, stack string, primary bool, count int) func() (*assets.Asset, string) {
	return func() (*assets.Asset, string) {
		return member(file, id, &assets.Stack{ID: stack, Primary: primary, AssetCount: count})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestRebuildStacks(t *testing.T) {
	src := collect(
		// S1: members in two month folders; the primary is not the first by name.
		m("2005/2005-01/100_0074.JPG", "a1", "S1", false, 2),
		m("2006/2006-05/100_0075.JPG", "a2", "S1", true, 2),
		// S2: 3 of its 4 members are here; the middle one is the primary.
		m("2006/2006-05/IMG_1.JPG", "b1", "S2", false, 4),
		m("2006/2006-05/IMG_2.JPG", "b2", "S2", true, 4),
		m("2006/2006-05/IMG_3.JPG", "b3", "S2", false, 4),
		// S3: one member here.
		m("2006/2006-05/IMG_9.JPG", "c1", "S3", true, 2),
		// S4: its primary is missing.
		m("2006/2006-05/IMG_5.JPG", "d1", "S4", false, 3),
		m("2006/2006-05/IMG_4.JPG", "d2", "S4", false, 3),
		// Not in a stack, or no ID here (not uploaded): ignored.
		func() (*assets.Asset, string) { return member("2006/2006-05/IMG_7.JPG", "e1", nil) },
		m("2006/2006-05/IMG_8.JPG", "", "S5", true, 2),
		// A file whose asset here is one already counted (the same server asset twice).
		m("2006/2006-05/IMG_2~1.JPG", "b2", "S2", false, 4),
	)
	var logBuf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logBuf, nil))
	f := &fakeStacks{}
	r, err := rebuildStacks(context.Background(), f, src, log)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"a2", "a1"},
		{"b2", "b1", "b3"},
		{"d2", "d1"}, // no primary here: the first by file name
	}
	if !slices.EqualFunc(f.creates, want, equalStrings) {
		t.Fatalf("POST /stacks %v, want %v", f.creates, want)
	}
	if r.created != 3 || r.present != 0 || r.notMade != 1 {
		t.Fatalf("result %+v", r)
	}
	logs := logBuf.String()
	for _, s := range []string{"S2", "S3", "S4"} {
		if !strings.Contains(logs, s) {
			t.Errorf("nothing logged for %s:\n%s", s, logs)
		}
	}
	if strings.Contains(logs, "S1") {
		t.Errorf("S1 is whole, yet logged:\n%s", logs)
	}

	// A rerun (or a resume after everything was uploaded) finds the stacks there and makes nothing.
	f = &fakeStacks{stacks: []immich.Stack{
		stack("n1", "a2", "a2", "a1"),
		stack("n2", "b2", "b2", "b1", "b3"),
		stack("n3", "d2", "d2", "d1"),
	}}
	r, err = rebuildStacks(context.Background(), f, src, log)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.creates) != 0 || r.created != 0 || r.present != 3 {
		t.Fatalf("rerun: POST /stacks %v, result %+v", f.creates, r)
	}
}

// A run cut short left a stack of the members uploaded by then (with another primary, since the
// primary was not uploaded yet). The next run makes the whole stack over it, with the right
// primary: Immich takes over a stack whose primary is among the new stack's assets.
func TestRebuildStacksCompletesAPartialStack(t *testing.T) {
	f := &fakeStacks{stacks: []immich.Stack{stack("partial", "b1", "b1", "b3")}}
	src := collect(
		m("2006/2006-05/IMG_1.JPG", "b1", "S2", false, 3),
		m("2006/2006-05/IMG_2.JPG", "b2", "S2", true, 3),
		m("2006/2006-05/IMG_3.JPG", "b3", "S2", false, 3),
	)
	r, err := rebuildStacks(context.Background(), f, src, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	if want := [][]string{{"b2", "b1", "b3"}}; r.created != 1 || r.notMade != 0 || !slices.EqualFunc(f.creates, want, equalStrings) {
		t.Fatalf("POST /stacks %v, result %+v; want %v", f.creates, r, want)
	}
}

// The groupers stacked members of a source stack earlier in the same run, by name and capture time
// (a RAW+JPEG pair with the RAW as cover, say). The rebuild does not fight that stack: one that is
// already the source stack is left alone, and one with another cover or fewer members is made
// again over it with the source primary (Immich takes it over: its primary is among the assets).
func TestRebuildStacksOverAGrouperStack(t *testing.T) {
	pair := []func() (*assets.Asset, string){
		m("2006/2006-05/IMG_1.CR2", "raw", "S1", false, 2),
		m("2006/2006-05/IMG_1.JPG", "jpg", "S1", true, 2),
	}
	triple := []func() (*assets.Asset, string){
		m("2006/2006-05/IMG_1.CR2", "raw", "S1", false, 3),
		m("2006/2006-05/IMG_1.HEIC", "heic", "S1", false, 3),
		m("2006/2006-05/IMG_1.JPG", "jpg", "S1", true, 3),
	}
	for _, tc := range []struct {
		name    string
		grouper []string // the grouper's stack, cover first
		source  []func() (*assets.Asset, string)
		want    [][]string // the POST /stacks calls of the rebuild
	}{
		{"the same stack", []string{"jpg", "raw"}, pair, nil},
		{"another cover", []string{"raw", "jpg"}, pair, [][]string{{"jpg", "raw"}}},
		{"fewer members", []string{"raw", "jpg"}, triple, [][]string{{"jpg", "raw", "heic"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeStacks{stacks: []immich.Stack{stack("grouper", tc.grouper[0], tc.grouper...)}}
			r, err := rebuildStacks(context.Background(), f, collect(tc.source...), slog.New(slog.DiscardHandler))
			if err != nil {
				t.Fatal(err)
			}
			if !slices.EqualFunc(f.creates, tc.want, equalStrings) || r.notMade != 0 || r.failed != 0 {
				t.Fatalf("POST /stacks %v, result %+v; want %v", f.creates, r, tc.want)
			}
		})
	}
}

// Two source stacks share one asset here (two files with the same content end as one server
// asset). POST /stacks for the second would move that asset out of the stack just made for the
// first, without a word. The checks see the stacks made earlier in the run: the second source
// stack is not made, with the warning of a member in another stack.
func TestRebuildTwoSourceStacksShareServerAsset(t *testing.T) {
	src := collect(
		m("2006/2006-05/IMG_1.JPG", "x", "S1", true, 2),
		m("2006/2006-05/IMG_2.JPG", "shared", "S1", false, 2),
		m("2007/2007-01/IMG_8.JPG", "y", "S2", true, 2),
		m("2007/2007-01/IMG_9.JPG", "shared", "S2", false, 2),
	)
	f := &fakeStacks{}
	var logBuf bytes.Buffer
	r, err := rebuildStacks(context.Background(), f, src, slog.New(slog.NewTextHandler(&logBuf, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if want := [][]string{{"x", "shared"}}; !slices.EqualFunc(f.creates, want, equalStrings) || r.created != 1 || r.notMade != 1 {
		t.Fatalf("POST /stacks %v, result %+v; want only %v, and S2 not made", f.creates, r, want)
	}
	var warning string
	for line := range strings.Lines(logBuf.String()) {
		if strings.Contains(line, "a member is in another stack") {
			warning = line
		}
	}
	if !strings.Contains(warning, "level=WARN") || !strings.Contains(warning, "S2") || !strings.Contains(warning, "stack-1") {
		t.Errorf("want a warning that a member of S2 is in stack-1, made for S1; log:\n%s", logBuf.String())
	}
}

// A member GET /stacks does not list (trashed or hidden), whose GET /assets/{id} then fails while
// the stack that is there is checked: the stack cannot be checked, so it is an error for that
// stack, and nothing is made (a failed lookup is no reason to make the stack again).
func TestRebuildFailedLookupOnExistingStack(t *testing.T) {
	src := collect(
		m("IMG_1.JPG", "b1", "S2", false, 3),
		m("IMG_2.JPG", "b2", "S2", true, 3),
		m("IMG_3.JPG", "b3", "S2", false, 3),
	)
	f := &fakeStacks{
		stacks:    []immich.Stack{stack("there", "b2", "b2", "b1")},
		infoFails: map[string]bool{"b3": true},
	}
	var logBuf bytes.Buffer
	r, err := rebuildStacks(context.Background(), f, src, slog.New(slog.NewTextHandler(&logBuf, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.creates) != 0 || r.created != 0 || r.present != 0 || r.failed != 1 {
		t.Fatalf("POST /stacks %v, result %+v; want none and 1 failed", f.creates, r)
	}
	if !strings.Contains(logBuf.String(), "level=ERROR") || !strings.Contains(logBuf.String(), "failed on purpose") {
		t.Errorf("the failed lookup is not logged as an error:\n%s", logBuf.String())
	}
}

// A member already in a stack with assets that are not in this source stack: that stack is left
// as it is, and nothing is made for this one.
func TestRebuildStacksLeavesOtherStacks(t *testing.T) {
	f := &fakeStacks{stacks: []immich.Stack{stack("other", "x1", "x1", "b1")}}
	src := collect(
		m("IMG_1.JPG", "b1", "S2", false, 2),
		m("IMG_2.JPG", "b2", "S2", true, 2),
	)
	var logBuf bytes.Buffer
	r, err := rebuildStacks(context.Background(), f, src, slog.New(slog.NewTextHandler(&logBuf, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.creates) != 0 || r.created != 0 || r.notMade != 1 {
		t.Fatalf("POST /stacks %v, result %+v", f.creates, r)
	}
	if !strings.Contains(logBuf.String(), "S2") {
		t.Fatalf("not logged:\n%s", logBuf.String())
	}
	// Someone else's stack is no failure of the run: a warning, not an error.
	if r.failed != 0 || strings.Contains(logBuf.String(), "level=ERROR") || !strings.Contains(logBuf.String(), "level=WARN") {
		t.Fatalf("result %+v; want a warning and no error, logged:\n%s", r, logBuf.String())
	}
}

// "stack made without its missing members" is logged only for a stack that is made: not when
// POST /stacks fails, not when a member is in another stack, and not on a rerun that finds the
// stack already there.
func TestRebuildStacksMissingMembersLoggedOnlyWhenMade(t *testing.T) {
	const missing = "stack made without its missing members"
	src := func() *sourceStacks {
		return collect(
			m("IMG_1.JPG", "b1", "S2", false, 4),
			m("IMG_2.JPG", "b2", "S2", true, 4),
			m("IMG_3.JPG", "b3", "S2", false, 4),
		)
	}
	rebuild := func(f *fakeStacks) string {
		var logBuf bytes.Buffer
		if _, err := rebuildStacks(context.Background(), f, src(), slog.New(slog.NewTextHandler(&logBuf, nil))); err != nil {
			t.Fatal(err)
		}
		return logBuf.String()
	}

	if logs := rebuild(&fakeStacks{}); !strings.Contains(logs, missing) {
		t.Errorf("made: %q not logged:\n%s", missing, logs)
	}
	if logs := rebuild(&fakeStacks{stacks: []immich.Stack{stack("there", "b2", "b2", "b1", "b3")}}); strings.Contains(logs, missing) {
		t.Errorf("already there: %q logged:\n%s", missing, logs)
	}
	if logs := rebuild(&fakeStacks{createErr: errors.New("POST /stacks fails on purpose")}); strings.Contains(logs, missing) {
		t.Errorf("POST /stacks fails: %q logged:\n%s", missing, logs)
	}
	if logs := rebuild(&fakeStacks{stacks: []immich.Stack{stack("other", "x1", "x1", "b1")}}); strings.Contains(logs, missing) {
		t.Errorf("a member in another stack: %q logged:\n%s", missing, logs)
	}
}

// createsThenRefuses lets n POST /stacks calls through, then returns err for the next ones.
type createsThenRefuses struct {
	*fakeStacks
	n   int
	err error
}

func (c *createsThenRefuses) CreateStack(ctx context.Context, ids []string) (string, error) {
	if c.n == 0 {
		return "", c.err
	}
	c.n--
	return c.fakeStacks.CreateStack(ctx, ids)
}

// When the server refuses POST /stacks midway (stack.create), its one warning still gives the
// stacks made and the stacks already there by then.
func TestRebuildStacksRefusedMidRunLogsCounts(t *testing.T) {
	f := &fakeStacks{stacks: []immich.Stack{stack("there", "a1", "a1", "a2")}}
	refused := forbidden(t, "stack.create", func(ctx context.Context, ic *immich.ImmichClient) error {
		_, err := ic.CreateStack(ctx, []string{"c1", "c2"})
		return err
	})
	api := &createsThenRefuses{fakeStacks: f, n: 1, err: refused}
	src := collect(
		m("IMG_1.JPG", "a1", "S1", true, 2), // already there
		m("IMG_2.JPG", "a2", "S1", false, 2),
		m("IMG_3.JPG", "b1", "S2", true, 2), // made
		m("IMG_4.JPG", "b2", "S2", false, 2),
		m("IMG_5.JPG", "c1", "S3", true, 2), // refused
		m("IMG_6.JPG", "c2", "S3", false, 2),
	)
	var logBuf bytes.Buffer
	r, err := rebuildStacks(context.Background(), api, src, slog.New(slog.NewTextHandler(&logBuf, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if r.present != 1 || r.created != 1 || r.refused != "stack.create" {
		t.Fatalf("result %+v; want 1 already there, 1 made, then stack.create refused", r)
	}
	var warning string
	for line := range strings.Lines(logBuf.String()) {
		if strings.Contains(line, "not rebuilt") {
			warning = line
		}
	}
	for _, s := range []string{"level=WARN", "stacks=1", "made=1", `"already there"=1`} {
		if !strings.Contains(warning, s) {
			t.Errorf("the warning does not hold %s:\n%s", s, logBuf.String())
		}
	}
}

// The same when the other stack's primary is in the trash (or hidden): GET /stacks names it as the
// primary but lists only our member. It is still another stack, left as it is.
func TestRebuildStacksLeavesOtherStacksWithATrashedPrimary(t *testing.T) {
	f := &fakeStacks{stacks: []immich.Stack{stack("other", "x1", "b1")}}
	src := collect(
		m("IMG_1.JPG", "b1", "S2", false, 2),
		m("IMG_2.JPG", "b2", "S2", true, 2),
	)
	var logBuf bytes.Buffer
	r, err := rebuildStacks(context.Background(), f, src, slog.New(slog.NewTextHandler(&logBuf, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.creates) != 0 || r.created != 0 || r.notMade != 1 {
		t.Fatalf("POST /stacks %v, result %+v", f.creates, r)
	}
	if !strings.Contains(logBuf.String(), "a member is in another stack") {
		t.Fatalf("not logged:\n%s", logBuf.String())
	}
}

// stackServer is a server whose stack calls fakeStacks answers.
type stackServer struct {
	immich.ImmichInterface
	f *fakeStacks
}

func (s stackServer) GetAllStacks(ctx context.Context) ([]immich.Stack, error) {
	return s.f.GetAllStacks(ctx)
}

func (s stackServer) CreateStack(ctx context.Context, ids []string) (string, error) {
	return s.f.CreateStack(ctx, ids)
}

func (s stackServer) GetAssetInfo(ctx context.Context, id string) (*immich.Asset, error) {
	return s.f.GetAssetInfo(ctx, id)
}

// The summary line holds for both sources of stacks: an archive's sidecars, and the source server
// of upload from-immich, which has no sidecars.
func TestRebuildSourceStacksSummary(t *testing.T) {
	uc, _ := newTestUpCmd(t)
	uc.client.Immich = stackServer{f: &fakeStacks{}}
	var logBuf bytes.Buffer
	uc.app.Log().Logger = slog.New(slog.NewTextHandler(&logBuf, nil))
	for _, p := range []func() (*assets.Asset, string){
		m("IMG_1.JPG", "b1", "S1", true, 2),
		m("IMG_2.JPG", "b2", "S1", false, 2),
	} {
		uc.stacks.add(p())
	}
	if err := uc.rebuildSourceStacks(context.Background()); err != nil {
		t.Fatal(err)
	}
	logs := logBuf.String()
	if !strings.Contains(logs, `msg="stacks from the source" made=1`) || strings.Contains(logs, "sidecar") {
		t.Fatalf("summary line:\n%s", logs)
	}
}

// forbidden is the error the immich client returns when the server refuses the call with 403, as
// Immich does when the API key lacks the permission the endpoint needs.
func forbidden(t *testing.T, permission string, call func(context.Context, *immich.ImmichClient) error) error {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message": "Missing required permission: ` + permission + `"}`))
	}))
	defer srv.Close()
	ic, err := immich.NewImmichClient(srv.URL, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	err = call(context.Background(), ic)
	if !immich.IsForbidden(err) {
		t.Fatalf("the refused call returned %v, want a 403", err)
	}
	return err
}

// Any other failure of GET /stacks or POST /stacks still fails the upload.
func TestRebuildSourceStacksOtherErrorsFail(t *testing.T) {
	for name, refuse := range map[string]func(*fakeStacks, error){
		"GET /stacks":  func(f *fakeStacks, err error) { f.listErr = err },
		"POST /stacks": func(f *fakeStacks, err error) { f.createErr = err },
	} {
		t.Run(name, func(t *testing.T) {
			f := &fakeStacks{}
			refuse(f, errors.New("connection reset on purpose"))
			uc, _ := newTestUpCmd(t)
			uc.client.Immich = stackServer{f: f}
			uc.app.Log().Logger = slog.New(slog.DiscardHandler)
			for _, p := range []func() (*assets.Asset, string){
				m("IMG_1.JPG", "b1", "S1", true, 2),
				m("IMG_2.JPG", "b2", "S1", false, 2),
			} {
				uc.stacks.add(p())
			}
			if err := uc.rebuildSourceStacks(context.Background()); err == nil {
				t.Fatal("the upload does not fail")
			}
		})
	}
}

// A member counts only as the server asset that holds this very file.
func TestStackMemberID(t *testing.T) {
	a := func(id, ck string) *assets.Asset { return &assets.Asset{ID: id, Checksum: ck} }
	for _, tc := range []struct {
		name string
		a    *assets.Asset
		adv  *Advice
		want string
	}{
		{"uploaded", a("new", "ck"), &Advice{Advice: NotOnServer}, "new"},
		{"upload failed", a("", "ck"), &Advice{Advice: NotOnServer}, ""},
		{"replaced a smaller file", a("new", "ck"), &Advice{Advice: SmallerOnServer, ServerAsset: a("old", "other")}, "new"},
		{"same checksum on the server", a("srv", "ck"), &Advice{Advice: SameOnServer, ServerAsset: a("srv", "ck")}, "srv"},
		{"same name, time and size, other content", a("srv", "ck"), &Advice{Advice: SameOnServer, ServerAsset: a("srv", "other")}, ""},
		{"a bigger file with its name and time", a("srv", "ck"), &Advice{Advice: BetterOnServer, ServerAsset: a("srv", "other")}, ""},
		{"the same content from another file", a("", "ck"), &Advice{Advice: AlreadyProcessed, ServerAsset: a("first", "ck")}, "first"},
	} {
		if got := stackMemberID(tc.a, tc.adv); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

// A server asset deleted during the upload (replaced by a bigger file) is no member, whenever
// its deletion is noted.
func TestSourceStacksForget(t *testing.T) {
	s := collect(m("IMG_1.JPG", "old", "S1", true, 3), m("IMG_2.JPG", "b", "S1", false, 3))
	s.forget("old")
	s.add(member("IMG_1~1.JPG", "old", &assets.Stack{ID: "S1", Primary: true, AssetCount: 3}))
	s.add(member("IMG_3.JPG", "c", &assets.Stack{ID: "S1", AssetCount: 3}))
	ids := make([]string, 0, len(s.m["S1"]))
	for _, x := range s.m["S1"] {
		ids = append(ids, x.id)
	}
	if !equalStrings(ids, []string{"b", "c"}) {
		t.Fatalf("members %v, want b, c", ids)
	}
}

// handleAsset notes an asset's source stack with the ID that holds this very file here: the asset
// uploaded earlier in the run with the same content (a duplicate in the input), or the server's
// asset with the same checksum. A server asset that only has its name, capture time and size is
// no member, and is logged.
func TestHandleAssetNotesSourceStackMembers(t *testing.T) {
	uc, _ := newTestUpCmd(t)
	var logBuf bytes.Buffer
	uc.app.Log().Logger = slog.New(slog.NewTextHandler(&logBuf, nil))
	ctx := context.Background()
	when := time.Date(2006, 5, 1, 10, 0, 0, 0, time.UTC)
	sum := func(s string) string {
		h := sha1.Sum([]byte(s))
		return base64.StdEncoding.EncodeToString(h[:])
	}
	fsys := fstest.MapFS{
		"IMG_1.JPG":   {Data: []byte("uploaded earlier in this run")},
		"IMG_1~1.JPG": {Data: []byte("uploaded earlier in this run")},
		"IMG_2.JPG":   {Data: []byte("on the server, same checksum")},
		"IMG_3.JPG":   {Data: []byte("same name, time and size")},
	}

	uploaded := &assets.Asset{ID: "up-1", File: fshelper.FSName(fsys, "IMG_1.JPG"), OriginalFileName: "IMG_1.JPG"}
	if _, err := uploaded.GetChecksum(); err != nil {
		t.Fatal(err)
	}
	uc.assetIndex.addLocalAsset(uploaded)
	for _, sa := range []*assets.Asset{
		{ID: "srv-2", Checksum: sum("on the server, same checksum"), OriginalFileName: "IMG_2.JPG", CaptureDate: when, FileSize: len(fsys["IMG_2.JPG"].Data)},
		{ID: "srv-3", Checksum: sum("other bytes on the server"), OriginalFileName: "IMG_3.JPG", CaptureDate: when, FileSize: len(fsys["IMG_3.JPG"].Data)},
	} {
		uc.assetIndex.add(sa, false)
	}

	for _, name := range []string{"IMG_1~1.JPG", "IMG_2.JPG", "IMG_3.JPG"} {
		a := &assets.Asset{
			File:             fshelper.FSName(fsys, name),
			OriginalFileName: name,
			CaptureDate:      when,
			FileSize:         len(fsys[name].Data),
		}
		a.FromApplication = &assets.Metadata{Stack: &assets.Stack{ID: "S1", AssetCount: 3}}
		if err := uc.handleAsset(ctx, a, nil); err != nil {
			t.Fatalf("handleAsset %s: %v", name, err)
		}
	}

	ids := make([]string, 0, len(uc.stacks.m["S1"]))
	for _, x := range uc.stacks.m["S1"] {
		ids = append(ids, x.id)
	}
	sort.Strings(ids)
	if !equalStrings(ids, []string{"srv-2", "up-1"}) {
		t.Fatalf("members %v, want srv-2 and up-1", ids)
	}
	if logs := logBuf.String(); !strings.Contains(logs, "stack member not stacked") || !strings.Contains(logs, "IMG_3.JPG") {
		t.Fatalf("IMG_3.JPG not logged as left out:\n%s", logs)
	}
}

// failingUpload is a server whose uploads fail.
type failingUpload struct{ immich.ImmichInterface }

func (failingUpload) AssetUpload(context.Context, *assets.Asset) (immich.AssetResponse, error) {
	return immich.AssetResponse{}, errors.New("upload failed on purpose")
}

// An asset that failed to upload is no member, even when it still has an ID: upload from-immich
// gives each asset its ID on the source server, which this server does not know.
func TestHandleAssetFailedUploadIsNoMember(t *testing.T) {
	uc, _ := newTestUpCmd(t)
	uc.client.Immich = failingUpload{}
	uc.app.Log().Logger = slog.New(slog.DiscardHandler)
	fsys := fstest.MapFS{"IMG_1.JPG": {Data: []byte("never uploaded")}}
	a := &assets.Asset{ID: "source-id", File: fshelper.FSName(fsys, "IMG_1.JPG"), OriginalFileName: "IMG_1.JPG"}
	a.FromApplication = &assets.Metadata{Stack: &assets.Stack{ID: "S1", Primary: true, AssetCount: 2}}
	if err := uc.handleAsset(context.Background(), a, nil); err == nil {
		t.Fatal("handleAsset: no error, want the failed upload")
	}
	if n := uc.stacks.len(); n != 0 {
		t.Fatalf("a failed upload was noted as a stack member: %v", uc.stacks.m)
	}
}
