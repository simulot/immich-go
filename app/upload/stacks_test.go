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
	"sort"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
)

// fakeStacks keeps stacks the way Immich v3.2.2 does (server/src/repositories/stack.repository.ts,
// create): a new stack takes over every stack whose primary is among its assets (and all of
// their members), moves its assets out of any other stack, and its first asset is the primary.
type fakeStacks struct {
	next       int
	primary    map[string]string // stack -> primary asset
	of         map[string]string // asset -> stack
	trashed    map[string]bool   // members GET /stacks leaves out
	unreported map[string]bool   // members GET /assets/{id} shows in no stack
	infoFails  map[string]bool   // members whose GET /assets/{id} fails
	listErr    error             // what GET /stacks returns instead of the stacks
	createErr  error             // what POST /stacks returns instead of making the stack
	creates    int
}

func newFakeStacks() *fakeStacks {
	return &fakeStacks{
		primary: map[string]string{}, of: map[string]string{}, trashed: map[string]bool{},
		unreported: map[string]bool{}, infoFails: map[string]bool{},
	}
}

func (f *fakeStacks) GetAssetInfo(_ context.Context, id string) (*immich.Asset, error) {
	if f.infoFails[id] {
		return &immich.Asset{}, fmt.Errorf("GetAssetInfo %s: failed on purpose", id)
	}
	a := &immich.Asset{ID: id}
	if sid, ok := f.of[id]; ok && !f.unreported[id] {
		a.Stack = &immich.AssetStack{ID: sid, PrimaryAssetID: f.primary[sid]}
	}
	return a, nil
}

func (f *fakeStacks) GetAllStacks(context.Context) ([]immich.Stack, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]immich.Stack, 0, len(f.primary))
	for id, p := range f.primary {
		s := immich.Stack{ID: id, PrimaryAssetID: p}
		for a, sid := range f.of {
			if sid == id && !f.trashed[a] {
				s.Assets = append(s.Assets, struct {
					ID string `json:"id"`
				}{a})
			}
		}
		out = append(out, s)
	}
	return out, nil
}

func (f *fakeStacks) CreateStack(_ context.Context, ids []string) (string, error) {
	f.creates++
	if f.createErr != nil {
		return "", f.createErr
	}
	all := map[string]bool{}
	for _, id := range ids {
		all[id] = true
	}
	for sid, p := range f.primary {
		if all[p] {
			for a, s := range f.of {
				if s == sid {
					all[a] = true
				}
			}
			delete(f.primary, sid)
		}
	}
	f.next++
	sid := fmt.Sprintf("stack-%d", f.next)
	f.primary[sid] = ids[0]
	for a := range all {
		f.of[a] = sid
	}
	return sid, nil
}

// members of each stack, primary first, then sorted.
func (f *fakeStacks) snapshot() map[string][]string {
	out := map[string][]string{}
	for sid, p := range f.primary {
		var rest []string
		for a, s := range f.of {
			if s == sid && a != p {
				rest = append(rest, a)
			}
		}
		sort.Strings(rest)
		out[sid] = append([]string{p}, rest...)
	}
	return out
}

func (f *fakeStacks) byPrimary() map[string][]string {
	out := map[string][]string{}
	for _, m := range f.snapshot() {
		out[m[0]] = m
	}
	return out
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
	f := newFakeStacks()
	r, err := rebuildStacks(context.Background(), f, src, log)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"a2": {"a2", "a1"},
		"b2": {"b2", "b1", "b3"},
		"d2": {"d2", "d1"}, // no primary here: the first by file name
	}
	got := f.byPrimary()
	if len(got) != len(want) {
		t.Fatalf("stacks %v, want %v", got, want)
	}
	for p, w := range want {
		if !equalStrings(got[p], w) {
			t.Fatalf("stacks %v, want %v", got, want)
		}
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

	// A rerun (or a resume after everything was uploaded) makes nothing.
	before := f.snapshot()
	creates := f.creates
	r, err = rebuildStacks(context.Background(), f, src, log)
	if err != nil {
		t.Fatal(err)
	}
	if f.creates != creates || r.created != 0 || r.present != 3 {
		t.Fatalf("rerun: %d creates, result %+v", f.creates-creates, r)
	}
	after := f.snapshot()
	if len(after) != len(before) {
		t.Fatalf("rerun changed the stacks: %v, then %v", before, after)
	}
	for sid, ms := range before {
		if !equalStrings(after[sid], ms) {
			t.Fatalf("rerun changed the stacks: %v, then %v", before, after)
		}
	}
}

// A run cut short left a stack of the members uploaded by then (with another primary, since the
// primary was not uploaded yet). The next run makes it whole, with the right primary: one stack.
func TestRebuildStacksCompletesAPartialStack(t *testing.T) {
	f := newFakeStacks()
	if _, err := f.CreateStack(context.Background(), []string{"b1", "b3"}); err != nil {
		t.Fatal(err)
	}
	src := collect(
		m("2006/2006-05/IMG_1.JPG", "b1", "S2", false, 3),
		m("2006/2006-05/IMG_2.JPG", "b2", "S2", true, 3),
		m("2006/2006-05/IMG_3.JPG", "b3", "S2", false, 3),
	)
	r, err := rebuildStacks(context.Background(), f, src, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	got := f.snapshot()
	if len(got) != 1 || r.created != 1 {
		t.Fatalf("stacks %v, result %+v", got, r)
	}
	for _, ms := range got {
		if !equalStrings(ms, []string{"b2", "b1", "b3"}) {
			t.Fatalf("stack %v, want b2 primary with b1, b3", ms)
		}
	}
}

// The groupers stacked members of a source stack earlier in the same run, by name and capture time
// (a RAW+JPEG pair with the RAW as cover, say). The rebuild does not fight that stack: one that is
// already the source stack is left alone, and one with another cover or fewer members is taken
// over by the source stack, with the source primary. Either way, one stack.
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
		creates int      // CreateStack calls by the rebuild
		want    []string // the one stack left, primary first
	}{
		{"the same stack", []string{"jpg", "raw"}, pair, 0, []string{"jpg", "raw"}},
		{"another cover", []string{"raw", "jpg"}, pair, 1, []string{"jpg", "raw"}},
		{"fewer members", []string{"raw", "jpg"}, triple, 1, []string{"jpg", "heic", "raw"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeStacks()
			if _, err := f.CreateStack(context.Background(), tc.grouper); err != nil {
				t.Fatal(err)
			}
			r, err := rebuildStacks(context.Background(), f, collect(tc.source...), slog.New(slog.DiscardHandler))
			if err != nil {
				t.Fatal(err)
			}
			if creates := f.creates - 1; creates != tc.creates || r.notMade != 0 || r.failed != 0 {
				t.Fatalf("%d creates, result %+v; want %d creates", creates, r, tc.creates)
			}
			got := f.snapshot()
			if len(got) != 1 {
				t.Fatalf("stacks %v, want one", got)
			}
			for _, ms := range got {
				if !equalStrings(ms, tc.want) {
					t.Fatalf("stack %v, want %v", ms, tc.want)
				}
			}
		})
	}
}

// A member put in the trash after the restore (as fill does for a trashed one) is not listed by
// GET /stacks: GET /assets/{id} shows it still in the stack, so a rerun makes nothing. When
// Immich reports it nowhere, the stack is made again, and Immich merges it: still one stack.
func TestRebuildStacksRerunWithTrashedMember(t *testing.T) {
	src := collect(
		m("IMG_1.JPG", "b1", "S2", false, 3),
		m("IMG_2.JPG", "b2", "S2", true, 3),
		m("IMG_3.JPG", "b3", "S2", false, 3),
	)
	f := newFakeStacks()
	log := slog.New(slog.DiscardHandler)
	if _, err := rebuildStacks(context.Background(), f, src, log); err != nil {
		t.Fatal(err)
	}
	first := f.snapshot()
	f.trashed["b3"] = true
	r, err := rebuildStacks(context.Background(), f, src, log)
	if err != nil {
		t.Fatal(err)
	}
	if f.creates != 1 || r.created != 0 || r.present != 1 {
		t.Fatalf("rerun with a trashed member: %d creates, result %+v", f.creates, r)
	}
	for sid, ms := range f.snapshot() {
		if !equalStrings(first[sid], ms) {
			t.Fatalf("rerun changed the stack: %v, then %v", first, f.snapshot())
		}
	}
	f.unreported["b3"] = true
	if r, err = rebuildStacks(context.Background(), f, src, log); err != nil {
		t.Fatal(err)
	}
	if r.created != 1 || len(f.primary) != 1 {
		t.Fatalf("a member reported nowhere: result %+v, %d stacks, want it made again as the one stack", r, len(f.primary))
	}
}

// A member GET /stacks does not list, whose GET /assets/{id} then fails: the stack that is there
// cannot be checked, so it is an error for that stack, and nothing is made (a failed lookup is no
// reason to make the stack again).
func TestRebuildStacksFailedLookup(t *testing.T) {
	src := collect(
		m("IMG_1.JPG", "b1", "S2", false, 3),
		m("IMG_2.JPG", "b2", "S2", true, 3),
		m("IMG_3.JPG", "b3", "S2", false, 3),
	)
	f := newFakeStacks()
	if _, err := rebuildStacks(context.Background(), f, src, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatal(err)
	}
	first := f.snapshot()
	f.trashed["b3"] = true
	f.infoFails["b3"] = true
	var logBuf bytes.Buffer
	r, err := rebuildStacks(context.Background(), f, src, slog.New(slog.NewTextHandler(&logBuf, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if f.creates != 1 || r.created != 0 || r.present != 0 || r.failed != 1 {
		t.Fatalf("rerun with a failed lookup: %d creates, result %+v; want no create and 1 failed", f.creates, r)
	}
	if !strings.Contains(logBuf.String(), "level=ERROR") || !strings.Contains(logBuf.String(), "failed on purpose") {
		t.Errorf("the failed lookup is not logged as an error:\n%s", logBuf.String())
	}
	for sid, ms := range f.snapshot() {
		if !equalStrings(first[sid], ms) {
			t.Fatalf("rerun changed the stack: %v, then %v", first, f.snapshot())
		}
	}
}

// A primary put in the trash (or hidden) after the restore: GET /stacks still names it as the
// stack's primary but lists it as no member. The stack is found by its primary all the same, and
// a rerun makes nothing.
func TestRebuildStacksRerunWithTrashedPrimary(t *testing.T) {
	src := collect(
		m("IMG_1.JPG", "b1", "S2", false, 3),
		m("IMG_2.JPG", "b2", "S2", true, 3),
		m("IMG_3.JPG", "b3", "S2", false, 3),
	)
	f := newFakeStacks()
	log := slog.New(slog.DiscardHandler)
	if _, err := rebuildStacks(context.Background(), f, src, log); err != nil {
		t.Fatal(err)
	}
	first := f.snapshot()
	f.trashed["b2"] = true
	r, err := rebuildStacks(context.Background(), f, src, log)
	if err != nil {
		t.Fatal(err)
	}
	if f.creates != 1 || r.created != 0 || r.present != 1 {
		t.Fatalf("rerun with the primary trashed: %d creates, result %+v", f.creates, r)
	}
	for sid, ms := range f.snapshot() {
		if !equalStrings(first[sid], ms) {
			t.Fatalf("rerun changed the stack: %v, then %v", first, f.snapshot())
		}
	}
}

// A member already in a stack with assets that are not in this source stack: that stack is left
// as it is, and nothing is made for this one.
func TestRebuildStacksLeavesOtherStacks(t *testing.T) {
	f := newFakeStacks()
	if _, err := f.CreateStack(context.Background(), []string{"x1", "b1"}); err != nil {
		t.Fatal(err)
	}
	src := collect(
		m("IMG_1.JPG", "b1", "S2", false, 2),
		m("IMG_2.JPG", "b2", "S2", true, 2),
	)
	var logBuf bytes.Buffer
	r, err := rebuildStacks(context.Background(), f, src, slog.New(slog.NewTextHandler(&logBuf, nil)))
	if err != nil {
		t.Fatal(err)
	}
	got := f.snapshot()
	if f.creates != 1 || r.created != 0 || r.notMade != 1 || len(got) != 1 {
		t.Fatalf("stacks %v, result %+v", got, r)
	}
	for _, ms := range got {
		if !equalStrings(ms, []string{"x1", "b1"}) {
			t.Fatalf("the other stack changed: %v", ms)
		}
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

	f := newFakeStacks()
	if logs := rebuild(f); !strings.Contains(logs, missing) {
		t.Errorf("made: %q not logged:\n%s", missing, logs)
	}
	if logs := rebuild(f); strings.Contains(logs, missing) {
		t.Errorf("already there: %q logged:\n%s", missing, logs)
	}

	f = newFakeStacks()
	f.createErr = errors.New("POST /stacks fails on purpose")
	if logs := rebuild(f); strings.Contains(logs, missing) {
		t.Errorf("POST /stacks fails: %q logged:\n%s", missing, logs)
	}

	f = newFakeStacks()
	if _, err := f.CreateStack(context.Background(), []string{"x1", "b1"}); err != nil {
		t.Fatal(err)
	}
	if logs := rebuild(f); strings.Contains(logs, missing) {
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
	f := newFakeStacks()
	if _, err := f.CreateStack(context.Background(), []string{"a1", "a2"}); err != nil {
		t.Fatal(err)
	}
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
	f := newFakeStacks()
	if _, err := f.CreateStack(context.Background(), []string{"x1", "b1"}); err != nil {
		t.Fatal(err)
	}
	f.trashed["x1"] = true
	src := collect(
		m("IMG_1.JPG", "b1", "S2", false, 2),
		m("IMG_2.JPG", "b2", "S2", true, 2),
	)
	var logBuf bytes.Buffer
	r, err := rebuildStacks(context.Background(), f, src, slog.New(slog.NewTextHandler(&logBuf, nil)))
	if err != nil {
		t.Fatal(err)
	}
	got := f.snapshot()
	if f.creates != 1 || r.created != 0 || r.notMade != 1 || len(got) != 1 {
		t.Fatalf("stacks %v, result %+v", got, r)
	}
	for _, ms := range got {
		if !equalStrings(ms, []string{"x1", "b1"}) {
			t.Fatalf("the other stack changed: %v", ms)
		}
	}
	if !strings.Contains(logBuf.String(), "a member is in another stack") {
		t.Fatalf("not logged:\n%s", logBuf.String())
	}
}

// stackServer is a server that keeps its stacks as fakeStacks does.
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
	uc.client.Immich = stackServer{f: newFakeStacks()}
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

// A key without a stack permission: the server refuses GET /stacks (stack.read) or POST /stacks
// (stack.create) with 403. The upload does not fail for it: one warning names the permission and
// says the stacks were not rebuilt, and no stack is made.
func TestRebuildSourceStacksWithoutStackPermission(t *testing.T) {
	for _, tc := range []struct {
		permission string
		call       func(context.Context, *immich.ImmichClient) error
		refuse     func(*fakeStacks, error)
		creates    int // POST /stacks calls: none after the first refused
	}{
		{
			permission: "stack.read",
			call: func(ctx context.Context, ic *immich.ImmichClient) error {
				_, err := ic.GetAllStacks(ctx)
				return err
			},
			refuse:  func(f *fakeStacks, err error) { f.listErr = err },
			creates: 0,
		},
		{
			permission: "stack.create",
			call: func(ctx context.Context, ic *immich.ImmichClient) error {
				_, err := ic.CreateStack(ctx, []string{"b1", "b2"})
				return err
			},
			refuse:  func(f *fakeStacks, err error) { f.createErr = err },
			creates: 1,
		},
	} {
		t.Run(tc.permission, func(t *testing.T) {
			f := newFakeStacks()
			tc.refuse(f, forbidden(t, tc.permission, tc.call))
			uc, _ := newTestUpCmd(t)
			uc.client.Immich = stackServer{f: f}
			var logBuf bytes.Buffer
			uc.app.Log().Logger = slog.New(slog.NewTextHandler(&logBuf, nil))
			for _, p := range []func() (*assets.Asset, string){
				m("IMG_1.JPG", "b1", "S1", true, 2),
				m("IMG_2.JPG", "b2", "S1", false, 2),
				m("IMG_3.JPG", "b3", "S2", true, 2),
				m("IMG_4.JPG", "b4", "S2", false, 2),
			} {
				uc.stacks.add(p())
			}

			if err := uc.rebuildSourceStacks(context.Background()); err != nil {
				t.Fatalf("the upload fails: %v", err)
			}
			if got := f.snapshot(); len(got) != 0 || f.creates != tc.creates {
				t.Errorf("stacks %v after %d POST /stacks, want none after %d", got, f.creates, tc.creates)
			}
			var warnings []string
			for line := range strings.Lines(logBuf.String()) {
				if strings.Contains(line, "level=WARN") || strings.Contains(line, "level=ERROR") {
					warnings = append(warnings, line)
				}
			}
			if len(warnings) != 1 || !strings.Contains(warnings[0], "level=WARN") ||
				!strings.Contains(warnings[0], "not rebuilt") || !strings.Contains(warnings[0], "lacks the "+tc.permission+" permission") {
				t.Errorf("want one warning that names %s and says the stacks were not rebuilt, got:\n%s", tc.permission, logBuf.String())
			}
		})
	}
}

// Any other failure of GET /stacks or POST /stacks still fails the upload.
func TestRebuildSourceStacksOtherErrorsFail(t *testing.T) {
	for name, refuse := range map[string]func(*fakeStacks, error){
		"GET /stacks":  func(f *fakeStacks, err error) { f.listErr = err },
		"POST /stacks": func(f *fakeStacks, err error) { f.createErr = err },
	} {
		t.Run(name, func(t *testing.T) {
			f := newFakeStacks()
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
