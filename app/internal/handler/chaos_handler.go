package handler

import (
	"net/http"
	"os"
	"time"
)

type chaosHandler struct{}

func (h *chaosHandler) Chaos(w http.ResponseWriter, r *http.Request) {
	// Write response and flush before killing the process
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"chaos initiated"}`))

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Kill the process after 50ms to ensure response delivery
	time.AfterFunc(50*time.Millisecond, func() {
		os.Exit(1)
	})
}
