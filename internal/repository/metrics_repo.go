package repository

import (
	"context"
	"sort"
	"time"

	"skip/internal/domain"
)

func (r *Store) GetMetricsSummary(ctx context.Context, now time.Time) (*domain.MetricsSummary, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	out := &domain.MetricsSummary{}

	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM tasks`).Scan(&out.TotalTasks); err != nil {
		return nil, err
	}
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM tasks WHERE status = 'completed'`).Scan(&out.CompletedTasks); err != nil {
		return nil, err
	}
	if out.TotalTasks > 0 {
		out.CompletionRatePct = int(float64(out.CompletedTasks) * 100.0 / float64(out.TotalTasks))
	}

	if err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM (
			SELECT task_id
			FROM task_attempts
			GROUP BY task_id
			HAVING MAX(attempt_no) >= 2
		) t`).Scan(&out.ReassignedTasks); err != nil {
		return nil, err
	}
	if out.TotalTasks > 0 {
		out.ReassignmentRatePct = int(float64(out.ReassignedTasks) * 100.0 / float64(out.TotalTasks))
	}

	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM queue_reports`).Scan(&out.TotalQueueReports); err != nil {
		return nil, err
	}
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM queue_reports WHERE expires_at <= ?`, now.Format(time.RFC3339)).Scan(&out.ExpiredQueueReports); err != nil {
		return nil, err
	}
	if out.TotalQueueReports > 0 {
		out.ExpiredSignalRatioPct = int(float64(out.ExpiredQueueReports) * 100.0 / float64(out.TotalQueueReports))
	}

	rows, err := r.DB.QueryContext(ctx, `
		SELECT strftime('%s', e.created_at) - strftime('%s', t.requested_at) AS sec
		FROM task_events e
		JOIN tasks t ON t.id = e.task_id
		WHERE e.to_status = 'accepted'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []int
	for rows.Next() {
		var sec int
		if err := rows.Scan(&sec); err != nil {
			return nil, err
		}
		if sec >= 0 {
			values = append(values, sec)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(values) > 0 {
		sort.Ints(values)
		v := values[len(values)/2]
		out.AcceptP50Seconds = &v
	}
	return out, nil
}
