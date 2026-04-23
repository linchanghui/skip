package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"skip/internal/domain"
)

func (s *Server) handleOpsAssignTask(w http.ResponseWriter, r *http.Request) {
	taskID := strings.TrimSpace(chi.URLParam(r, "id"))
	var in domain.AssignTaskInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}
	task, err := s.dispatchService().AssignTaskByOps(r.Context(), taskID, in)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}
