package service

import (
	"context"
	"strings"
	"time"

	"skip/internal/domain"
	"skip/internal/repository"
)

type TaskService struct {
	Repo *repository.Store
	Now  func() time.Time
}

func (s *TaskService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *TaskService) CreateTask(ctx context.Context, in domain.CreateTaskInput) (*domain.Task, error) {
	in.UserID = strings.TrimSpace(in.UserID)
	in.StoreID = strings.TrimSpace(in.StoreID)
	if in.UserID == "" || in.StoreID == "" {
		return nil, domainErr("user_id and store_id are required")
	}
	if _, err := domain.ParseTaskType(string(in.TaskType)); err != nil {
		return nil, domainErr("invalid task_type")
	}
	return s.Repo.CreateTask(ctx, in, s.now())
}

func (s *TaskService) GetTask(ctx context.Context, taskID string) (*domain.TaskDetail, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, domainErr("id is required")
	}
	return s.Repo.GetTask(ctx, taskID, 50)
}

func (s *TaskService) CancelTask(ctx context.Context, taskID string, in domain.CancelTaskInput) (*domain.Task, error) {
	taskID = strings.TrimSpace(taskID)
	in.UserID = strings.TrimSpace(in.UserID)
	if taskID == "" || in.UserID == "" {
		return nil, domainErr("id and user_id are required")
	}
	return s.Repo.CancelTask(ctx, taskID, in.UserID, s.now())
}
