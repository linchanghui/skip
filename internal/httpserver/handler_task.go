package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"skip/internal/domain"
	"skip/internal/service"
)

func (s *Server) taskService() *service.TaskService {
	return &service.TaskService{Repo: s.Repo}
}

func (s *Server) dispatchService() *service.DispatchService {
	return &service.DispatchService{Repo: s.Repo}
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}
	task, err := s.taskService().CreateTask(r.Context(), in)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	taskID := strings.TrimSpace(chi.URLParam(r, "id"))
	detail, err := s.taskService().GetTask(r.Context(), taskID)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	taskID := strings.TrimSpace(chi.URLParam(r, "id"))
	var in domain.CancelTaskInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}
	task, err := s.taskService().CancelTask(r.Context(), taskID, in)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleTaskAccept(w http.ResponseWriter, r *http.Request) {
	s.handleRunnerTaskAction(w, r, func(taskID string, in domain.TaskActionByRunnerInput) (*domain.Task, error) {
		return s.dispatchService().AcceptTask(r.Context(), taskID, in)
	})
}

func (s *Server) handleTaskArrive(w http.ResponseWriter, r *http.Request) {
	s.handleRunnerTaskAction(w, r, func(taskID string, in domain.TaskActionByRunnerInput) (*domain.Task, error) {
		return s.dispatchService().ArriveTask(r.Context(), taskID, in)
	})
}

func (s *Server) handleTaskComplete(w http.ResponseWriter, r *http.Request) {
	s.handleRunnerTaskAction(w, r, func(taskID string, in domain.TaskActionByRunnerInput) (*domain.Task, error) {
		return s.dispatchService().CompleteTask(r.Context(), taskID, in)
	})
}

func (s *Server) handleRunnerTaskAction(
	w http.ResponseWriter,
	r *http.Request,
	fn func(taskID string, in domain.TaskActionByRunnerInput) (*domain.Task, error),
) {
	taskID := strings.TrimSpace(chi.URLParam(r, "id"))
	var in domain.TaskActionByRunnerInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}
	task, err := fn(taskID, in)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleTaskProofCreate(w http.ResponseWriter, r *http.Request) {
	taskID := strings.TrimSpace(chi.URLParam(r, "id"))
	var in domain.CreateTaskProofInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}
	proof, err := s.Repo.CreateTaskProof(r.Context(), taskID, in, time.Now().UTC())
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, proof)
}
