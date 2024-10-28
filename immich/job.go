package immich

import "context"

type Job struct {
	JobCounts struct {
		Active    int `json:"active"`
		Completed int `json:"completed"`
		Failed    int `json:"failed"`
		Delayed   int `json:"delayed"`
		Waiting   int `json:"waiting"`
		Paused    int `json:"paused"`
	} `json:"jobCounts"`
	QueueStatus struct {
		IsActive bool `json:"isActive"`
		IsPaused bool `json:"isPaused"`
	} `json:"queueStatus"`
}

type SendJobCommandResponse struct {
	JobCounts struct {
		Active    int `json:"active"`
		Completed int `json:"completed"`
		Delayed   int `json:"delayed"`
		Failed    int `json:"failed"`
		Paused    int `json:"paused"`
		Waiting   int `json:"waiting"`
	} `json:"jobCounts"`
	QueueStatus struct {
		IsActive bool `json:"isActive"`
		IsPause  bool `json:"isPause"`
	}
}

type JobID string

const (
	StorageTemplateMigration JobID = "storageTemplateMigration"
)

type JobCommand string

const (
	Start       JobCommand = "start"
	Pause       JobCommand = "pause"
	Resume      JobCommand = "resume"
	Empty       JobCommand = "empty"
	ClearFailed JobCommand = "clear-failed"
)

type JobName string

const (
	PersonCleanup JobName = "person-cleanup"
	TagCleanup    JobName = "tag-cleanup"
	UserCleanup   JobName = "user-cleanup"
)

func (ic *ImmichClient) GetJobs(ctx context.Context) (map[string]Job, error) {
	var resp map[string]Job
	err := ic.newServerCall(ctx, EndPointGetJobs).
		do(getRequest("/jobs", setAcceptJSON()), responseJSON(&resp))
	return resp, err
}

func (ic *ImmichClient) SendJobCommand(
	ctx context.Context,
	jobID JobID,
	command JobCommand,
	force bool,
) (resp SendJobCommandResponse, err error) {
	err = ic.newServerCall(ctx, EndPointSendJobCommand).do(putRequest("/jobs/"+string(jobID),
		setJSONBody(struct {
			Command JobCommand `json:"command"`
			Force   bool       `json:"force"`
		}{Command: command, Force: force})), responseJSON(&resp))
	return
}

func (ic *ImmichClient) CreateJob(ctx context.Context, name JobName) error {
	return ic.newServerCall(ctx, EndPointCreateJob).do(postRequest("/jobs",
		"application/json",
		setJSONBody(struct {
			Name JobName `json:"name"`
		}{Name: name})))
}
