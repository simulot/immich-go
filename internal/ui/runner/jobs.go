package runner

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"time"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/ui/core/messages"
	"github.com/simulot/immich-go/internal/ui/core/state"
)

// DefaultJobsPollInterval controls how often the watcher queries Immich for job stats.
const DefaultJobsPollInterval = 250 * time.Millisecond

// JobsWatcherConfig configures the background job watcher.
type JobsWatcherConfig struct {
	Client    immich.ImmichJobInterface
	Publisher messages.Publisher
	Interval  time.Duration
	Logger    *slog.Logger
	Clock     func() time.Time
}

// StartJobsWatcher launches a goroutine that periodically fetches server job stats
// and publishes them through the provided Publisher. The returned cancel function
// stops the watcher; calling it multiple times is safe.
func StartJobsWatcher(ctx context.Context, cfg JobsWatcherConfig) context.CancelFunc {
	if cfg.Client == nil || cfg.Publisher == nil {
		return func() {}
	}

	interval := cfg.Interval
	if interval <= 0 {
		interval = DefaultJobsPollInterval
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}

	watchCtx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-watchCtx.Done():
				return
			case <-ticker.C:
				summaries, err := fetchJobSummaries(watchCtx, cfg.Client, clock)
				if err != nil {
					if cfg.Logger != nil && !errors.Is(err, context.Canceled) {
						cfg.Logger.Debug("job watcher: fetch failed", "error", err)
					}
					continue
				}
				cfg.Publisher.UpdateJobs(watchCtx, summaries)
			}
		}
	}()

	return cancel
}

func fetchJobSummaries(ctx context.Context, client immich.ImmichJobInterface, clock func() time.Time) ([]state.JobSummary, error) {
	jobs, err := client.GetJobs(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]state.JobSummary, 0, len(jobs))
	now := clock()
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
