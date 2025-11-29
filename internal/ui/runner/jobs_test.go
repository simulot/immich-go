package runner

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/ui/core/messages"
	"github.com/simulot/immich-go/internal/ui/core/state"
)

func TestConvertJob(t *testing.T) {
	ts := time.Unix(1_700_000_000, 0)
	job := immich.Job{}
	job.JobCounts.Active = 3
	job.JobCounts.Waiting = 2
	job.JobCounts.Completed = 10
	job.JobCounts.Failed = 1

	summary := convertJob("ingest", job, ts)

	if summary.Name != "ingest" || summary.Kind != "ingest" {
		t.Fatalf("unexpected name/kind: %#v", summary)
	}
	if summary.Active != 3 || summary.Waiting != 2 {
		t.Fatalf("unexpected active or waiting counts: %#v", summary)
	}
	if summary.Pending != 5 {
		t.Fatalf("pending should sum active+waiting, got %d", summary.Pending)
	}
	if summary.Completed != 10 || summary.Failed != 1 {
		t.Fatalf("unexpected completed/failed counts: %#v", summary)
	}
	if !summary.UpdatedAt.Equal(ts) {
		t.Fatalf("expected timestamp %v, got %v", ts, summary.UpdatedAt)
	}
}

func TestFetchJobSummariesSortsAndStamps(t *testing.T) {
	client := &fakeJobClient{
		jobs: map[string]immich.Job{
			"b-job": jobWithCounts(1, 2, 3),
			"a-job": jobWithCounts(5, 0, 7),
		},
	}

	ts := time.Unix(1_800_000_000, 0)
	summaries, err := fetchJobSummaries(context.Background(), client, func() time.Time { return ts })
	if err != nil {
		t.Fatalf("fetchJobSummaries returned error: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}
	if summaries[0].Name != "a-job" || summaries[1].Name != "b-job" {
		t.Fatalf("summaries not sorted alphabetically: %#v", summaries)
	}
	if summaries[0].UpdatedAt != ts || summaries[1].UpdatedAt != ts {
		t.Fatalf("expected timestamps to be %v", ts)
	}
}

func TestStartJobsWatcherPublishesUpdates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := &fakeJobClient{
		jobs: map[string]immich.Job{
			"ingest": jobWithCounts(1, 1, 0),
		},
	}

	pub := &testPublisher{}
	stop := StartJobsWatcher(ctx, JobsWatcherConfig{
		Client:    client,
		Publisher: pub,
		Interval:  10 * time.Millisecond,
		Clock: func() time.Time {
			return time.Unix(1_900_000_000, 0)
		},
	})
	defer stop()

	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		if pub.jobUpdateCount() > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if pub.jobUpdateCount() == 0 {
		t.Fatalf("expected at least one job update to be published")
	}
}

type fakeJobClient struct {
	mu   sync.Mutex
	jobs map[string]immich.Job
	err  error
}

func (f *fakeJobClient) GetJobs(context.Context) (map[string]immich.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	snapshot := make(map[string]immich.Job, len(f.jobs))
	for k, v := range f.jobs {
		snapshot[k] = v
	}
	return snapshot, nil
}

func (f *fakeJobClient) SendJobCommand(context.Context, string, immich.JobCommand, bool) (immich.SendJobCommandResponse, error) {
	return immich.SendJobCommandResponse{}, nil
}

func (f *fakeJobClient) CreateJob(context.Context, immich.JobName) error {
	return nil
}

type testPublisher struct {
	mu   sync.Mutex
	jobs [][]state.JobSummary
}

func (p *testPublisher) AssetQueued(context.Context, state.AssetEvent)   {}
func (p *testPublisher) AssetUploaded(context.Context, state.AssetEvent) {}
func (p *testPublisher) AssetFailed(context.Context, state.AssetEvent)   {}
func (p *testPublisher) UpdateStats(context.Context, state.RunStats)     {}
func (p *testPublisher) AppendLog(context.Context, state.LogEvent)       {}
func (p *testPublisher) Close()                                          {}

func (p *testPublisher) UpdateJobs(_ context.Context, jobs []state.JobSummary) {
	p.mu.Lock()
	defer p.mu.Unlock()
	copyJobs := make([]state.JobSummary, len(jobs))
	copy(copyJobs, jobs)
	p.jobs = append(p.jobs, copyJobs)
}

func (p *testPublisher) jobUpdateCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.jobs)
}

var _ messages.Publisher = (*testPublisher)(nil)

func jobWithCounts(active, waiting, completed int) immich.Job {
	job := immich.Job{}
	job.JobCounts.Active = active
	job.JobCounts.Waiting = waiting
	job.JobCounts.Completed = completed
	return job
}
