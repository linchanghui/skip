package httpserver

import (
	"errors"
	"net/http"
	"strings"

	"skip/internal/repository"
	"skip/internal/service"
)

func writeServiceErr(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	var ve service.ValidationError
	if errors.As(err, &ve) {
		writeErr(w, 400, "bad_request", ve.Message)
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		writeErr(w, 404, "not_found", "resource not found")
		return
	}
	if strings.Contains(strings.ToLower(err.Error()), "invalid transition") {
		writeErr(w, 409, "conflict", err.Error())
		return
	}
	writeErr(w, 500, "internal", "internal error")
}
