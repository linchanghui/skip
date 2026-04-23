package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"skip/internal/domain"
)

func (r *Store) CreateTaskProof(ctx context.Context, taskID string, in domain.CreateTaskProofInput, now time.Time) (*domain.TaskProof, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var exists int
	if err := r.DB.QueryRowContext(ctx, `SELECT 1 FROM tasks WHERE id = ?`, taskID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO task_proofs (task_id, runner_id, proof_type, media_url, note, captured_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		taskID, in.RunnerID, in.ProofType, in.MediaURL, in.Note, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, task_id, runner_id, proof_type, media_url, note, captured_at, created_at
		FROM task_proofs
		WHERE id = ?`, id,
	)
	var out domain.TaskProof
	var runnerID, mediaURL, note sql.NullString
	var capturedAt, createdAt string
	if err := row.Scan(&out.ID, &out.TaskID, &runnerID, &out.ProofType, &mediaURL, &note, &capturedAt, &createdAt); err != nil {
		return nil, err
	}
	out.RunnerID = nullStringPtr(runnerID)
	out.MediaURL = nullStringPtr(mediaURL)
	out.Note = nullStringPtr(note)
	out.CapturedAt = parseDBTime(capturedAt)
	out.CreatedAt = parseDBTime(createdAt)
	return &out, nil
}
