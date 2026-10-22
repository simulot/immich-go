package upload

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
)

// The stacks of an immich-go archive come from the source, not from names and capture times.
// `archive from-immich` writes each asset's stack on the source server into its .JSON sidecar
// (assets.Metadata.Stack); upload notes, for each uploaded asset, the ID it has on this server,
// and after the last upload rebuilds exactly those stacks, with the same primary. Members can sit
// in different folders. upload from-immich reads the same stacks from the source server.

type stackMember struct {
	id      string // the asset's ID on this server
	primary bool   // it was the stack's primary on the source server
	count   int    // the stack's size as the source server reported it
	file    fshelper.FSAndName
}

// sourceStacks collects the members of each source stack while the assets are uploaded.
type sourceStacks struct {
	mu   sync.Mutex
	m    map[string][]stackMember // source stack ID -> members
	gone map[string]bool          // server assets deleted during the upload (replaced by a bigger file)
}

// add notes a's source stack, if its sidecar names one, with id, the asset's ID on this server.
func (s *sourceStacks) add(a *assets.Asset, id string) {
	if id == "" || a == nil || a.FromApplication == nil || a.FromApplication.Stack == nil || a.FromApplication.Stack.ID == "" {
		return
	}
	st := a.FromApplication.Stack
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gone[id] {
		return
	}
	if s.m == nil {
		s.m = map[string][]stackMember{}
	}
	s.m[st.ID] = append(s.m[st.ID], stackMember{id: id, primary: st.Primary, count: st.AssetCount, file: a.File})
}

// forget drops a server asset that was deleted during the upload from every stack.
func (s *sourceStacks) forget(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gone == nil {
		s.gone = map[string]bool{}
	}
	s.gone[id] = true
	for k, ms := range s.m {
		kept := ms[:0]
		for _, m := range ms {
			if m.id != id {
				kept = append(kept, m)
			}
		}
		if len(kept) == 0 {
			delete(s.m, k)
		} else {
			s.m[k] = kept
		}
	}
}

// stackMemberID is the ID of the server asset that holds this very file (the same content), or ""
// when there is none: not uploaded, or the server keeps another file in its place (the same name,
// capture time and maybe size; ShouldUpload). A stack is rebuilt only from assets known to be the
// archived files themselves.
func stackMemberID(a *assets.Asset, adv *Advice) string {
	switch adv.Advice {
	case NotOnServer, ForceUpload, SmallerOnServer: // this file was uploaded (or matched by checksum)
		return a.ID
	case AlreadyProcessed: // the same content, uploaded from another file of this run
		if adv.ServerAsset != nil {
			return adv.ServerAsset.ID
		}
	case SameOnServer:
		if adv.ServerAsset != nil && adv.ServerAsset.Checksum != "" && adv.ServerAsset.Checksum == a.Checksum {
			return a.ID
		}
	}
	return ""
}

func (s *sourceStacks) len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.m)
}

type stackAPI interface {
	GetAllStacks(ctx context.Context) ([]immich.Stack, error)
	CreateStack(ctx context.Context, ids []string) (string, error)
	GetAssetInfo(ctx context.Context, id string) (*immich.Asset, error)
}

type stackResult struct {
	created int    // stacks made
	present int    // stacks already there, exactly (a rerun)
	notMade int    // source stacks with 1 member here, or whose members are in other stacks
	failed  int    // CreateStack errors, and stacks there that could not be checked
	refused string // the stack permission the API key lacks: the stacks left were not rebuilt
	stacked []fshelper.FSAndName
}

// refuse notes that the server refused a call with 403 because the API key lacks permission, and
// logs that the n stacks left are not rebuilt, with the stacks made and already there by then. It
// is no reason to fail the upload: every file is on the server, only the stacks are missing.
func (r *stackResult) refuse(log *slog.Logger, permission string, n int, err error) {
	r.refused = permission
	log.Warn("stacks from the source not rebuilt: the API key lacks the "+permission+" permission", "stacks", n,
		"made", r.created, "already there", r.present, "err", err)
}

// rebuildStacks makes, for each source stack with 2 or more members here, one stack of exactly
// those members, its source primary first. It lists the server's stacks once: a stack that is
// already there with the same members and primary is left alone, so reruns and resumes make no
// duplicate. A stack whose members are all in this one, left by a run that was cut short or made
// by the groupers (series, burst, RAW+JPEG...) earlier in this run, is taken over by the new stack
// (Immich merges a stack whose primary is among the new stack's assets). A member in a stack with
// other assets is left as it is, and that source stack is not made, with a warning: it does not
// fail the upload. Missing members are logged once the stack is made; a stack with 1 member here
// is not made. When the server refuses GET /stacks (stack.read) or POST /stacks (stack.create)
// with 403, the stacks left are not rebuilt, with one warning and no error.
func rebuildStacks(ctx context.Context, api stackAPI, src *sourceStacks, log *slog.Logger) (stackResult, error) {
	var r stackResult
	src.mu.Lock()
	byStack := make(map[string][]stackMember, len(src.m))
	for k, v := range src.m {
		byStack[k] = append([]stackMember(nil), v...)
	}
	src.mu.Unlock()
	if len(byStack) == 0 {
		return r, nil
	}

	existing, err := api.GetAllStacks(ctx)
	if immich.IsForbidden(err) {
		r.refuse(log, "stack.read", len(byStack), err)
		return r, nil
	}
	if err != nil {
		return r, fmt.Errorf("can't list the server's stacks: %w", err)
	}
	stackOf := map[string]int{} // asset ID -> index in existing (its primary, and the members listed)
	for i := range existing {
		stackOf[existing[i].PrimaryAssetID] = i
		for _, a := range existing[i].Assets {
			stackOf[a.ID] = i
		}
	}

	ids := make([]string, 0, len(byStack))
	for k := range byStack {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	for n, sid := range ids {
		// One member per asset here (two files can end as one asset), in file name order.
		seen := map[string]int{}
		var ms []stackMember
		want := 0
		for _, m := range byStack[sid] {
			want = max(want, m.count)
			if i, ok := seen[m.id]; ok {
				ms[i].primary = ms[i].primary || m.primary
				continue
			}
			seen[m.id] = len(ms)
			ms = append(ms, m)
		}
		sort.SliceStable(ms, func(i, j int) bool {
			if ms[i].file.FullName() != ms[j].file.FullName() {
				return ms[i].file.FullName() < ms[j].file.FullName()
			}
			return ms[i].id < ms[j].id
		})
		if len(ms) < 2 {
			log.Warn("stack not made: only one of its members is here", "source stack", sid,
				"members on the source", want, "file", ms[0].file)
			r.notMade++
			continue
		}
		p, np := -1, 0
		for i, m := range ms {
			if m.primary {
				np++
				if p < 0 {
					p = i
				}
			}
		}
		switch {
		case p < 0:
			p = 0
			log.Warn("the stack's primary is not here: the first member by file name is its primary", "source stack", sid, "primary", ms[0].file)
		case np > 1:
			log.Warn("the sidecars name more than one primary: the first by file name is the primary", "source stack", sid, "primary", ms[p].file)
		}
		order := []string{ms[p].id}
		for i, m := range ms {
			if i != p {
				order = append(order, m.id)
			}
		}
		members := map[string]bool{}
		for _, id := range order {
			members[id] = true
		}

		// Already there, exactly? A lookup that fails is an error for this stack, never a reason
		// to make it again.
		if i, ok := stackOf[order[0]]; ok && existing[i].PrimaryAssetID == order[0] {
			same, err := sameStack(ctx, api, existing[i], order, members)
			if err != nil {
				log.Error("stack not made: can't check the stack that is there", "source stack", sid, "stack", existing[i].ID, "err", err)
				r.failed++
				continue
			}
			if same {
				r.present++
				for _, m := range ms {
					r.stacked = append(r.stacked, m.file)
				}
				continue
			}
		}
		// A member in a stack with other assets: not ours to change. GET /stacks leaves out trashed
		// and hidden assets, so a primary that is not ours makes it another stack too.
		var other []string
		for _, id := range order {
			if i, ok := stackOf[id]; ok {
				mixed := !members[existing[i].PrimaryAssetID]
				for _, a := range existing[i].Assets {
					mixed = mixed || !members[a.ID]
				}
				if mixed {
					other = append(other, existing[i].ID)
				}
			}
		}
		if len(other) > 0 {
			log.Warn("stack not made: a member is in another stack, with assets that are not in this one", "source stack", sid,
				"other stacks", strings.Join(other, ","))
			r.notMade++
			continue
		}
		newID, err := api.CreateStack(ctx, order)
		if immich.IsForbidden(err) { // the next ones would be refused too
			r.refuse(log, "stack.create", len(ids)-n, err)
			return r, nil
		}
		if err != nil {
			log.Error("can't make the stack", "source stack", sid, "err", err)
			r.failed++
			continue
		}
		log.Debug("stack made", "source stack", sid, "stack", newID, "primary", ms[p].file, "members", len(order))
		if want > len(ms) {
			log.Warn("stack made without its missing members", "source stack", sid,
				"members here", len(ms), "members on the source", want)
		}
		r.created++
		for _, m := range ms {
			r.stacked = append(r.stacked, m.file)
		}
	}
	return r, nil
}

// sameStack tells whether st is the stack of exactly these members, with order[0] its primary.
// GET /stacks leaves out trashed and hidden members, so a member it does not list is looked up
// with GET /assets/{id} (which reports its stack while another member is on the timeline); a
// member neither shows in st means it is not (and the stack is made again: Immich merges it). A
// failed GET /assets/{id} is returned as the error: it tells nothing about the stack.
func sameStack(ctx context.Context, api stackAPI, st immich.Stack, order []string, members map[string]bool) (bool, error) {
	listed := map[string]bool{st.PrimaryAssetID: true}
	for _, a := range st.Assets {
		if !members[a.ID] {
			return false, nil
		}
		listed[a.ID] = true
	}
	for _, id := range order {
		if listed[id] {
			continue
		}
		a, err := api.GetAssetInfo(ctx, id)
		if err != nil {
			return false, err
		}
		if a == nil || a.Stack == nil || a.Stack.ID != st.ID {
			return false, nil
		}
	}
	return true, nil
}

// rebuildSourceStacks runs rebuildStacks when the source named stacks (the .JSON sidecars of an
// archive, or the source server for upload from-immich), and logs the result.
func (uc *UpCmd) rebuildSourceStacks(ctx context.Context) error {
	if uc.stacks.len() == 0 {
		return nil
	}
	r, err := rebuildStacks(ctx, uc.client.Immich, &uc.stacks, uc.app.Log().Logger)
	if err != nil {
		return err
	}
	if uc.app.FileProcessor() != nil {
		for _, f := range r.stacked {
			uc.app.FileProcessor().RecordNonAsset(ctx, f, 0, fileevent.ProcessedStacked)
		}
	}
	if r.refused == "" { // otherwise the warning says the stacks were not rebuilt
		uc.app.Log().Info("stacks from the source", "made", r.created, "already there", r.present, "not made", r.notMade, "failed", r.failed)
	}
	if r.failed > 0 {
		return fmt.Errorf("%d stacks could not be made", r.failed)
	}
	return nil
}
