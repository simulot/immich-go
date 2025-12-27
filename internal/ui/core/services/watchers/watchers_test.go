package watchers

import (
	"context"
	"testing"
	"time"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/ui/core/state"
)

type mockJobClient struct {
	jobs  map[string]immich.Job
	err   error
	calls int
}

func (m *mockJobClient) GetJobs(ctx context.Context) (map[string]immich.Job, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.jobs, nil
}

func (m *mockJobClient) SendJobCommand(ctx context.Context, jobID string, command immich.JobCommand, force bool) (immich.SendJobCommandResponse, error) {
	return immich.SendJobCommandResponse{}, nil
}

func (m *mockJobClient) CreateJob(ctx context.Context, name immich.JobName) error {
	return nil
}

func TestJobsSourceEmitsUpdates(t *testing.T) {
	job := immich.Job{}
	job.JobCounts.Active = 1
	job.JobCounts.Waiting = 2
	job.JobCounts.Completed = 10
	job.JobCounts.Failed = 0

	jobs := map[string]immich.Job{"imageTagging": job}
	client := &mockJobClient{jobs: jobs}
	source := NewJobsSource(client, 10*time.Millisecond, nil)
	stream := source.Subscribe(context.Background())

	select {
	case evt := <-stream:
		if evt.Type != "jobs_updated" {
			t.Fatalf("expected jobs_updated, got %s", evt.Type)
		}
		summaries, ok := evt.Payload.([]state.JobSummary)
		if !ok {
			t.Fatalf("expected []JobSummary, got %T", evt.Payload)
		}
		if len(summaries) != 1 || summaries[0].Name != "imageTagging" {
			t.Fatalf("unexpected jobs: %+v", summaries)
		}
		if summaries[0].Active != 1 || summaries[0].Waiting != 2 {
			t.Fatalf("unexpected job counts: %+v", summaries[0])
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for jobs event")
	}

	source.Close()
}

func TestJobsSourceSkipsOnError(t *testing.T) {
	client := &mockJobClient{err: context.Canceled}
	source := NewJobsSource(client, 10*time.Millisecond, nil)
	stream := source.Subscribe(context.Background())

	select {
	case _, ok := <-stream:
		if ok {
			t.Fatalf("expected no event on context canceled")
		}
	case <-time.After(100 * time.Millisecond):
		// expected: no data due to error
	}

	source.Close()
}

type mockInventoryClient struct {
	stats immich.UserStatistics
	err   error
}

func (m *mockInventoryClient) GetAssetStatistics(ctx context.Context) (immich.UserStatistics, error) {
	if m.err != nil {
		return immich.UserStatistics{}, m.err
	}
	return m.stats, nil
}

func TestInventorySourceEmitsUpdates(t *testing.T) {
	stats := immich.UserStatistics{Images: 100, Videos: 50, Total: 150}
	client := &mockInventoryClient{stats: stats}
	source := NewInventorySource(client, 10*time.Millisecond, nil)
	stream := source.Subscribe(context.Background())

	select {
	case evt := <-stream:
		if evt.Type != "inventory_updated" {
			t.Fatalf("expected inventory_updated, got %s", evt.Type)
		}
		inv, ok := evt.Payload.(state.ServerInventory)
		if !ok {
			t.Fatalf("expected ServerInventory, got %T", evt.Payload)
		}
		if inv.Photos != 100 || inv.Videos != 50 || inv.Total != 150 {
			t.Fatalf("unexpected inventory: %+v", inv)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for inventory event")
	}

	source.Close()
}

func TestInventorySourceSkipsOnError(t *testing.T) {
	client := &mockInventoryClient{err: context.Canceled}
	source := NewInventorySource(client, 10*time.Millisecond, nil)
	stream := source.Subscribe(context.Background())

	select {
	case _, ok := <-stream:
		if ok {
			t.Fatalf("expected no event on context canceled")
		}
	case <-time.After(100 * time.Millisecond):
		// expected: no data due to error
	}

	source.Close()
}
