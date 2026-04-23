package service

import (
	"context"
	"strings"
	"time"

	"skip/internal/domain"
	"skip/internal/repository"
)

type RunnerService struct {
	Repo *repository.Store
	Now  func() time.Time
}

func (s *RunnerService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *RunnerService) Apply(ctx context.Context, in domain.RunnerApplyInput) (*domain.Runner, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Phone = strings.TrimSpace(in.Phone)
	if in.Name == "" || in.Phone == "" {
		return nil, domainErr("name and phone are required")
	}
	return s.Repo.ApplyRunner(ctx, in, s.now())
}

func (s *RunnerService) SetAvailability(ctx context.Context, runnerID string, in domain.RunnerAvailabilityInput) (*domain.RunnerAvailability, error) {
	runnerID = strings.TrimSpace(runnerID)
	if runnerID == "" {
		return nil, domainErr("id is required")
	}
	return s.Repo.SetRunnerAvailability(ctx, runnerID, in, s.now())
}
