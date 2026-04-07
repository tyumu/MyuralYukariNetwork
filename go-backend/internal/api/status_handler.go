package api

import (
	"context"
	"net/http"
	"time"

	"go-backend/shared"
)

// Status returns health status and dev logs.
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), statusRequestTimeout)
	defer cancel()

	services := map[string]string{}

	if healthy, _ := h.memoryClient.Health(ctx); healthy {
		services["memory"] = "ok"
	} else {
		services["memory"] = "error"
	}

	if healthy, _ := h.llmClient.Health(ctx); healthy {
		services["llama_cpp"] = "ok"
	} else {
		services["llama_cpp"] = "error"
	}

	okCount := 0
	for _, s := range services {
		if s == "ok" {
			okCount++
		}
	}

	status := "error"
	if okCount == len(services) {
		status = "healthy"
	} else if okCount > 0 {
		status = "degraded"
	}

	resp := shared.StatusResponse{
		Status:      status,
		Services:    services,
		TimestampMs: time.Now().UnixMilli(),
		Logs:        h.logger.GetDevLogs(20),
	}

	writeJSON(w, resp, http.StatusOK)
}
