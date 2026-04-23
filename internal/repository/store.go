package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"skip/internal/domain"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	DB *sql.DB
}

func (r *Store) SeedDemo(ctx context.Context) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var n int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM stores`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		stores := []struct {
			id, name, category, terminal, floor string
			lat, lng                            float64
			areaID                              string
		}{
			{"sb-jewel", "Starbucks (Jewel Changi Airport)", "coffee", "Jewel", "L1", 1.36137, 103.98915, "changi"},
			{"sb-t3", "Starbucks (Changi Airport Terminal 3)", "coffee", "T3", "Public", 1.35525, 103.98865, "changi"},
		}
		for _, s := range stores {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO stores (id, name, category, terminal, floor, lat, lng, area_id, external_ref)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'seed')`,
				s.id, s.name, s.category, s.terminal, s.floor, s.lat, s.lng, s.areaID,
			); err != nil {
				return err
			}
		}
		reports := []struct {
			storeID string
			level   domain.BusyLevel
			ql, wm  int
			note    string
		}{
			{"sb-jewel", domain.BusyModerate, 6, 12, "seed"},
			{"sb-t3", domain.BusyBusy, 14, 22, "seed"},
		}
		for _, rep := range reports {
			res, err := tx.ExecContext(ctx, `
				INSERT INTO status_reports (store_id, busy_level, queue_length, wait_minutes_est, source, note)
				VALUES (?, ?, ?, ?, ?, ?)`,
				rep.storeID, string(rep.level), rep.ql, rep.wm, string(domain.SourceOperator), rep.note,
			)
			if err != nil {
				return err
			}
			id, err := res.LastInsertId()
			if err != nil {
				return err
			}
			var reportedAt string
			if err := tx.QueryRowContext(ctx, `SELECT reported_at FROM status_reports WHERE id = ?`, id).Scan(&reportedAt); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT OR REPLACE INTO store_status_latest (store_id, busy_level, queue_length, wait_minutes_est, source, as_of)
				VALUES (?, ?, ?, ?, ?, ?)`,
				rep.storeID, string(rep.level), rep.ql, rep.wm, string(domain.SourceOperator), reportedAt,
			); err != nil {
				return err
			}
		}
	}

	// MVP mock dataset: insert fixed rows with INSERT OR IGNORE so startup is idempotent.
	runners := []struct {
		id, name, phone, status, area, agreement string
		score                                    float64
	}{
		{"runner-alex", "Alex Tan", "+6591000001", "active", "changi", "v1", 0.94},
		{"runner-bao", "Bao Lin", "+6591000002", "active", "changi", "v1", 0.88},
		{"runner-chen", "Chen Wei", "+6591000003", "probation", "changi", "v1", 0.72},
	}
	for _, r := range runners {
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO runners
				(id, name, phone, status, service_area, reliability_score, agreement_version)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			r.id, r.name, r.phone, r.status, r.area, r.score, r.agreement,
		); err != nil {
			return err
		}
	}

	availability := []struct {
		runnerID     string
		isOnline     int
		lat, lng     float64
		activeTaskID *string
	}{
		{"runner-alex", 1, 1.36110, 103.98900, nil},
		{"runner-bao", 1, 1.35560, 103.98890, strPtr("task-003")},
		{"runner-chen", 0, 1.35630, 103.98950, nil},
	}
	for _, a := range availability {
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO runner_availability
				(runner_id, is_online, current_lat, current_lng, active_task_id)
			VALUES (?, ?, ?, ?, ?)`,
			a.runnerID, a.isOnline, a.lat, a.lng, a.activeTaskID,
		); err != nil {
			return err
		}
	}

	tasks := []struct {
		id, userID, storeID, status, requestedAt, acceptBy, arriveBy string
		runnerID                                                     *string
		price                                                        int
		failReason, cancelledBy                                      *string
	}{
		{"task-001", "user-001", "sb-jewel", "completed", "2026-04-23T03:10:00Z", "2026-04-23T03:15:00Z", "2026-04-23T03:28:00Z", strPtr("runner-alex"), 780, nil, nil},
		{"task-002", "user-002", "sb-t3", "matching", "2026-04-23T04:40:00Z", "2026-04-23T04:45:00Z", "", nil, 680, nil, nil},
		{"task-003", "user-003", "sb-t3", "accepted", "2026-04-23T05:20:00Z", "2026-04-23T05:25:00Z", "2026-04-23T05:38:00Z", strPtr("runner-bao"), 820, nil, nil},
	}
	for _, t := range tasks {
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO tasks
				(id, user_id, store_id, task_type, status, requested_at, accepted_runner_id, quoted_price_cents, sla_accept_by, sla_arrive_by, fail_reason, cancelled_by)
			VALUES (?, ?, ?, 'queue_for_me', ?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?)`,
			t.id, t.userID, t.storeID, t.status, t.requestedAt, t.runnerID, t.price, t.acceptBy, t.arriveBy, t.failReason, t.cancelledBy,
		); err != nil {
			return err
		}
	}

	attempts := []struct {
		taskID, strategy, candidateIDs, result, startedAt, endedAt string
		attemptNo                                                  int
		selectedRunnerID                                           *string
	}{
		{"task-001", "auto_batch", `["runner-alex","runner-bao"]`, "accepted", "2026-04-23T03:10:10Z", "2026-04-23T03:11:01Z", 1, strPtr("runner-alex")},
		{"task-002", "auto_batch", `["runner-chen","runner-bao"]`, "timeout", "2026-04-23T04:40:10Z", "2026-04-23T04:42:30Z", 1, nil},
		{"task-002", "manual_assign", `["runner-alex"]`, "pending", "2026-04-23T04:42:35Z", "", 2, nil},
		{"task-003", "auto_batch", `["runner-bao","runner-alex"]`, "accepted", "2026-04-23T05:20:20Z", "2026-04-23T05:21:08Z", 1, strPtr("runner-bao")},
	}
	for _, a := range attempts {
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO task_attempts
				(task_id, attempt_no, strategy, candidate_runner_ids, selected_runner_id, result, started_at, ended_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''))`,
			a.taskID, a.attemptNo, a.strategy, a.candidateIDs, a.selectedRunnerID, a.result, a.startedAt, a.endedAt,
		); err != nil {
			return err
		}
	}

	events := []struct {
		taskID, fromStatus, toStatus, actorType, actorID, payload, createdAt string
	}{
		{"task-001", "", "created", "user", "user-001", `{"note":"please queue for me"}`, "2026-04-23T03:10:00Z"},
		{"task-001", "created", "matching", "system", "dispatch", `{"attempt_no":1}`, "2026-04-23T03:10:10Z"},
		{"task-001", "matching", "accepted", "runner", "runner-alex", `{"eta_min":11}`, "2026-04-23T03:11:01Z"},
		{"task-001", "accepted", "arrived", "runner", "runner-alex", `{"at":"Jewel L1"}`, "2026-04-23T03:20:05Z"},
		{"task-001", "arrived", "queuing", "runner", "runner-alex", `{"queue_length":7}`, "2026-04-23T03:21:10Z"},
		{"task-001", "queuing", "completed", "runner", "runner-alex", `{"handoff":"done"}`, "2026-04-23T03:31:40Z"},
		{"task-002", "", "created", "user", "user-002", `{"note":"T3 please"}`, "2026-04-23T04:40:00Z"},
		{"task-002", "created", "matching", "system", "dispatch", `{"attempt_no":1}`, "2026-04-23T04:40:10Z"},
		{"task-003", "", "created", "user", "user-003", `{"note":"urgent"}`, "2026-04-23T05:20:00Z"},
		{"task-003", "created", "matching", "system", "dispatch", `{"attempt_no":1}`, "2026-04-23T05:20:20Z"},
		{"task-003", "matching", "accepted", "runner", "runner-bao", `{"eta_min":8}`, "2026-04-23T05:21:08Z"},
	}
	for _, e := range events {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO task_events
				(task_id, from_status, to_status, actor_type, actor_id, payload, created_at)
			SELECT ?, NULLIF(?, ''), ?, ?, ?, ?, ?
			WHERE NOT EXISTS (
				SELECT 1 FROM task_events
				WHERE task_id = ? AND to_status = ? AND created_at = ?
			)`,
			e.taskID, e.fromStatus, e.toStatus, e.actorType, e.actorID, e.payload, e.createdAt,
			e.taskID, e.toStatus, e.createdAt,
		); err != nil {
			return err
		}
	}

	proofs := []struct {
		taskID, runnerID, proofType, mediaURL, note, capturedAt string
	}{
		{"task-001", "runner-alex", "arrived_photo", "https://example.com/mock/arrived-task-001.jpg", "arrived at entrance", "2026-04-23T03:20:08Z"},
		{"task-001", "runner-alex", "queue_progress_photo", "https://example.com/mock/queue-task-001.jpg", "queue moving", "2026-04-23T03:24:12Z"},
		{"task-001", "runner-alex", "completion_photo", "https://example.com/mock/done-task-001.jpg", "handoff completed", "2026-04-23T03:31:42Z"},
	}
	for _, p := range proofs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO task_proofs
				(task_id, runner_id, proof_type, media_url, note, captured_at)
			SELECT ?, ?, ?, ?, ?, ?
			WHERE NOT EXISTS (
				SELECT 1 FROM task_proofs
				WHERE task_id = ? AND proof_type = ? AND captured_at = ?
			)`,
			p.taskID, p.runnerID, p.proofType, p.mediaURL, p.note, p.capturedAt,
			p.taskID, p.proofType, p.capturedAt,
		); err != nil {
			return err
		}
	}

	queueReports := []struct {
		storeID, reporterType, reporterID, busyLevel, evidenceURL, confidence, reportedAt, expiresAt string
		queueLength, waitMinutes                                                                     *int
	}{
		{"sb-jewel", "runner", "runner-alex", "moderate", "https://example.com/mock/qr-jewel.jpg", "normal", "2026-04-23T05:30:00Z", "2026-04-23T05:55:00Z", intPtr(6), intPtr(11)},
		{"sb-t3", "operator", "ops-001", "busy", "", "low", "2026-04-23T04:00:00Z", "2026-04-23T04:25:00Z", intPtr(12), intPtr(20)},
	}
	for _, q := range queueReports {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO queue_reports
				(store_id, reporter_type, reporter_id, queue_length, wait_minutes_est, busy_level, evidence_url, confidence_flag, reported_at, expires_at)
			SELECT ?, ?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?
			WHERE NOT EXISTS (
				SELECT 1 FROM queue_reports
				WHERE store_id = ? AND reporter_type = ? AND reported_at = ?
			)`,
			q.storeID, q.reporterType, q.reporterID, q.queueLength, q.waitMinutes, q.busyLevel, q.evidenceURL, q.confidence, q.reportedAt, q.expiresAt,
			q.storeID, q.reporterType, q.reportedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func strPtr(v string) *string {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func (r *Store) ListByArea(ctx context.Context, areaID string) ([]domain.Store, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT
			s.id, s.name, s.category, s.terminal, s.floor, s.lat, s.lng, s.area_id, s.external_ref,
			l.busy_level, l.queue_length, l.wait_minutes_est, l.source, l.as_of
		FROM stores s
		LEFT JOIN store_status_latest l ON l.store_id = s.id
		WHERE s.area_id = ?
		ORDER BY s.name`,
		areaID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Store
	for rows.Next() {
		var (
			id, name, category, area string
			terminal, floor, extRef  sql.NullString
			lat, lng                 float64
			lBusy                    sql.NullString
			lQL, lWM                 sql.NullInt64
			lSrc                     sql.NullString
			lAsOf                    sql.NullString
		)
		if err := rows.Scan(&id, &name, &category, &terminal, &floor, &lat, &lng, &area, &extRef,
			&lBusy, &lQL, &lWM, &lSrc, &lAsOf); err != nil {
			return nil, err
		}
		st := domain.Store{
			ID:       id,
			Name:     name,
			AreaID:   area,
			Category: category,
			Location: domain.LatLng{Lat: lat, Lng: lng},
		}
		if terminal.Valid {
			v := terminal.String
			st.Terminal = &v
		}
		if floor.Valid {
			v := floor.String
			st.Floor = &v
		}
		if extRef.Valid {
			v := extRef.String
			st.ExternalRef = &v
		}
		if lBusy.Valid && lSrc.Valid && lAsOf.Valid {
			t, err := time.Parse(time.RFC3339, lAsOf.String)
			if err != nil {
				t, _ = time.Parse("2006-01-02 15:04:05", lAsOf.String)
			}
			bl, _ := domain.ParseBusyLevel(lBusy.String)
			src, _ := domain.ParseStatusSource(lSrc.String)
			ls := domain.LatestStatus{
				BusyLevel: bl,
				AsOf:      t.UTC(),
				Source:    src,
			}
			if lQL.Valid {
				v := int(lQL.Int64)
				ls.QueueLength = &v
			}
			if lWM.Valid {
				v := int(lWM.Int64)
				ls.WaitMinutesEst = &v
			}
			st.LatestStatus = &ls
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

func (r *Store) GetWithHistory(ctx context.Context, id string, historyLimit int) (*domain.StoreDetail, error) {
	if historyLimit <= 0 {
		historyLimit = 20
	}
	if historyLimit > 100 {
		historyLimit = 100
	}

	var (
		name, category, area    string
		terminal, floor, extRef sql.NullString
		lat, lng                float64
	)
	err := r.DB.QueryRowContext(ctx, `
		SELECT name, category, terminal, floor, lat, lng, area_id, external_ref
		FROM stores WHERE id = ?`, id,
	).Scan(&name, &category, &terminal, &floor, &lat, &lng, &area, &extRef)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	detail := &domain.StoreDetail{
		Store: domain.Store{
			ID:       id,
			Name:     name,
			AreaID:   area,
			Category: category,
			Location: domain.LatLng{Lat: lat, Lng: lng},
		},
	}
	if terminal.Valid {
		v := terminal.String
		detail.Terminal = &v
	}
	if floor.Valid {
		v := floor.String
		detail.Floor = &v
	}
	if extRef.Valid {
		v := extRef.String
		detail.ExternalRef = &v
	}

	row := r.DB.QueryRowContext(ctx, `
		SELECT busy_level, queue_length, wait_minutes_est, source, as_of
		FROM store_status_latest WHERE store_id = ?`, id,
	)
	var (
		lBusy, lSrc, lAsOf string
		lQL, lWM           sql.NullInt64
	)
	if err := row.Scan(&lBusy, &lQL, &lWM, &lSrc, &lAsOf); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	} else if err == nil {
		t, err := time.Parse(time.RFC3339, lAsOf)
		if err != nil {
			t, _ = time.Parse("2006-01-02 15:04:05", lAsOf)
		}
		bl, _ := domain.ParseBusyLevel(lBusy)
		src, _ := domain.ParseStatusSource(lSrc)
		ls := domain.LatestStatus{BusyLevel: bl, AsOf: t.UTC(), Source: src}
		if lQL.Valid {
			v := int(lQL.Int64)
			ls.QueueLength = &v
		}
		if lWM.Valid {
			v := int(lWM.Int64)
			ls.WaitMinutesEst = &v
		}
		detail.LatestStatus = &ls
	}

	hrows, err := r.DB.QueryContext(ctx, `
		SELECT id, store_id, busy_level, queue_length, wait_minutes_est, source, reporter_id, reported_at, note
		FROM status_reports WHERE store_id = ? ORDER BY reported_at DESC, id DESC LIMIT ?`,
		id, historyLimit,
	)
	if err != nil {
		return nil, err
	}
	defer hrows.Close()

	for hrows.Next() {
		var rep domain.StatusReport
		var ql, wm sql.NullInt64
		var rid, note sql.NullString
		var reported string
		if err := hrows.Scan(&rep.ID, &rep.StoreID, &rep.BusyLevel, &ql, &wm, &rep.Source, &rid, &reported, &note); err != nil {
			return nil, err
		}
		if ql.Valid {
			v := int(ql.Int64)
			rep.QueueLength = &v
		}
		if wm.Valid {
			v := int(wm.Int64)
			rep.WaitMinutesEst = &v
		}
		if rid.Valid {
			v := rid.String
			rep.ReporterID = &v
		}
		if note.Valid {
			v := note.String
			rep.Note = &v
		}
		t, err := time.Parse(time.RFC3339, reported)
		if err != nil {
			t, _ = time.Parse("2006-01-02 15:04:05", reported)
		}
		rep.ReportedAt = t.UTC()
		detail.StatusHistory = append(detail.StatusHistory, rep)
	}
	return detail, hrows.Err()
}

func (r *Store) InsertStatusReport(ctx context.Context, storeID string, in domain.StatusReportInput, reporterID *string) (*domain.StatusReport, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var one int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM stores WHERE id = ?`, storeID).Scan(&one); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO status_reports (store_id, busy_level, queue_length, wait_minutes_est, source, reporter_id, note)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		storeID, string(in.BusyLevel), in.QueueLength, in.WaitMinutesEst, string(in.Source), reporterID, in.Note,
	)
	if err != nil {
		return nil, err
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	var reportedAt string
	if err := tx.QueryRowContext(ctx, `SELECT reported_at FROM status_reports WHERE id = ?`, newID).Scan(&reportedAt); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT OR REPLACE INTO store_status_latest (store_id, busy_level, queue_length, wait_minutes_est, source, as_of)
		VALUES (?, ?, ?, ?, ?, ?)`,
		storeID, string(in.BusyLevel), in.QueueLength, in.WaitMinutesEst, string(in.Source), reportedAt,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	t, err := time.Parse(time.RFC3339, reportedAt)
	if err != nil {
		t, _ = time.Parse("2006-01-02 15:04:05", reportedAt)
	}
	out := &domain.StatusReport{
		ID:             newID,
		StoreID:        storeID,
		BusyLevel:      in.BusyLevel,
		QueueLength:    in.QueueLength,
		WaitMinutesEst: in.WaitMinutesEst,
		Source:         in.Source,
		ReporterID:     reporterID,
		ReportedAt:     t.UTC(),
		Note:           in.Note,
	}
	return out, nil
}

// AreaChangi returns static demo metadata (no DB).
func AreaChangi() domain.Area {
	return domain.Area{
		ID:   "changi",
		Name: "Singapore Changi Airport (demo)",
		Center: domain.LatLng{
			Lat: 1.3596,
			Lng: 103.9891,
		},
		DefaultZoom: 14,
		BBox: domain.BBox{
			South: 1.342,
			West:  103.975,
			North: 1.375,
			East:  104.005,
		},
	}
}
