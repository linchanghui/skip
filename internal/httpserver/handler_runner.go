package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"skip/internal/domain"
	"skip/internal/service"
)

func (s *Server) runnerService() *service.RunnerService {
	return &service.RunnerService{Repo: s.Repo}
}

func (s *Server) handleRunnerApply(w http.ResponseWriter, r *http.Request) {
	var in domain.RunnerApplyInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}
	runner, err := s.runnerService().Apply(r.Context(), in)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, runner)
}

func (s *Server) handleRunnerAvailability(w http.ResponseWriter, r *http.Request) {
	runnerID := strings.TrimSpace(chi.URLParam(r, "id"))
	var in domain.RunnerAvailabilityInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}
	availability, err := s.runnerService().SetAvailability(r.Context(), runnerID, in)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, availability)
}
