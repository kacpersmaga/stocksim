package handler

import (
	"net/http"

	"github.com/remitly-task/stocksim/internal/domain"
)

type logHandler struct {
	log LogServicer
}

type getLogResponse struct {
	Entries []domain.LogEntry `json:"entries"`
}

func (h *logHandler) GetLog(w http.ResponseWriter, r *http.Request) {
	entries, err := h.log.GetLog(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if entries == nil {
		entries = []domain.LogEntry{}
	}
	writeJSON(w, http.StatusOK, getLogResponse{Entries: entries})
}
