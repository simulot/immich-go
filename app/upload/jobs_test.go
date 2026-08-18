package upload

import (
	"context"
	"errors"
	"io"
	"slices"
	"testing"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
	"github.com/spf13/cobra"
)

// jobsClient stubs the admin client's job endpoints and records the commands sent.
type jobsClient struct {
	immich.ImmichInterface
	paused      []string // queues reported as paused by GetJobs
	failPauseOf string   // queue whose pause command fails, if set
	commands    []string // "command:queue", in order
}

func (c *jobsClient) GetJobs(context.Context) (map[string]immich.Job, error) {
	jobs := map[string]immich.Job{}
	for _, name := range c.paused {
		var j immich.Job
		j.QueueStatus.IsPaused = true
		jobs[name] = j
	}
	return jobs, nil
}

func (c *jobsClient) SendJobCommand(_ context.Context, jobID string, command immich.JobCommand, _ bool) (immich.SendJobCommandResponse, error) {
	c.commands = append(c.commands, string(command)+":"+jobID)
	if command == immich.Pause && jobID == c.failPauseOf {
		return immich.SendJobCommandResponse{}, errors.New("pause failed")
	}
	return immich.SendJobCommandResponse{}, nil
}

func newJobsUpCmd(t *testing.T, client *jobsClient) *UpCmd {
	t.Helper()
	a := app.New(context.Background(), &cobra.Command{})
	a.Log().SetLogWriter(io.Discard)
	return &UpCmd{app: a, client: app.Client{AdminImmich: client}}
}

func TestPauseResumeJobsSkipsAlreadyPausedQueues(t *testing.T) {
	client := &jobsClient{paused: []string{"thumbnailGeneration"}}
	uc := newJobsUpCmd(t, client)
	ctx := context.Background()

	if err := uc.pauseJobs(ctx); err != nil {
		t.Fatalf("pauseJobs: %v", err)
	}
	if err := uc.resumeJobs(ctx); err != nil {
		t.Fatalf("resumeJobs: %v", err)
	}

	want := []string{
		"pause:metadataExtraction", "pause:videoConversion", "pause:faceDetection", "pause:smartSearch",
		"resume:metadataExtraction", "resume:videoConversion", "resume:faceDetection", "resume:smartSearch",
	}
	if !slices.Equal(client.commands, want) {
		t.Errorf("commands sent:\n got %v\nwant %v", client.commands, want)
	}
}

func TestResumeJobsWithoutPauseSendsNothing(t *testing.T) {
	client := &jobsClient{}
	uc := newJobsUpCmd(t, client)

	if err := uc.resumeJobs(context.Background()); err != nil {
		t.Fatalf("resumeJobs: %v", err)
	}
	if len(client.commands) != 0 {
		t.Errorf("commands sent without a prior pause: %v", client.commands)
	}
}

func TestPauseJobsFailureResumesQueuesPausedSoFar(t *testing.T) {
	client := &jobsClient{failPauseOf: "videoConversion"}
	uc := newJobsUpCmd(t, client)

	if err := uc.pauseJobs(context.Background()); err == nil {
		t.Fatal("pauseJobs: expected an error")
	}

	want := []string{
		"pause:thumbnailGeneration", "pause:metadataExtraction", "pause:videoConversion",
		"resume:thumbnailGeneration", "resume:metadataExtraction",
	}
	if !slices.Equal(client.commands, want) {
		t.Errorf("commands sent:\n got %v\nwant %v", client.commands, want)
	}
}
