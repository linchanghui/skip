package service

import (
	"context"
	"strings"
	"time"

	"skip/internal/domain"
	"skip/internal/repository"
)

type DispatchService struct {
	Repo *repository.Store
	Now  func() time.Time
}

func (s *DispatchService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *DispatchService) AcceptTask(ctx context.Context, taskID string, in domain.TaskActionByRunnerInput) (*domain.Task, error) {
	taskID = strings.TrimSpace(taskID)
	in.RunnerID = strings.TrimSpace(in.RunnerID)
	if taskID == "" || in.RunnerID == "" {
		return nil, domainErr("id and runner_id are required")
	}
	return s.Repo.AcceptTask(ctx, taskID, in.RunnerID, s.now())
}

func (s *DispatchService) ArriveTask(ctx context.Context, taskID string, in domain.TaskActionByRunnerInput) (*domain.Task, error) {
	taskID = strings.TrimSpace(taskID)
	in.RunnerID = strings.TrimSpace(in.RunnerID)
	if taskID == "" || in.RunnerID == "" {
		return nil, domainErr("id and runner_id are required")
	}
	return s.Repo.ArriveTask(ctx, taskID, in.RunnerID, s.now(), in.Note)
}

func (s *DispatchService) CompleteTask(ctx context.Context, taskID string, in domain.TaskActionByRunnerInput) (*domain.Task, error) {
	taskID = strings.TrimSpace(taskID)
	in.RunnerID = strings.TrimSpace(in.RunnerID)
	if taskID == "" || in.RunnerID == "" {
		return nil, domainErr("id and runner_id are required")
	}
	return s.Repo.CompleteTask(ctx, taskID, in.RunnerID, s.now(), in.Note)
}

func (s *DispatchService) AssignTaskByOps(ctx context.Context, taskID string, in domain.AssignTaskInput) (*domain.Task, error) {
	taskID = strings.TrimSpace(taskID)
	in.RunnerID = strings.TrimSpace(in.RunnerID)
	in.OpsID = strings.TrimSpace(in.OpsID)
	if taskID == "" || in.RunnerID == "" || in.OpsID == "" {
		return nil, domainErr("id, runner_id and ops_id are required")
	}
	return s.Repo.AssignTaskByOps(ctx, taskID, in.RunnerID, in.OpsID, s.now())
}
