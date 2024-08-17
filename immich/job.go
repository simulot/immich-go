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

func (ic *ImmichClient) GetJobs(ctx context.Context) (map[string]Job, error) {
	var resp map[string]Job
	err := ic.newServerCall(ctx, EndPointGetJobs).do(getRequest("/jobs", setAcceptJSON()), responseJSON(&resp))
	return resp, err
}
