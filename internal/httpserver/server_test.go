package httpserver

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"skip/internal/db"
	"skip/internal/repository"
)

func newTestServer(t *testing.T, adminKey string) *httptest.Server {
	t.Helper()
	dsn := "file::memory:?cache=shared&_pragma=foreign_keys(1)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
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
	t.Cleanup(func() { _ = sqlDB.Close() })

	s := &Server{
		Log:      slog.Default(),
		Repo:     repo,
		AdminKey: adminKey,
	}
	return httptest.NewServer(s.Handler())
}

func TestHealthz(t *testing.T) {
	ts := newTestServer(t, "secret")
	defer ts.Close()

	res, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	b, _ := io.ReadAll(res.Body)
	if string(b) != "ok" {
		t.Fatalf("body %q", b)
	}
}

func TestListStores(t *testing.T) {
	ts := newTestServer(t, "secret")
	defer ts.Close()

	res, err := http.Get(ts.URL + "/v1/stores?area_id=changi")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), `"stores"`) || !strings.Contains(string(body), "sb-jewel") {
		t.Fatalf("body %s", body)
	}
}

func TestPostStatusRequiresAdminKeyConfigured(t *testing.T) {
	ts := newTestServer(t, "")
	defer ts.Close()

	res, err := http.Post(ts.URL+"/v1/stores/sb-t3/status-reports", "application/json", strings.NewReader(`{"busy_level":"quiet","source":"operator"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestPostStatusWithKey(t *testing.T) {
	ts := newTestServer(t, "k9")
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/stores/sb-t3/status-reports", strings.NewReader(`{"busy_level":"moderate","queue_length":2,"source":"operator"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Key", "k9")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status %d body %s", res.StatusCode, body)
	}
}

func TestCreateAndGetTask(t *testing.T) {
	ts := newTestServer(t, "secret")
	defer ts.Close()

	res, err := http.Post(ts.URL+"/v1/tasks", "application/json", strings.NewReader(`{"user_id":"u-1","store_id":"sb-jewel","task_type":"queue_for_me"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("create status %d body %s", res.StatusCode, body)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), `"status":"matching"`) || !strings.Contains(string(body), `"id":"task-`) {
		t.Fatalf("create body %s", body)
	}
}

func TestQueueSignal(t *testing.T) {
	ts := newTestServer(t, "secret")
	defer ts.Close()

	res, err := http.Get(ts.URL + "/v1/stores/sb-jewel/queue-signal")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), `"store_id":"sb-jewel"`) {
		t.Fatalf("body %s", body)
	}
}

func TestMetricsSummary(t *testing.T) {
	ts := newTestServer(t, "secret")
	defer ts.Close()

	res, err := http.Get(ts.URL + "/v1/metrics/summary")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), `"total_tasks"`) || !strings.Contains(string(body), `"expired_signal_ratio_pct"`) {
		t.Fatalf("body %s", body)
	}
}
