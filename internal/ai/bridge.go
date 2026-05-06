package ai

import "context"

type WorkerRequest struct {
	Action  string
	Payload map[string]any
}

type WorkerResponse struct {
	Status string
	Data   map[string]any
}

type Bridge interface {
	Run(context.Context, WorkerRequest) (WorkerResponse, error)
}
