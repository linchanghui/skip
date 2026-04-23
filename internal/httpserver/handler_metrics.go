package httpserver

import (
	"net/http"
	"time"
)

func (s *Server) handleMetricsSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.Repo.GetMetricsSummary(r.Context(), time.Now().UTC())
	if err != nil {
		s.Log.Error("metrics summary", "err", err)
		writeErr(w, http.StatusInternalServerError, "internal", "database error")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
