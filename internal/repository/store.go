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
	var n int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM stores`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stores := []struct {
		id, name, category, terminal, floor string
		lat, lng                             float64
		areaID                               string
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
	return tx.Commit()
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
			terminal, floor, extRef sql.NullString
			lat, lng                  float64
			lBusy                     sql.NullString
			lQL, lWM                  sql.NullInt64
			lSrc                      sql.NullString
			lAsOf                     sql.NullString
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
		name, category, area string
		terminal, floor, extRef sql.NullString
		lat, lng float64
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
		ID:              newID,
		StoreID:         storeID,
		BusyLevel:       in.BusyLevel,
		QueueLength:     in.QueueLength,
		WaitMinutesEst:  in.WaitMinutesEst,
		Source:          in.Source,
		ReporterID:      reporterID,
		ReportedAt:      t.UTC(),
		Note:            in.Note,
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

