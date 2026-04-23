package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"skip/internal/domain"
)

func (r *Store) ApplyRunner(ctx context.Context, in domain.RunnerApplyInput, now time.Time) (*domain.Runner, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	runnerID := fmt.Sprintf("runner-%d", now.UnixNano())
	area := "changi"
	if in.ServiceArea != nil && *in.ServiceArea != "" {
		area = *in.ServiceArea
	}
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO runners
			(id, name, phone, status, service_area, reliability_score, agreement_version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 0.5, 'v1', ?, ?)`,
		runnerID, in.Name, in.Phone, string(domain.RunnerCandidate), area, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	return r.GetRunner(ctx, runnerID)
}

func (r *Store) GetRunner(ctx context.Context, runnerID string) (*domain.Runner, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, name, phone, status, service_area, reliability_score, agreement_version, created_at, updated_at
		FROM runners
		WHERE id = ?`, runnerID,
	)
	var out domain.Runner
	var status, createdAt, updatedAt string
	var agreement sql.NullString
	if err := row.Scan(&out.ID, &out.Name, &out.Phone, &status, &out.ServiceArea, &out.ReliabilityScore, &agreement, &createdAt, &updatedAt); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	rs, _ := domain.ParseRunnerStatus(status)
	out.Status = rs
	out.AgreementVersion = nullStringPtr(agreement)
	out.CreatedAt = parseDBTime(createdAt)
	out.UpdatedAt = parseDBTime(updatedAt)
	return &out, nil
}

func (r *Store) SetRunnerAvailability(ctx context.Context, runnerID string, in domain.RunnerAvailabilityInput, now time.Time) (*domain.RunnerAvailability, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var exists int
	if err := r.DB.QueryRowContext(ctx, `SELECT 1 FROM runners WHERE id = ?`, runnerID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	var lat, lng any
	if in.Location != nil {
		lat = in.Location.Lat
		lng = in.Location.Lng
	}
	isOnline := 0
	if in.IsOnline {
		isOnline = 1
	}
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO runner_availability (runner_id, is_online, current_lat, current_lng, last_ping_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(runner_id) DO UPDATE SET
			is_online = excluded.is_online,
			current_lat = COALESCE(excluded.current_lat, runner_availability.current_lat),
			current_lng = COALESCE(excluded.current_lng, runner_availability.current_lng),
			last_ping_at = excluded.last_ping_at,
			updated_at = excluded.updated_at`,
		runnerID, isOnline, lat, lng, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}

	row := r.DB.QueryRowContext(ctx, `
		SELECT runner_id, is_online, current_lat, current_lng, active_task_id, last_ping_at, updated_at
		FROM runner_availability WHERE runner_id = ?`, runnerID,
	)
	var out domain.RunnerAvailability
	var online int
	var latN, lngN sql.NullFloat64
	var activeTaskID sql.NullString
	var lastPing, updated string
	if err := row.Scan(&out.RunnerID, &online, &latN, &lngN, &activeTaskID, &lastPing, &updated); err != nil {
		return nil, err
	}
	out.IsOnline = online == 1
	if latN.Valid && lngN.Valid {
		out.Location = &domain.LatLng{Lat: latN.Float64, Lng: lngN.Float64}
	}
	out.ActiveTaskID = nullStringPtr(activeTaskID)
	out.LastPingAt = parseDBTime(lastPing)
	out.UpdatedAt = parseDBTime(updated)
	return &out, nil
}
