package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"skip/internal/domain"
)

func (r *Store) CreateQueueReport(ctx context.Context, storeID string, in domain.CreateQueueReportInput, now time.Time) (*domain.QueueReport, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	ttl := 30
	if in.TTLMinutes != nil && *in.TTLMinutes > 0 {
		ttl = *in.TTLMinutes
	}
	expiresAt := now.Add(time.Duration(ttl) * time.Minute)

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM stores WHERE id = ?`, storeID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO queue_reports
			(store_id, reporter_type, reporter_id, queue_length, wait_minutes_est, busy_level, evidence_url, confidence_flag, reported_at, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'normal', ?, ?, ?)`,
		storeID, string(in.ReporterType), in.ReporterID, in.QueueLength, in.WaitMinutesEst, string(in.BusyLevel), in.EvidenceURL,
		now.Format(time.RFC3339), expiresAt.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	// Keep map/list/detail in sync: queue reports should immediately surface as latest crowd signal.
	// Only overwrite when this report is newer than the current latest status.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO store_status_latest
			(store_id, busy_level, queue_length, wait_minutes_est, source, as_of)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(store_id) DO UPDATE SET
			busy_level = excluded.busy_level,
			queue_length = excluded.queue_length,
			wait_minutes_est = excluded.wait_minutes_est,
			source = excluded.source,
			as_of = excluded.as_of
		WHERE excluded.as_of >= store_status_latest.as_of`,
		storeID, string(in.BusyLevel), in.QueueLength, in.WaitMinutesEst, string(domain.SourceCrowd), now.Format(time.RFC3339),
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetQueueReportByID(ctx, id)
}

func (r *Store) GetQueueReportByID(ctx context.Context, id int64) (*domain.QueueReport, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, store_id, reporter_type, reporter_id, queue_length, wait_minutes_est,
		       busy_level, evidence_url, confidence_flag, reported_at, expires_at, created_at
		FROM queue_reports WHERE id = ?`, id,
	)
	return scanQueueReportRow(row)
}

func (r *Store) HideQueueReport(ctx context.Context, id int64) (*domain.QueueReport, error) {
	res, err := r.DB.ExecContext(ctx, `
		UPDATE queue_reports SET confidence_flag='low' WHERE id = ?`, id,
	)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.GetQueueReportByID(ctx, id)
}

func (r *Store) GetQueueSignal(ctx context.Context, storeID string, now time.Time) (*domain.QueueSignal, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var exists int
	if err := r.DB.QueryRowContext(ctx, `SELECT 1 FROM stores WHERE id = ?`, storeID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	row := r.DB.QueryRowContext(ctx, `
		SELECT id, store_id, reporter_type, reporter_id, queue_length, wait_minutes_est,
		       busy_level, evidence_url, confidence_flag, reported_at, expires_at, created_at
		FROM queue_reports
		WHERE store_id = ?
		ORDER BY reported_at DESC, id DESC
		LIMIT 1`, storeID,
	)
	qr, err := scanQueueReportRow(row)
	if errors.Is(err, ErrNotFound) {
		return &domain.QueueSignal{
			StoreID:             storeID,
			StatusExpired:       true,
			LastUpdatedAt:       time.Unix(0, 0).UTC(),
			LastUpdatedXMinsAgo: 0,
			Signal:              nil,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	minAgo := int(now.Sub(qr.ReportedAt).Minutes())
	if minAgo < 0 {
		minAgo = 0
	}
	expired := now.After(qr.ExpiresAt)
	signal := qr
	if expired {
		signal = nil
	}
	return &domain.QueueSignal{
		StoreID:             storeID,
		StatusExpired:       expired,
		LastUpdatedAt:       qr.ReportedAt,
		LastUpdatedXMinsAgo: minAgo,
		Signal:              signal,
	}, nil
}

func scanQueueReportRow(row *sql.Row) (*domain.QueueReport, error) {
	var out domain.QueueReport
	var (
		reporterType, busyLevel, reportedAt, expiresAt, createdAt string
		reporterID, evidenceURL                                   sql.NullString
		queueLength, waitMinutes                                  sql.NullInt64
	)
	if err := row.Scan(&out.ID, &out.StoreID, &reporterType, &reporterID, &queueLength, &waitMinutes, &busyLevel,
		&evidenceURL, &out.ConfidenceFlag, &reportedAt, &expiresAt, &createdAt); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	rt, _ := domain.ParseReporterType(reporterType)
	bl, _ := domain.ParseBusyLevel(busyLevel)
	out.ReporterType = rt
	out.BusyLevel = bl
	out.ReporterID = nullStringPtr(reporterID)
	out.QueueLength = nullIntPtr(queueLength)
	out.WaitMinutesEst = nullIntPtr(waitMinutes)
	out.EvidenceURL = nullStringPtr(evidenceURL)
	out.ReportedAt = parseDBTime(reportedAt)
	out.ExpiresAt = parseDBTime(expiresAt)
	out.CreatedAt = parseDBTime(createdAt)
	return &out, nil
}
