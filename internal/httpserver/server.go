package httpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"skip/internal/domain"
	"skip/internal/repository"
)

type Server struct {
	Log      *slog.Logger
	Repo     *repository.Store
	AdminKey string
	// If StaticDir is set, unmatched GET/HEAD routes fall back to static files or index.html (SPA).
	StaticDir string
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(cors)

	r.Get("/healthz", s.handleHealthz)
	r.Get("/v1/areas/changi", s.handleAreaChangi)
	r.Get("/v1/stores", s.handleListStores)
	r.Get("/v1/stores/{id}", s.handleGetStore)
	r.Post("/v1/stores/{id}/status-reports", s.handlePostStatus)
	r.Post("/v1/tasks", s.handleCreateTask)
	r.Get("/v1/tasks", s.handleListTasks)
	r.Get("/v1/tasks/{id}", s.handleGetTask)
	r.Post("/v1/tasks/{id}/cancel", s.handleCancelTask)
	r.Post("/v1/runners/apply", s.handleRunnerApply)
	r.Post("/v1/runners/{id}/availability", s.handleRunnerAvailability)
	r.Post("/v1/tasks/{id}/accept", s.handleTaskAccept)
	r.Post("/v1/tasks/{id}/arrive", s.handleTaskArrive)
	r.Post("/v1/tasks/{id}/complete", s.handleTaskComplete)
	r.Post("/v1/tasks/{id}/proofs", s.handleTaskProofCreate)
	r.Post("/v1/stores/{id}/queue-reports", s.handleQueueReportCreate)
	r.Get("/v1/stores/{id}/queue-signal", s.handleQueueSignalGet)
	r.Get("/v1/metrics/summary", s.handleMetricsSummary)
	r.Post("/v1/ops/tasks/{id}/assign", s.handleOpsAssignTask)
	r.Post("/v1/ops/queue-reports/{id}/hide", s.handleOpsHideQueueReport)

	if strings.TrimSpace(s.StaticDir) != "" {
		dir := strings.TrimSpace(s.StaticDir)
		r.NotFound(s.staticOrSPA(dir))
	}

	return r
}

func (s *Server) staticOrSPA(root string) http.HandlerFunc {
	base, err := filepath.Abs(root)
	if err != nil {
		base = root
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		rel := strings.TrimPrefix(r.URL.Path, "/")
		if rel == "" {
			http.ServeFile(w, r, filepath.Join(root, "index.html"))
			return
		}
		urlPath := path.Clean("/" + rel)
		if strings.HasPrefix(urlPath, "/..") {
			http.NotFound(w, r)
			return
		}
		trim := strings.TrimPrefix(urlPath, "/")
		candidate := filepath.Join(root, filepath.FromSlash(trim))
		real, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			real = candidate
		}
		absBase, err := filepath.EvalSymlinks(base)
		if err != nil {
			absBase = base
		}
		realAbs, err := filepath.Abs(real)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		absBaseAbs, err := filepath.Abs(absBase)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if !strings.HasPrefix(realAbs, absBaseAbs+string(os.PathSeparator)) && realAbs != absBaseAbs {
			http.NotFound(w, r)
			return
		}
		if fi, err := os.Stat(realAbs); err == nil && !fi.IsDir() {
			http.ServeFile(w, r, realAbs)
			return
		}
		http.ServeFile(w, r, filepath.Join(root, "index.html"))
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Admin-Key")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleAreaChangi(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, repository.AreaChangi())
}

func (s *Server) handleListStores(w http.ResponseWriter, r *http.Request) {
	area := strings.TrimSpace(r.URL.Query().Get("area_id"))
	if area == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "area_id is required")
		return
	}
	stores, err := s.Repo.ListByArea(r.Context(), area)
	if err != nil {
		s.Log.Error("list stores", "err", err)
		writeErr(w, http.StatusInternalServerError, "internal", "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stores": stores})
}

func (s *Server) handleGetStore(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	limit := 20
	if v := strings.TrimSpace(r.URL.Query().Get("history_limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 100 {
			writeErr(w, http.StatusBadRequest, "bad_request", "history_limit must be 0-100")
			return
		}
		limit = n
	}
	detail, err := s.Repo.GetWithHistory(r.Context(), id, limit)
	if errors.Is(err, repository.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not_found", "store not found")
		return
	}
	if err != nil {
		s.Log.Error("get store", "err", err)
		writeErr(w, http.StatusInternalServerError, "internal", "database error")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handlePostStatus(w http.ResponseWriter, r *http.Request) {
	if s.AdminKey == "" {
		writeErr(w, http.StatusServiceUnavailable, "admin_not_configured", "SKIP_ADMIN_API_KEY is not set; refusing writes")
		return
	}
	key := strings.TrimSpace(r.Header.Get("X-Admin-Key"))
	if key != s.AdminKey {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "invalid or missing X-Admin-Key")
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "id is required")
		return
	}
	var body domain.StatusReportInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}
	if _, err := domain.ParseBusyLevel(string(body.BusyLevel)); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid busy_level")
		return
	}
	if _, err := domain.ParseStatusSource(string(body.Source)); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid source")
		return
	}
	rep, err := s.Repo.InsertStatusReport(r.Context(), id, body, nil)
	if errors.Is(err, repository.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not_found", "store not found")
		return
	}
	if err != nil {
		s.Log.Error("insert status", "err", err)
		writeErr(w, http.StatusInternalServerError, "internal", "database error")
		return
	}
	writeJSON(w, http.StatusCreated, rep)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": msg,
		},
	})
}
