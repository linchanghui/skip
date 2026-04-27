package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"skip/internal/domain"
)

func (r *Store) CreateTask(ctx context.Context, in domain.CreateTaskInput, now time.Time) (*domain.Task, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	taskID := fmt.Sprintf("task-%d", now.UnixNano())
	slaAcceptBy := now.Add(5 * time.Minute)

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM stores WHERE id = ?`, in.StoreID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO tasks
			(id, user_id, store_id, task_type, status, requested_at, sla_accept_by)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		taskID, in.UserID, in.StoreID, string(in.TaskType), string(domain.TaskStatusCreated), now.Format(time.RFC3339), slaAcceptBy.Format(time.RFC3339),
	); err != nil {
		return nil, err
	}

	payload := map[string]any{}
	if in.Note != nil && strings.TrimSpace(*in.Note) != "" {
		payload["note"] = strings.TrimSpace(*in.Note)
	}
	if err := insertTaskEventTx(ctx, tx, taskID, nil, domain.TaskStatusCreated, "user", &in.UserID, payload, now); err != nil {
		return nil, err
	}
	if err := insertTaskEventTx(ctx, tx, taskID, ptrTaskStatus(domain.TaskStatusCreated), domain.TaskStatusMatching, "system", strPtr("dispatch"), map[string]any{"attempt_no": 1}, now.Add(time.Second)); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO task_attempts (task_id, attempt_no, strategy, result, started_at)
		VALUES (?, 1, 'auto_batch', 'pending', ?)`,
		taskID, now.Add(time.Second).Format(time.RFC3339),
	); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`,
		string(domain.TaskStatusMatching), now.Add(time.Second).Format(time.RFC3339), taskID,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	detail, err := r.GetTask(ctx, taskID, 50)
	if err != nil {
		return nil, err
	}
	return &detail.Task, nil
}

func (r *Store) GetTask(ctx context.Context, taskID string, eventLimit int) (*domain.TaskDetail, error) {
	if eventLimit <= 0 {
		eventLimit = 20
	}
	if eventLimit > 200 {
		eventLimit = 200
	}
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, user_id, store_id, task_type, status, accepted_runner_id, quoted_price_cents,
		       requested_at, sla_accept_by, sla_arrive_by, fail_reason, cancelled_by, created_at, updated_at
		FROM tasks
		WHERE id = ?`, taskID,
	)
	var (
		t                            domain.Task
		taskType, status             string
		acceptedRunnerID, failReason sql.NullString
		cancelledBy, requestedAt     sql.NullString
		slaAcceptBy, slaArriveBy     sql.NullString
		createdAt, updatedAt         sql.NullString
		quotedPrice                  sql.NullInt64
	)
	if err := row.Scan(&t.ID, &t.UserID, &t.StoreID, &taskType, &status, &acceptedRunnerID, &quotedPrice,
		&requestedAt, &slaAcceptBy, &slaArriveBy, &failReason, &cancelledBy, &createdAt, &updatedAt); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	tt, _ := domain.ParseTaskType(taskType)
	ts, _ := domain.ParseTaskStatus(status)
	t.TaskType = tt
	t.Status = ts
	t.AcceptedRunnerID = nullStringPtr(acceptedRunnerID)
	t.QuotedPriceCents = nullIntPtr(quotedPrice)
	t.FailReason = nullStringPtr(failReason)
	t.CancelledBy = nullStringPtr(cancelledBy)
	if requestedAt.Valid {
		t.RequestedAt = parseDBTime(requestedAt.String)
	}
	if slaAcceptBy.Valid {
		t.SLAAcceptBy = parseDBTime(slaAcceptBy.String)
	}
	if slaArriveBy.Valid {
		v := parseDBTime(slaArriveBy.String)
		t.SLAArriveBy = &v
	}
	if createdAt.Valid {
		t.CreatedAt = parseDBTime(createdAt.String)
	}
	if updatedAt.Valid {
		t.UpdatedAt = parseDBTime(updatedAt.String)
	}

	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, task_id, from_status, to_status, actor_type, actor_id, payload, created_at
		FROM task_events
		WHERE task_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?`, taskID, eventLimit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []domain.TaskEvent
	for rows.Next() {
		var e domain.TaskEvent
		var toStatus, actorType, createdAtStr string
		var fromStatus sql.NullString
		var actorID, payload sql.NullString
		if err := rows.Scan(&e.ID, &e.TaskID, &fromStatus, &toStatus, &actorType, &actorID, &payload, &createdAtStr); err != nil {
			return nil, err
		}
		if fromStatus.Valid {
			if fs, err := domain.ParseTaskStatus(fromStatus.String); err == nil {
				e.FromStatus = &fs
			}
		}
		if ts, err := domain.ParseTaskStatus(toStatus); err == nil {
			e.ToStatus = ts
		} else {
			e.ToStatus = domain.TaskStatus(toStatus)
		}
		e.ActorType = actorType
		e.ActorID = nullStringPtr(actorID)
		e.Payload = decodeJSONMap(payload)
		e.CreatedAt = parseDBTime(createdAtStr)
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &domain.TaskDetail{Task: t, Events: events}, nil
}

func (r *Store) ListTasks(ctx context.Context, statuses []domain.TaskStatus, runnerID string, limit int) ([]domain.Task, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	query := `
		SELECT id, user_id, store_id, task_type, status, accepted_runner_id, quoted_price_cents,
		       requested_at, sla_accept_by, sla_arrive_by, fail_reason, cancelled_by, created_at, updated_at
		FROM tasks`
	args := make([]any, 0, len(statuses)+2)
	conds := make([]string, 0, 2)

	if len(statuses) > 0 {
		holders := make([]string, 0, len(statuses))
		for _, st := range statuses {
			holders = append(holders, "?")
			args = append(args, string(st))
		}
		conds = append(conds, "status IN ("+strings.Join(holders, ",")+")")
	}
	if strings.TrimSpace(runnerID) != "" {
		conds = append(conds, "accepted_runner_id = ?")
		args = append(args, strings.TrimSpace(runnerID))
	}
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += " ORDER BY requested_at DESC, id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Task, 0, limit)
	for rows.Next() {
		var (
			t                            domain.Task
			taskType, status             string
			acceptedRunnerID, failReason sql.NullString
			cancelledBy, requestedAt     sql.NullString
			slaAcceptBy, slaArriveBy     sql.NullString
			createdAt, updatedAt         sql.NullString
			quotedPrice                  sql.NullInt64
		)
		if err := rows.Scan(&t.ID, &t.UserID, &t.StoreID, &taskType, &status, &acceptedRunnerID, &quotedPrice,
			&requestedAt, &slaAcceptBy, &slaArriveBy, &failReason, &cancelledBy, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		tt, _ := domain.ParseTaskType(taskType)
		ts, _ := domain.ParseTaskStatus(status)
		t.TaskType = tt
		t.Status = ts
		t.AcceptedRunnerID = nullStringPtr(acceptedRunnerID)
		t.QuotedPriceCents = nullIntPtr(quotedPrice)
		t.FailReason = nullStringPtr(failReason)
		t.CancelledBy = nullStringPtr(cancelledBy)
		if requestedAt.Valid {
			t.RequestedAt = parseDBTime(requestedAt.String)
		}
		if slaAcceptBy.Valid {
			t.SLAAcceptBy = parseDBTime(slaAcceptBy.String)
		}
		if slaArriveBy.Valid {
			v := parseDBTime(slaArriveBy.String)
			t.SLAArriveBy = &v
		}
		if createdAt.Valid {
			t.CreatedAt = parseDBTime(createdAt.String)
		}
		if updatedAt.Valid {
			t.UpdatedAt = parseDBTime(updatedAt.String)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Store) CancelTask(ctx context.Context, taskID, userID string, now time.Time) (*domain.Task, error) {
	return r.transitionTask(ctx, taskID, []domain.TaskStatus{
		domain.TaskStatusCreated, domain.TaskStatusMatching,
	}, domain.TaskStatusCancelled, "user", &userID, nil, nil, &userID, now, nil)
}

func (r *Store) AcceptTask(ctx context.Context, taskID, runnerID string, now time.Time) (*domain.Task, error) {
	slaArriveBy := now.Add(15 * time.Minute)
	return r.transitionTask(ctx, taskID, []domain.TaskStatus{domain.TaskStatusMatching}, domain.TaskStatusAccepted, "runner", &runnerID, &runnerID, &slaArriveBy, nil, now, map[string]any{"eta_min": 15})
}

func (r *Store) ArriveTask(ctx context.Context, taskID, runnerID string, now time.Time, note *string) (*domain.Task, error) {
	payload := map[string]any{}
	if note != nil && strings.TrimSpace(*note) != "" {
		payload["note"] = strings.TrimSpace(*note)
	}
	return r.transitionTask(ctx, taskID, []domain.TaskStatus{domain.TaskStatusAccepted}, domain.TaskStatusArrived, "runner", &runnerID, nil, nil, nil, now, payload)
}

func (r *Store) CompleteTask(ctx context.Context, taskID, runnerID string, now time.Time, note *string) (*domain.Task, error) {
	payload := map[string]any{}
	if note != nil && strings.TrimSpace(*note) != "" {
		payload["note"] = strings.TrimSpace(*note)
	}
	return r.transitionTask(ctx, taskID, []domain.TaskStatus{domain.TaskStatusArrived, domain.TaskStatusQueuing}, domain.TaskStatusCompleted, "runner", &runnerID, nil, nil, nil, now, payload)
}

func (r *Store) AssignTaskByOps(ctx context.Context, taskID, runnerID, opsID string, now time.Time) (*domain.Task, error) {
	slaArriveBy := now.Add(15 * time.Minute)
	return r.transitionTask(ctx, taskID, []domain.TaskStatus{domain.TaskStatusMatching}, domain.TaskStatusAccepted, "ops", &opsID, &runnerID, &slaArriveBy, nil, now, map[string]any{"manual_assign": true})
}

func (r *Store) transitionTask(
	ctx context.Context,
	taskID string,
	from []domain.TaskStatus,
	to domain.TaskStatus,
	actorType string,
	actorID *string,
	acceptedRunnerID *string,
	slaArriveBy *time.Time,
	cancelledBy *string,
	now time.Time,
	payload map[string]any,
) (*domain.Task, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var oldStatusStr string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, taskID).Scan(&oldStatusStr); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	oldStatus, _ := domain.ParseTaskStatus(oldStatusStr)
	allowed := false
	for _, s := range from {
		if s == oldStatus {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("invalid transition: %s -> %s", oldStatus, to)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE tasks
		SET status = ?,
		    accepted_runner_id = COALESCE(?, accepted_runner_id),
		    sla_arrive_by = COALESCE(?, sla_arrive_by),
		    cancelled_by = COALESCE(?, cancelled_by),
		    updated_at = ?
		WHERE id = ?`,
		string(to), acceptedRunnerID, nullableTimeString(slaArriveBy), cancelledBy, now.Format(time.RFC3339), taskID,
	); err != nil {
		return nil, err
	}

	if err := insertTaskEventTx(ctx, tx, taskID, &oldStatus, to, actorType, actorID, payload, now); err != nil {
		return nil, err
	}
	if to == domain.TaskStatusAccepted {
		if _, err := tx.ExecContext(ctx, `
			UPDATE task_attempts SET result='accepted', selected_runner_id=?, ended_at=?
			WHERE task_id=? AND result='pending'`,
			acceptedRunnerID, now.Format(time.RFC3339), taskID,
		); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	d, err := r.GetTask(ctx, taskID, 1)
	if err != nil {
		return nil, err
	}
	return &d.Task, nil
}

func insertTaskEventTx(
	ctx context.Context,
	tx *sql.Tx,
	taskID string,
	from *domain.TaskStatus,
	to domain.TaskStatus,
	actorType string,
	actorID *string,
	payload map[string]any,
	at time.Time,
) error {
	var fromStr *string
	if from != nil {
		s := string(*from)
		fromStr = &s
	}
	var payloadStr *string
	if payload != nil {
		b, _ := json.Marshal(payload)
		s := string(b)
		payloadStr = &s
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO task_events (task_id, from_status, to_status, actor_type, actor_id, payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		taskID, fromStr, string(to), actorType, actorID, payloadStr, at.Format(time.RFC3339),
	)
	return err
}

func nullableTimeString(v *time.Time) *string {
	if v == nil {
		return nil
	}
	s := v.UTC().Format(time.RFC3339)
	return &s
}

func ptrTaskStatus(v domain.TaskStatus) *domain.TaskStatus {
	return &v
}
