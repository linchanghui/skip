package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"skip/internal/db"
	"skip/internal/domain"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := "file::memory:?cache=shared&_pragma=foreign_keys(1)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	ctx := context.Background()
	if err := db.Migrate(ctx, func(ctx context.Context, q string) error {
		_, err := sqlDB.ExecContext(ctx, q)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	repo := &Store{DB: sqlDB}
	if err := repo.SeedDemo(ctx); err != nil {
		t.Fatal(err)
	}
	return sqlDB
}

func TestSeedAndList(t *testing.T) {
	sqlDB := openTestDB(t)
	repo := &Store{DB: sqlDB}
	ctx := context.Background()
	stores, err := repo.ListByArea(ctx, "changi")
	if err != nil {
		t.Fatal(err)
	}
	if len(stores) != 2 {
		t.Fatalf("stores: want 2 got %d", len(stores))
	}
	if stores[0].LatestStatus == nil || stores[1].LatestStatus == nil {
		t.Fatal("expected latest_status")
	}
}

func TestGetWithHistory(t *testing.T) {
	sqlDB := openTestDB(t)
	repo := &Store{DB: sqlDB}
	ctx := context.Background()
	d, err := repo.GetWithHistory(ctx, "sb-jewel", 10)
	if err != nil {
		t.Fatal(err)
	}
	if d.ID != "sb-jewel" || len(d.StatusHistory) < 1 {
		t.Fatalf("detail: %+v", d)
	}
}

func TestInsertStatusReport(t *testing.T) {
	sqlDB := openTestDB(t)
	repo := &Store{DB: sqlDB}
	ctx := context.Background()
	ql := 3
	rep, err := repo.InsertStatusReport(ctx, "sb-t3", domain.StatusReportInput{
		BusyLevel:   domain.BusyQuiet,
		QueueLength: &ql,
		Source:      domain.SourceOperator,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.BusyLevel != domain.BusyQuiet {
		t.Fatalf("level %s", rep.BusyLevel)
	}
	d, err := repo.GetWithHistory(ctx, "sb-t3", 5)
	if err != nil {
		t.Fatal(err)
	}
	if d.LatestStatus == nil || d.LatestStatus.BusyLevel != domain.BusyQuiet {
		t.Fatalf("latest: %+v", d.LatestStatus)
	}
}

func TestTaskAndQueueFlows(t *testing.T) {
	sqlDB := openTestDB(t)
	repo := &Store{DB: sqlDB}
	ctx := context.Background()

	task, err := repo.CreateTask(ctx, domain.CreateTaskInput{
		UserID:   "user-test",
		StoreID:  "sb-jewel",
		TaskType: domain.TaskTypeQueueForMe,
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != domain.TaskStatusMatching {
		t.Fatalf("want matching got %s", task.Status)
	}
	accepted, err := repo.AcceptTask(ctx, task.ID, "runner-alex", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != domain.TaskStatusAccepted {
		t.Fatalf("want accepted got %s", accepted.Status)
	}

	ttl := 20
	report, err := repo.CreateQueueReport(ctx, "sb-jewel", domain.CreateQueueReportInput{
		ReporterType: domain.ReporterRunner,
		ReporterID:   strPtr("runner-alex"),
		BusyLevel:    domain.BusyModerate,
		TTLMinutes:   &ttl,
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if report.ID == 0 {
		t.Fatal("expected queue report id")
	}
	sig, err := repo.GetQueueSignal(ctx, "sb-jewel", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if sig.StoreID != "sb-jewel" {
		t.Fatalf("signal: %+v", sig)
	}
}

func TestQueueReportUpdatesLatestStatusForMap(t *testing.T) {
	sqlDB := openTestDB(t)
	repo := &Store{DB: sqlDB}
	ctx := context.Background()

	ql := 3
	wm := 100
	ttl := 30
	_, err := repo.CreateQueueReport(ctx, "sb-t3", domain.CreateQueueReportInput{
		ReporterType:   domain.ReporterRunner,
		ReporterID:     strPtr("runner-alex"),
		QueueLength:    &ql,
		WaitMinutesEst: &wm,
		BusyLevel:      domain.BusyModerate,
		TTLMinutes:     &ttl,
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	stores, err := repo.ListByArea(ctx, "changi")
	if err != nil {
		t.Fatal(err)
	}
	var t3 *domain.Store
	for i := range stores {
		if stores[i].ID == "sb-t3" {
			t3 = &stores[i]
			break
		}
	}
	if t3 == nil || t3.LatestStatus == nil {
		t.Fatalf("expected latest status for sb-t3, got %+v", t3)
	}
	if t3.LatestStatus.BusyLevel != domain.BusyModerate {
		t.Fatalf("want moderate got %s", t3.LatestStatus.BusyLevel)
	}
	if t3.LatestStatus.WaitMinutesEst == nil || *t3.LatestStatus.WaitMinutesEst != 100 {
		t.Fatalf("unexpected wait: %+v", t3.LatestStatus.WaitMinutesEst)
	}
}

func TestMetricsSummary(t *testing.T) {
	sqlDB := openTestDB(t)
	repo := &Store{DB: sqlDB}
	ctx := context.Background()
	sum, err := repo.GetMetricsSummary(ctx, time.Date(2026, 4, 23, 6, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if sum.TotalTasks < 1 {
		t.Fatalf("expected seeded tasks, got %+v", sum)
	}
	if sum.TotalQueueReports < 1 {
		t.Fatalf("expected seeded queue reports, got %+v", sum)
	}
}

func TestListTasksFilters(t *testing.T) {
	sqlDB := openTestDB(t)
	repo := &Store{DB: sqlDB}
	ctx := context.Background()

	matching, err := repo.ListTasks(ctx, []domain.TaskStatus{domain.TaskStatusMatching}, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(matching) == 0 {
		t.Fatal("expected matching tasks from seed")
	}
	for _, task := range matching {
		if task.Status != domain.TaskStatusMatching {
			t.Fatalf("unexpected status in filtered result: %s", task.Status)
		}
	}

	myActive, err := repo.ListTasks(ctx, []domain.TaskStatus{domain.TaskStatusAccepted}, "runner-bao", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(myActive) == 0 {
		t.Fatal("expected runner-bao accepted task from seed")
	}
	for _, task := range myActive {
		if task.AcceptedRunnerID == nil || *task.AcceptedRunnerID != "runner-bao" {
			t.Fatalf("unexpected runner in filtered result: %+v", task)
		}
	}
}
