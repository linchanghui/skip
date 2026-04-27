package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"skip/internal/db"
	"skip/internal/domain"
	"skip/internal/repository"
)

func openTestRepo(t *testing.T) *repository.Store {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
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
	repo := &repository.Store{DB: sqlDB}
	if err := repo.SeedDemo(ctx); err != nil {
		t.Fatal(err)
	}
	return repo
}

func TestTaskServiceCreateTask(t *testing.T) {
	repo := openTestRepo(t)
	svc := &TaskService{
		Repo: repo,
		Now:  func() time.Time { return time.Date(2026, 4, 23, 8, 0, 0, 0, time.UTC) },
	}
	task, err := svc.CreateTask(context.Background(), domain.CreateTaskInput{
		UserID:   "u1",
		StoreID:  "sb-jewel",
		TaskType: domain.TaskTypeQueueForMe,
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != domain.TaskStatusMatching {
		t.Fatalf("want matching got %s", task.Status)
	}
}

func TestTaskServiceCreateTaskValidation(t *testing.T) {
	repo := openTestRepo(t)
	svc := &TaskService{Repo: repo}
	_, err := svc.CreateTask(context.Background(), domain.CreateTaskInput{
		UserID:   "",
		StoreID:  "sb-jewel",
		TaskType: domain.TaskTypeQueueForMe,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestQueueServiceTTLValidation(t *testing.T) {
	repo := openTestRepo(t)
	svc := &QueueService{Repo: repo}
	ttl := 500
	_, err := svc.Report(context.Background(), "sb-jewel", domain.CreateQueueReportInput{
		ReporterType: domain.ReporterRunner,
		BusyLevel:    domain.BusyModerate,
		TTLMinutes:   &ttl,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestTaskServiceListTasks(t *testing.T) {
	repo := openTestRepo(t)
	svc := &TaskService{Repo: repo}

	tasks, err := svc.ListTasks(context.Background(), []string{"matching"}, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) == 0 {
		t.Fatal("expected matching tasks")
	}
	for _, task := range tasks {
		if task.Status != domain.TaskStatusMatching {
			t.Fatalf("unexpected status: %s", task.Status)
		}
	}
}

func TestTaskServiceListTasksValidation(t *testing.T) {
	repo := openTestRepo(t)
	svc := &TaskService{Repo: repo}
	if _, err := svc.ListTasks(context.Background(), []string{"bad-status"}, "", 20); err == nil {
		t.Fatal("expected invalid status error")
	}
	if _, err := svc.ListTasks(context.Background(), []string{"matching"}, "", 101); err == nil {
		t.Fatal("expected invalid limit error")
	}
}
