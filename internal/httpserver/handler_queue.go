package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"skip/internal/domain"
	"skip/internal/service"
)

func (s *Server) queueService() *service.QueueService {
	return &service.QueueService{Repo: s.Repo}
}

func (s *Server) handleQueueReportCreate(w http.ResponseWriter, r *http.Request) {
	storeID := strings.TrimSpace(chi.URLParam(r, "id"))
	var in domain.CreateQueueReportInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}
	report, err := s.queueService().Report(r.Context(), storeID, in)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, report)
}

func (s *Server) handleQueueSignalGet(w http.ResponseWriter, r *http.Request) {
	storeID := strings.TrimSpace(chi.URLParam(r, "id"))
	signal, err := s.queueService().GetSignal(r.Context(), storeID)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, signal)
}

func (s *Server) handleOpsHideQueueReport(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, "bad_request", "id must be a positive integer")
		return
	}
	report, err := s.queueService().HideReport(r.Context(), id)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}
