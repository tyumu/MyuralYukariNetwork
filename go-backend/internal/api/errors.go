package api

import (
	"context"
	"errors"
	"net"
	"net/http"
)

func classifyGenerationError(err error) (string, int) {
	if errors.Is(err, context.DeadlineExceeded) {
		return "Generation timeout", http.StatusGatewayTimeout
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "Generation timeout", http.StatusGatewayTimeout
	}

	return "LLM service unavailable", http.StatusBadGateway
}
