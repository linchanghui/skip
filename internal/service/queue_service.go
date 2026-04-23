package service

import (
	"context"
	"strings"
	"time"

	"skip/internal/domain"
	"skip/internal/repository"
)

type QueueService struct {
	Repo *repository.Store
	Now  func() time.Time
}

func (s *QueueService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *QueueService) Report(ctx context.Context, storeID string, in domain.CreateQueueReportInput) (*domain.QueueReport, error) {
	storeID = strings.TrimSpace(storeID)
	if storeID == "" {
		return nil, domainErr("id is required")
	}
	if _, err := domain.ParseReporterType(string(in.ReporterType)); err != nil {
		return nil, domainErr("invalid reporter_type")
	}
	if _, err := domain.ParseBusyLevel(string(in.BusyLevel)); err != nil {
		return nil, domainErr("invalid busy_level")
	}
	if in.TTLMinutes != nil {
		if *in.TTLMinutes <= 0 || *in.TTLMinutes > 180 {
			return nil, domainErr("ttl_minutes must be 1-180")
		}
	}
	return s.Repo.CreateQueueReport(ctx, storeID, in, s.now())
}

func (s *QueueService) GetSignal(ctx context.Context, storeID string) (*domain.QueueSignal, error) {
	storeID = strings.TrimSpace(storeID)
	if storeID == "" {
		return nil, domainErr("id is required")
	}
	return s.Repo.GetQueueSignal(ctx, storeID, s.now())
}

func (s *QueueService) HideReport(ctx context.Context, id int64) (*domain.QueueReport, error) {
	if id <= 0 {
		return nil, domainErr("id must be positive")
	}
	return s.Repo.HideQueueReport(ctx, id)
}
