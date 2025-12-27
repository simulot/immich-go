package watchers

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/ui/core/messages"
	"github.com/simulot/immich-go/internal/ui/core/state"
)

const DefaultJobsPollInterval = 250 * time.Millisecond

// JobsSource polls Immich for job statistics and emits EventJobsUpdated.
type JobsSource struct {
	client   immich.ImmichJobInterface
	interval time.Duration
	logger   *slog.Logger
	clock    func() time.Time
	ch       chan messages.Event
	mu       sync.Mutex
	closed   bool
}

// Ensure JobsSource implements messages.EventSource.
var _ messages.EventSource = (*JobsSource)(nil)

// NewJobsSource constructs a jobs event source.
func NewJobsSource(client immich.ImmichJobInterface, interval time.Duration, logger *slog.Logger) *JobsSource {
	if interval <= 0 {
		interval = DefaultJobsPollInterval
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(nil, nil))
	}
	return &JobsSource{
		client:   client,
		interval: interval,
		logger:   logger,
		clock:    time.Now,
		ch:       make(chan messages.Event, 1),
	}
}

// Subscribe returns the stream for this jobs source.
func (js *JobsSource) Subscribe(ctx context.Context) messages.Stream {
	go js.poll(ctx)
	return messages.Stream(js.ch)
}

// Close stops the source and closes its stream.
func (js *JobsSource) Close() error {
	js.mu.Lock()
	if js.closed {
		js.mu.Unlock()
		return nil
	}
	js.closed = true
	close(js.ch)
	js.mu.Unlock()
	return nil
}

func (js *JobsSource) poll(ctx context.Context) {
	defer func() {
		js.mu.Lock()
		if !js.closed {
			js.closed = true
			close(js.ch)
		}
		js.mu.Unlock()
	}()

	if js.client == nil {
		return
	}

	ticker := time.NewTicker(js.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			summaries, err := js.fetchJobSummaries(ctx)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					js.logger.Debug("job watcher: fetch failed", "error", err)
				}
				continue
			}
			js.send(messages.Event{Type: messages.EventJobsUpdated, Payload: summaries})
		}
	}
}

func (js *JobsSource) fetchJobSummaries(ctx context.Context) ([]state.JobSummary, error) {
	jobs, err := js.client.GetJobs(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]state.JobSummary, 0, len(jobs))
	now := js.clock()
	for name, job := range jobs {
		summaries = append(summaries, convertJob(name, job, now))
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})
	return summaries, nil
}

func convertJob(name string, job immich.Job, ts time.Time) state.JobSummary {
	pending := job.JobCounts.Active + job.JobCounts.Waiting
	return state.JobSummary{
		Name:      name,
		Kind:      name,
		Active:    job.JobCounts.Active,
		Waiting:   job.JobCounts.Waiting,
		Pending:   pending,
		Completed: job.JobCounts.Completed,
		Failed:    job.JobCounts.Failed,
		UpdatedAt: ts,
	}
}

func (js *JobsSource) send(evt messages.Event) {
	js.mu.Lock()
	if js.closed {
		js.mu.Unlock()
		return
	}
	select {
	case js.ch <- evt:
	default:
	}
	js.mu.Unlock()
}
